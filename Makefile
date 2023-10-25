VERSION ?= dev
ARCH ?= amd64

release:
	@GOARCH=$(ARCH) GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" .
	@tar -czf leo-$(VERSION)-$(ARCH).tar.gz leo
