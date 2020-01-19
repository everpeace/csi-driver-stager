module github.com/everpeace/csi-driver-stager

go 1.13

require (
	github.com/container-storage-interface/spec v1.2.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/uuid v1.1.1
	github.com/kubernetes-csi/csi-lib-utils v0.7.0 // indirect
	github.com/kubernetes-csi/csi-test/v3 v3.0.0
	github.com/kubernetes-csi/drivers v1.0.2

	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.17.2
	github.com/spf13/cobra v0.0.5
	google.golang.org/grpc v1.26.0

	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.1-beta.0
	k8s.io/client-go v0.17.0
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
)
