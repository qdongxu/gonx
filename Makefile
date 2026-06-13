.PHONY: build test fmt lint vet clean

BINARY := gonx
CMD_DIR := ./cmd/gonx

build:
	go build -o $(BINARY) $(CMD_DIR)

test:
	go test -v ./...

fmt:
	go fmt ./...

lint: fmt vet
	@echo "Lint complete. Add golangci-lint when ready."

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

.DEFAULT_GOAL := build
