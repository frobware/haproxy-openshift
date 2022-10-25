GO ?= CGO_ENABLED=1 GO111MODULE=on GOFLAGS=-mod=vendor go

perf-test-hydra: generate fmt vet vendor tidy
	$(GO) build -o $@ ./...

generate:
	@$(GO) generate ./...

fmt:
	@$(GO) fmt ./...

vet:
	@$(GO) vet ./...

vendor:
	@$(GO) mod vendor

tidy:
	@$(GO) mod tidy

.PHONY: generate fmt vet vendor tidy
