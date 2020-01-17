package volume

import (
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/everpeace/csi-driver-stager/pkg/stager/util"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

const (
	DockerConfigJsonKey     = ".dockerconfigjson"
	StageInImageKey         = "stage-in/image"
	StageInTlsVerifyKey     = "stage-in/tlsverify"
	StageOutEnabledKey      = "stage-out/enabled"
	StageOutImageRepoKey    = "stage-out/repository"
	StageOutTagGeneratorKey = "stage-out/taggenerator"
	StageOutSquashKey       = "stage-out/squash"
	StageOutTlsVerifyKey    = "stage-out/tlsverify"
)

type Spec struct {
	VolumeID         string
	ReadOnly         bool
	TargetPath       string
	DockerConfigJson string
	StageInSpec      StageInSpec
	StageOutSpec     StageOutSpec
	PodInfo          util.PodInfo
}

type StageInSpec struct {
	TlsVerify bool
	Image     string
}

type StageOutSpec struct {
	Enabled         bool
	Squash          bool
	TlsVerify       bool
	ImageRepository string
	TagGenerator    TagGenerator
}

func NewSpec(req *csi.NodePublishVolumeRequest) (*Spec, error) {
	zlog.Trace().Interface("request", req).Msg("volume.NewSpec called")
	spec := &Spec{}

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, errors.New("Volume ID not provided")
	}
	spec.VolumeID = volumeID

	target := req.GetTargetPath()
	if target == "" {
		return nil, errors.New("Staging target not provided")
	}
	spec.TargetPath = target

	secrets := req.GetSecrets()
	if len(secrets) > 0 {
		dockerConfigJson, ok := secrets[DockerConfigJsonKey]
		if !ok {
			return nil, errors.Errorf("secret must have key='%s'", DockerConfigJsonKey)
		}
		spec.DockerConfigJson = dockerConfigJson
	}

	spec.ReadOnly = req.GetReadonly()

	podInfo, err := util.PodInfoFrom(req.VolumeContext)
	if err != nil {
		return nil, err
	}
	spec.PodInfo = podInfo

	stageInConfig, err := NewStageInSpec(req)
	if err != nil {
		return nil, err
	}
	spec.StageInSpec = stageInConfig

	stageOutConfig, err := NewStageOutSpec(req)
	if err != nil {
		return nil, err
	}
	spec.StageOutSpec = stageOutConfig

	return spec, err
}

func NewStageInSpec(req *csi.NodePublishVolumeRequest) (StageInSpec, error) {
	zlog.Trace().Interface("request", req).Msg("volume.NewStageInSpec called")
	spec := StageInSpec{}
	context := req.GetVolumeContext()

	image, ok := context[StageInImageKey]
	if !ok {
		return spec, errors.Errorf("it must specify %s", StageInImageKey)
	}
	spec.Image = image

	tlsVerifyStr, ok := context[StageInTlsVerifyKey]
	if !ok {
		return spec, errors.Errorf("it must specify %s", StageInTlsVerifyKey)
	}
	var err error
	if spec.TlsVerify, err = strconv.ParseBool(tlsVerifyStr); err != nil {
		return spec, errors.Errorf("%s must be boolean", StageInTlsVerifyKey)
	}

	return spec, nil
}

func NewStageOutSpec(req *csi.NodePublishVolumeRequest) (StageOutSpec, error) {
	zlog.Trace().Interface("request", req).Msg("volume.NewStageOutSpec called")
	spec := StageOutSpec{}
	context := req.GetVolumeContext()

	var err error
	enabledStr, ok := context[StageOutEnabledKey]
	if !ok {
		spec.Enabled = false
		return spec, nil
	}
	if spec.Enabled, err = strconv.ParseBool(enabledStr); err != nil {
		return spec, errors.Errorf("%s must be boolean", StageOutEnabledKey)
	}

	if !spec.Enabled {
		return spec, nil
	}

	if spec.ImageRepository, ok = context[StageOutImageRepoKey]; !ok {
		return spec, errors.Errorf("it must specify %s", StageOutImageRepoKey)
	}

	tgName, ok := context[StageOutTagGeneratorKey]
	if !ok {
		return spec, errors.Errorf("it must specify %s", StageOutTagGeneratorKey)
	}
	if spec.TagGenerator, err = newTagGenerator(tgName); err != nil {
		return spec, errors.Wrapf(err, "wrong taggenerator")
	}

	squashStr, ok := context[StageOutSquashKey]
	if !ok {
		return spec, errors.Errorf("it must specify %s", StageOutSquashKey)
	}
	if spec.Squash, err = strconv.ParseBool(squashStr); err != nil {
		return spec, errors.Errorf("%s must be boolean", StageOutSquashKey)
	}

	tlsVerifyStr, ok := context[StageOutTlsVerifyKey]
	if !ok {
		return spec, errors.Errorf("it must specify %s", StageOutTlsVerifyKey)
	}
	if spec.TlsVerify, err = strconv.ParseBool(tlsVerifyStr); err != nil {
		return spec, errors.Errorf("%s must be boolean", StageOutTlsVerifyKey)
	}

	return spec, nil
}
