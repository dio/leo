package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/build"
	"github.com/dio/leo/compute"
	"github.com/dio/leo/envoy"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "leo <command> [flags]",
		Short: "Your artifacts builder",
	}

	zone         string
	instanceName string
	machineType  string
	machineImage string

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

	computeCreateCmd = &cobra.Command{
		Use:   "create [flags]",
		Short: "Create a compute",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			i := &compute.Instance{
				ProjectID: os.Getenv("GCLOUD_PROJECT"),
				Zone:      zone,
				Name:      instanceName,
			}
			return i.Create(cmd.Context(), machineType, machineImage)
		},
	}

	computeDeleteCmd = &cobra.Command{
		Use:   "delete [flags]",
		Short: "Delete a compute",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			i := &compute.Instance{
				ProjectID: os.Getenv("GCLOUD_PROJECT"),
				Zone:      zone,
				Name:      instanceName,
			}
			return i.Delete(cmd.Context())
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

	resolveCmd = &cobra.Command{
		Use:   "resolve [flags]",
		Short: "Resolve workspace from a reference",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := arg.Version(args[0])
			r := arg.Repo(v.Name())
			switch r.Name() {
			case "envoy":
				target, err := envoy.ResolveWorkspace(cmd.Context(), v)
				if err != nil {
					return err
				}
				fmt.Print(target)
			}
			return nil
		},
	}

	overrideEnvoy string
	patchSource   string
	fipsBuild     bool
	remoteCache   string
	target        string
	arch          string
	version       string
	repo          string
	dir           string

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

	proxyReleaseCmd = &cobra.Command{
		Use:   "release [flags]",
		Short: "Proxy release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			builder, err := build.NewProxyBuilder(args[0], overrideEnvoy, patchSource, remoteCache, fipsBuild, &build.Output{
				Target: target,
				Arch:   arch,
				Repo:   repo,
				Dir:    dir,
			})
			if err != nil {
				return err
			}
			return builder.Release(cmd.Context())
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

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("leo", version)
			return nil
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
	computeCmd.PersistentFlags().StringVar(&machineType, "machine-type", "n2-standard-8", "Machine type")
	computeCmd.PersistentFlags().StringVar(&machineImage, "machine-image", "builder-amd64", "Machine image")
	computeCmd.AddCommand(computeStartCmd)
	computeCmd.AddCommand(computeStopCmd)
	computeCmd.AddCommand(computeCreateCmd)
	computeCmd.AddCommand(computeDeleteCmd)

	proxyCmd.PersistentFlags().StringVar(&overrideEnvoy, "override-envoy", "", "Override Envoy repository. For example: tetratelabs/envoy@88a80e6bbbee56de8c3899c75eaf36c46fad1aa7")
	proxyCmd.PersistentFlags().StringVar(&patchSource, "patch-source", "github://dio/leo", "Patch source. For example: file://patches")
	proxyCmd.PersistentFlags().BoolVar(&fipsBuild, "fips-build", false, "FIPS build")
	proxyCmd.PersistentFlags().StringVar(&remoteCache, "remote-cache", "", "Remote cache. E.g. us-central1, asia-south2")
	proxyOutputCmd.Flags().StringVar(&target, "target", "istio-proxy", "Build target, i.e. envoy, istio-proxy")
	proxyOutputCmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "Builder architecture")
	proxyReleaseCmd.Flags().StringVar(&target, "target", "istio-proxy", "Build target, i.e. envoy, istio-proxy")
	proxyReleaseCmd.Flags().StringVar(&repo, "repo", "tetrateio/proxy-archives", "Archives repo")
	proxyReleaseCmd.Flags().StringVar(&dir, "dir", "./out", "Assets directory")
	proxyReleaseCmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "Builder architecture")

	proxyCmd.AddCommand(proxyInfoCmd)
	proxyCmd.AddCommand(proxyOutputCmd)
	proxyCmd.AddCommand(proxyBuildCmd)
	proxyCmd.AddCommand(proxyReleaseCmd)

	rootCmd.AddCommand(computeCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(resolveCmd)
	rootCmd.AddCommand(versionCmd)
}
