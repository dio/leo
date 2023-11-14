package istioproxy_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

// build-wasm: istio-proxy-status
// 	export PATH=$(PATH) CC=$(CC) CXX=$(CXX) && bazel $(BAZEL_STARTUP_ARGS) build --override_repository=envoy=/work/envoy-2e4228b0ee73ae640c92e0974c91e251997a3d2f $(BAZEL_BUILD_ARGS) $(BAZEL_CONFIG_REL) //extensions:stats.wasm
// 	export PATH=$(PATH) CC=$(CC) CXX=$(CXX) && bazel $(BAZEL_STARTUP_ARGS) build --override_repository=envoy=/work/envoy-2e4228b0ee73ae640c92e0974c91e251997a3d2f $(BAZEL_BUILD_ARGS) $(BAZEL_CONFIG_REL) //extensions:metadata_exchange.wasm
// 	export PATH=$(PATH) CC=$(CC) CXX=$(CXX) && bazel $(BAZEL_STARTUP_ARGS) build --override_repository=envoy=/work/envoy-2e4228b0ee73ae640c92e0974c91e251997a3d2f $(BAZEL_BUILD_ARGS) $(BAZEL_CONFIG_REL) //extensions:attributegen.wasm
// 	export PATH=$(PATH) CC=$(CC) CXX=$(CXX) && bazel $(BAZEL_STARTUP_ARGS) build --override_repository=envoy=/work/envoy-2e4228b0ee73ae640c92e0974c91e251997a3d2f $(BAZEL_BUILD_ARGS) $(BAZEL_CONFIG_REL) @envoy//test/tools/wee8_compile:wee8_compile_tool
// 	bazel-bin/external/envoy/test/tools/wee8_compile/wee8_compile_tool bazel-bin/extensions/stats.wasm bazel-bin/extensions/stats.compiled.wasm
// 	bazel-bin/external/envoy/test/tools/wee8_compile/wee8_compile_tool bazel-bin/extensions/metadata_exchange.wasm bazel-bin/extensions/metadata_exchange.compiled.wasm
// 	bazel-bin/external/envoy/test/tools/wee8_compile/wee8_compile_tool bazel-bin/extensions/attributegen.wasm bazel-bin/extensions/attributegen.compiled.wasm

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
