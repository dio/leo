package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"slices"

	"github.com/dio/leo/env"
	"github.com/dio/sh"
)

type List []string

func (l List) Contains(name string) bool {
	return slices.Contains(l, name)
}

type TagsList struct {
	Child List `json:"child"`
	Tags  List `json:"tags"`
}

func GetTagsList(ctx context.Context, repo string) (*TagsList, error) {
	args := []string{
		"-fsSL",
		fmt.Sprintf("https://us-central1-docker.pkg.dev/v2/%s/tags/list", repo),
	}
	args = append(args, token()...)

	out, err := sh.Output(ctx, "curl", args...)
	if err != nil {
		return nil, err
	}

	var l TagsList
	if err := json.Unmarshal([]byte(out), &l); err != nil {
		return nil, err
	}
	return &l, nil
}

func GetImagesForVersion(ctx context.Context, repo, version string) ([]string, error) {
	l, err := GetTagsList(ctx, repo)
	if err != nil {
		return nil, err
	}

	var images []string
	for _, i := range l.Child {
		l, err := GetTagsList(ctx, path.Join(repo, i))
		if err != nil {
			return nil, err
		}
		if l.Tags.Contains(version) {
			images = append(images, i)
		}
	}
	return images, nil
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
