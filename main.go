package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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

	proxyCmd = &cobra.Command{
		Use:   "proxy <command> [flags]",
		Short: "Proxy related tasks",
	}

	proxyInfoCmd = &cobra.Command{
		Use:   "info [flags]",
		Short: "Proxy build info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, fipsBuild)
			if err != nil {
				return err
			}
			return builder.Info(cmd.Context())
		},
	}

	proxyBuildCmd = &cobra.Command{
		Use:   "build [flags]",
		Short: "Build proxy based-on flavors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, fipsBuild)
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
	proxyBuildCmd.Flags().StringVar(&overrideEnvoy, "override-envoy", "", "Override Envoy repository. For example: tetratelabs/envoy@88a80e6bbbee56de8c3899c75eaf36c46fad1aa7")
	proxyBuildCmd.Flags().StringVar(&patchSource, "patch-source", "github://dio/leo-patches", "Patch source. For example: file://patches")
	proxyBuildCmd.Flags().BoolVar(&fipsBuild, "fips-build", false, "FIPS build")
	proxyCmd.AddCommand(proxyBuildCmd)
	proxyCmd.AddCommand(proxyInfoCmd)
	rootCmd.AddCommand(proxyCmd)
}
