package image_test

import (
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	api "github.com/everpeace/csi-driver-stager/pkg/stager/api/image"
	"github.com/everpeace/csi-driver-stager/pkg/stager/image"
	"github.com/everpeace/csi-driver-stager/pkg/stager/util"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"path/filepath"
)

var _ = Describe("Taggenerator", func() {
	var volumeID string
	var targetPath string

	createVolume := func(generatorName, generatorArg string) (*image.Volume, error) {
		return image.NewVolume(&csi.NodePublishVolumeRequest{
			VolumeId:   volumeID,
			TargetPath: targetPath,
			VolumeContext: map[string]string{
				api.StageInImageKey:               "busybox",
				api.StageOutImageRepoKey:          "test",
				api.StageOutTagGeneratorKey:       generatorName,
				api.StageOutTagGeneratorArgKey:    generatorArg,
				util.PodInfoNamespaceKey:          "test-ns",
				util.PodInfoNameKey:               "test-name",
				util.PodInfoUIDKey:                volumeID,
				util.PodInfoServiceAccountNameKey: "test-sa",
			},
		}, fakeClock)
	}

	BeforeEach(func() {
		volumeID = uuid.New().String()
		targetPath = filepath.Join("/tmp", "targetpath", volumeID)
	})

	Describe("pod info generartors", func() {
		Describe("'podName'", func() {
			It("returns podName as tag", func() {
				vol, err := createVolume("podName", "")
				Expect(err).NotTo(HaveOccurred())
				tag, err := vol.TagGenerator.Generate(vol)
				Expect(err).NotTo(HaveOccurred())
				Expect(tag).Should(Equal("test-name"))
			})
		})
		Describe("'podNamespace'", func() {
			It("returns pod namespace as tag", func() {
				vol, err := createVolume("podNamespace", "")
				Expect(err).NotTo(HaveOccurred())
				tag, err := vol.TagGenerator.Generate(vol)
				Expect(err).NotTo(HaveOccurred())
				Expect(tag).Should(Equal("test-ns"))
			})
		})
		Describe("'podUid'", func() {
			It("returns pod uid as tag", func() {
				vol, err := createVolume("podUid", "")
				Expect(err).NotTo(HaveOccurred())
				tag, err := vol.TagGenerator.Generate(vol)
				Expect(err).NotTo(HaveOccurred())
				Expect(tag).Should(Equal(vol.VolumeID))
			})
		})
		Describe("'podServiceAccount'", func() {
			It("returns pod sa as tag", func() {
				vol, err := createVolume("podServiceAccount", "")
				Expect(err).NotTo(HaveOccurred())
				tag, err := vol.TagGenerator.Generate(vol)
				Expect(err).NotTo(HaveOccurred())
				Expect(tag).Should(Equal("test-sa"))
			})
		})
	})

	Describe("'timestamp'", func() {
		It("returns current timestamp as tag", func() {
			vol, err := createVolume("timestamp", "")
			Expect(err).NotTo(HaveOccurred())
			tag, err := vol.TagGenerator.Generate(vol)
			Expect(err).NotTo(HaveOccurred())
			Expect(tag).Should(Equal(fmt.Sprintf("%d", fakeNow.UTC().Unix())))
		})
	})

	Describe("'fixed'", func() {
		It("returns fixes string from arg  as tag", func() {
			fixedTag := "my-value"
			vol, err := createVolume("fixed", fixedTag)
			Expect(err).NotTo(HaveOccurred())
			tag, err := vol.TagGenerator.Generate(vol)
			Expect(err).NotTo(HaveOccurred())
			Expect(tag).Should(Equal(fixedTag))
		})
	})

	Describe("'template'", func() {
		It("returns rendered value by arg as tag", func() {
			vol, err := createVolume(
				"template",
				"{{.podNamespace}}-{{.podName}}-{{.podUID}}-{{.podServiceAccount}}-{{.volumeID}}-{{.timestamp}}",
			)
			expectedTag := fmt.Sprintf(
				"test-ns-test-name-%s-test-sa-%s-%d",
				volumeID, volumeID, fakeNow.UTC().Unix(),
			)
			Expect(err).NotTo(HaveOccurred())
			tag, err := vol.TagGenerator.Generate(vol)
			Expect(err).NotTo(HaveOccurred())
			Expect(tag).Should(Equal(expectedTag))
		})
	})
})
