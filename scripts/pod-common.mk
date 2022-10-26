.PHONY: build
build: fmt vet test ## Build pod executable binary
	go build -o bin/${EXECUTABLE_NAME} main.go

.PHONY: run
run: build ## Run the pod executable from your host.
	go run ./main.go

.PHONY: test
test: fmt vet ## Run pod test tests for the pods
	go test ./... -coverprofile cover.out
