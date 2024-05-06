package patch

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dio/leo/github"
	"github.com/dio/sh"
)

type Info struct {
	Name   string
	Ref    string
	Suffix string
}

type Getter interface {
	Get(context.Context, Info) ([]byte, error)
}

func Get(ctx context.Context, info Info, getter Getter) ([]byte, error) {
	return getter.Get(ctx, info)
}

type GitHubGetter struct {
	Repo string
}

func (g GitHubGetter) Get(ctx context.Context, info Info) ([]byte, error) {
	idx := strings.LastIndex(info.Ref, ".")
	minorName := info.Ref[0:idx]

	// E.g. 1.29.0-fips.patch.
	patchFile := info.Ref + info.Suffix + ".patch"
	content, err := github.GetRaw(ctx, g.Repo, path.Join("patches", info.Name, patchFile), "main")
	if err == nil {
		return []byte(content + "\n"), nil
	}

	// We search for minor with suffix. E.g. 1.29-fips.patch.
	patchFile = minorName + info.Suffix + ".patch"
	content, err = github.GetRaw(ctx, g.Repo, path.Join("patches", info.Name, patchFile), "main")
	if err == nil {
		return []byte(content + "\n"), nil
	}

	// E.g. 1.29.0.patch.
	patchFile = info.Ref + ".patch"
	content, err = github.GetRaw(ctx, g.Repo, path.Join("patches", info.Name, patchFile), "main")
	if err == nil {
		return []byte(content + "\n"), nil
	}

	// We search for minor. E.g. 1.29.patch.
	patchFile = minorName + ".patch"
	content, err = github.GetRaw(ctx, g.Repo, path.Join("patches", info.Name, patchFile), "main")
	if err == nil {
		return []byte(content + "\n"), nil
	}

	return []byte{}, nil
}

type FSGetter struct {
	Dir string
}

func (g FSGetter) Get(_ context.Context, info Info) ([]byte, error) {
	baseDir := filepath.Join(g.Dir, info.Name)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var content []byte
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		idx := strings.LastIndex(info.Ref, ".")
		minorName := info.Ref[0:idx]

		patchFile := info.Ref + info.Suffix + ".patch"
		if entry.Name() == patchFile { // For example -fips.
			content, err = os.ReadFile(filepath.Join(baseDir, patchFile))
			if err == nil {
				return content, nil
			}
		}

		patchFile = minorName + info.Suffix + ".patch"
		if entry.Name() == patchFile { // For example -fips.
			content, err = os.ReadFile(filepath.Join(baseDir, patchFile))
			if err == nil {
				return content, nil
			}
		}

		patchFile = info.Ref + ".patch"
		if entry.Name() == patchFile {
			content, err = os.ReadFile(filepath.Join(baseDir, patchFile))
			if err == nil {
				return content, nil
			}
		}

		// We search for minor.
		patchFile = minorName + ".patch"
		if entry.Name() == patchFile {
			content, err = os.ReadFile(filepath.Join(baseDir, patchFile))
			if err == nil {
				return content, nil
			}
		}
	}

	return []byte{}, nil
}

func Apply(ctx context.Context, info Info, patchGetter Getter, dst string) error {
	patchData, err := patchGetter.Get(ctx, info)
	if err != nil {
		return err
	}

	patchFile, err := os.CreateTemp(os.TempDir(), "*.leo.patch")
	if err != nil {
		return err
	}
	defer func() {
		_ = patchFile.Close()
		_ = os.Remove(patchFile.Name())
	}()

	_, err = patchFile.Write(patchData)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "patching", info.Name, "with", patchFile.Name())
	return sh.Run(ctx, "patch", "-p1", "-i", patchFile.Name(), "-d", dst)
}

type Source string

func (s Source) IsLocal() bool {
	return strings.HasPrefix(string(s), "file://")
}

func (s Source) Path() (string, error) {
	parsed, err := url.Parse(string(s))
	if err != nil {
		return "", nil
	}
	return strings.TrimPrefix(string(s), parsed.Scheme+"://"), nil
}
