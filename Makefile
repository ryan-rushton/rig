.PHONY: build install run clean

BINARY := rig
BIN_DIR := ./bin

build:
	go build -o $(BIN_DIR)/$(BINARY) .

install:
	go install .

run:
	go run . $(ARGS)

clean:
	rm -rf $(BIN_DIR)
