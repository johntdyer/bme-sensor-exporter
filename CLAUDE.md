# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Prometheus metrics exporter for the BME280 temperature/humidity/pressure sensor running on a Raspberry Pi. Written in Go, it reads sensor data over I2C and exposes it via an HTTP `/metrics` endpoint.

## Common Commands

```bash
# Build for local development
go build ./...

# Build for Raspberry Pi (ARM)
GOOS=linux GOARCH=arm GOARM=7 go build -o bme-sensor-exporter ./cmd/bme-sensor-exporter

# Run tests
go test ./...

# Run a single test
go test ./internal/sensor/... -run TestFunctionName

# Run tests with race detector
go test -race ./...

# Lint (assumes golangci-lint installed)
golangci-lint run

# Format code
gofmt -w .
goimports -w .
```

## Architecture

The app follows a standard Go project layout:

- `cmd/bme-sensor-exporter/` — main entrypoint, wires dependencies together
- `internal/sensor/` — BME280 I2C communication and data parsing
- `internal/collector/` — Prometheus collector implementation wrapping the sensor

The sensor package handles low-level I2C reads and applies the BME280 compensation formulas to raw ADC values. The collector package implements `prometheus.Collector` and calls into the sensor package on each scrape.

## Key Dependencies

- `periph.io/x/periph` or `golang.org/x/exp/io/i2c` — I2C communication
- `github.com/prometheus/client_golang` — Prometheus metrics exposition

## Raspberry Pi Deployment

The binary is cross-compiled on a dev machine and copied to the Pi. The I2C bus must be enabled on the Pi (`raspi-config` → Interface Options → I2C). The default I2C address for BME280 is `0x76` (or `0x77` if SDO is pulled high).

The exporter runs as a systemd service listening on `:9100` (or configurable port) and exposes metrics at `/metrics`.
