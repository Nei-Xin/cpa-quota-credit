# Makefile for cpa-quota-credit plugin

PLUGIN_NAME := cpa-quota-credit
BIN_DIR := ./bin

.PHONY: all build build-linux build-windows build-darwin test clean

all: test build

test:
	go test -v ./...

build:
	@mkdir -p $(BIN_DIR)
	go build -buildmode=c-shared -o $(BIN_DIR)/$(PLUGIN_NAME).so main.go

build-linux:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o $(BIN_DIR)/$(PLUGIN_NAME).so main.go

build-windows:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -buildmode=c-shared -o $(BIN_DIR)/$(PLUGIN_NAME).dll main.go

build-darwin:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildmode=c-shared -o $(BIN_DIR)/$(PLUGIN_NAME).dylib main.go

clean:
	rm -rf $(BIN_DIR)
