package build

import (
	"context"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/patch"
)

func NewProxyBuilder(target, overrideEnvoy, patchSource string, fipsBuild bool) (*ProxyBuilder, error) {
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
		target:      arg.Version(target),
		envoy:       arg.Version(overrideEnvoy),
		patchGetter: patchGetter,
		fipsBuild:   fipsBuild,
	}, nil
}

type ProxyBuilder struct {
	target      arg.Version
	envoy       arg.Version
	patchGetter patch.Getter
	fipsBuild   bool
}

func (b *ProxyBuilder) Info(ctx context.Context) error {
	switch b.target.Name() {
	case "istio":
		builder := &IstioProxyBuilder{
			Version:   b.target.Version(),
			Envoy:     b.envoy,
			Patch:     b.patchGetter,
			FIPSBuild: b.fipsBuild,
		}
		return builder.Info(ctx)
	}

	return nil
}

func (b *ProxyBuilder) Build(ctx context.Context) error {
	switch b.target.Name() {
	case "istio":
		builder := &IstioProxyBuilder{
			Version:   b.target.Version(),
			Envoy:     b.envoy,
			Patch:     b.patchGetter,
			FIPSBuild: b.fipsBuild,
		}
		return builder.Build(ctx)
	}

	return nil
}
