GO ?= GO111MODULE=auto GOFLAGS=-mod=vendor go

perf-test-hydra: gen fmt vet vendor tidy
	CGOENABLED=1 $(GO) build -x -o $@ ./...

gen:
	$(GO) generate ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

vendor:
	$(GO) mod vendor

tidy:
	$(GO) mod tidy

.PHONY: perf-test-hydra gen fmt vet vendor tidy
