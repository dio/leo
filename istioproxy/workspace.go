package istioproxy

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/bazelbuild/buildtools/build"
)

type Envoy struct {
	SHA     string
	SHA256  string
	Org     string
	Repo    string
	Version string
}

func EnvoyFromWorkspace(workspace string) (*Envoy, error) {
	scanner := bufio.NewScanner(strings.NewReader(workspace))
	values := map[string]string{
		"ENVOY_SHA":    "",
		"ENVOY_SHA256": "",
		"ENVOY_ORG":    "",
		"ENVOY_REPO":   "",
	}

	for scanner.Scan() {
		line := scanner.Text()
		for key := range values {
			if strings.Contains(line, key+" = ") {
				re, err := regexp.Compile(fmt.Sprintf(`%s = "([^"]+)"`, key))
				if err != nil {
					return nil, err
				}
				values[key] = re.FindStringSubmatch(line)[1]
				break
			}
		}
	}

	repo := &Envoy{
		SHA:    values["ENVOY_SHA"],
		SHA256: values["ENVOY_SHA256"],
		Org:    values["ENVOY_ORG"],
		Repo:   values["ENVOY_REPO"],
	}

	return repo, nil
}

func istioProxyEnvoyBinaryTarget(dir string) (string, string, error) {
	// First find BUILD in the "current" dir.
	src := filepath.Join(dir, "src", "envoy", "BUILD")
	if _, err := os.Stat(src); err == nil {
		// Parse and get the "target" string
		data, err := os.ReadFile(src)
		if err == nil {
			f, err := build.ParseBuild("BUILD", data)
			if err == nil {
				for _, rule := range f.Rules("envoy_cc_binary") {
					if rule.Name() == "envoy" {
						return "//src/envoy:envoy", "bazel-bin/src/envoy/envoy", nil
					}
				}
			}
		}
	}

	return "//:envoy", "bazel-bin/envoy", nil
}

func writeNamedWorkspaceStatus(name, revision, status, proxyDir string) error {
	content := fmt.Sprintf(`#!/bin/bash
echo "BUILD_SCM_REVISION %s"
echo "BUILD_SCM_STATUS Distribution/%s"
echo "BUILD_CONFIG Release"
`, revision, status)
	return os.WriteFile(filepath.Join(proxyDir, "bazel", "bazel_get_workspace_status_"+name), []byte(content), os.ModePerm)
}

func WriteWorkspaceStatus(proxyDir, envoyName, envoySHA string) error {
	status := "istio/proxy"
	if envoyName != "envoyproxy/envoy" {
		status = path.Join(status, envoyName, envoySHA[0:7])
	}
	if err := writeNamedWorkspaceStatus("istio-proxy", strings.TrimPrefix(filepath.Base(proxyDir), "proxy-"), status, proxyDir); err != nil {
		return err
	}

	if err := writeNamedWorkspaceStatus("envoy", envoySHA, envoyName, proxyDir); err != nil {
		return err
	}

	if err := writeNamedWorkspaceStatus("envoy-contrib", envoySHA, envoyName+"/contrib", proxyDir); err != nil {
		return err
	}

	return nil
}

type TargetOptions struct {
	ProxyDir     string
	ProxySHA     string
	EnvoyDir     string
	EnvoySHA     string
	IstioVersion string
	EnvoyVersion string
	RemoteCache  string // Remote cache values: us-central1 or asia-south2.
	FIPSBuild    bool
	Gperftools   bool
	Wasm         bool
	EnvoyRepo    string
}

func PrepareBuilder(proxyDir, remote string) error {
	var remoteCache string
	if len(remote) > 0 {
		remoteCache = "--google_default_credentials --remote_cache=https://storage.googleapis.com/tetrate-istio-subscription-proxy-builder-" + remote
	}

	if err := os.WriteFile(filepath.Join(proxyDir, "common", "scripts", "Dockerfile"),
		[]byte(`# Generated.
ARG IMG

FROM ubuntu:20.04 AS linux_headers_amd64
RUN apt-get -q update && apt-get install -yqq --no-install-recommends curl ca-certificates
RUN curl -sSLO https://github.com/Kitware/CMake/releases/download/v3.29.2/cmake-3.29.2-linux-x86_64.tar.gz && \
  tar -xzf cmake-3.29.2-linux-x86_64.tar.gz -C /usr/  --strip-components=1 && \
  rm cmake-3.29.2-linux-x86_64.tar.gz

FROM ubuntu:20.04 AS linux_headers_arm64
RUN apt-get -q update && apt-get install -yqq --no-install-recommends curl ca-certificates
RUN curl -sSLO https://github.com/Kitware/CMake/releases/download/v3.29.2/cmake-3.29.2-linux-aarch64.tar.gz && \
  tar -xzvf cmake-3.29.2-linux-aarch64.tar.gz -C /usr/ --strip-components=1 && \
  rm cmake-3.29.2-linux-aarch64.tar.gz


FROM linux_headers_${TARGETARCH} AS linux_headers
RUN apt-get -q update && apt-get install -yqq --no-install-recommends linux-libc-dev

FROM $IMG
COPY --from=linux_headers /usr/include/linux/tcp.h /usr/include/linux/tcp.h
# Get cmake from kitware
COPY --from=linux_headers /usr/bin/cmake /usr/bin/cmake
COPY --from=linux_headers /usr/share/cmake-3.29 /usr/share/cmake-3.29

RUN su-exec 0:0 apt-get -o APT::Get::AllowUnauthenticated=true -o Acquire::AllowInsecureRepositories=true -o Acquire::AllowDowngradeToInsecureRepositories=true -q update && \
  su-exec 0:0 apt-get -o APT::Get::AllowUnauthenticated=true -o Acquire::AllowInsecureRepositories=true -o Acquire::AllowDowngradeToInsecureRepositories=true install -yqq --no-install-recommends rsync cmake libtinfo6

ENV BAZEL_BUILD_ARGS="`+remoteCache+`"`),
		os.ModePerm); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(proxyDir, "common", "scripts", "setup_env.sh"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = f.WriteString(`
# Generated.
docker build --build-arg="IMG=${IMG}" "${SCRIPT_DIR}" -t leo/builder:1
IMG="leo/builder:1"
`)

	return err
}

func AddMakeTargets(opts TargetOptions) error {
	f, err := os.OpenFile(filepath.Join(opts.ProxyDir, "Makefile.core.mk"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	istioProxyTarget, err := IstioProxyTarget(opts)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(istioProxyTarget); err != nil {
		return err
	}

	istioProxyCentos7Target, err := IstioProxyCentos7Target(opts)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(istioProxyCentos7Target); err != nil {
		return err
	}

	envoyTarget, err := EnvoyTarget(opts)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(envoyTarget); err != nil {
		return err
	}

	envoyCentos7Target, err := EnvoyCentos7Target(opts)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(envoyCentos7Target); err != nil {
		return err
	}

	envoyContribTarget, err := EnvoyContribTarget(opts)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(envoyContribTarget); err != nil {
		return err
	}

	return nil
}

const configLibcxx = "build --config=libc++"

func buildConfigFlags(proxyDir string, gperftools bool) (string, error) {
	var gperftoolsFlag string
	if gperftools {
		gperftoolsFlag = " --define=tcmalloc=gperftools "
	}
	// build --config=libc++20
	data, err := os.ReadFile(filepath.Join(proxyDir, ".bazelrc"))
	if err != nil {
		return "", err
	}
	text := string(data)
	// When "configLibcxx" is enabled in proxy's .bazelrc, we don't need to specify it again.
	// This to remedy: WARNING: The following configs were expanded more than once: [libc++, clang]. For repeatable flags, repeats are counted twice and may lead to unexpected behavior.
	if strings.Contains(text, configLibcxx) {
		return "--config=release" + gperftoolsFlag, nil
	}

	// For older version, we need to explicitly enable --config=libc++.
	return "--config=release --config=libc++" + gperftoolsFlag, nil
}

func IstioProxyCentos7Target(opts TargetOptions) (string, error) {
	target, binaryPath, err := istioProxyEnvoyBinaryTarget(opts.ProxyDir)
	if err != nil {
		return "", err
	}

	buildConfig, err := buildConfigFlags(opts.ProxyDir, opts.Gperftools)
	if err != nil {
		return "", err
	}

	var remoteCache string
	if len(opts.RemoteCache) > 0 {
		remoteCache = "--google_default_credentials --remote_cache=https://storage.googleapis.com/tetrate-istio-subscription-proxy-builder-" + opts.RemoteCache
	}

	var boringssl string
	if opts.FIPSBuild {
		boringssl = "--define=boringssl=fips"
	}

	var ldLibraryPath string
	if runtime.GOARCH == "amd64" {
		// TODO(dio): Normalize this. gcr.io/tetratelabs/envoy-build-centos:1 has intalled llvm in /opt/llvm vs. /usr/lib/llvm.
		ldLibraryPath = "--action_env=LD_LIBRARY_PATH=/opt/llvm/lib/x86_64-unknown-linux-gnu --host_action_env=LD_LIBRARY_PATH=/opt/llvm/lib/x86_64-unknown-linux-gnu"
	}

	var targzSuffix string
	if opts.FIPSBuild {
		targzSuffix = "-fips"
	}
	targz := "istio-proxy-centos7" + targzSuffix + "-" + runtime.GOARCH + ".tar.gz"
	content := `
istio-proxy-centos7-status:
	cp -f bazel/bazel_get_workspace_status_istio-proxy bazel/bazel_get_workspace_status

istio-proxy-centos7: istio-proxy-centos7-status
	bazel build %s %s --stamp --override_repository=envoy=/work%s %s %s %s
	mkdir -p /work/out/usr/local/bin
	cp -f %s /work/out/usr/local/bin/envoy
	tar -czf /work/out/%s -C /work/out usr
	rm -fr /work/out/usr
`
	return fmt.Sprintf(content,
		buildConfig,
		boringssl,
		strings.Replace(opts.EnvoyDir, opts.ProxyDir, "", 1),
		target+".stripped",
		remoteCache,
		ldLibraryPath,
		binaryPath+".stripped",
		targz), nil
}

func IstioProxyTarget(opts TargetOptions) (string, error) {
	target, binaryPath, err := istioProxyEnvoyBinaryTarget(opts.ProxyDir)
	if err != nil {
		return "", err
	}

	buildConfig, err := buildConfigFlags(opts.ProxyDir, opts.Gperftools)
	if err != nil {
		return "", err
	}

	var remoteCache string
	if len(opts.RemoteCache) > 0 {
		remoteCache = "--google_default_credentials --remote_cache=https://storage.googleapis.com/tetrate-istio-subscription-proxy-builder-" + opts.RemoteCache
	}

	var boringssl string
	if opts.FIPSBuild {
		boringssl = "--define=boringssl=fips"
	}

	var ldLibraryPath string
	if runtime.GOARCH == "amd64" {
		ldLibraryPath = "--action_env=LD_LIBRARY_PATH=/usr/lib/llvm/lib/x86_64-unknown-linux-gnu --host_action_env=LD_LIBRARY_PATH=/usr/lib/llvm/lib/x86_64-unknown-linux-gnu"
	}

	wasmTarget := buildWasmTarget(filepath.Join(opts.ProxyDir, "Makefile.core.mk"), strings.Replace(opts.EnvoyDir, opts.ProxyDir, "", 1))
	var buildWasm string
	var copyWasm string
	if opts.Wasm && len(wasmTarget) > 0 {
		buildWasm = "build-wasm"
		copyWasm = `
	cp -f bazel-bin/extensions/*.wasm /work/out/
`
	}

	var targzSuffix string
	if opts.FIPSBuild {
		targzSuffix = "-fips"
	}
	targz := "istio-proxy" + targzSuffix + "-" + runtime.GOARCH + ".tar.gz"
	content := `
istio-proxy-status:
	cp -f bazel/bazel_get_workspace_status_istio-proxy bazel/bazel_get_workspace_status

istio-proxy: istio-proxy-status %s
	bazel build %s %s --stamp --override_repository=envoy=/work%s %s %s %s
	mkdir -p /work/out/usr/local/bin
	cp -f %s /work/out/usr/local/bin/envoy
	tar -czf /work/out/%s -C /work/out usr
	rm -fr /work/out/usr
%s
`
	if opts.Wasm && len(wasmTarget) > 0 {
		content += buildWasmTarget(filepath.Join(opts.ProxyDir, "Makefile.core.mk"), strings.Replace(opts.EnvoyDir, opts.ProxyDir, "", 1))
	}

	return fmt.Sprintf(content,
		buildWasm,
		buildConfig,
		boringssl,
		strings.Replace(opts.EnvoyDir, opts.ProxyDir, "", 1),
		target+".stripped",
		remoteCache,
		ldLibraryPath,
		binaryPath+".stripped",
		targz,
		copyWasm,
	), nil
}

func EnvoyTarget(opts TargetOptions) (string, error) {
	target := "@envoy//source/exe:envoy-static.stripped"
	binaryPath := "bazel-bin/external/envoy/source/exe/envoy-static.stripped"
	var remoteCache string
	if len(opts.RemoteCache) > 0 {
		remoteCache = "--google_default_credentials --remote_cache=https://storage.googleapis.com/tetrate-istio-subscription-proxy-builder-" + opts.RemoteCache
	}

	buildConfig, err := buildConfigFlags(opts.ProxyDir, opts.Gperftools)
	if err != nil {
		return "", err
	}

	var boringssl string
	if opts.FIPSBuild {
		boringssl = "--define=boringssl=fips"
	}
	// Write a WORKSPACE to source/extensions.
	if err := os.WriteFile(filepath.Join(opts.EnvoyDir, "source", "extensions", "WORKSPACE"), []byte{}, os.ModePerm); err != nil {
		return "", err
	}

	var ldLibraryPath string
	if runtime.GOARCH == "amd64" {
		ldLibraryPath = "--action_env=LD_LIBRARY_PATH=/usr/lib/llvm/lib/x86_64-unknown-linux-gnu --host_action_env=LD_LIBRARY_PATH=/usr/lib/llvm/lib/x86_64-unknown-linux-gnu"
	}

	var targzSuffix string
	if opts.FIPSBuild {
		targzSuffix = "-fips"
	}
	targz := "envoy" + targzSuffix + "-" + runtime.GOARCH + ".tar.gz"
	content := `
envoy-status:
	cp -f bazel/bazel_get_workspace_status_envoy bazel/bazel_get_workspace_status

envoy: envoy-status
	bazel build %s %s --stamp --override_repository=envoy=/work%s --override_repository=envoy_build_config=/work%s %s %s %s
	mkdir -p /work/out
	cp -f %s %s/envoy
	tar -czf /work/out/%s -C %s envoy
`
	return fmt.Sprintf(content,
		buildConfig,
		boringssl,
		strings.Replace(opts.EnvoyDir, opts.ProxyDir, "", 1),
		strings.Replace(filepath.Join(opts.EnvoyDir, "source", "extensions"), opts.ProxyDir, "", 1),
		target,
		remoteCache,
		ldLibraryPath,

		// Rename binary.
		binaryPath,
		filepath.Dir(binaryPath),

		// tar -czf.
		targz,
		filepath.Dir(binaryPath)), nil
}

func EnvoyCentos7Target(opts TargetOptions) (string, error) {
	target := "@envoy//source/exe:envoy-static.stripped"
	binaryPath := "bazel-bin/external/envoy/source/exe/envoy-static.stripped"
	var remoteCache string
	if len(opts.RemoteCache) > 0 {
		remoteCache = "--google_default_credentials --remote_cache=https://storage.googleapis.com/tetrate-istio-subscription-proxy-builder-" + opts.RemoteCache
	}

	buildConfig, err := buildConfigFlags(opts.ProxyDir, opts.Gperftools)
	if err != nil {
		return "", err
	}

	var boringssl string
	if opts.FIPSBuild {
		boringssl = "--define=boringssl=fips"
	}
	// Write a WORKSPACE to source/extensions.
	if err := os.WriteFile(filepath.Join(opts.EnvoyDir, "source", "extensions", "WORKSPACE"), []byte{}, os.ModePerm); err != nil {
		return "", err
	}

	var ldLibraryPath string
	if runtime.GOARCH == "amd64" {
		ldLibraryPath = "--action_env=LD_LIBRARY_PATH=/usr/lib/llvm/lib/x86_64-unknown-linux-gnu --host_action_env=LD_LIBRARY_PATH=/usr/lib/llvm/lib/x86_64-unknown-linux-gnu"
	}

	var targzSuffix string
	if opts.FIPSBuild {
		targzSuffix = "-fips"
	}
	targz := "envoy" + targzSuffix + "-" + runtime.GOARCH + ".tar.gz"
	content := `
envoy-centos7-status:
	cp -f bazel/bazel_get_workspace_status_envoy bazel/bazel_get_workspace_status

envoy-centos7: envoy-centos7-status
	bazel build %s %s --stamp --override_repository=envoy=/work%s --override_repository=envoy_build_config=/work%s %s %s %s
	mkdir -p /work/out
	cp -f %s %s/envoy
	tar -czf /work/out/%s -C %s envoy
`
	return fmt.Sprintf(content,
		buildConfig,
		boringssl,
		strings.Replace(opts.EnvoyDir, opts.ProxyDir, "", 1),
		strings.Replace(filepath.Join(opts.EnvoyDir, "source", "extensions"), opts.ProxyDir, "", 1),
		target,
		remoteCache,
		ldLibraryPath,

		// Rename binary.
		binaryPath,
		filepath.Dir(binaryPath),

		// tar -czf.
		targz,
		filepath.Dir(binaryPath)), nil
}

func EnvoyContribTarget(opts TargetOptions) (string, error) {
	target := "@envoy//contrib/exe:envoy-static.stripped"
	binaryPath := "bazel-bin/external/envoy/contrib/exe/envoy-static.stripped"
	var remoteCache string
	if len(opts.RemoteCache) > 0 {
		remoteCache = "--google_default_credentials --remote_cache=https://storage.googleapis.com/tetrate-istio-subscription-proxy-builder-" + opts.RemoteCache
	}

	buildConfig, err := buildConfigFlags(opts.ProxyDir, opts.Gperftools)
	if err != nil {
		return "", err
	}

	var boringssl string
	if opts.FIPSBuild {
		boringssl = "--define=boringssl=fips"
	}

	var ldLibraryPath string
	if runtime.GOARCH == "amd64" {
		ldLibraryPath = "--action_env=LD_LIBRARY_PATH=/usr/lib/llvm/lib/x86_64-unknown-linux-gnu --host_action_env=LD_LIBRARY_PATH=/usr/lib/llvm/lib/x86_64-unknown-linux-gnu"
	}

	// TODO(dio): Allow to disable some contrib extenstions, since it is problematic with clang-12.

	var targzSuffix string
	if opts.FIPSBuild {
		targzSuffix = "-fips"
	}
	targz := "envoy-contrib" + targzSuffix + "-" + runtime.GOARCH + ".tar.gz"
	content := `
envoy-contrib-status:
	cp -f bazel/bazel_get_workspace_status_envoy-contrib bazel/bazel_get_workspace_status

envoy-contrib: envoy-contrib-status
	bazel build %s %s --stamp --override_repository=envoy=/work%s %s %s %s
	mkdir -p /work/out
	cp -f %s %s/envoy
	tar -czf /work/out/%s -C %s envoy
`
	return fmt.Sprintf(content,
		buildConfig,
		boringssl,
		strings.Replace(opts.EnvoyDir, opts.ProxyDir, "", 1),
		target,
		remoteCache,
		ldLibraryPath,

		// Rename binary.
		binaryPath,
		filepath.Dir(binaryPath),

		// tar -czf.
		targz,
		filepath.Dir(binaryPath)), nil
}

func buildWasmTarget(makefileCoreMk, override string) string {
	b, err := os.ReadFile(makefileCoreMk)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	var target string
	var start bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "build_wasm:") && !start {
			start = true
		}

		if start {
			if len(strings.TrimSpace(line)) == 0 ||
				// Or if we found another target.
				(strings.HasSuffix(strings.TrimSpace(line), ":") && len(target) > 0) {
				break
			}
			target += line + "\n"
		}
	}
	target = strings.Replace(target, "build_wasm:", "build-wasm: istio-proxy-status", 1)
	target = strings.ReplaceAll(target, "$(BAZEL_BUILD_ARGS)", "$(BAZEL_BUILD_ARGS) --override_repository=envoy=/work"+override)

	return target
}
