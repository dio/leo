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
	Istio      arg.Version
	IstioProxy arg.Version
	Version    string
	Envoy      arg.Version
	Patch      patch.Getter
	FIPSBuild  bool
	// Currently, we use boolean to determine if we need to build dynamic modules.
	DynamicModulesBuild bool
	Wasm                bool
	PatchInfoName       string
	PatchSuffix         string
	Gperftools          bool

	remoteCache string
	output      *Output
}

func (b *IstioProxyBuilder) info(ctx context.Context) (string, string, error) {
	istioRepo := "istio/istio"
	if len(b.Istio.Repo().Owner()) != 0 {
		istioRepo = string(b.Istio.Repo())
	}

	istioRef, err := github.ResolveCommitSHA(ctx, istioRepo, b.Version)
	if err != nil {
		return "", "", err
	}
	b.Version = istioRef

	// When IstioProxy is not set, we need to resolve the proxy from the istio.deps.
	if b.IstioProxy.IsEmpty() {
		deps, err := istio.GetDeps(ctx, istioRepo, b.Version)
		if err != nil {
			return "", "", err
		}
		b.IstioProxy = arg.Version(fmt.Sprintf("istio/proxy@%s", deps.Get("proxy").SHA))
	} else {
		b.IstioProxy = arg.Version(fmt.Sprintf("%s@%s", b.IstioProxy.Name(), b.IstioProxy.Version()))
	}

	// When "envoy" is empty, we need to resolve our envoy from the istio.deps.
	if b.Envoy.IsEmpty() {
		istioProxyWorkspace, err := github.GetRaw(ctx, b.IstioProxy.Name(), "WORKSPACE", b.IstioProxy.Version())
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
	return b.IstioProxy.Version(), envoyVersion, err
}

func (b *IstioProxyBuilder) Info(ctx context.Context) error {
	istioProxyRef, envoyVersion, err := b.info(ctx)
	if err != nil {
		return err
	}

	if b.IstioProxy == "" {
		b.IstioProxy = arg.Version(fmt.Sprintf("istio/proxy@%s", istioProxyRef))
	}

	fmt.Fprintf(os.Stderr, `build info:
  istio: %s
  workspace: %s
  envoy: %s
  envoyVersion: %s
  fips: %v
  dynamic-modules: %v
`, b.Istio, b.IstioProxy, b.Envoy, envoyVersion, b.FIPSBuild, b.DynamicModulesBuild)
	return nil
}

func (b *IstioProxyBuilder) Output(ctx context.Context) error {
	istioProxyRef, _, err := b.info(ctx)
	if err != nil {
		return err
	}

	out := path.Join("work", "proxy-"+istioProxyRef, "out", "*")
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

	if b.DynamicModulesBuild {
		remoteProxyDir = "-dynamic-modules"
		tag = path.Join(tag, "dynamic-modules")
		title += "+dynamic-modules"
	}
	if b.FIPSBuild {
		remoteProxyDir += "-fips"
	}
	if b.Envoy.Name() != "envoyproxy/envoy" {
		remoteProxyDir += "-" + arg.Repo(b.Envoy.Name()).Owner()
	}
	suffix := b.PatchSuffix + ".tar.gz"
	if b.output.Arch != "amd64" {
		suffix = "-" + b.output.Arch + b.PatchSuffix + ".tar.gz"
	}

	for _, file := range files {
		if !strings.HasSuffix(file, ".tar.gz") {
			continue
		}
		// Upload to GCS.
		remoteFile := path.Join("tetrate-istio-subscription-build", remoteProxyDir, "envoy-"+remoteProxyRef+suffix)
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
			baseName := filepath.Base(file)
			remoteFile := path.Join("tetrate-istio-subscription-build",
				remoteProxyDir, strings.Replace(baseName, ".wasm", "-"+istioProxyRef+".wasm", 1))
			if strings.HasSuffix(file, ".compiled.wasm") {
				remoteFile = path.Join("tetrate-istio-subscription-build",
					remoteProxyDir, strings.Replace(baseName, ".compiled.wasm", "-"+istioProxyRef+".compiled.wasm", 1))
			}

			if err := sh.RunV(ctx, "gsutil", "cp", file, "gs://"+remoteFile); err != nil {
				return err
			}
			continue
		}
	}

	notes := fmt.Sprintf(`
- https://github.com/%s/commits/%s
- https://github.com/%s/commits/%s
- https://github.com/%s/commits/%s
`, b.Istio.Repo(), b.Version[0:7], b.IstioProxy.Name(), b.IstioProxy.Version()[0:7], b.Envoy.Name(), b.Envoy.Version()[0:7])

	if err := sh.RunV(ctx, "gh", "release", "view", tag, "-R", b.output.Repo); err != nil {
		if err := sh.RunV(ctx, "gh", append([]string{"release", "create", tag, "-n", notes, "-t", title, "-R", b.output.Repo}, files...)...); err == nil {
			return err
		}
	} else {
		return sh.RunV(ctx, "gh", append([]string{"release", "upload", tag, "--clobber", "-R", b.output.Repo}, files...)...)
	}
	return nil
}

func (b *IstioProxyBuilder) Build(ctx context.Context) error {
	istioProxyRef, envoyVersion, err := b.info(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, `build info:
  istio: %s
  workspace: %s
  envoy: %s
  envoyVersion: %s
  fips: %v
  dynamic-modules: %v
`, b.Istio, b.IstioProxy, b.Envoy, envoyVersion, b.FIPSBuild, b.DynamicModulesBuild)

	istioProxyDir, err := utils.GetTarballAndExtract(ctx, b.IstioProxy.Name(), istioProxyRef, "work")
	if err != nil {
		return err
	}

	envoyDir, err := utils.GetTarballAndExtract(ctx, b.Envoy.Name(), b.Envoy.Version(), istioProxyDir)
	if err != nil {
		return err
	}

	var suffix string
	if b.DynamicModulesBuild {
		suffix = "-dynamic-modules"
		// When we have DynamicModulesBuild, we need to add the dynamic modules to the workspace.
		// This is a hack since we use istio/proxy workspace vs. envoy workspace.
		istioProxyWorkspace, err := github.GetRaw(ctx, b.IstioProxy.Name(), "WORKSPACE", b.IstioProxy.Version())
		if err != nil {
			return err
		}
		modifiedIstioProxyWorkspace := istioproxy.AddDynamicModules(istioProxyWorkspace)
		if err := os.WriteFile(filepath.Join(istioProxyDir, "WORKSPACE"), []byte(modifiedIstioProxyWorkspace), os.ModePerm); err != nil {
			return err
		}
	}

	if b.FIPSBuild {
		suffix = "-fips"
	}

	if len(b.PatchInfoName) == 0 {
		b.PatchInfoName = "envoy"
	}

	// Patch envoy
	err = patch.Apply(ctx, patch.Info{
		Name: b.PatchInfoName,
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
			Name: b.PatchInfoName,
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
		ProxyDir:            istioProxyDir,
		ProxySHA:            istioProxyRef,
		EnvoyDir:            envoyDir,
		EnvoySHA:            b.Envoy.Version(),
		EnvoyVersion:        envoyVersion,
		EnvoyRepo:           b.Envoy.Name(),
		IstioVersion:        b.Version,
		FIPSBuild:           b.FIPSBuild,
		DynamicModulesBuild: b.DynamicModulesBuild,
		Gperftools:          b.Gperftools,
		Wasm:                b.Wasm,
		RemoteCache:         b.remoteCache,
	}); err != nil {
		return err
	}

	if err := istioproxy.PrepareBuilder(istioProxyDir, b.remoteCache); err != nil {
		return err
	}

	fmt.Print(istioProxyDir)

	return nil
}
