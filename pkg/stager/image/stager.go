package image

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/everpeace/csi-driver-stager/pkg/stager/image/buildah"
	"github.com/everpeace/csi-driver-stager/pkg/stager/util"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

type Stager struct {
	Buildah  *buildah.Client
	GcPeriod time.Duration
	Recorder record.EventRecorder
}

func (stager *Stager) publishEventIfSupported(vol *Volume, reason, message string) {
	if stager.Recorder == nil {
		zlog.Warn().Interface("ObjectMeta", vol.podMeta).Str("Reason", reason).Str("EventMessage", message).Msg("skip publishing event")
		return
	}
	stager.Recorder.Eventf(&corev1.Pod{ObjectMeta: vol.podMeta}, corev1.EventTypeNormal, reason, message)
}

func (stager *Stager) StageIn(vol *Volume) error {
	switch vol.Phase {
	case PhaseInitState:
		isExist, err := stager.Buildah.IsContainerExist(vol.VolumeID)
		if err != nil {
			return errors.Wrapf(err, "check Buildah container(name=%s) existence failed", vol.VolumeID)
		}
		if isExist {
			vol.Phase = PhaseContainerCreated
			return stager.StageIn(vol)
		}

		stager.publishEventIfSupported(vol, "StageInStarted", fmt.Sprintf("volumeID=%s image=%s", vol.VolumeID, vol.Spec.StageInSpec.Image))
		if err := stager.Buildah.From(vol.VolumeID, vol.Spec.StageInSpec.Image, vol.DockerConfigJson, vol.Spec.StageInSpec.TlsVerify); err != nil {
			stager.publishEventIfSupported(vol, "StageInFailed", fmt.Sprintf("volumeID=%s image=%s error=%s", vol.VolumeID, vol.Spec.StageInSpec.Image, err.Error()))
			return errors.Wrapf(err, "can't create Buildah container(name=%s)", vol.VolumeID)
		}
		stager.publishEventIfSupported(vol, "StageInSucceeded", fmt.Sprintf("volumeID=%s image=%s", vol.VolumeID, vol.Spec.StageInSpec.Image))

		vol.Phase = PhaseContainerCreated
		return stager.StageIn(vol)

	case PhaseContainerCreated:
		provisionRoot, err := stager.Buildah.Mount(vol.VolumeID)
		if err != nil {
			return errors.Wrapf(err, "can't mount Buildah container(name=%s)", vol.VolumeID)
		}
		vol.ProvisionedRoot = provisionRoot
		vol.Phase = PhaseContainerMounted
		return stager.StageIn(vol)

	case PhaseContainerMounted:
		options := []string{"bind"}
		if vol.ReadOnly {
			options = append(options, "ro")
		}
		if err := util.MountTargetPath(vol.ProvisionedRoot, vol.TargetPath, options); err != nil {
			return errors.Wrapf(err,
				"can't mount Buildah container(name=%s)'vol provisioned root(=%s) to volume targetPath(=%s)",
				vol.VolumeID, vol.ProvisionedRoot, vol.TargetPath,
			)
		}
		vol.Phase = PhaseTargetPathMounted
		return stager.StageIn(vol)

	case PhaseTargetPathMounted:
		vol.Phase = PhasePublished
		return stager.StageIn(vol)
	case PhasePublished:
		return nil
	default:
		return errors.Errorf("internal error in publishing volume. volumeID=%s, phase=%s", vol.VolumeID, vol.Phase)
	}
}

func (stager *Stager) RollBackStageIn(vol *Volume) error {
	switch vol.Phase {
	case PhaseInitState:
		return nil
	case PhaseContainerCreated:
		if err := stager.Buildah.Delete(vol.VolumeID); err != nil {
			return errors.Wrapf(err, "can't delete Buildah container(name=%s)", vol.VolumeID)
		}
		vol.Phase = PhaseInitState
		return stager.RollBackStageIn(vol)
	case PhaseContainerMounted:
		if err := stager.Buildah.Umount(vol.VolumeID); err != nil {
			return errors.Wrapf(err, "can't umount Buildah container(name=%s)", vol.VolumeID)
		}
		vol.Phase = PhaseContainerCreated
		return stager.RollBackStageIn(vol)
	case PhaseTargetPathMounted:
		if err := util.UnmountTargetPath(vol.TargetPath); err != nil {
			return errors.Wrapf(err, "can't unmount volume(volumeID=%s) targetPath(=%s)", vol.VolumeID, vol.TargetPath)
		}
		vol.Phase = PhaseContainerMounted
		return stager.RollBackStageIn(vol)
	default:
		return errors.Errorf("internal error in rolling back publishing volume. volumeID=%s, phase=%s", vol.VolumeID, vol.Phase)
	}
}

func (stager *Stager) StageOut(vol *Volume) error {
	switch vol.Phase {
	case PhasePublished:
		if err := util.UnmountTargetPath(vol.TargetPath); err != nil {
			return errors.Wrapf(err, "can't unmount volume(volumeID=%s) targetPath(=%s)", vol.VolumeID, vol.TargetPath)
		}
		vol.Phase = PhaseTargetPathUnMounted
		return stager.StageOut(vol)

	case PhaseTargetPathUnMounted:
		if !vol.Spec.StageOutSpec.Enabled {
			vol.Phase = PhaseContainerImagePushed
			return stager.StageOut(vol)
		}
		generatedTag, err := vol.TagGenerator.Generate(vol)
		if err != nil {
			return errors.Wrapf(err, "failed to generate image tag to stage out")
		}
		vol.ImageToPush = fmt.Sprintf("%s:%s", vol.Spec.StageOutSpec.ImageRepository, generatedTag)
		if err := stager.Buildah.Commit(vol.VolumeID, vol.ImageToPush, vol.Spec.StageOutSpec.Squash); err != nil {
			return errors.Wrapf(err, "can't commit Buildah container(name=%s)", vol.VolumeID)
		}
		vol.Phase = PhaseContainerCommitted
		return stager.StageOut(vol)
	case PhaseContainerCommitted:
		if err := stager.Buildah.Umount(vol.VolumeID); err != nil {
			return errors.Wrapf(err, "can't umount Buildah container(name=%s)", vol.VolumeID)
		}
		vol.Phase = PhaseContainerUnMounted
		return stager.StageOut(vol)
	case PhaseContainerUnMounted:
		if err := stager.Buildah.Push(vol.VolumeID, vol.ImageToPush, vol.DockerConfigJson, vol.Spec.StageOutSpec.TlsVerify); err != nil {
			stager.publishEventIfSupported(vol, "StageOutFailed", fmt.Sprintf("volumeID=%s image=%s error=%s", vol.VolumeID, vol.ImageToPush, err.Error()))
			return errors.Wrapf(err, "can't push image(=%s)", vol.ImageToPush)
		}
		stager.publishEventIfSupported(vol, "StageOutSucceeded", fmt.Sprintf("volumeID=%s image=%s", vol.VolumeID, vol.ImageToPush))
		vol.Phase = PhaseContainerImagePushed
		return stager.StageOut(vol)
	case PhaseContainerImagePushed:
		if err := stager.Buildah.Delete(vol.VolumeID); err != nil {
			return errors.Wrapf(err, "can't delete Buildah container(name=%s)", vol.VolumeID)
		}
		vol.Phase = PhaseContainerDeleted
		return stager.StageOut(vol)
	case PhaseContainerDeleted:
		vol.Phase = PhaseUnPublished
		return stager.StageOut(vol)
	case PhaseUnPublished:
		return nil
	default:
		return errors.Errorf("internal error in unPublishVolume. volumeID=%s, volumePhase=%s", vol.VolumeID, vol.Phase)
	}
}

func (stager *Stager) StartGarbageCollection(stop chan struct{}) {
	if stager.GcPeriod == 0 {
		zlog.Info().Msg("builadh garbage collector disabled")
		return
	}
	zlog.Info().Msg("starting builadh garbage collector")
	wait.Until(stager.Buildah.GarbageCollectOnce, stager.GcPeriod, stop)
	zlog.Info().Msg("stopped builadh garbage collector")
}
