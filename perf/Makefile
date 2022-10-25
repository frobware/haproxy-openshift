GO ?= CGO_ENABLED=0 GOFLAGS=-mod=readonly GO111MODULE=auto go

perf-test-hydra: generate fmt vet
	$(GO) build -mod=vendor -o $@ ./...

generate:
	@$(GO) generate ./...

fmt:
	@$(GO) fmt ./...

vet:
	@$(GO) vet ./...

vendor:
	@$(GO) mod vendor
	@$(GO) mod tidy

tidy:
	@$(GO) mod tidy

.PHONY: generate fmt vet vendor tidy
