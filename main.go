package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/dio/leo/build"
	"github.com/dio/leo/compute"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "leo <command> [flags]",
		Short: "Your artifacts builder",
	}

	zone         string
	instanceName string

	computeCmd = &cobra.Command{
		Use:   "compute <command> [flags]",
		Short: "Start and stop compute",
	}

	computeStartCmd = &cobra.Command{
		Use:   "start [flags]",
		Short: "Start a compute",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			i := &compute.Instance{
				ProjectID: os.Getenv("GCLOUD_PROJECT"),
				Zone:      zone,
				Name:      instanceName,
			}
			return i.Start(cmd.Context())
		},
	}

	computeStopCmd = &cobra.Command{
		Use:   "stop [flags]",
		Short: "Stop a compute",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			i := &compute.Instance{
				ProjectID: os.Getenv("GCLOUD_PROJECT"),
				Zone:      zone,
				Name:      instanceName,
			}
			return i.Stop(cmd.Context())
		},
	}

	overrideEnvoy string
	patchSource   string
	fipsBuild     bool
	remoteCache   string
	target        string
	arch          string

	proxyCmd = &cobra.Command{
		Use:   "proxy <command> [flags]",
		Short: "Proxy related tasks",
	}

	proxyInfoCmd = &cobra.Command{
		Use:   "info [flags]",
		Short: "Proxy build info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, remoteCache, fipsBuild, nil)
			if err != nil {
				return err
			}
			return builder.Info(cmd.Context())
		},
	}

	proxyOutputCmd = &cobra.Command{
		Use:   "output [flags]",
		Short: "Proxy build output",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, remoteCache, fipsBuild, &build.Output{
				Target: target,
				Arch:   arch,
			})
			if err != nil {
				return err
			}
			return builder.Output(cmd.Context())
		},
	}

	proxyBuildCmd = &cobra.Command{
		Use:   "build [flags]",
		Short: "Build proxy based-on flavors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, remoteCache, fipsBuild, nil)
			if err != nil {
				return err
			}
			return builder.Build(cmd.Context())
		},
	}
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	computeCmd.PersistentFlags().StringVar(&zone, "zone", "", "Zone")
	computeCmd.PersistentFlags().StringVar(&instanceName, "instance", "", "Instance name")
	computeCmd.AddCommand(computeStartCmd)
	computeCmd.AddCommand(computeStopCmd)

	proxyCmd.PersistentFlags().StringVar(&overrideEnvoy, "override-envoy", "", "Override Envoy repository. For example: tetratelabs/envoy@88a80e6bbbee56de8c3899c75eaf36c46fad1aa7")
	proxyCmd.PersistentFlags().StringVar(&patchSource, "patch-source", "github://dio/leo", "Patch source. For example: file://patches")
	proxyCmd.PersistentFlags().BoolVar(&fipsBuild, "fips-build", false, "FIPS build")
	proxyCmd.PersistentFlags().StringVar(&remoteCache, "remote-cache", "", "Remote cache. E.g. us-central1, asia-south2")
	proxyOutputCmd.Flags().StringVar(&target, "target", "istio-proxy", "Build target, i.e. envoy, istio-proxy")
	proxyOutputCmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "Builder architecture")

	proxyCmd.AddCommand(proxyInfoCmd)
	proxyCmd.AddCommand(proxyOutputCmd)
	proxyCmd.AddCommand(proxyBuildCmd)

	rootCmd.AddCommand(computeCmd)
	rootCmd.AddCommand(proxyCmd)
}
