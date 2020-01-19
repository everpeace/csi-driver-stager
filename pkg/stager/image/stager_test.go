package image

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	api "github.com/everpeace/csi-driver-stager/pkg/stager/api/image"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/everpeace/csi-driver-stager/pkg/stager/image/buildah"
	"github.com/everpeace/csi-driver-stager/pkg/stager/util"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

var (
	testDriverName = "test.image.stager.csi.k8s.io"
	stager         *Stager
)

func TestImageStagerVolume(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Image Stager Test")
}

var _ = BeforeSuite(func() {
	zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	stager = &Stager{
		Buildah: &buildah.Client{
			DriverName: testDriverName,
			ExecPath:   "buildah",
		},
	}
})

var _ = Describe("Stage-In(StageIn) & Stage-Out(StageOut)", func() {
	var volumeID string
	var targetPath string

	BeforeEach(func() {
		volumeID = uuid.New().String()
		targetPath = filepath.Join("/tmp", "targetpath", volumeID)
		Expect(os.MkdirAll(targetPath, 0777)).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(exec.Command("rm", "-rf", targetPath).Run()).NotTo(HaveOccurred())
		Expect(exec.Command("buildah", "delete", "--all").Run()).NotTo(HaveOccurred())
		Expect(exec.Command("buildah", "rmi", "--all").Run()).NotTo(HaveOccurred())
	})

	Context("Stage-In(StageIn)", func() {
		It("should mount targetPath to pulled container", func() {
			vol, err := NewVolume(&csi.NodePublishVolumeRequest{
				VolumeId:   volumeID,
				TargetPath: targetPath,
				VolumeContext: map[string]string{
					api.StageInImageKey:               "busybox",
					util.PodInfoNamespaceKey:          "test-ns",
					util.PodInfoNameKey:               "test-name",
					util.PodInfoUIDKey:                volumeID,
					util.PodInfoServiceAccountNameKey: "test-sa",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = stager.StageIn(vol)
			Expect(err).NotTo(HaveOccurred())
			Expect(vol.Phase).Should(Equal(PhasePublished))

			output, err := exec.Command("ls", targetPath).Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).Should(Equal("bin\ndev\netc\nhome\nroot\ntmp\nusr\nvar\n"))

			// unmount it for delete the directory safely in AfterEach
			Expect(exec.Command("umount", targetPath).Run()).NotTo(HaveOccurred())
		})

		It("should rollback when error in stage-in", func() {
			vol, err := NewVolume(&csi.NodePublishVolumeRequest{
				VolumeId: volumeID,
				// this causes mount error in stage-in
				TargetPath: filepath.Join("/tmp", "not-existed"),
				VolumeContext: map[string]string{
					api.StageInImageKey:               "busybox",
					util.PodInfoNamespaceKey:          "test-ns",
					util.PodInfoNameKey:               "test-name",
					util.PodInfoUIDKey:                volumeID,
					util.PodInfoServiceAccountNameKey: "test-sa",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = stager.StageIn(vol)
			Expect(err).To(HaveOccurred())
			Expect(vol.Phase).Should(Equal(PhaseContainerMounted))

			err = stager.RollBackStageIn(vol)
			Expect(err).NotTo(HaveOccurred())
			Expect(vol.Phase).Should(Equal(PhaseInitState))
		})
	})
	Context("Stage-Out(StageOut)", func() {
		It("should pushed modified container to image", func() {
			stageOutRepo := "registory:5000/misc/misc"
			vol, err := NewVolume(&csi.NodePublishVolumeRequest{
				VolumeId:   volumeID,
				TargetPath: targetPath,
				VolumeContext: map[string]string{
					api.StageInImageKey:               "busybox",
					api.StageOutImageRepoKey:          stageOutRepo,
					api.StageOutTagGeneratorKey:       "podUid",
					api.StageOutSquashKey:             "false",
					api.StageOutTlsVerifyKey:          "false",
					util.PodInfoNamespaceKey:          "test-ns",
					util.PodInfoNameKey:               "test-name",
					util.PodInfoUIDKey:                volumeID,
					util.PodInfoServiceAccountNameKey: "test-sa",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			// Stage-In first
			err = stager.StageIn(vol)
			Expect(err).NotTo(HaveOccurred())
			Expect(vol.Phase).Should(Equal(PhasePublished))

			// create file in the container root
			addedFileName := "hello"
			Expect(ioutil.WriteFile(
				filepath.Join(targetPath, addedFileName),
				([]byte)(addedFileName),
				0777,
			)).NotTo(HaveOccurred())

			// Stage-out
			err = stager.StageOut(vol)
			Expect(err).NotTo(HaveOccurred())
			Expect(vol.Phase).Should(Equal(PhaseUnPublished))
			expectedImageToPush := fmt.Sprintf("%s:%s", stageOutRepo, volumeID)
			Expect(vol.imageToPush).Should(Equal(expectedImageToPush))

			// after stage-out, targetPath should be empty
			output, err := exec.Command("ls", targetPath).Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).Should(Equal(""))

			// confirm pushed image can be pulled and have 'hello' file.
			name := uuid.New().String()
			Expect(exec.Command(
				"buildah", "from", "--pull-always", "--tls-verify=false",
				fmt.Sprintf("--name=%s", name),
				expectedImageToPush,
			).Run()).NotTo(HaveOccurred())
			output, err = exec.Command(
				"buildah", "run", name, "cat", fmt.Sprintf("/%s", addedFileName),
			).Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).Should(Equal(addedFileName))
		})
	})
})
