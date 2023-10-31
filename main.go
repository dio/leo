package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"cloud.google.com/go/pubsub"
	"github.com/dio/leo/arg"
	"github.com/dio/leo/build"
	"github.com/dio/leo/compute"
	"github.com/dio/leo/envoy"
	"github.com/dio/leo/github"
	"github.com/dio/leo/queue"
	"github.com/magefile/mage/sh"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "leo <command> [flags]",
		Short: "Your artifacts builder",
	}

	queueTarget    string
	queueVersion   string
	queueArguments string
	queueSkip      bool

	queueCmd = &cobra.Command{
		Use:   "queue <command> [flags]",
		Short: "Pull from and push to queue",
	}

	queuePullCmd = &cobra.Command{
		Use:   "pull [flags]",
		Short: "Pull from queue",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var mu sync.Mutex
			proxyBuilder := "tetrateio/proxy-builder"
			return queue.Pull(cmd.Context(), "builds", func(ctx context.Context, msg *pubsub.Message) {
				mu.Lock()
				if queueSkip {
					msg.Ack()
					mu.Unlock()
					return
				}

				inProgress, _ := github.WorkflowRuns(proxyBuilder, "in_progress")
				pending, _ := github.WorkflowRuns(proxyBuilder, "pending")
				c := inProgress + pending
				fmt.Println("we have", c, "runs")

				// So we can immediately queue a task.
				if c >= 2 {
					fmt.Println("we need to wait")
					_ = queue.Publish(ctx, "builds", msg.Data)
				} else {
					data := map[string]string{}
					if err := json.Unmarshal(msg.Data, &data); err == nil {
						// Run workflow!
						switch data["name"] {
						case "build.yaml":
							err = sh.RunV("gh", "workflow", "run", data["name"],
								"-f", "target="+data["target"],
								"-f", "istio-version="+data["istioVersion"],
								"-f", "arguments="+data["arguments"],
								"-R", proxyBuilder,
							)
							if err != nil {
								_ = queue.Publish(ctx, "builds", msg.Data)
							}
						case "build-envoy.yaml":
							err = sh.RunV("gh", "workflow", "run", data["name"],
								"-f", "target="+data["target"],
								"-f", "envoy="+data["envoy"],
								"-f", "arguments="+data["arguments"],
								"-R", proxyBuilder,
							)
							if err != nil {
								_ = queue.Publish(ctx, "builds", msg.Data)
							}
						}
					}
				}

				msg.Ack()
				mu.Unlock()
				os.Exit(0)
			})
		},
	}

	queuePushCmd = &cobra.Command{
		Use:   "push [flags]",
		Short: "Push to queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			switch args[0] {
			case "build.yaml":
				i := queue.InputsBuild{
					Name:         "build.yaml",
					Target:       queueTarget,
					Arguments:    queueArguments,
					IstioVersion: queueVersion,
				}
				msg, err := json.Marshal(i)
				if err != nil {
					return err
				}
				return queue.Publish(ctx, "builds", msg)

			case "build-envoy.yaml":
				i := queue.InputsBuildEnvoy{
					Name:      "build-envoy.yaml",
					Target:    queueTarget,
					Arguments: queueArguments,
					Envoy:     queueVersion,
				}
				msg, err := json.Marshal(i)
				if err != nil {
					return err
				}
				return queue.Publish(ctx, "builds", msg)
			}

			return nil
		},
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

	resolveCmd = &cobra.Command{
		Use:   "resolve [flags]",
		Short: "Resolve workspace from a reference",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := arg.Version(args[0])
			r := arg.Repo(v.Name())
			switch r.Name() {
			case "envoy":
				target, err := envoy.ResolveWorkspace(v)
				if err != nil {
					return err
				}
				fmt.Print(target)
			}
			return nil
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
	queueCmd.AddCommand(queuePullCmd)
	queueCmd.AddCommand(queuePushCmd)
	queuePullCmd.Flags().BoolVar(&queueSkip, "skip", false, "skip")
	queuePushCmd.Flags().StringVar(&queueTarget, "target", "", "target")
	queuePushCmd.Flags().StringVar(&queueVersion, "version", "", "version")
	queuePushCmd.Flags().StringVar(&queueArguments, "arguments", "", "arguments")

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
	proxyReleaseCmd.Flags().StringVar(&target, "target", "istio-proxy", "Build target, i.e. envoy, istio-proxy")
	proxyReleaseCmd.Flags().StringVar(&repo, "repo", "tetrateio/proxy-archives", "Archives repo")
	proxyReleaseCmd.Flags().StringVar(&dir, "dir", "./out", "Assets directory")

	proxyCmd.AddCommand(proxyInfoCmd)
	proxyCmd.AddCommand(proxyOutputCmd)
	proxyCmd.AddCommand(proxyBuildCmd)
	proxyCmd.AddCommand(proxyReleaseCmd)

	rootCmd.AddCommand(computeCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(resolveCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(queueCmd)
}
