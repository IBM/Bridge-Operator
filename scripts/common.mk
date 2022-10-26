# This file is included in other Makefiles. It provides universal, shared definitions.
# Import this file before any other "*.mk" files. Defining REPO_ROOT is a convenient way
# to provide the absolute path to the repo's root directory:
#   REPO_ROOT := $(abspath ../..)
#   include $(REPO_ROOT)/tools/common.mk
# More than convenient, REPO_ROOT is actually required, as sometimes the path is needed
# to locate required tools used during builds. If not defined, an error is thrown below.
#
# The definitions below can be overridden when appropriate in each Makefile:
# MAKEFLAGS
#   Unless there is a good reason to override the definition, use += as shown here.
#
# MAKEFLAGS_RECURSIVE
#   Flags passed to make when doing the recursive invocations of make.
#
# SUBDIRS
#   The child directories to visit when building "recursive" targets like "all".
#   By default, this variable contains the list of child directories with Makefiles,
#   sorted alphabetically. You might need to override the variable definition to
#   to impose a different traversal order!
MAKEFLAGS           += --warn-undefined-variables
MAKEFLAGS_RECURSIVE ?= --print-directory
DOCKER_CMD          ?= docker

# Used for version tagging release artifacts.
GIT_HASH            ?= $(shell git show --pretty="%H" --abbrev-commit |head -1)
TIMESTAMP           ?= $(shell date +"%Y%m%d-%H%M%S")

# Sed is used to strip leading "./" and "/Makefile" leaving the directory name:
SUBDIRS := $(shell find . -name Makefile -mindepth 2 -maxdepth 2 | sed -e 's?^./??' -e 's?/Makefile??' | sort)

# The targets:
# Note that for convenience, singular and plural names for some common target names
# are provided.

# If the user doesn't specify a target on invocation, always default to this one:
.DEFAULT_GOAL := all

# Error and warning messages:
define repo_root_error_message
  ERROR: REPO_ROOT must be defined in the Makefile ${PWD}/Makefile.
endef

define all_todo_warning_message
  WARNING: All make targets are TODO in ${PWD}/Makefile
endef

define executable_name_error_message
  ERROR: EXECUTABLE_NAME must be defined in the Makefile ${PWD}/Makefile.
endef
define  image_tag_base_error_message
  ERROR: IMAGE_TAG_BASE must be defined in the Makefile: ${PWD}/Makefile.
endef

ifndef REPO_ROOT
$(error ${repo_root_error_message})
endif

ifndef EXECUTABLE_NAME
$(error ${executable_name_error_message})
endif

ifndef IMAGE_TAG_BASE
$(error ${image_tag_base_error_message})
endif

ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

export PATH := $(MYGOBIN):$(PATH)
export MYGOBIN=${GOBIN}


# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General
.PHONY: all
all: build


# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: clean
clean : ## Remove the executables
	rm -f bin/${EXECUTABLE_NAME} bin/${EXECUTABLE_NAME}_test cover.out 
