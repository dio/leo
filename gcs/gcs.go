package gcs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dio/leo/env"
	"github.com/dio/sh"
)

// For example, to download: https://storage.cloud.google.com/tetrate-istio-distro-build/istio/1.19.3-tetrate0/istio-1.19.3-tetrate0-linux-amd64.tar.gz
func DownloadIstioLinuxTarball(ctx context.Context, dir, version string) (string, error) {
	_ = os.MkdirAll(dir, os.ModePerm)
	targz := filepath.Join(dir, version+".tar.gz")
	args := []string{
		"-fsSL",
		"-o",
		targz,
		fmt.Sprintf("https://storage.cloud.google.com/tetrate-istio-distro-build/istio/%s/istio-%s-linux-amd64.tar.gz", version, version),
	}
	args = append(args, token()...)

	if err := sh.Run(ctx, "curl", args...); err != nil {
		return "", err
	}
	return targz, nil
}

func token() []string {
	val := env.GCLOUD_TOKEN
	if len(val) > 0 {
		return []string{
			"-H",
			fmt.Sprintf("Authorization: Bearer %s", val),
		}
	}
	return []string{}
}
