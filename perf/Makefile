GO ?= go

perf-test-hydra: generate fmt vet
	CGO_ENABLED=1 GO111MODULE=on GOFLAGS=-mod=vendor go build -o $@ ./...

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

release: vendor tidy

.PHONY: generate fmt vet vendor tidy release
