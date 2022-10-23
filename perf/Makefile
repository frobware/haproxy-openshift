GO_FILES := $(wildcard ./cmd/perf-test-hydra/*.go)

# run:
# 	CGOENABLED=0 go run ./cmd/perf-test-hydra/main.go $(filter-out main.go,$(GO_FILES))

perf-test-hydra-race: gen fmt vet
	CGOENABLED=1 go build -race -o $@ $(GO_FILES)

gen:
	@go generate ./...

fmt:
	@go fmt ./...

vet:
	@go vet ./...

.PHONY: run perf-test-hydra gen fmt vet
