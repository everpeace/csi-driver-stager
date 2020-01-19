package main

import "github.com/everpeace/csi-driver-stager/cmd"

var (
	Version  string
	Revision string
)

func init() {
	cmd.Version = Version
	cmd.Revision = Revision
}

func main() {
	cmd.Execute()
}
