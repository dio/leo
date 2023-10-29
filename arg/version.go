package arg

import (
	"strings"
)

type Version string

func (v Version) IsEmpty() bool {
	return len(string(v)) == 0
}

func (v Version) Name() string {
	parts := v.parse()
	return parts[0]
}

func (v Version) Repo() Repo {
	parts := v.parse()
	return Repo(parts[0])
}

func (v Version) Version() string {
	parts := v.parse()
	return parts[1]
}

func (v Version) VersionV() string {
	parts := v.parse()
	if !strings.HasPrefix(parts[1], "v") {
		return "v" + parts[1]
	}
	return parts[1]
}

func (v Version) parse() []string {
	str := string(v)
	return strings.Split(str, "@")
}
