package arg

import "path/filepath"

type Repo string

func (r Repo) Name() string {
	return filepath.Base(string(r))
}

func (r Repo) Owner() string {
	owner := filepath.Dir(string(r))
	if owner == "." {
		return ""
	}
	return owner
}
