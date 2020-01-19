package image

import (
	"bytes"
	"fmt"
	gotemplate "text/template"

	"github.com/pkg/errors"
)

type tagGenerator interface {
	Generate(*Volume) (string, error)
}

func newTagGenerator(name string) (tagGenerator, error) {
	switch name {
	case "fixed":
		return &FuncTagGenerator{fixedTGFunc}, nil
	case "volumeId", "volumdID":
		return &FuncTagGenerator{volumeIdTGFunc}, nil
	case "timestamp":
		return &FuncTagGenerator{timestampTGFunc}, nil
	case "podName":
		return &FuncTagGenerator{podNameTGFunc}, nil
	case "podNamespace":
		return &FuncTagGenerator{podNamespaceTGFunc}, nil
	case "podUid", "podUID":
		return &FuncTagGenerator{podUIDTGFunc}, nil
	case "podServiceAccount":
		return &FuncTagGenerator{podServiceAccountTGFunc}, nil
	case "template":
		return &FuncTagGenerator{templateTGFunc}, nil
	default:
		return nil, errors.Errorf("tag generator=%s doesn't support", name)
	}
}

type FuncTagGenerator struct {
	f func(*Volume) (string, error)
}

func (f FuncTagGenerator) Generate(spec *Volume) (string, error) {
	return f.f(spec)
}

func volumeIdTGFunc(volume *Volume) (string, error) {
	return volume.VolumeID, nil
}

func fixedTGFunc(volume *Volume) (string, error) {
	return volume.Spec.StageOutSpec.TagGeneratorArg, nil
}

func timestampTGFunc(volume *Volume) (string, error) {
	return fmt.Sprintf("%d", volume.Clock.Now().UTC().Unix()), nil
}

func podNameTGFunc(volume *Volume) (string, error) {
	return volume.PodInfo.Name, nil
}

func podNamespaceTGFunc(volume *Volume) (string, error) {
	return volume.PodInfo.Namespace, nil
}

func podUIDTGFunc(volume *Volume) (string, error) {
	return string(volume.PodInfo.UID), nil
}

func podServiceAccountTGFunc(volume *Volume) (string, error) {
	return volume.PodInfo.ServiceAccountName, nil
}

func templateTGFunc(volume *Volume) (string, error) {
	context := map[string]string{
		"timestamp":         fmt.Sprintf("%d", volume.Clock.Now().UTC().Unix()),
		"volumeId":          volume.VolumeID,
		"volumeID":          volume.VolumeID,
		"podNamespace":      volume.PodInfo.Namespace,
		"podName":           volume.PodInfo.Name,
		"podUid":            string(volume.PodInfo.UID),
		"podUID":            string(volume.PodInfo.UID),
		"podServiceAccount": volume.PodInfo.ServiceAccountName,
	}

	tmpl, err := gotemplate.New("templateTGFunc tag generator").Parse(volume.Spec.StageOutSpec.TagGeneratorArg)
	if err != nil {
		return "", errors.Wrap(err, "failed generating image tag")
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, context); err != nil {
		return "", errors.Wrap(err, "failed generating image tag")
	}

	return rendered.String(), nil
}
