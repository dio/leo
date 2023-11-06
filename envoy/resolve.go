package envoy

import (
	"context"
	"errors"
	"strings"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/github"
	"github.com/dio/leo/istio"
	"github.com/dio/leo/istioproxy"
)

// ResolveWorkspace returns istio version that can serve building an envoy version.
func ResolveWorkspace(ctx context.Context, v arg.Version) (string, error) {
	target, err := getVersion(ctx, v)
	if err != nil {
		return "", err
	}

	// Firstly, check if master can serve us.
	master, err := getReferencedVersion(ctx, "master")
	if err != nil {
		return "", err
	}
	if target == master {
		return github.ResolveCommitSHA(ctx, "istio/istio", "master")
	}

	lastPage, err := github.GetLastReleasePageNumber(ctx, "istio/istio")
	if err != nil {
		return "", err
	}

	// If not, we need to scan all releases.
	for page := 1; page <= lastPage; page++ {
		releases, err := github.GetReleases(ctx, "istio/istio", page)
		if err != nil {
			return "", err
		}
		for _, release := range releases {
			ref, err := getReferencedVersion(ctx, release.TagName)
			if err != nil {
				return "", err
			}
			if target == ref {
				return github.ResolveCommitSHA(ctx, "istio/istio", release.TagName)
			}
		}
	}

	return "", errors.New("cannot resolve")
}

func getVersion(ctx context.Context, v arg.Version) (string, error) {
	versionTxt, err := github.GetRaw(ctx, v.Name(), "VERSION.txt", v.Version())
	if err != nil {
		return "", err
	}

	target := strings.TrimSpace(string(versionTxt))

	// We hope that matching minor version can use the same build-tools.
	return target[0:strings.LastIndex(target, ".")], nil
}

func getReferencedVersion(ctx context.Context, istioRef string) (string, error) {
	deps, err := istio.GetDeps(ctx, "istio/istio", istioRef)
	if err != nil {
		return "", err
	}

	workspace, err := github.GetRaw(ctx, "istio/proxy", "WORKSPACE", deps.Get("proxy").SHA)
	if err != nil {
		return "", err
	}

	e, err := istioproxy.EnvoyFromWorkspace(workspace)
	if err != nil {
		return "", err
	}
	return getVersion(ctx, arg.Version(e.Org+"/"+e.Repo+"@"+e.SHA))
}
