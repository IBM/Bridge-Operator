# Executable name
EXECUTABLE_NAME := dummy
# Image base name
IMAGE_TAG_BASE :=  dummy

# Define the repo root
REPO_ROOT := $(abspath ../..)
include ${REPO_ROOT}/scripts/tools.mk
include $(REPO_ROOT)/scripts/common.mk
include ${REPO_ROOT}/scripts/go-common.mk
.PHONY: build
build: fmt vet test ## Build go module
	go build ./...


.PHONY: test
test: fmt vet ## Run tests for the module
	go test ./... -coverprofile cover.out

