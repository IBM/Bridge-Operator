# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
VERSION := 0.0.1
IMAGE_TAG_BASE :=  quay.io/ibmdpdev/ray-pod

# Executable name
EXECUTABLE_NAME := ray-pod

# Define the repo root
REPO_ROOT := $(abspath ../..)
include $(REPO_ROOT)/scripts/common.mk
include $(REPO_ROOT)/scripts/docker-common.mk
