package istio

import (
	"encoding/json"
	"fmt"

	"github.com/dio/leo/github"
)

// Deps lists all deps.
type Deps []Dep

// Get gets dep by name.
func (d Deps) Get(name string) Dep {
	for _, item := range d {
		if item.Name == name {
			return item
		}
	}
	return Dep{}
}

// Dep holds dep.
type Dep struct {
	Name string `json:"repoName"`
	SHA  string `json:"lastStableSHA"`
}

// GetDeps gets deps of a specific istio revision.
func GetDeps(ref string) (Deps, error) {
	data, err := github.GetRaw("istio/istio", "istio.deps", ref)
	if err != nil {
		return Deps{}, fmt.Errorf("failed to get istio.deps: %w", err)
	}
	var deps Deps
	if err := json.Unmarshal([]byte(data), &deps); err != nil {
		return Deps{}, err
	}
	return deps, nil
}
