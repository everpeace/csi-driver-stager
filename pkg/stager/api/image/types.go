package image

import (
	zlog "github.com/rs/zerolog/log"
)

type StagerSpec struct {
	StageInSpec  StageInSpec
	StageOutSpec StageOutSpec
}

func NewSpec(context map[string]string, defaultStageInImage string) (*StagerSpec, error) {
	zlog.Trace().Interface("context", context).Msg("NewSpec called")

	stageInSpec, err := NewStageInSpec(context, defaultStageInImage)
	if err != nil {
		return nil, err
	}

	stageOutSpec, err := NewStageOutSpec(context)
	if err != nil {
		return nil, err
	}

	return &StagerSpec{
		StageInSpec:  stageInSpec,
		StageOutSpec: stageOutSpec,
	}, err
}
