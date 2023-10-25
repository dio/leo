package utils

import (
	"os"
	"path"

	"github.com/dio/leo/arg"
	"github.com/dio/leo/github"
	"github.com/magefile/mage/sh"
)

func GetTarballAndExtract(repo, ref, dir string) (string, error) {
	tmp, err := os.MkdirTemp(os.TempDir(), "leo.*")
	if err != nil {
		return "", err
	}

	targz, err := github.GetTarball(repo, ref, tmp)
	if err != nil {
		return "", err
	}

	_ = os.MkdirAll(dir, os.ModePerm)
	if err := sh.Run("tar", "-C", dir, "-xzf", targz); err != nil {
		return "", err
	}

	return path.Join(dir, arg.Repo(repo).Name()+"-"+ref), nil
}
