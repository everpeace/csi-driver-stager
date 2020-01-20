package main

import (
	"flag"
	"github.com/everpeace/csi-driver-stager/cmd"
	"k8s.io/klog"
)

var (
	Version  string
	Revision string
)

func init() {
	cmd.Version = Version
	cmd.Revision = Revision

	// hack to make flag.Parsed return true such that glog is happy
	// about the flags having been parsed
	fs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(fs)
	_ = fs.Parse([]string{"-logtostderr=true", "-v=0"})
	flag.CommandLine = fs
}

func main() {
	cmd.Execute()
}
