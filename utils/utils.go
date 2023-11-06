package utils

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/github"
	"github.com/dio/sh"
)

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
