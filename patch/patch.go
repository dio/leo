package patch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
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
	List(context.Context, string, string) ([]Info, error)
}

func Get(ctx context.Context, info Info, getter Getter) ([]byte, error) {
	return getter.Get(ctx, info)
}

type GitHubGetter struct {
	Repo string
	Ref  string
}

func (g GitHubGetter) Get(ctx context.Context, info Info) ([]byte, error) {

	ref := g.Ref
	if ref == "" {
		ref = "main"
	}

	fmt.Fprintln(os.Stderr, "Searching for patch", info.Name+"/"+info.Ref, "in", g.Repo+"@"+ref)

	// Try getting the file from the ref branch first
	content, err := github.GetRaw(ctx, g.Repo, info.Name, ref)
	if err == nil {
		return []byte(content + "\n"), nil
	}

	idx := strings.LastIndex(info.Ref, ".")
	minorName := info.Ref[0:idx]

	// E.g. 1.29.0-fips.patch.
	patchFile := info.Ref + info.Suffix + ".patch"
	content, err = github.GetRaw(ctx, g.Repo, path.Join("patches", info.Name, patchFile), ref)
	if err == nil {
		return []byte(content + "\n"), nil
	}

	// We search for minor with suffix. E.g. 1.29-fips.patch.
	patchFile = minorName + info.Suffix + ".patch"
	content, err = github.GetRaw(ctx, g.Repo, path.Join("patches", info.Name, patchFile), ref)
	if err == nil {
		return []byte(content + "\n"), nil
	}

	// E.g. 1.29.0.patch.
	patchFile = info.Ref + ".patch"
	content, err = github.GetRaw(ctx, g.Repo, path.Join("patches", info.Name, patchFile), ref)
	if err == nil {
		return []byte(content + "\n"), nil
	}

	// We search for minor. E.g. 1.29.patch.
	patchFile = minorName + ".patch"
	content, err = github.GetRaw(ctx, g.Repo, path.Join("patches", info.Name, patchFile), ref)
	if err == nil {
		return []byte(content + "\n"), nil
	}

	return []byte{}, errors.New("patch not found")
}

func (g GitHubGetter) List(ctx context.Context, path, prefix string) ([]Info, error) {
	var list []Info

	names, err := github.GetPatchList(ctx, g.Repo, g.Ref, path, prefix)
	if err != nil {
		return list, err
	}

	for _, name := range names {
		list = append(list, Info{
			Name: name,
			Ref:  g.Ref,
		})
	}

	return list, nil
}

type FSGetter struct {
	Dir string
}

func (g FSGetter) Get(_ context.Context, info Info) ([]byte, error) {
	var (
		content []byte
		err     error
	)

	content, err = os.ReadFile(filepath.Join(g.Dir, info.Name))
	if err == nil {
		return content, nil
	}

	baseDir := filepath.Join(g.Dir, info.Name)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

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

	return []byte{}, errors.New("patch not found")
}

func (f FSGetter) List(ctx context.Context, patchPath, prefix string) ([]Info, error) {
	var list []Info

	entries, err := os.ReadDir(path.Join(f.Dir, patchPath))
	if err != nil {
		return list, err
	}

	slices.SortFunc(entries, func(i, j fs.DirEntry) int {
		return strings.Compare(i.Name(), j.Name())
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), prefix+"-") {
			list = append(list, Info{
				Name: path.Join(patchPath, entry.Name()),
			})
		}
	}

	return list, nil
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

// ApplyDir applies all patches in the patchDir directory with the given prefix into the dst directory.
func ApplyDir(ctx context.Context, patchGetter Getter, patchDir, prefix, dst string) error {

	infos, err := patchGetter.List(ctx, patchDir, prefix)
	if err != nil {
		return err
	}

	for _, info := range infos {
		if err := Apply(ctx, info, patchGetter, dst); err != nil {
			return err
		}
	}

	return nil
}

type Source string

func (s Source) IsLocal() bool {
	return strings.HasPrefix(string(s), "file://")
}

func (s Source) Path() string {
	parsed, err := url.Parse(string(s))
	if err != nil {
		return ""
	}
	p := strings.TrimPrefix(string(s), parsed.Scheme+"://")

	parts := strings.Split(p, "@")
	if len(parts) != 2 {
		return p
	}
	return parts[0]
}

func (s Source) Ref() string {
	parts := strings.Split(string(s), "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
