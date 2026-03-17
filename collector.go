package main

import "github.com/prometheus/client_golang/prometheus"

type sensor interface {
	EnvData() (temperature, pressure, humidity float64, err error)
}

type bme280Collector struct {
	sensor      sensor
	temperature *prometheus.Desc
	pressure    *prometheus.Desc
	humidity    *prometheus.Desc
}

func newCollector(s sensor) *bme280Collector {
	return &bme280Collector{
		sensor: s,
		temperature: prometheus.NewDesc(
			"bme280_temperature_celsius",
			"Temperature in Celsius",
			nil, nil,
		),
		pressure: prometheus.NewDesc(
			"bme280_pressure_hpa",
			"Atmospheric pressure in hPa",
			nil, nil,
		),
		humidity: prometheus.NewDesc(
			"bme280_humidity_percent",
			"Relative humidity in percent",
			nil, nil,
		),
	}
}

func (c *bme280Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.temperature
	ch <- c.pressure
	ch <- c.humidity
}

func (c *bme280Collector) Collect(ch chan<- prometheus.Metric) {
	t, p, h, err := c.sensor.EnvData()
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.temperature, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.temperature, prometheus.GaugeValue, t)
	ch <- prometheus.MustNewConstMetric(c.pressure, prometheus.GaugeValue, p)
	ch <- prometheus.MustNewConstMetric(c.humidity, prometheus.GaugeValue, h)
}
