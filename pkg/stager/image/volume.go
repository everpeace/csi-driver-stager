package image

import (
	api "github.com/everpeace/csi-driver-stager/pkg/stager/api/image"
	"k8s.io/utils/clock"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/everpeace/csi-driver-stager/pkg/stager/util"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Phase string

const (
	DockerConfigJsonKey = ".dockerconfigjson"

	// publish states
	PhaseInitState         Phase = "InitState"
	PhaseContainerCreated  Phase = "ContainerCreated"
	PhaseContainerMounted  Phase = "ContainerMounted"
	PhaseTargetPathMounted Phase = "TargetPathMounted"
	PhasePublished         Phase = "Published"

	// unpublish states
	PhaseTargetPathUnMounted  Phase = "TargetPathUnMounted"
	PhaseContainerCommitted   Phase = "ContainerCommitted"
	PhaseContainerUnMounted   Phase = "ContainerUnMounted"
	PhaseContainerImagePushed Phase = "ContainerImagePushed"
	PhaseContainerDeleted     Phase = "ContainerDeleted"
	PhaseUnPublished          Phase = "UnPublished"
)

type Volume struct {
	Clock clock.Clock

	// User defined spec
	Spec         api.StagerSpec
	TagGenerator tagGenerator

	// values from PublishVolumeRequest
	ReadOnly         bool
	VolumeID         string
	TargetPath       string
	DockerConfigJson string
	PodInfo          util.PodInfo
	podMeta          metav1.ObjectMeta

	// Status
	Phase           Phase
	ProvisionedRoot string
	ImageToPush     string
}

func NewVolume(req *csi.NodePublishVolumeRequest, clock clock.Clock) (*Volume, error) {
	zlog.Trace().Interface("request", req).Msg("volume.NewVolume called")

	spec, err := api.NewSpec(req.VolumeContext)
	if err != nil {
		return nil, errors.Wrapf(err, "can't load stager spec")
	}
	tagGenerator, err := newTagGenerator(spec.StageOutSpec.TagGenerator)
	if err != nil {
		return nil, errors.Wrapf(err, "can't load stager spec")
	}

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, errors.New("Volume ID not provided")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, errors.New("Staging targetPath not provided")
	}

	var dockerConfigJson string
	if secrets := req.GetSecrets(); len(secrets) > 0 {
		dckrcfgjson, ok := secrets[DockerConfigJsonKey]
		if !ok {
			return nil, errors.Errorf("secret must have key='%s'", DockerConfigJsonKey)
		}
		dockerConfigJson = dckrcfgjson
	}

	podInfo, err := util.NewPodInfo(req.VolumeContext)
	if err != nil {
		return nil, err
	}

	return &Volume{
		Clock:            clock,
		Spec:             *spec,
		TagGenerator:     tagGenerator,
		VolumeID:         volumeID,
		TargetPath:       targetPath,
		DockerConfigJson: dockerConfigJson,
		ReadOnly:         req.GetReadonly(),
		PodInfo:          podInfo,
		Phase:            PhaseInitState,
		podMeta: metav1.ObjectMeta{
			Namespace: podInfo.Namespace,
			Name:      podInfo.Name,
			UID:       podInfo.UID,
		},
	}, nil
}
