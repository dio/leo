package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver"
	"github.com/dio/leo/env"
	"github.com/magefile/mage/sh"
)

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
	if _, err := semver.NewVersion(ref); err != nil {
		return ref, nil
	}

	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		fmt.Sprintf("https://api.github.com/repos/%s/git/ref/tags/%s", repo, ref),
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
