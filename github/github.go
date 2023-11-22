package github

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/dio/leo/env"
	"github.com/dio/sh"
)

type Release struct {
	TagName string `json:"tag_name"`
}

func GetReleases(ctx context.Context, repo string, page int) ([]Release, error) {
	// https://api.github.com/repos/istio/istio/releases?page=1
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/releases?page=%d", repo, page),
	}
	args = append(args, token()...)

	out, err := sh.Output(ctx, "curl", args...)
	if err != nil {
		return nil, err
	}

	releases := make([]Release, 0)
	if err := json.Unmarshal([]byte(out), &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

var (
	pagePattern = `page=(\d+)`
	pageRe      = regexp.MustCompile(pagePattern)
)

func GetLastReleasePageNumber(ctx context.Context, repo string) (int, error) {
	args := []string{
		"-fsSLI",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/releases", repo),
	}
	args = append(args, token()...)

	out, err := sh.Output(ctx, "curl", args...)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "link:") {
			matches := pageRe.FindAllStringSubmatch(line, -1)
			page := matches[len(matches)-1][1]
			return strconv.Atoi(page)
		}
	}

	return 0, nil
}

func GetRaw(ctx context.Context, repo, file, ref string) (string, error) {
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.raw",
		fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", repo, file, ref),
	}
	args = append(args, token()...)

	return sh.Output(ctx, "curl", args...)
}

func GetTarball(ctx context.Context, repo, ref, dir string) (string, error) {
	_ = os.MkdirAll(dir, os.ModePerm)
	targz := filepath.Join(dir, ref+".tar.gz")
	args := []string{
		"-fsSL",
		"-o",
		targz,
		fmt.Sprintf("https://github.com/%s/archive/%s.tar.gz", repo, ref),
	}
	args = append(args, user()...)

	if err := sh.Run(ctx, "curl", args...); err != nil {
		return "", err
	}
	return targz, nil
}

type Ref struct {
	Object RefObject `json:"object"`
}

type RefObject struct {
	SHA string `json:"sha"`
}

type Runs struct {
	Count int `json:"total_count"`
}

func WorkflowRuns(ctx context.Context, repo, status string) (int, error) {
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/actions/runs?status=%s", repo, status),
	}
	args = append(args, token()...)

	out, err := sh.Output(ctx, "curl", args...)
	if err != nil {
		return 0, err
	}
	var r Runs
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return 0, err
	}
	return r.Count, nil
}

func ResolveCommitSHA(ctx context.Context, repo, ref string) (string, error) {
	// Check if the given ref is from commits
	sha, err := getCommit(ctx, repo, ref)
	if err == nil {
		return sha, nil
	}

	// Check if the given ref is a "head" (i.e. branch).
	sha, err = GetRefSHA(ctx, repo, ref, "heads")
	if err == nil {
		return sha, nil
	}

	// If not, we check if it is a commit SHA.
	// TODO(dio): Validate commit SHA.
	if _, err := semver.NewVersion(ref); err != nil {
		return ref, nil
	}

	// Since this is a valid semver, we check it as a tag.
	return GetRefSHA(ctx, repo, ref, "tags")
}

func getCommit(ctx context.Context, repo, ref string) (string, error) {
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, ref),
	}
	args = append(args, token()...)

	out, err := sh.Output(ctx, "curl", args...)
	if err != nil {
		return "", err
	}
	var r RefObject
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return "", err
	}
	return r.SHA, nil
}

func GetRefSHA(ctx context.Context, repo, ref, refType string) (string, error) {
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/git/ref/%s/%s", repo, refType, ref),
	}
	args = append(args, token()...)

	out, err := sh.Output(ctx, "curl", args...)
	if err != nil {
		return "", err
	}
	var r Ref
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return "", err
	}

	if refType != "tags" {
		return r.Object.SHA, nil
	}

	// When the refType is tags, we need to resolve it once again IF it is not a commit.
	args = []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, r.Object.SHA),
	}
	args = append(args, token()...)
	out, err = sh.Output(ctx, "curl", args...)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal([]byte(out), &r); err == nil {
		return r.Object.SHA, err
	}

	args = []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/git/tags/%s", repo, r.Object.SHA),
	}
	args = append(args, token()...)

	out, err = sh.Output(ctx, "curl", args...)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return "", err
	}
	return r.Object.SHA, nil
}

// GetNewerRelease gets release newer than version (patch, minor, or major).
func GetNewerMinorRelease(ctx context.Context, version string) (string, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}

	lastPage, err := GetLastReleasePageNumber(ctx, "istio/istio")
	if err != nil {
		return "", err
	}

	for page := 1; page <= lastPage; page++ {
		releases, err := GetReleases(ctx, "istio/istio", page)
		if err != nil {
			return "", err
		}
		for _, release := range releases {
			if strings.Contains(release.TagName, "-") {
				continue
			}

			r, err := semver.NewVersion(release.TagName)
			if err != nil {
				return "", err
			}

			if r.Minor() > v.Minor() {
				return release.TagName, nil
			}
		}
	}
	return "", errors.New("not found")
}

// GetNewerRelease gets release newer than version (patch, minor, or major).
func GetNewerPatchRelease(ctx context.Context, version string) (string, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}
	majorMinor := fmt.Sprintf("%d.%d", v.Major(), v.Minor())

	lastPage, err := GetLastReleasePageNumber(ctx, "istio/istio")
	if err != nil {
		return "", err
	}

	for page := 1; page <= lastPage; page++ {
		releases, err := GetReleases(ctx, "istio/istio", page)
		if err != nil {
			return "", err
		}
		for _, release := range releases {
			if strings.Contains(release.TagName, "-") {
				continue
			}

			if !strings.HasPrefix(release.TagName, majorMinor) {
				continue
			}

			r, err := semver.NewVersion(release.TagName)
			if err != nil {
				return "", err
			}

			if r.Patch() > v.Patch() {
				return release.TagName, nil
			}
		}
	}
	return "", errors.New("not found")
}

func token() []string {
	val := env.GH_TOKEN
	if len(val) > 0 {
		return []string{
			"-H",
			// Reference: https://github.com/octokit/auth-token.js/blob/902a172693d08de998250bf4d8acb1fdb22377a4/src/with-authorization-prefix.ts#L6-L12
			fmt.Sprintf("Authorization: token %s", val),
		}
	}
	return []string{}
}

func user() []string {
	val := env.GH_TOKEN
	if len(val) > 0 {
		return []string{
			"-u",
			fmt.Sprintf(":%s", val),
		}
	}
	return []string{}
}
