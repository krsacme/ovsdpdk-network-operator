package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

const (
	componentName = "ovsdpdk-network-prepare"
)

var (
	rootCmd = &cobra.Command{
		Use:   componentName,
		Short: "Run OvS-DPDK Node Prepare Steps",
	}
)

func init() {
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		glog.Exitf("Error executing ovsdpdk-network-prepare: %v", err)
	}
}
