
# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
VERSION := 0.0.1
IMAGE_TAG_BASE :=  quay.io/ibmdpdev/slurm-pod
DOCKERFILE := Dockerfile_slurm


# Executable name
EXECUTABLE_NAME := slurm-pod

# Define the repo root
REPO_ROOT := $(abspath ../..)
include ${REPO_ROOT}/scripts/tools.mk
include $(REPO_ROOT)/scripts/common.mk
include ${REPO_ROOT}/scripts/go-common.mk
include ${REPO_ROOT}/scripts/pod-common.mk
include ${REPO_ROOT}/scripts/docker-common.mk
