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

func NewStageInSpec(context map[string]string, defaultStageInImage string) (StageInSpec, error) {
	zlog.Trace().Interface("context", context).Msg("NewStageInSpec called")

	// prepare defaults
	spec := StageInSpec{}
	spec.Image = defaultStageInImage
	spec.TlsVerify = true

	if image, ok := context[StageInImageKey]; ok {
		spec.Image = image
	}

	if tlsVerifyStr, ok := context[StageInTlsVerifyKey]; ok {
		tlsVerify, err := strconv.ParseBool(tlsVerifyStr)
		if err != nil {
			return spec, errors.Errorf("%s must be boolean", StageInTlsVerifyKey)
		}
		spec.TlsVerify = tlsVerify
	}

	return spec, nil
}
