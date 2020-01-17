package volume

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type TagGenerator interface {
	Generate(Spec) string
}

func newSimpleTagGenerator(name string) (TagGenerator, error) {
	switch name {
	case "timestamp":
		return &FuncTagGenerator{timestamp}, nil
	case "podname":
		return &FuncTagGenerator{podName}, nil
	case "podnamespace":
		return &FuncTagGenerator{podNamespace}, nil
	case "poduid":
		return &FuncTagGenerator{podUID}, nil
	default:
		return nil, errors.Errorf("taggenerator=%s doesn't support", name)
	}
}

func newTagGenerator(spec string) (TagGenerator, error) {
	names := strings.Split(spec, "-")
	if len(names) <= 1 {
		return newSimpleTagGenerator(names[0])
	}
	return newCompositeTagGenerator(names[0], names[1:])
}

type FuncTagGenerator struct {
	f func(Spec) string
}

func (f FuncTagGenerator) Generate(spec Spec) string {
	return f.f(spec)
}

type CompositeTagGenerator struct {
	tgs []TagGenerator
}

func (c CompositeTagGenerator) Generate(spec Spec) string {
	tagParts := []string{}
	for _, tg := range c.tgs {
		tagParts = append(tagParts, tg.Generate(spec))
	}
	return strings.Join(tagParts, "-")
}

func newCompositeTagGenerator(name string, names []string) (TagGenerator, error) {
	tg0, err := newSimpleTagGenerator(name)
	if err != nil {
		return nil, err
	}
	tgs := []TagGenerator{tg0}
	for _, n := range names {
		tg, err := newSimpleTagGenerator(n)
		if err != nil {
			return nil, err
		}
		tgs = append(tgs, tg)
	}

	return CompositeTagGenerator{tgs}, nil
}

func timestamp(spec Spec) string {
	return fmt.Sprintf("%d", time.Now().UTC().Unix())
}

func podName(spec Spec) string {
	return spec.PodInfo.Name
}

func podNamespace(spec Spec) string {
	return spec.PodInfo.Namespace
}

func podUID(spec Spec) string {
	return string(spec.PodInfo.UID)
}
