##@ Development Tools Setup

.PHONY: install-tools 
install-tools: download-tools  ## Download tools needed for development.

# Hack to allow for the continuation characters to be used and the help to be
# printed
.PHONY: download-tools
download-tools: \
	$(MYGOBIN)/controller-gen \
	$(MYGOBIN)/kustomize \
	$(MYGOBIN)/setup-envtest \
	$(MYGOBIN)/golangci-lint \
	$(MYGOBIN)/opm \
	$(MYGOBIN)/operator-sdk

.PHONY: uninstall-tools
uninstall-tools: ## Remove tools needed for development.
	rm -f $(MYGOBIN)/controller-gen
	rm -f $(MYGOBIN)/kustomize
	rm -f $(MYGOBIN)/setup-envtest
	rm -f $(MYGOBIN)/golangci-lint
	rm -f $(MYGOBIN)/opm
	rm -f $(MYGOBIN)/operator-sdk

$(MYGOBIN)/controller-gen: 
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0 

$(MYGOBIN)/kustomize :
	go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.5

$(MYGOBIN)/setup-envtest :
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

$(MYGOBIN)/golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.47.2

$(MYGOBIN)/opm :
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sfSLo $(MYGOBIN)/opm https://github.com/operator-framework/operator-registry/releases/download/v1.24.0/$${OS}-$${ARCH}-opm && \
	chmod +x $(MYGOBIN)/opm 

$(MYGOBIN)/operator-sdk :
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -fsSLo $(MYGOBIN)/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/v1.22.1/operator-sdk_$${OS}_$${ARCH} && \
	chmod +x $(MYGOBIN)/operator-sdk 

