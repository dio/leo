# Version can be defined:
# (1) in a $VERSION shell variable, which takes precedence; or
# (2) in the VERSION file, in which we will append "-dev" to it
ifeq ($(VERSION),)
VERSION_FROM_FILE := $(shell cat VERSION)
ifeq ($(VERSION_FROM_FILE),)
$(error VERSION not detected. Make sure it's stored in the VERSION file or defined in VERSION variable)
endif
VERSION := $(VERSION_FROM_FILE)-dev
endif

export VERSION

# Base version of Istio image to use
BASE_VERSION ?= master-2023-10-12T19-01-47
ISTIO_BASE_REGISTRY ?= gcr.io/istio-release

export GO111MODULE ?= on
export GOPROXY ?= https://proxy.golang.org
export GOSUMDB ?= sum.golang.org
