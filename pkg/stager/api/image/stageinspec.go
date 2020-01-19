package image

import (
	"strconv"

	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

const (
	StageInImageKey     = "stage-in/image"
	StageInTlsVerifyKey = "stage-in/tlsVerify"
)

type StageInSpec struct {
	TlsVerify bool
	Image     string
}

func NewStageInSpec(context map[string]string) (StageInSpec, error) {
	zlog.Trace().Interface("context", context).Msg("NewStageInSpec called")

	// prepare defaults
	spec := StageInSpec{}
	spec.TlsVerify = true

	image, ok := context[StageInImageKey]
	if !ok {
		return spec, errors.Errorf("it must specify %s", StageInImageKey)
	}
	spec.Image = image

	if tlsVerifyStr, ok := context[StageInTlsVerifyKey]; ok {
		tlsVerify, err := strconv.ParseBool(tlsVerifyStr)
		if err != nil {
			return spec, errors.Errorf("%s must be boolean", StageInTlsVerifyKey)
		}
		spec.TlsVerify = tlsVerify
	}

	return spec, nil
}
