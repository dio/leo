package envoy

import (
	"errors"
	"strings"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/github"
	"github.com/dio/leo/istio"
	"github.com/dio/leo/istioproxy"
)

// ResolveWorkspace returns istio version that can serve building an envoy version.
func ResolveWorkspace(v arg.Version) (string, error) {
	target, err := getVersion(v)
	if err != nil {
		return "", err
	}

	// Firstly, check if master can serve us.
	master, err := getReferencedVersion("master")
	if err != nil {
		return "", err
	}
	if target == master {
		return github.ResolveCommitSHA("istio/istio", "master")
	}

	lastPage, err := github.GetLastReleasePageNumber("istio/istio")
	if err != nil {
		return "", err
	}

	// If not, we need to scan all releases.
	for page := 1; page <= lastPage; page++ {
		releases, err := github.GetReleases("istio/istio", page)
		if err != nil {
			return "", err
		}
		for _, release := range releases {
			ref, err := getReferencedVersion(release.TagName)
			if err != nil {
				return "", err
			}
			if target == ref {
				return github.ResolveCommitSHA("istio/istio", release.TagName)
			}
		}
	}

	return "", errors.New("cannot resolve")
}

func getVersion(v arg.Version) (string, error) {
	versionTxt, err := github.GetRaw(v.Name(), "VERSION.txt", v.Version())
	if err != nil {
		return "", err
	}

	target := strings.TrimSpace(string(versionTxt))

	// We hope that matching minor version can use the same build-tools.
	return target[0:strings.LastIndex(target, ".")], nil
}

func getReferencedVersion(istioRef string) (string, error) {
	deps, err := istio.GetDeps("istio/istio", istioRef)
	if err != nil {
		return "", err
	}

	workspace, err := github.GetRaw("istio/proxy", "WORKSPACE", deps.Get("proxy").SHA)
	if err != nil {
		return "", err
	}

	e, err := istioproxy.EnvoyFromWorkspace(workspace)
	if err != nil {
		return "", err
	}
	return getVersion(arg.Version(e.Org + "/" + e.Repo + "@" + e.SHA))
}
