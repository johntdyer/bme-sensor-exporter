package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type mockSensor struct {
	temperature float64
	pressure    float64
	humidity    float64
	err         error
}

func (m *mockSensor) EnvData() (float64, float64, float64, error) {
	return m.temperature, m.pressure, m.humidity, m.err
}

func TestCollector(t *testing.T) {
	mock := &mockSensor{
		temperature: 22.5,
		pressure:    1013.25,
		humidity:    55.0,
	}

	collector := newCollector(mock)
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)

	expected := `
		# HELP bme280_humidity_percent Relative humidity in percent
		# TYPE bme280_humidity_percent gauge
		bme280_humidity_percent 55
		# HELP bme280_pressure_hpa Atmospheric pressure in hPa
		# TYPE bme280_pressure_hpa gauge
		bme280_pressure_hpa 1013.25
		# HELP bme280_temperature_celsius Temperature in Celsius
		# TYPE bme280_temperature_celsius gauge
		bme280_temperature_celsius 22.5
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected)); err != nil {
		t.Fatal(err)
	}
}

func TestCollectorSensorError(t *testing.T) {
	mock := &mockSensor{err: errors.New("i2c read failed")}

	collector := newCollector(mock)
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)

	_, err := reg.Gather()
	if err == nil {
		t.Fatal("expected error when sensor fails, got nil")
	}
}
