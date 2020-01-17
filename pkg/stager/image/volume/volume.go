package volume

import (
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/everpeace/csi-driver-stager/pkg/stager/image/buildah"
	"github.com/everpeace/csi-driver-stager/pkg/stager/util"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Phase string

var (
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
	PhaseCnotainerDeleted     Phase = "ContainerDeleted"
	PhaseUnPublished          Phase = "UnPublished"
)

type Volume struct {
	Spec            Spec
	Phase           Phase
	ProvisionedRoot string
	podMeta         metav1.ObjectMeta
	imageToPush     string
}

func New(req *csi.NodePublishVolumeRequest) (*Volume, error) {
	zlog.Trace().Interface("request", req).Msg("volume.New called")
	spec, err := NewSpec(req)
	if err != nil {
		return nil, errors.Wrapf(err, "can't initialize volume")
	}
	return &Volume{
		Spec:  *spec,
		Phase: PhaseInitState,
		podMeta: metav1.ObjectMeta{
			Namespace: spec.PodInfo.Namespace,
			Name:      spec.PodInfo.Name,
			UID:       spec.PodInfo.UID,
		},
	}, nil
}

func (vol *Volume) Publish(buildah *buildah.Client) error {
	switch vol.Phase {
	case PhaseInitState:
		isExist, err := buildah.IsContainerExist(vol.Spec.VolumeID)
		if err != nil {
			return errors.Wrapf(err, "check buildah container(name=%s) existence failed", vol.Spec.VolumeID)
		}
		if isExist {
			vol.Phase = PhaseContainerCreated
			return vol.Publish(buildah)
		}
		if err := buildah.From(vol.Spec.VolumeID, vol.Spec.StageInSpec.Image, vol.Spec.DockerConfigJson, vol.Spec.StageInSpec.TlsVerify); err != nil {
			return errors.Wrapf(err, "can't create buildah container(name=%s)", vol.Spec.VolumeID)
		}
		vol.Phase = PhaseContainerCreated
		return vol.Publish(buildah)

	case PhaseContainerCreated:
		provisionRoot, err := buildah.Mount(vol.Spec.VolumeID)
		if err != nil {
			return errors.Wrapf(err, "can't mount buildah container(name=%s)", vol.Spec.VolumeID)
		}
		vol.ProvisionedRoot = provisionRoot
		vol.Phase = PhaseContainerMounted
		return vol.Publish(buildah)

	case PhaseContainerMounted:
		options := []string{"bind"}
		if vol.Spec.ReadOnly {
			options = append(options, "ro")
		}
		if err := util.MountTargetPath(vol.ProvisionedRoot, vol.Spec.TargetPath, options); err != nil {
			return errors.Wrapf(err,
				"can't mount buildah container(name=%s)'vol provisioned root(=%s) to volume targetPath(=%s)",
				vol.Spec.VolumeID, vol.ProvisionedRoot, vol.Spec.TargetPath,
			)
		}
		vol.Phase = PhaseTargetPathMounted
		return vol.Publish(buildah)

	case PhaseTargetPathMounted:
		vol.Phase = PhasePublished
		return vol.Publish(buildah)
	case PhasePublished:
		return nil
	default:
		return errors.Errorf("internal error in publishing volume. volumeID=%s, phase=%s", vol.Spec.VolumeID, vol.Phase)
	}
}

func (vol *Volume) RollBackPublish(buildah *buildah.Client) error {
	switch vol.Phase {
	case PhaseInitState:
		return nil
	case PhaseContainerCreated:
		if err := buildah.Delete(vol.Spec.VolumeID); err != nil {
			return errors.Wrapf(err, "can't delete buildah container(name=%s)", vol.Spec.VolumeID)
		}
		vol.Phase = PhaseInitState
		return vol.RollBackPublish(buildah)
	case PhaseContainerMounted:
		if err := buildah.Umount(vol.Spec.VolumeID); err != nil {
			return errors.Wrapf(err, "can't umount buildah container(name=%s)", vol.Spec.VolumeID)
		}
		vol.Phase = PhaseContainerCreated
		return vol.RollBackPublish(buildah)
	case PhaseTargetPathMounted:
		if err := util.UnmountTargetPath(vol.Spec.TargetPath); err != nil {
			return errors.Wrapf(err, "can't unmount volume(volumeID=%s) targetPath(=%s)", vol.Spec.VolumeID, vol.Spec.TargetPath)
		}
		vol.Phase = PhaseContainerMounted
		return vol.RollBackPublish(buildah)
	default:
		return errors.Errorf("internal error in rolling back publishing volume. volumeID=%s, phase=%s", vol.Spec.VolumeID, vol.Phase)
	}
}

func (vol *Volume) UnPublish(buildah *buildah.Client) error {
	switch vol.Phase {
	case PhasePublished:
		if err := util.UnmountTargetPath(vol.Spec.TargetPath); err != nil {
			return errors.Wrapf(err, "can't unmount volume(volumeID=%s) targetPath(=%s)", vol.Spec.VolumeID, vol.Spec.TargetPath)
		}
		vol.Phase = PhaseTargetPathUnMounted
		return vol.UnPublish(buildah)

	case PhaseTargetPathUnMounted:
		if !vol.Spec.StageOutSpec.Enabled {
			vol.Phase = PhaseContainerImagePushed
			return vol.UnPublish(buildah)
		}

		generatedTag := vol.Spec.StageOutSpec.TagGenerator.Generate(vol.Spec)
		vol.imageToPush = fmt.Sprintf("%s:%s", vol.Spec.StageOutSpec.ImageRepository, generatedTag)
		if err := buildah.Commit(vol.Spec.VolumeID, vol.imageToPush, vol.Spec.StageOutSpec.Squash); err != nil {
			return errors.Wrapf(err, "can't commit buildah container(name=%s)", vol.Spec.VolumeID)
		}
		vol.Phase = PhaseContainerCommitted
		return vol.UnPublish(buildah)
	case PhaseContainerCommitted:
		if err := buildah.Umount(vol.Spec.VolumeID); err != nil {
			return errors.Wrapf(err, "can't umount buildah container(name=%s)", vol.Spec.VolumeID)
		}
		vol.Phase = PhaseContainerUnMounted
		return vol.UnPublish(buildah)
	case PhaseContainerUnMounted:
		if err := buildah.Push(vol.Spec.VolumeID, vol.imageToPush, vol.Spec.DockerConfigJson, vol.Spec.StageOutSpec.TlsVerify); err != nil {
			return errors.Wrapf(err, "can't push image(=%s)", vol.imageToPush)
		}
		vol.Phase = PhaseContainerImagePushed
		return vol.UnPublish(buildah)
	case PhaseContainerImagePushed:
		if err := buildah.Delete(vol.Spec.VolumeID); err != nil {
			return errors.Wrapf(err, "can't delete buildah container(name=%s)", vol.Spec.VolumeID)
		}
		vol.Phase = PhaseCnotainerDeleted
		return vol.UnPublish(buildah)
	case PhaseCnotainerDeleted:
		vol.Phase = PhaseUnPublished
		return vol.UnPublish(buildah)
	case PhaseUnPublished:
		return nil
	default:
		return errors.Errorf("internal error in unPublishVolume. volumeID=%s, volumePhase=%s", vol.Spec.VolumeID, vol.Phase)
	}
}
