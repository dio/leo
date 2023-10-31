VERSION ?= dev
ARCH ?= arm64
OS ?= linux

tarball:
	@GOARCH=$(ARCH) GOOS=$(OS) CGO_ENABLED=0 go build -ldflags="-s -w -X 'main.version=$(VERSION)'" .
	@tar -czf leo-$(VERSION)-$(ARCH).tar.gz leo
