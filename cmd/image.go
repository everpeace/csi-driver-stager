/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"os"
	"time"

	"github.com/everpeace/csi-driver-stager/pkg/stager/driver/imagedriver"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/clock"

	zlog "github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type ImageCmdOptions struct {
	BuildahPath      string
	BuildahTimeout   time.Duration
	BuildahGcTimeout time.Duration
	BuildahGcPeriod  time.Duration
}

// imageCmd represents the Image command
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "start container Image csi driver",
	Long:  `start containerr csi driver`,
	Run: func(cmd *cobra.Command, args []string) {
		zlog.Info().
			Str("Driver", imagedriver.DriverName).
			Str("Version", Version).
			Str("Revision", Revision).
			Interface("Options", &Options).
			Msg("Starting")

		config, err := clientcmd.BuildConfigFromFlags(Options.MasterURL, Options.Kubeconfig)
		if err != nil {
			panic(err.Error())
		}
		kubeClient, err := kubernetes.NewForConfig(rest.AddUserAgent(config, "csi-driver-stager"))
		if err != nil {
			panic(err.Error())
		}

		driver := imagedriver.NewDriver(
			Version, Options.NodeID, Options.Endpoint,
			Options.Image.BuildahPath, Options.Image.BuildahTimeout,
			Options.Image.BuildahGcTimeout, Options.Image.BuildahGcPeriod,
			kubeClient, clock.RealClock{},
		)

		if err := driver.Run(); err != nil {
			zlog.Err(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)

	imageCmd.Flags().StringVar(&Options.Image.BuildahPath, "buildahPath", "/bin/buildah", "buildah binary path")
	imageCmd.Flags().DurationVar(&Options.Image.BuildahTimeout, "buildahTimeout", 10*time.Minute, "timeout to execute buildah commands")
	imageCmd.Flags().DurationVar(&Options.Image.BuildahGcTimeout, "buildahGcTimeout", 60*time.Minute, "timeout to execute buildah gc command")
	imageCmd.Flags().DurationVar(&Options.Image.BuildahGcPeriod, "buildahGcPeriod", 24*time.Hour, "period for performing buildah gc")
}
