package patch

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dio/leo/github"
	"github.com/magefile/mage/sh"
)

type Info struct {
	Name   string
	Ref    string
	Suffix string
}

type Getter interface {
	Get(Info) ([]byte, error)
}

func Get(info Info, getter Getter) ([]byte, error) {
	return getter.Get(info)
}

type GitHubGetter struct {
	Repo string
}

func (g GitHubGetter) Get(info Info) ([]byte, error) {
	patchFile := info.Ref + info.Suffix + ".patch"
	content, err := github.GetRaw(g.Repo, path.Join("patches", info.Name, patchFile), "main")
	if err == nil {
		return []byte(content + "\n"), nil
	}

	patchFile = info.Ref + ".patch"
	content, err = github.GetRaw(g.Repo, path.Join("patches", info.Name, patchFile), "main")
	if err == nil {
		return []byte(content + "\n"), nil
	}

	// We search for minor.
	idx := strings.LastIndex(info.Ref, ".")
	patchFile = info.Ref[0:idx] + ".patch"
	content, err = github.GetRaw(g.Repo, path.Join("patches", info.Name, patchFile), "main")
	if err == nil {
		return []byte(content + "\n"), nil
	}

	return []byte{}, nil
}

type FSGetter struct {
	Dir string
}

func (g FSGetter) Get(info Info) ([]byte, error) {
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
		patchFile := info.Ref + info.Suffix + ".patch"
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
		idx := strings.LastIndex(info.Ref, ".")
		patchFile = info.Ref[0:idx] + ".patch"
		if entry.Name() == patchFile {
			content, err = os.ReadFile(filepath.Join(baseDir, patchFile))
			if err == nil {
				return content, nil
			}
		}
	}

	return []byte{}, nil
}

func Apply(info Info, patchGetter Getter, dst string) error {
	patchData, err := patchGetter.Get(info)
	if err != nil {
		return err
	}

	patchFile, err := os.CreateTemp(os.TempDir(), "*.leo.patch")
	if err != nil {
		return err
	}
	defer func() {
		_ = patchFile.Close()
		// _ = os.Remove(patchFile.Name())
	}()

	_, err = patchFile.Write(patchData)
	if err != nil {
		return err
	}
	return sh.Run("patch", "-p1", "-i", patchFile.Name(), "-d", dst)
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
