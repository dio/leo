package istio

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

type BaseImage struct {
	Registry string `json:"registry"`
	Version  string `json:"version"`
}

func BaseImageFromMakefileCore(makefile string) (*BaseImage, error) {
	scanner := bufio.NewScanner(strings.NewReader(makefile))
	values := map[string]string{
		"BASE_VERSION":        "",
		"ISTIO_BASE_REGISTRY": "",
	}
	for scanner.Scan() {
		line := scanner.Text()
		for key := range values {
			if strings.Contains(line, key+" ?= ") {
				re, err := regexp.Compile(fmt.Sprintf(`%s\s*\?\s*=\s*(.*)`, key))
				if err != nil {
					return nil, err
				}
				values[key] = re.FindStringSubmatch(line)[1]
				break
			}
		}
	}
	return &BaseImage{
		Version:  values["BASE_VERSION"],
		Registry: values["ISTIO_BASE_REGISTRY"],
	}, nil
}
