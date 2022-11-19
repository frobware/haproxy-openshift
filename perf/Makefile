GO ?= go

perf-test-hydra: generate fmt vet
	# Default to CGO_ENABLED=1 so that we can use ./dns/libmydns.so
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
	@$(GO) run honnef.co/go/tools/cmd/staticcheck@latest ./...

.PHONY: generate fmt vet vendor tidy release
