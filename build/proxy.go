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

func NewProxyBuilder(target, overrideIstioProxy, overrideEnvoy, patchSource, patchSourceName, remoteCache, patchSuffix, dynamicModulesBuild string,
	fipsBuild, wasm, gperftools bool, output *Output) (*ProxyBuilder, error) {
	var patchGetter patch.Getter

	patchGetterSource := patch.Source(patchSource)
	patchPath, err := patchGetterSource.Path()
	if err != nil {
		return nil, err
	}
	if patchGetterSource.IsLocal() {
		patchGetter = &patch.FSGetter{
			Dir: patchPath, // TODO(dio): Allow to override this.
		}
	} else {
		patchGetter = &patch.GitHubGetter{
			Repo: patchPath, // TODO(dio): Allow to override this.
		}
	}
	return &ProxyBuilder{
		target:              arg.Version(target),
		envoy:               arg.Version(overrideEnvoy),
		istioProxy:          arg.Version(overrideIstioProxy),
		patchGetter:         patchGetter,
		fipsBuild:           fipsBuild,
		dynamicModulesBuild: dynamicModulesBuild,
		gperftools:          gperftools,
		wasm:                wasm,
		output:              output,
		remoteCache:         remoteCache,
		patchInfoName:       patchSourceName,
		patchSuffix:         patchSuffix,
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

	// these are for output
	output *Output
}

func (b *ProxyBuilder) Info(ctx context.Context) error {
	switch b.target.Repo().Name() {
	case "istio":
		builder := &IstioProxyBuilder{
			Istio:               b.target,
			Version:             b.target.Version(),
			Envoy:               b.envoy,
			IstioProxy:          b.istioProxy,
			Patch:               b.patchGetter,
			FIPSBuild:           b.fipsBuild,
			DynamicModulesBuild: b.dynamicModulesBuild,
			Gperftools:          b.gperftools,
			Wasm:                b.wasm,
			remoteCache:         b.remoteCache,
			PatchInfoName:       b.patchInfoName,
			PatchSuffix:         b.patchSuffix,
		}
		return builder.Info(ctx)
	}

	return nil
}

func (b *ProxyBuilder) Output(ctx context.Context) error {
	switch b.target.Repo().Name() {
	case "istio":
		builder := &IstioProxyBuilder{
			Istio:               b.target,
			Version:             b.target.Version(),
			Envoy:               b.envoy,
			IstioProxy:          b.istioProxy,
			Patch:               b.patchGetter,
			FIPSBuild:           b.fipsBuild,
			DynamicModulesBuild: b.dynamicModulesBuild,
			Wasm:                b.wasm,
			Gperftools:          b.gperftools,
			output:              b.output,
			remoteCache:         b.remoteCache,
			PatchInfoName:       b.patchInfoName,
			PatchSuffix:         b.patchSuffix,
		}
		return builder.Output(ctx)
	}

	return nil
}

func (b *ProxyBuilder) Release(ctx context.Context) error {
	switch b.target.Repo().Name() {
	case "istio":
		builder := &IstioProxyBuilder{
			Istio:               b.target,
			Version:             b.target.Version(),
			Envoy:               b.envoy,
			IstioProxy:          b.istioProxy,
			Patch:               b.patchGetter,
			FIPSBuild:           b.fipsBuild,
			DynamicModulesBuild: b.dynamicModulesBuild,
			Gperftools:          b.gperftools,
			Wasm:                b.wasm,
			output:              b.output,
			remoteCache:         b.remoteCache,
			PatchInfoName:       b.patchInfoName,
			PatchSuffix:         b.patchSuffix,
		}
		return builder.Release(ctx)
	}

	return nil
}

func (b *ProxyBuilder) Build(ctx context.Context) error {
	switch b.target.Repo().Name() {
	case "istio":
		builder := &IstioProxyBuilder{
			Istio:               b.target,
			Version:             b.target.Version(),
			Envoy:               b.envoy,
			IstioProxy:          b.istioProxy,
			Patch:               b.patchGetter,
			FIPSBuild:           b.fipsBuild,
			DynamicModulesBuild: b.dynamicModulesBuild,
			Gperftools:          b.gperftools,
			Wasm:                b.wasm,
			remoteCache:         b.remoteCache,
			PatchInfoName:       b.patchInfoName,
			PatchSuffix:         b.patchSuffix,
		}
		return builder.Build(ctx)
	}

	return nil
}
