package image

import (
	"strconv"

	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

const (
	StageOutImageRepoKey       = "stage-out/repository"
	StageOutTagGeneratorKey    = "stage-out/tagGenerator"
	StageOutTagGeneratorArgKey = "stage-out/tagGeneratorArg"
	StageOutSquashKey          = "stage-out/squash"
	StageOutTlsVerifyKey       = "stage-out/tlsVerify"
)

type StageOutSpec struct {
	Enabled         bool
	Squash          bool
	TlsVerify       bool
	ImageRepository string
	TagGenerator    string
	TagGeneratorArg string
}

func NewStageOutSpec(context map[string]string) (StageOutSpec, error) {
	zlog.Trace().Interface("context", context).Msg("NewStageOutSpec called")

	// prepare defaults
	spec := StageOutSpec{}
	spec.TlsVerify = true
	spec.TagGenerator = "timestamp"

	// read values from context
	imageRepository, ok := context[StageOutImageRepoKey]
	if !ok {
		zlog.Debug().Msgf("StageOut is disabled because '%s' key is not set", StageOutImageRepoKey)
		return spec, nil
	}
	spec.Enabled = true
	spec.ImageRepository = imageRepository

	if tgName, ok := context[StageOutTagGeneratorKey]; ok {
		spec.TagGenerator = tgName
	}

	if tgArg, ok := context[StageOutTagGeneratorArgKey]; ok {
		spec.TagGeneratorArg = tgArg
	}

	if squashStr, ok := context[StageOutSquashKey]; ok {
		squash, err := strconv.ParseBool(squashStr)
		if err != nil {
			return spec, errors.Errorf("%s must be boolean", StageOutSquashKey)
		}
		spec.Squash = squash
	}

	if tlsVerifyStr, ok := context[StageOutTlsVerifyKey]; ok {
		tlsVerify, err := strconv.ParseBool(tlsVerifyStr)
		if err != nil {
			return spec, errors.Errorf("%s must be boolean", StageOutTlsVerifyKey)
		}
		spec.TlsVerify = tlsVerify
	}

	return spec, nil
}
