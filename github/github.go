package github

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/dio/leo/env"
	"github.com/magefile/mage/sh"
)

type Release struct {
	TagName string `json:"tag_name"`
}

func GetReleases(repo string, page int) ([]Release, error) {
	// https://api.github.com/repos/istio/istio/releases?page=1
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/releases?page=%d", repo, page),
	}
	args = append(args, token()...)

	out, err := sh.Output("curl", args...)
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

func GetLastReleasePageNumber(repo string) (int, error) {
	args := []string{
		"-fsSLI",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/releases", repo),
	}
	args = append(args, token()...)

	out, err := sh.Output("curl", args...)
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

func GetRaw(repo, file, ref string) (string, error) {
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.raw",
		fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", repo, file, ref),
	}
	args = append(args, token()...)

	return sh.Output("curl", args...)
}

func GetTarball(repo, ref, dir string) (string, error) {
	_ = os.MkdirAll(dir, os.ModePerm)
	targz := filepath.Join(dir, ref+".tar.gz")
	args := []string{
		"-fsSL",
		"-o",
		targz,
		fmt.Sprintf("https://github.com/%s/archive/%s.tar.gz", repo, ref),
	}
	args = append(args, user()...)

	if err := sh.Run("curl", args...); err != nil {
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

func ResolveCommitSHA(repo, ref string) (string, error) {
	// Check if the given ref is a "head" (i.e. branch).
	sha, err := getRefSHA(repo, ref, "heads")
	if err == nil {
		return sha, nil
	}

	// If not, we check if it is a commit SHA.
	// TODO(dio): Validate commit SHA.
	if _, err := semver.NewVersion(ref); err != nil {
		return ref, nil
	}

	// Since this is a valid semver, we check it as a tag.
	return getRefSHA(repo, ref, "tags")
}

func getRefSHA(repo, ref, refType string) (string, error) {
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/git/ref/%s/%s", repo, refType, ref),
	}
	args = append(args, token()...)

	out, err := sh.Output("curl", args...)
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

	// When the refType is tags, we need to resolve it once again.
	args = []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/git/tags/%s", repo, r.Object.SHA),
	}
	args = append(args, token()...)

	out, err = sh.Output("curl", args...)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return "", err
	}
	return r.Object.SHA, nil
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
