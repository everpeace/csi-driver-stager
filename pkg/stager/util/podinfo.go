package util

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	PodInfoNamespaceKey          = "csi.storage.k8s.io/pod.namespace"
	PodInfoNameKey               = "csi.storage.k8s.io/pod.name"
	PodInfoUIDKey                = "csi.storage.k8s.io/pod.uid"
	PodInfoServiceAccountNameKey = "csi.storage.k8s.io/serviceAccount.name"
)

type PodInfo struct {
	Namespace          string
	Name               string
	ServiceAccountName string
	UID                types.UID
}

func NewPodInfo(context map[string]string) (PodInfo, error) {
	podInfo := PodInfo{}

	podNamespace, ok := context[PodInfoNamespaceKey]
	if !ok {
		return podInfo, errors.New("CSIDriver's spec.podInfoOnMount must be true")
	}
	podInfo.Namespace = podNamespace

	podName, ok := context[PodInfoNameKey]
	if !ok {
		return podInfo, errors.New("CSIDriver's spec.podInfoOnMount must be true")
	}
	podInfo.Name = podName

	podUid, ok := context[PodInfoUIDKey]
	if !ok {
		return podInfo, errors.New("CSIDriver's spec.podInfoOnMount must be true")
	}
	podInfo.UID = types.UID(podUid)

	sa, ok := context[PodInfoServiceAccountNameKey]
	if !ok {
		return podInfo, errors.New("CSIDriver's spec.podInfoOnMount must be true")
	}
	podInfo.ServiceAccountName = sa

	return podInfo, nil
}
