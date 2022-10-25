CGOENABLED=1
export CGOENABLED

perf-test-hydra: gen fmt vet tidy
	go build -o $@ ./...

gen:
	@go generate ./...

fmt:
	@go fmt ./...

vet:
	@go vet ./...

tidy:
	@go mod tidy

.PHONY: perf-test-hydra gen fmt vet tidy
