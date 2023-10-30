package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/github"
	"github.com/dio/leo/istio"
	"github.com/dio/leo/istioproxy"
	"github.com/dio/leo/patch"
	"github.com/dio/leo/utils"
)

type IstioProxyBuilder struct {
	Version   string
	Envoy     arg.Version
	Patch     patch.Getter
	FIPSBuild bool

	remoteCache string
	output      *Output
}

func (b *IstioProxyBuilder) info(ctx context.Context) (string, string, error) {
	istioRef, err := github.ResolveCommitSHA("istio/istio", b.Version)
	if err != nil {
		return "", "", err
	}
	b.Version = istioRef

	deps, err := istio.GetDeps(b.Version)
	if err != nil {
		return "", "", err
	}
	istioProxyRef := deps.Get("proxy").SHA

	// When "envoy" is empty, we need to resolve our envoy from the istio.deps.
	if b.Envoy.IsEmpty() {
		istioProxyWorkspace, err := github.GetRaw("istio/proxy", "WORKSPACE", istioProxyRef)
		if err != nil {
			return "", "", err
		}
		envoyRepo, err := istioproxy.EnvoyFromWorkspace(istioProxyWorkspace)
		if err != nil {
			return "", "", err
		}
		b.Envoy = arg.Version(fmt.Sprintf("%s/%s@%s", envoyRepo.Org, envoyRepo.Repo, envoyRepo.SHA))
	} else {
		envoySHA, err := github.ResolveCommitSHA(b.Envoy.Name(), b.Envoy.Version())
		if err != nil {
			return "", "", err
		}
		b.Envoy = arg.Version(b.Envoy.Name() + "@" + envoySHA)
	}

	envoyVersion, err := github.GetRaw(b.Envoy.Name(), "VERSION.txt", b.Envoy.Version())
	if err != nil {
		return "", "", err
	}
	return istioProxyRef, envoyVersion, err
}

func (b *IstioProxyBuilder) Info(ctx context.Context) error {
	istioProxyRef, envoyVersion, err := b.info(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, `build info:
  workspace: istio/proxy@%s
  envoy: %s
  envoyVersion: %s
  fips: %v
`, istioProxyRef, b.Envoy, envoyVersion, b.FIPSBuild)
	return nil
}

func (b *IstioProxyBuilder) Output(ctx context.Context) error {
	istioProxyRef, envoyVersion, err := b.info(ctx)
	if err != nil {
		return err
	}

	var targzSuffix string
	if b.FIPSBuild {
		targzSuffix = "fips-"
	}

	prefix := path.Join("work", "proxy-"+istioProxyRef, "out")

	// TODO(dio): Synchronize this with the make targets.
	switch b.output.Target {
	case "istio-proxy":
		fmt.Printf("%s/istio-proxy-%s-%s-%s-%s%s.tar.gz", prefix, b.Version[0:7], istioProxyRef[0:7], b.Envoy.Version()[0:7], targzSuffix, b.output.Arch)
	case "envoy":
		fmt.Printf("%s/envoy-%s-%s-%s%s.tar.gz", prefix, envoyVersion, b.Envoy.Version()[0:7], targzSuffix, b.output.Arch)
	case "envoy-contrib":
		fmt.Printf("%s/envoy-contrib-%s-%s-%s%s.tar.gz", prefix, envoyVersion, b.Envoy.Version()[0:7], targzSuffix, b.output.Arch)
	default:
		return errors.New("invalid target")
	}

	return nil
}

func (b *IstioProxyBuilder) Build(ctx context.Context) error {
	istioProxyRef, envoyVersion, err := b.info(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, `build info:
  workspace: istio/proxy@%s
  envoy: %s
  envoyVersion: %s
  fips: %v
`, istioProxyRef, b.Envoy, envoyVersion, b.FIPSBuild)

	istioProxyDir, err := utils.GetTarballAndExtract("istio/proxy", istioProxyRef, "work")
	if err != nil {
		return err
	}

	envoyDir, err := utils.GetTarballAndExtract(b.Envoy.Name(), b.Envoy.Version(), istioProxyDir)
	if err != nil {
		return err
	}

	var suffix string
	if b.FIPSBuild {
		suffix = "-fips"
	}
	// Patch envoy
	if err := patch.Apply(patch.Info{
		Name: "envoy",
		// Always trim -dev. But this probably misleading since the patch will be valid for envoyVersion.patch+1.
		// For example: A patch that valid 1.24.10-dev, probably invalid for 1.24.10.
		Ref:    strings.TrimSuffix(envoyVersion, "-dev"),
		Suffix: suffix,
	},
		b.Patch, envoyDir); err != nil {
		return err
	}

	if err := istioproxy.WriteWorkspaceStatus(istioProxyDir, b.Envoy.Name(), b.Envoy.Version()); err != nil {
		return err
	}

	if err := istioproxy.AddMakeTargets(istioproxy.TargetOptions{
		ProxyDir:     istioProxyDir,
		ProxySHA:     istioProxyRef,
		EnvoyDir:     envoyDir,
		EnvoySHA:     b.Envoy.Version(),
		EnvoyVersion: envoyVersion,
		IstioVersion: b.Version,
		FIPSBuild:    b.FIPSBuild,
		RemoteCache:  b.remoteCache,
	}); err != nil {
		return err
	}

	if err := istioproxy.PrepareBuilder(istioProxyDir); err != nil {
		return err
	}

	fmt.Print(istioProxyDir)

	return nil
}
