BINARY_NAME=mrmintchain
BINARY_CHAIN_NAME=ethermintd
BUILD_PATH=./cmd

GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif

INSTALL_PATH := $(GOBIN)/$(BINARY_NAME)
INSTALL_CHAIN_PATH := $(GOBIN)/$(BINARY_CHAIN_NAME)

.PHONY: all build install clean

all: build

build:
	go build -o $(BINARY_NAME) $(BUILD_PATH)

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)"
	@mkdir -p $(GOBIN)
	@cp $(BINARY_NAME) $(INSTALL_PATH)

	@echo "Installing $(BINARY_CHAIN_NAME) to $(INSTALL_CHAIN_PATH)"
	@cp chain/$(BINARY_CHAIN_NAME) $(INSTALL_CHAIN_PATH)


clean:
	rm -f $(BINARY_NAME)