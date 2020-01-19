package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print version",
	Long:  `print version`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := zerolog.New(os.Stderr)
		if Options.LogPretty {
			logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}
		logger.Log().Str("Version", Version).Str("Revision", Revision).Msg("")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
