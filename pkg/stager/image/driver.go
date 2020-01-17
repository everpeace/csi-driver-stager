package image

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/everpeace/csi-driver-stager/pkg/stager/image/buildah"
	"github.com/everpeace/csi-driver-stager/pkg/stager/image/volume"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	zlog "github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

const (
	DriverName = "image.stager.csi.k8s.io"
)

type Driver struct {
	vendorVesion string
	endpoint     string
	nodeID       string

	srv        *grpc.Server
	kubeClient kubernetes.Interface

	buildah *buildah.Client

	statuses map[string]*volume.Volume
}

func NewDriver(vendorVesion, nodeID, endpoint, buildahPath string, buildahTimeout, garbageCollectPeriod time.Duration, kubeClient kubernetes.Interface) *Driver {
	zlog.Debug().
		Str("Driver", DriverName).
		Str("VendorVersion", vendorVesion).
		Str("NodeID", nodeID).
		Msg("initialing driver")

	return &Driver{
		vendorVesion: vendorVesion,
		endpoint:     endpoint,
		kubeClient:   kubeClient,
		buildah: &buildah.Client{
			DriverName: DriverName,
			ExecPath:   buildahPath,
			Timeout:    buildahTimeout,
		},
	}
}

func (d *Driver) Run() error {
	zlog.Info().
		Str("Driver", DriverName).
		Str("VendorVersion", d.vendorVesion).
		Str("NodeID", d.nodeID).
		Msg("starting driver")

	stop := make(chan struct{})
	go func() { d.buildah.StartGarbageCollection(stop) }()

	scheme, addr, err := csicommon.ParseEndpoint(d.endpoint)
	if err != nil {
		zlog.Err(err)
		os.Exit(1)
	}

	listener, err := net.Listen(scheme, addr)
	if err != nil {
		zlog.Err(err)
		os.Exit(1)
	}

	logErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			zlog.Err(err)
		}
		return resp, err
	}
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logErr),
	}
	d.srv = grpc.NewServer(opts...)

	csi.RegisterIdentityServer(d.srv, d)
	csi.RegisterNodeServer(d.srv, d)

	zlog.Info().
		Str("address", listener.Addr().String()).
		Msg("listening for connections on address")
	return d.srv.Serve(listener)
}
