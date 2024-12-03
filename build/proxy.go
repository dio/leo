package build

import (
	"context"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/patch"
)

type Output struct {
	Target string
	Arch   string
	Repo   string
	Dir    string
}

func NewProxyBuilder(target,
	overrideIstioProxy, overrideEnvoy,
	patchSource, patchSourceName,
	remoteCache, patchSuffix, dynamicModulesBuild,
	additionalPatchDir, additionalPatchDirSource string,
	fipsBuild, wasm, gperftools bool,
	output *Output) (*ProxyBuilder, error) {
	var patchGetter patch.Getter
	var additionalPatchGetter patch.Getter

	patchGetterSource := patch.Source(patchSource)
	patchPath := patchGetterSource.Path()

	if patchGetterSource.IsLocal() {
		patchGetter = &patch.FSGetter{
			Dir: patchPath, // TODO(dio): Allow to override this.
		}
	} else {
		patchGetter = &patch.GitHubGetter{
			Repo: patchPath, // TODO(dio): Allow to override this.
		}
	}

	if additionalPatchDirSource != patchSource {
		additionalPatchGetterSource := patch.Source(additionalPatchDirSource)
		additionalPatchPath := additionalPatchGetterSource.Path()

		if additionalPatchGetterSource.IsLocal() {
			additionalPatchGetter = &patch.FSGetter{
				Dir: additionalPatchPath, // TODO(dio): Allow to override this.
			}
		} else {
			additionalPatchGetter = &patch.GitHubGetter{
				Repo: additionalPatchPath, // TODO(dio): Allow to override this.
				Ref:  additionalPatchGetterSource.Ref(),
			}
		}
	}

	return &ProxyBuilder{
		target:                arg.Version(target),
		envoy:                 arg.Version(overrideEnvoy),
		istioProxy:            arg.Version(overrideIstioProxy),
		patchGetter:           patchGetter,
		fipsBuild:             fipsBuild,
		dynamicModulesBuild:   dynamicModulesBuild,
		gperftools:            gperftools,
		wasm:                  wasm,
		output:                output,
		remoteCache:           remoteCache,
		patchInfoName:         patchSourceName,
		patchSuffix:           patchSuffix,
		additionalPatchDir:    additionalPatchDir,
		additionalPatchGetter: additionalPatchGetter,
	}, nil
}

type ProxyBuilder struct {
	target              arg.Version
	envoy               arg.Version
	istioProxy          arg.Version
	patchGetter         patch.Getter
	fipsBuild           bool
	wasm                bool
	gperftools          bool
	remoteCache         string
	patchInfoName       string
	dynamicModulesBuild string
	patchSuffix         string

	// Additional patches support.
	// The patches are placed in the additionalPatchDir directory and applied after the main patch.
	// The proxy patch filenames are prefixed with the 'proxy-' and the envoy patch filenames
	// are prefixed with 'envoy-'.
	additionalPatchDir    string
	additionalPatchGetter patch.Getter

	// these are for output
	output *Output
}

func (b *ProxyBuilder) Info(ctx context.Context) error {
	switch b.target.Repo().Name() {
	case "tetrateio-proxy":
		fallthrough
	case "istio":
		builder := &IstioProxyBuilder{
			Istio:                 b.target,
			Version:               b.target.Version(),
			Envoy:                 b.envoy,
			IstioProxy:            b.istioProxy,
			Patch:                 b.patchGetter,
			FIPSBuild:             b.fipsBuild,
			DynamicModulesBuild:   b.dynamicModulesBuild,
			Gperftools:            b.gperftools,
			Wasm:                  b.wasm,
			remoteCache:           b.remoteCache,
			PatchInfoName:         b.patchInfoName,
			PatchSuffix:           b.patchSuffix,
			AdditionalPatchDir:    b.additionalPatchDir,
			AdditionalPatchGetter: b.additionalPatchGetter,
		}
		return builder.Info(ctx)
	}

	return nil
}

func (b *ProxyBuilder) Output(ctx context.Context) error {
	switch b.target.Repo().Name() {
	case "tetrateio-proxy":
		fallthrough
	case "istio":
		builder := &IstioProxyBuilder{
			Istio:                 b.target,
			Version:               b.target.Version(),
			Envoy:                 b.envoy,
			IstioProxy:            b.istioProxy,
			Patch:                 b.patchGetter,
			FIPSBuild:             b.fipsBuild,
			DynamicModulesBuild:   b.dynamicModulesBuild,
			Wasm:                  b.wasm,
			Gperftools:            b.gperftools,
			output:                b.output,
			remoteCache:           b.remoteCache,
			PatchInfoName:         b.patchInfoName,
			PatchSuffix:           b.patchSuffix,
			AdditionalPatchDir:    b.additionalPatchDir,
			AdditionalPatchGetter: b.additionalPatchGetter,
		}
		return builder.Output(ctx)
	}

	return nil
}

func (b *ProxyBuilder) Release(ctx context.Context) error {
	switch b.target.Repo().Name() {
	case "tetrateio-proxy":
		fallthrough
	case "istio":
		builder := &IstioProxyBuilder{
			Istio:                 b.target,
			Version:               b.target.Version(),
			Envoy:                 b.envoy,
			IstioProxy:            b.istioProxy,
			Patch:                 b.patchGetter,
			FIPSBuild:             b.fipsBuild,
			DynamicModulesBuild:   b.dynamicModulesBuild,
			Gperftools:            b.gperftools,
			Wasm:                  b.wasm,
			output:                b.output,
			remoteCache:           b.remoteCache,
			PatchInfoName:         b.patchInfoName,
			PatchSuffix:           b.patchSuffix,
			AdditionalPatchDir:    b.additionalPatchDir,
			AdditionalPatchGetter: b.additionalPatchGetter,
		}

		return builder.Release(ctx)
	}

	return nil
}

func (b *ProxyBuilder) Build(ctx context.Context) error {
	switch b.target.Repo().Name() {
	case "tetrateio-proxy":
		fallthrough
	case "istio":
		builder := &IstioProxyBuilder{
			Istio:                 b.target,
			Version:               b.target.Version(),
			Envoy:                 b.envoy,
			IstioProxy:            b.istioProxy,
			Patch:                 b.patchGetter,
			FIPSBuild:             b.fipsBuild,
			DynamicModulesBuild:   b.dynamicModulesBuild,
			Gperftools:            b.gperftools,
			Wasm:                  b.wasm,
			remoteCache:           b.remoteCache,
			PatchInfoName:         b.patchInfoName,
			PatchSuffix:           b.patchSuffix,
			AdditionalPatchDir:    b.additionalPatchDir,
			AdditionalPatchGetter: b.additionalPatchGetter,
		}
		return builder.Build(ctx)
	}

	return nil
}
