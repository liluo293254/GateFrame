.PHONY: build test run tidy

BIN_DIR ?= ../../.ddev/bin

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/user-service ./cmd/user-service

test:
	go test ./...

run: build
	$(BIN_DIR)/user-service

tidy:
	go mod tidy
