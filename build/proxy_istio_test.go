package build

import (
	"testing"

	"github.com/dio/leo/arg"
)

func TestGetRemoteProxyDir(t *testing.T) {
	tests := []struct {
		name                   string
		builder                IstioProxyBuilder
		expectedRemoteProxyDir string
	}{
		{
			name: "istio-proxy",
			builder: IstioProxyBuilder{
				Istio:  arg.Version("istio"),
				Envoy:  arg.Version("envoyproxy/envoy"),
				output: &Output{Target: "istio-proxy"},
			},
			expectedRemoteProxyDir: "proxy",
		},
		{
			name: "tetrateio-proxy",
			builder: IstioProxyBuilder{
				Istio:  arg.Version("tetrateio-proxy"),
				Envoy:  arg.Version("envoyproxy/envoy"),
				output: &Output{Target: "istio-proxy"},
			},
			expectedRemoteProxyDir: "tetrateio-proxy",
		},
		{
			name: "tetrateio-proxy",
			builder: IstioProxyBuilder{
				Istio:  arg.Version("tetrateio-proxy"),
				Envoy:  arg.Version("istio/envoy"),
				output: &Output{Target: "istio-proxy"},
			},
			expectedRemoteProxyDir: "tetrateio-proxy",
		},
		{
			name: "istio-proxy-centos7",
			builder: IstioProxyBuilder{
				Istio:  arg.Version("istio"),
				Envoy:  arg.Version("envoyproxy/envoy"),
				output: &Output{Target: "istio-proxy-centos7"},
			},
			expectedRemoteProxyDir: "proxy",
		},
		{
			name: "envoy-contrib",
			builder: IstioProxyBuilder{
				Istio:  arg.Version("istio"),
				Envoy:  arg.Version("envoyproxy/envoy"),
				output: &Output{Target: "envoy-contrib"},
			},
			expectedRemoteProxyDir: "envoy-contrib",
		},
		{
			name: "envoy",
			builder: IstioProxyBuilder{
				Istio:  arg.Version("istio"),
				Envoy:  arg.Version("envoyproxy/envoy"),
				output: &Output{Target: "envoy"},
			},
			expectedRemoteProxyDir: "envoy",
		},
		{
			name: "envoy-centos7",
			builder: IstioProxyBuilder{
				Istio:  arg.Version("istio"),
				Envoy:  arg.Version("envoyproxy/envoy"),
				output: &Output{Target: "envoy-centos7"},
			},
			expectedRemoteProxyDir: "envoy",
		},
		{
			name: "dynamic-modules",
			builder: IstioProxyBuilder{
				Istio:               arg.Version("istio"),
				Envoy:               arg.Version("envoyproxy/envoy"),
				DynamicModulesBuild: "some-module",
				output:              &Output{Target: "istio-proxy"},
			},
			expectedRemoteProxyDir: "proxy-dynamic-modules",
		},
		{
			name: "fips",
			builder: IstioProxyBuilder{
				Istio:     arg.Version("istio"),
				Envoy:     arg.Version("envoyproxy/envoy"),
				FIPSBuild: true,
				output:    &Output{Target: "istio-proxy"},
			},
			expectedRemoteProxyDir: "proxy-fips",
		},
		{
			name: "tetrateio-proxy-fips",
			builder: IstioProxyBuilder{
				Istio:     arg.Version("tetrateio-proxy"),
				Envoy:     arg.Version("envoyproxy/envoy"),
				FIPSBuild: true,
				output:    &Output{Target: "istio-proxy"},
			},
			expectedRemoteProxyDir: "tetrateio-proxy-fips",
		},
		{
			name: "tetrateio-proxy-fips",
			builder: IstioProxyBuilder{
				Istio:     arg.Version("tetrateio-proxy"),
				Envoy:     arg.Version("istio/envoy"),
				FIPSBuild: true,
				output:    &Output{Target: "istio-proxy"},
			},
			expectedRemoteProxyDir: "tetrateio-proxy-fips",
		},
		{
			name: "custom-envoy",
			builder: IstioProxyBuilder{
				Istio:  arg.Version("istio"),
				Envoy:  arg.Version("custom/envoy"),
				output: &Output{Target: "istio-proxy"},
			},
			expectedRemoteProxyDir: "proxy-custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.builder.getRemoteProxyDir(); got != tt.expectedRemoteProxyDir {
				t.Errorf("getRemoteProxyDir() = %v, want %v", got, tt.expectedRemoteProxyDir)
			}
		})
	}
}
