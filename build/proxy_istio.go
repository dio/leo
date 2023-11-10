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
	"github.com/dio/sh"
)

type IstioProxyBuilder struct {
	Version   string
	Envoy     arg.Version
	Patch     patch.Getter
	FIPSBuild bool
	Wasm      bool

	remoteCache string
	output      *Output
}

func (b *IstioProxyBuilder) info(ctx context.Context) (string, string, error) {
	istioRef, err := github.ResolveCommitSHA(ctx, "istio/istio", b.Version)
	if err != nil {
		return "", "", err
	}
	b.Version = istioRef

	deps, err := istio.GetDeps(ctx, "istio/istio", b.Version)
	if err != nil {
		return "", "", err
	}
	istioProxyRef := deps.Get("proxy").SHA

	// When "envoy" is empty, we need to resolve our envoy from the istio.deps.
	if b.Envoy.IsEmpty() {
		istioProxyWorkspace, err := github.GetRaw(ctx, "istio/proxy", "WORKSPACE", istioProxyRef)
		if err != nil {
			return "", "", err
		}
		envoyRepo, err := istioproxy.EnvoyFromWorkspace(istioProxyWorkspace)
		if err != nil {
			return "", "", err
		}
		b.Envoy = arg.Version(fmt.Sprintf("%s/%s@%s", envoyRepo.Org, envoyRepo.Repo, envoyRepo.SHA))
	} else {
		envoySHA, err := github.ResolveCommitSHA(ctx, b.Envoy.Name(), b.Envoy.Version())
		if err != nil {
			return "", "", err
		}
		b.Envoy = arg.Version(b.Envoy.Name() + "@" + envoySHA)
	}

	envoyVersion, err := github.GetRaw(ctx, b.Envoy.Name(), "VERSION.txt", b.Envoy.Version())
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
	wasmFiles, err := filepath.Glob(path.Join("work", "proxy-"+istioProxyRef, "out", "*.wasm"))
	if err != nil {
		return err
	}
	if len(wasmFiles) > 0 {
		out += " " + path.Join("work", "proxy-"+istioProxyRef, "out", "*.wasm")
	}

	fmt.Print(out)

	return nil
}

func (b *IstioProxyBuilder) Release(ctx context.Context) error {
	istioProxyRef, _, err := b.info(ctx)
	if err != nil {
		return err
	}

	var (
		tag            string
		title          string
		remoteProxyDir string
		remoteProxyRef string
	)
	switch b.output.Target {
	case "istio-proxy":
		tag = path.Join("istio", b.Version[0:7], "proxy", istioProxyRef[0:7], b.Envoy.Name(), b.Envoy.Version()[0:7])
		title = "istio-proxy@" + istioProxyRef[0:7]
		remoteProxyDir = "proxy"
		remoteProxyRef = "alpha-" + istioProxyRef

	case "istio-proxy-centos7":
		tag = path.Join("istio", b.Version[0:7], "proxy", istioProxyRef[0:7], b.Envoy.Name(), b.Envoy.Version()[0:7])
		title = "istio-proxy@" + istioProxyRef[0:7]
		remoteProxyDir = "proxy"
		remoteProxyRef = "centos-alpha-" + istioProxyRef

	case "envoy-contrib":
		tag = path.Join(b.Envoy.Name(), b.Envoy.Version()[0:7])
		title = b.Envoy.Name() + "-contrib@" + b.Envoy.Version()[0:7]
		remoteProxyDir = "envoy-contrib"
		remoteProxyRef = b.Envoy.Version()

	case "envoy":
		tag = path.Join(b.Envoy.Name(), b.Envoy.Version()[0:7])
		title = b.Envoy.Name() + "@" + b.Envoy.Version()[0:7]
		remoteProxyDir = "envoy"
		remoteProxyRef = b.Envoy.Version()

	case "envoy-centos7":
		tag = path.Join(b.Envoy.Name(), b.Envoy.Version()[0:7])
		title = b.Envoy.Name() + "@" + b.Envoy.Version()[0:7]
		remoteProxyDir = "envoy"
		remoteProxyRef = "centos-" + b.Envoy.Version()
	}

	out := path.Join(b.output.Dir, "*.tar.gz")
	files, err := filepath.Glob(out)
	if err != nil {
		return err
	}

	if b.FIPSBuild {
		remoteProxyDir += "-fips"
	}
	if b.Envoy.Name() != "envoyproxy/envoy" {
		remoteProxyDir += "-" + arg.Repo(b.Envoy.Name()).Owner()
	}
	suffix := ".tar.gz"
	if b.output.Arch != "amd64" {
		suffix = "-" + b.output.Arch + ".tar.gz"
	}

	for _, file := range files {
		if !strings.HasSuffix(file, ".tar.gz") {
			continue
		}
		// Upload to GCS.
		remoteFile := path.Join("tetrate-istio-distro-build", remoteProxyDir, "envoy-"+remoteProxyRef+suffix)
		if err := sh.RunV(ctx, "gsutil", "cp", file, "gs://"+remoteFile); err != nil {
			return err
		}
	}

	// Upload wasm files.
	wasmOut := path.Join(b.output.Dir, "*.wasm")
	wasmFiles, err := filepath.Glob(wasmOut)
	if err != nil {
		return err
	}
	for _, file := range wasmFiles {
		if strings.HasSuffix(file, ".wasm") {
			remoteFile := path.Join("tetrate-istio-distro-build",
				remoteProxyDir, strings.Replace(file, ".wasm", "-"+istioProxyRef+".wasm", 1))
			if strings.HasSuffix(file, ".compiled.wasm") {
				remoteFile = path.Join("tetrate-istio-distro-build",
					remoteProxyDir, strings.Replace(file, ".compiled.wasm", "-"+istioProxyRef+".compiled.wasm", 1))
			}

			if err := sh.RunV(ctx, "gsutil", "cp", file, "gs://"+remoteFile); err != nil {
				return err
			}
			continue
		}
	}

	notes := fmt.Sprintf(`
- https://github.com/istio/istio/commits/%s
- https://github.com/istio/proxy/commits/%s
- https://github.com/%s/commits/%s
`, b.Version[0:7], istioProxyRef[0:7], b.Envoy.Name(), b.Envoy.Version()[0:7])

	if err := sh.RunV(ctx, "gh", "release", "view", tag, "-R", b.output.Repo); err != nil {
		if err := sh.RunV(ctx, "gh", append([]string{"release", "create", tag, "-n", notes, "-t", title, "-R", b.output.Repo}, files...)...); err == nil {
			return err
		}
	}
	return sh.RunV(ctx, "gh", append([]string{"release", "upload", tag, "--clobber", "-R", b.output.Repo}, files...)...)
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

	istioProxyDir, err := utils.GetTarballAndExtract(ctx, "istio/proxy", istioProxyRef, "work")
	if err != nil {
		return err
	}

	envoyDir, err := utils.GetTarballAndExtract(ctx, b.Envoy.Name(), b.Envoy.Version(), istioProxyDir)
	if err != nil {
		return err
	}

	var suffix string
	if b.FIPSBuild {
		suffix = "-fips"
	}
	// Patch envoy
	err = patch.Apply(ctx, patch.Info{
		Name: "envoy",
		// Always trim -dev. But this probably misleading since the patch will be valid for envoyVersion.patch+1.
		// For example: A patch that valid 1.24.10-dev, probably invalid for 1.24.10.
		Ref:    strings.TrimSuffix(envoyVersion, "-dev"),
		Suffix: suffix,
	}, b.Patch, envoyDir)

	if err != nil {
		// When we have no suffix, no fallback.
		if len(suffix) == 0 {
			return err
		}
		_ = os.RemoveAll(envoyDir)
		envoyDir, err = utils.GetTarballAndExtract(ctx, b.Envoy.Name(), b.Envoy.Version(), istioProxyDir)
		if err != nil {
			return err
		}
		if err = patch.Apply(ctx, patch.Info{
			Name: "envoy",
			// Always trim -dev. But this probably misleading since the patch will be valid for envoyVersion.patch+1.
			// For example: A patch that valid 1.24.10-dev, probably invalid for 1.24.10.
			Ref: strings.TrimSuffix(envoyVersion, "-dev"),
		}, b.Patch, envoyDir); err != nil {
			return err
		}
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
		Wasm:         b.Wasm,
		RemoteCache:  b.remoteCache,
	}); err != nil {
		return err
	}

	if err := istioproxy.PrepareBuilder(istioProxyDir, b.remoteCache); err != nil {
		return err
	}

	fmt.Print(istioProxyDir)

	return nil
}
