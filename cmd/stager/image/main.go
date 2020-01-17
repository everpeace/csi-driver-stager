package main

import (
	"flag"
	"os"
	"time"

	"github.com/everpeace/csi-driver-stager/pkg/stager/image"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

var (
	Version  string
	Revision string

	logpretty = flag.Bool("logpretty", false, "sets pretty logging")
	loglevel  = flag.String("loglevel", "info", "sets log level")
	endpoint  = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	nodeID    = flag.String("nodeid", "", "node id")

	buildahPath     = flag.String("buildahpath", "/bin/buildah", "buildah binary path")
	buildahTimeout  = flag.Duration("buildahtimeout", 10*time.Minute, "timeout to execute buildah command.")
	buildahGcPeriod = flag.Duration("buildahgcperiod", 24*time.Hour, "period for buildah garbage collection")
	masterURL       = flag.String("masterURL", "", "kubernetes master url")
	kubeconfig      = flag.String("kubeconfig", "", "kubeconfig path")
)

func initZeroLog() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *logpretty {
		zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	logLevel, err := zerolog.ParseLevel(*loglevel)
	if err != nil {
		zlog.Error().Str("loglevel", *loglevel).Msg("can't parse loglevel")
		os.Exit(1)
	}
	zerolog.SetGlobalLevel(logLevel)
}

func main() {
	flag.Parse()
	initZeroLog()

	zlog.Info().
		Str("Driver", image.DriverName).
		Str("Version", Version).
		Str("Revision", Revision).
		Str("NodeID", *nodeID).
		Msg("started")

	handle()

	os.Exit(0)
}

func handle() {
	config, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(rest.AddUserAgent(config, "mpi-operator"))
	if err != nil {
		panic(err.Error())
	}

	driver := image.NewDriver(Version, *nodeID, *endpoint, *buildahPath, *buildahTimeout, *buildahGcPeriod, kubeClient)

	if err := driver.Run(); err != nil {
		zlog.Err(err)
		os.Exit(1)
	}
}
