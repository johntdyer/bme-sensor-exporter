package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
)

type configuredPin struct {
	name    string
	pullStr string
	pin     gpio.PinIO
}

type gpioCollector struct {
	pins []configuredPin
	desc *prometheus.Desc
}

// newGPIOCollector parses a comma-separated pin spec string and returns a
// collector that reads each pin's level on every Prometheus scrape.
//
// Format: "GPIO17:up,GPIO18:down,GPIO27"
// Pull direction defaults to "up" if omitted. Valid values: up, down, float.
//
// Returns nil, nil when spec is empty (feature disabled).
func newGPIOCollector(spec string) (*gpioCollector, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}

	var pins []configuredPin
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		name, pullStr, _ := strings.Cut(part, ":")
		name = strings.TrimSpace(name)
		if pullStr == "" {
			pullStr = "up"
		}

		var pull gpio.Pull
		switch strings.ToLower(pullStr) {
		case "up":
			pull = gpio.PullUp
		case "down":
			pull = gpio.PullDown
		case "float", "none":
			pull = gpio.Float
		default:
			return nil, fmt.Errorf("pin %s: unknown pull direction %q (use up, down, or float)", name, pullStr)
		}

		p := gpioreg.ByName(name)
		if p == nil {
			return nil, fmt.Errorf("GPIO pin %q not found — check pin name and ensure GPIO is enabled", name)
		}
		if err := p.In(pull, gpio.NoEdge); err != nil {
			return nil, fmt.Errorf("failed to configure pin %s as input with pull-%s: %w", name, pullStr, err)
		}

		pins = append(pins, configuredPin{
			name:    name,
			pullStr: strings.ToLower(pullStr),
			pin:     p,
		})
	}

	return &gpioCollector{
		pins: pins,
		desc: prometheus.NewDesc(
			"gpio_pin_state",
			"GPIO pin level: 1 = high, 0 = low",
			[]string{"pin", "pull"},
			nil,
		),
	}, nil
}

func (c *gpioCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *gpioCollector) Collect(ch chan<- prometheus.Metric) {
	for _, cp := range c.pins {
		val := 0.0
		if cp.pin.Read() == gpio.High {
			val = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			c.desc,
			prometheus.GaugeValue,
			val,
			cp.name, cp.pullStr,
		)
	}
}
