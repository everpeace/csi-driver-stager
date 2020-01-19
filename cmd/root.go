package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	zlog "github.com/rs/zerolog/log"
)

var (
	Version  string
	Revision string
	Options  CmdOptions
)

type CmdOptions struct {
	LogPretty  bool
	Loglevel   string
	Endpoint   string
	NodeID     string
	MasterURL  string
	Kubeconfig string
	Image      ImageCmdOptions
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "csi-driver-stager",
	Short: "CSI Driver performing stage-in/stage-out",
	Long:  `CSI Driver performing stage-in/stage-out`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initZeroLog()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		zlog.Error().Err(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&Options.LogPretty, "logpretty", false, "sets pretty logging")
	rootCmd.PersistentFlags().StringVar(&Options.Loglevel, "loglevel", "info", "sets log level")
	rootCmd.PersistentFlags().StringVar(&Options.Endpoint, "endpoint", "unix://tmp/csi.sock", "CSI Endpoint")
	rootCmd.PersistentFlags().StringVar(&Options.NodeID, "nodeid", "", "node id")
	rootCmd.PersistentFlags().StringVar(&Options.MasterURL, "masterURL", "", "kubernetes master url")
	rootCmd.PersistentFlags().StringVar(&Options.Kubeconfig, "kubeconfig", "", "kubeconfig path")
}

func initZeroLog() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if Options.LogPretty {
		zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	logLevel, err := zerolog.ParseLevel(Options.Loglevel)
	if err != nil {
		zlog.Error().Str("Loglevel", Options.Loglevel).Msg("can't parse Loglevel")
		os.Exit(1)
	}
	zerolog.SetGlobalLevel(logLevel)
}
