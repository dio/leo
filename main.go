package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/dio/leo/build"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "leo <command> [flags]",
		Short: "Your artifacts builder",
	}

	overrideEnvoy string
	patchSource   string
	fipsBuild     bool
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
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, fipsBuild, nil)
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
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, fipsBuild, &build.Output{
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
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, fipsBuild, nil)
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
	proxyInfoCmd.Flags().StringVar(&overrideEnvoy, "override-envoy", "", "Override Envoy repository. For example: tetratelabs/envoy@88a80e6bbbee56de8c3899c75eaf36c46fad1aa7")
	proxyInfoCmd.Flags().StringVar(&patchSource, "patch-source", "github://dio/leo", "Patch source. For example: file://patches")
	proxyInfoCmd.Flags().BoolVar(&fipsBuild, "fips-build", false, "FIPS build")
	proxyOutputCmd.Flags().StringVar(&overrideEnvoy, "override-envoy", "", "Override Envoy repository. For example: tetratelabs/envoy@88a80e6bbbee56de8c3899c75eaf36c46fad1aa7")
	proxyOutputCmd.Flags().StringVar(&patchSource, "patch-source", "github://dio/leo", "Patch source. For example: file://patches")
	proxyOutputCmd.Flags().BoolVar(&fipsBuild, "fips-build", false, "FIPS build")
	proxyOutputCmd.Flags().StringVar(&target, "target", "istio-proxy", "Build target, i.e. envoy, istio-proxy")
	proxyOutputCmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "Builder architecture")
	proxyBuildCmd.Flags().StringVar(&overrideEnvoy, "override-envoy", "", "Override Envoy repository. For example: tetratelabs/envoy@88a80e6bbbee56de8c3899c75eaf36c46fad1aa7")
	proxyBuildCmd.Flags().StringVar(&patchSource, "patch-source", "github://dio/leo", "Patch source. For example: file://patches")
	proxyBuildCmd.Flags().BoolVar(&fipsBuild, "fips-build", false, "FIPS build")

	proxyCmd.AddCommand(proxyInfoCmd)
	proxyCmd.AddCommand(proxyOutputCmd)
	proxyCmd.AddCommand(proxyBuildCmd)

	rootCmd.AddCommand(proxyCmd)
}
