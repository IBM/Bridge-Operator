##@ Common golang build targets
# GO Version
GO_VERSION = 1.18

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: ## Run go linter against code
	golangci-lint run

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy -go=${GO_VERSION}

.PHONY: change-go-version
change-go-version:
        go mod edit -go=${GO_VERSION}
        go mod tidy -go=${GO_VERSION}

.PHONY: upgrade-dependencies
upgrade-dependencies: ## Update golang module dependencies
	go get -u ./...
	go mod tidy -go=${GO_VERSION}