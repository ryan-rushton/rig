.PHONY: build install run clean test lint fmt check

BINARY := rig
BIN_DIR := ./bin

## build: compile the binary
build:
	go build -o $(BIN_DIR)/$(BINARY) .

## install: install to $GOPATH/bin
install:
	go install .

## run: run without building (pass args via ARGS=)
run:
	go run . $(ARGS)

## test: run all tests
test:
	go test ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: format and fix lint issues in-place
fmt:
	golangci-lint run --fix ./...

## check: fmt + lint + test (use before committing)
check: fmt lint test

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR) dist

## help: show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
