build_wasm:
	export PATH=$(PATH) CC=$(CC) CXX=$(CXX) && bazel $(BAZEL_STARTUP_ARGS) build $(BAZEL_BUILD_ARGS) $(BAZEL_CONFIG_REL) //extensions:metadata_exchange.wasm
	export PATH=$(PATH) CC=$(CC) CXX=$(CXX) && bazel $(BAZEL_STARTUP_ARGS) build $(BAZEL_BUILD_ARGS) $(BAZEL_CONFIG_REL) @envoy//test/tools/wee8_compile:wee8_compile_tool
	bazel-bin/external/envoy/test/tools/wee8_compile/wee8_compile_tool bazel-bin/extensions/metadata_exchange.wasm bazel-bin/extensions/metadata_exchange.compiled.wasm
