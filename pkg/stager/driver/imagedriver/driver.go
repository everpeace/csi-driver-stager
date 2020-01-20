package imagedriver

import (
	"context"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/record"
	"net"
	"os"
	"time"

	"github.com/everpeace/csi-driver-stager/pkg/stager/image"
	corev1 "k8s.io/api/core/v1"
	clientgokubescheme "k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/clock"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/everpeace/csi-driver-stager/pkg/stager/image/buildah"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	zlog "github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

const (
	DriverName = "image.stager.csi.k8s.io"
)

type Driver struct {
	clock        clock.Clock
	vendorVesion string
	endpoint     string
	nodeID       string

	srv        *grpc.Server
	kubeClient kubernetes.Interface
	recorder   record.EventRecorder

	stager *image.Stager

	defaultStageInImage string
	statuses            map[string]*image.Volume
}

func NewDriver(
	vendorVesion, nodeID, endpoint, defaultStageInImage string,
	buildahPath string, buildahTimeout, buildahGcTimeout, buildahGcPeriod time.Duration,
	kubeClient kubernetes.Interface,
	clock clock.Clock) *Driver {
	zlog.Debug().
		Str("Driver", DriverName).
		Str("VendorVersion", vendorVesion).
		Str("NodeID", nodeID).
		Msg("initialing driver")

	var recorder record.EventRecorder
	if kubeClient != nil {
		eventBroadcaster := record.NewBroadcaster()
		eventBroadcaster.StartLogging(glog.Infof)
		eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
		recorder = eventBroadcaster.NewRecorder(clientgokubescheme.Scheme, corev1.EventSource{Component: DriverName})
	} else {
		zlog.Warn().Msg("the driver won't publish any kubernetes events because it is initialized without kubernetes client")
	}

	return &Driver{
		clock:               clock,
		vendorVesion:        vendorVesion,
		endpoint:            endpoint,
		nodeID:              nodeID,
		kubeClient:          kubeClient,
		recorder:            recorder,
		defaultStageInImage: defaultStageInImage,
		statuses:            map[string]*image.Volume{},
		stager: &image.Stager{
			Buildah: &buildah.Client{
				DriverName: DriverName,
				ExecPath:   buildahPath,
				Timeout:    buildahTimeout,
				GcTimeout:  buildahGcTimeout,
			},
			GcPeriod: buildahGcPeriod,
			Recorder: recorder,
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
	go func() { d.stager.StartGarbageCollection(stop) }()

	scheme, addr, err := csicommon.ParseEndpoint(d.endpoint)
	if err != nil {
		zlog.Error().Err(err).Msg("")
		os.Exit(1)
	}

	listener, err := net.Listen(scheme, addr)
	if err != nil {
		zlog.Error().Err(err).Msg("")
		os.Exit(1)
	}

	logErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			zlog.Error().Err(err).Msg("")
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

func (d *Driver) Shutdown() {
	zlog.Info().
		Str("Driver", DriverName).
		Str("VendorVersion", d.vendorVesion).
		Str("NodeID", d.nodeID).
		Msg("shutting down driver gracefully")
	d.srv.GracefulStop()
}
