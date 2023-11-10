package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/github"
	"github.com/dio/sh"
)

type version struct {
	Version string `json:"version"`
}

func GetLatestGoVersion(ctx context.Context) (string, error) {
	args := []string{
		"-fsSL",
		"-H", "Accept: application/vnd.github.v3.json",
		"https://go.dev/dl/?mode=json",
	}
	out, err := sh.Output(ctx, "curl", args...)
	if err != nil {
		return "", err
	}
	var versions []version
	if err := json.Unmarshal([]byte(out), &versions); err != nil {
		return "", err
	}
	if len(versions) == 0 {
		// https://go.dev/dl/go1.21.4.linux-amd64.tar.gz.
		return "go1.21.4", nil
	}

	return versions[0].Version, nil
}

func GetTarballAndExtract(ctx context.Context, repo, ref, dir string) (string, error) {
	tmp, err := os.MkdirTemp(os.TempDir(), "leo.*")
	if err != nil {
		return "", err
	}

	targz, err := github.GetTarball(ctx, repo, ref, tmp)
	if err != nil {
		return "", err
	}

	_ = os.MkdirAll(dir, os.ModePerm)
	if err := sh.Run(ctx, "tar", "-C", dir, "-xzf", targz); err != nil {
		return "", err
	}

	return path.Join(dir, arg.Repo(repo).Name()+"-"+ref), nil
}

func GitCloneAndCheckout(ctx context.Context, repo, ref, dir string) (string, error) {
	dst := filepath.Join(dir, ref)
	_ = os.RemoveAll(dst)
	_ = os.MkdirAll(dst, os.ModePerm)
	if err := sh.Run(ctx, "git", "clone", "https://github.com/"+repo+".git", dst); err != nil {
		return "", err
	}

	return dst, sh.Run(ctx, "git", "-C", dst, "checkout", ref)
}

func ReplaceContent(name, s, r string) error {
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}

	out := bytes.Replace(data, []byte(s), []byte(r), 1)
	return os.WriteFile(path.Join(name), out, os.ModePerm)
}

func ReplaceMatchedLine(name, replaced string, replacer func(string) string) error {
	input, err := os.Open(name)
	if err != nil {
		return err
	}
	defer input.Close()

	scanner := bufio.NewScanner(input)

	var out bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		out.WriteString(replacer(line) + "\n")
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(replaced, out.Bytes(), os.ModePerm)
}
