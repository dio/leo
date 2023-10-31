package build

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/github"
	"github.com/dio/leo/istio"
	"github.com/dio/leo/istioproxy"
	"github.com/dio/leo/patch"
	"github.com/dio/leo/utils"
	"github.com/magefile/mage/sh"
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
	istioProxyRef, _, err := b.info(ctx)
	if err != nil {
		return err
	}

	out := path.Join("work", "proxy-"+istioProxyRef, "out", "*.tar.gz")
	fmt.Print(out)

	return nil
}

func (b *IstioProxyBuilder) Release(ctx context.Context) error {
	istioProxyRef, _, err := b.info(ctx)
	if err != nil {
		return err
	}

	var (
		tag   string
		title string
	)
	switch b.output.Target {
	case "istio-proxy":
		tag = path.Join("istio", b.Version[0:7], "proxy", istioProxyRef[0:7], b.Envoy.Name(), b.Envoy.Version()[0:7])
		title = "istio-proxy@" + istioProxyRef[0:7]
	case "envoy-contrib":
		tag = path.Join(b.Envoy.Name(), b.Envoy.Version()[0:7])
		title = b.Envoy.Name() + "-contrib@" + b.Envoy.Version()[0:7]
	case "envoy":
		tag = path.Join(b.Envoy.Name()+"@", b.Envoy.Version()[0:7])
		title = b.Envoy.Name() + b.Envoy.Version()[0:7]
	}

	out := path.Join(b.output.Dir, "*.tar.gz")
	files, err := filepath.Glob(out)
	if err != nil {
		return err
	}

	notes := fmt.Sprintf(`
- https://github.com/istio/istio/commits/%s
- https://github.com/istio/proxy/commits/%s
- https://github.com/%s/commits/%s
`, b.Version[0:7], istioProxyRef[0:7], b.Envoy.Name(), b.Envoy.Version()[0:7])

	if err := sh.RunV("gh", "release", "view", tag, "-R", b.output.Repo); err != nil {
		if err := sh.RunV("gh", append([]string{"release", "create", tag, "-n", notes, "-t", title, "-R", b.output.Repo}, files...)...); err == nil {
			return err
		}
	} else {
		return sh.RunV("gh", append([]string{"release", "upload", tag, "--clobber", "-R", b.output.Repo}, files...)...)
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
	err = patch.Apply(patch.Info{
		Name: "envoy",
		// Always trim -dev. But this probably misleading since the patch will be valid for envoyVersion.patch+1.
		// For example: A patch that valid 1.24.10-dev, probably invalid for 1.24.10.
		Ref:    strings.TrimSuffix(envoyVersion, "-dev"),
		Suffix: suffix,
	}, b.Patch, envoyDir)

	if err != nil && len(suffix) == 0 {
		return err
	}

	_ = os.RemoveAll(envoyDir)
	envoyDir, err = utils.GetTarballAndExtract(b.Envoy.Name(), b.Envoy.Version(), istioProxyDir)
	if err != nil {
		return err
	}
	if err = patch.Apply(patch.Info{
		Name: "envoy",
		// Always trim -dev. But this probably misleading since the patch will be valid for envoyVersion.patch+1.
		// For example: A patch that valid 1.24.10-dev, probably invalid for 1.24.10.
		Ref: strings.TrimSuffix(envoyVersion, "-dev"),
	}, b.Patch, envoyDir); err != nil {
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
		EnvoyRepo:    b.Envoy.Name(),
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
