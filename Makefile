BINARY := bme-sensor-exporter
BUILD_DIR := build
PI_HOST ?= pi@raspberrypi.local

.PHONY: build build-pi test clean deploy

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) .

build-pi:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY) .

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)

deploy: build-pi
	scp $(BUILD_DIR)/$(BINARY) $(PI_HOST):~/$(BINARY)
