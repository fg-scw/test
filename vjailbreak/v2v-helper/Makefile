SHELL :=  /usr/bin/env bash
PKG_CONFIG_PATH ?= /usr/local/lib/pkgconfig
CGO_CFLAGS := $(shell PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) pkg-config --cflags libnbd)
CGO_LDFLAGS := $(shell PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) pkg-config --libs libnbd)

LDFLAGS += "-s -w -X github.com/platform9/vjailbreak/v2v-helper/pkg/version.Version=$(RELEASE_VERSION)"
BUILD_OPTIONS += -ldflags=$(LDFLAGS) -buildvcs=false

export CGO_CFLAGS
export CGO_LDFLAGS
export LD_LIBRARY_PATH := /usr/local/lib:$(LD_LIBRARY_PATH)

all: build test
.PHONY: all build test

build:
	@echo "Building v2v-helper..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(BUILD_OPTIONS) -a -o manager *.go

gen-mock:
	go generate ./...

test:
	@echo "Running tests for v2v-helper"
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go test ./... -v