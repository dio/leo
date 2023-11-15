package istioproxy_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestExtractBuildWasm(t *testing.T) {
	b, _ := os.ReadFile("testdata/Makefile.core.mk")
	scanner := bufio.NewScanner(strings.NewReader(string(b)))

	var target string
	var start bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "build_wasm:") && !start {
			start = true
		}

		if start && len(strings.TrimSpace(line)) > 0 {
			target += line + "\n"
		}
	}

	target = strings.Replace(target, "build_wasm:", "build-wasm: istio-proxy-status", 1)
	target = strings.ReplaceAll(target, "$(BAZEL_BUILD_ARGS)", "$(BAZEL_BUILD_ARGS) --override_repository=envoy=/work/envoy-2e4228b0ee73ae640c92e0974c91e251997a3d2f")
	fmt.Println(target)
}
