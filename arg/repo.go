package arg

import "path/filepath"

type Repo string

func (r Repo) Name() string {
	return filepath.Base(string(r))
}

func (r Repo) Owner() string {
	return filepath.Dir(string(r))
}
