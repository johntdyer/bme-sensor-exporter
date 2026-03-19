package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type configuredPin struct {
	name    string
	offset  int
	pullStr string
}

type gpioCollector struct {
	pins []configuredPin
	desc *prometheus.Desc
}

// newGPIOCollector parses a comma-separated pin spec, configures each pin as
// an input via pinctrl, and returns a collector that reads levels on each scrape.
//
// Format: "GPIO17:up,GPIO18:down,GPIO27" or bare offsets "5:up,6:down"
// Pull direction defaults to "up". Valid values: up, down, float.
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
		pullStr = strings.ToLower(pullStr)

		offset, err := parsePinOffset(name)
		if err != nil {
			return nil, fmt.Errorf("invalid pin %q: %w", name, err)
		}

		switch pullStr {
		case "up", "down", "float", "none":
		default:
			return nil, fmt.Errorf("pin %s: unknown pull direction %q (use up, down, or float)", name, pullStr)
		}

		pins = append(pins, configuredPin{name: name, offset: offset, pullStr: pullStr})
	}

	c := &gpioCollector{
		pins: pins,
		desc: prometheus.NewDesc(
			"gpio_pin_state",
			"GPIO pin level: 1 = high, 0 = low",
			[]string{"pin", "pull"},
			nil,
		),
	}

	if err := c.configurePins(); err != nil {
		return nil, err
	}
	return c, nil
}

// configurePins runs "pinctrl set <offset> ip <pull>" for each pin at startup.
func (c *gpioCollector) configurePins() error {
	for _, cp := range c.pins {
		args := []string{"set", strconv.Itoa(cp.offset), "ip"}
		switch cp.pullStr {
		case "up":
			args = append(args, "pu")
		case "down":
			args = append(args, "pd")
		// float/none: omit pull arg, kernel keeps existing bias
		}

		if out, err := exec.Command("pinctrl", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("pinctrl set for pin %s failed: %w\n%s", cp.name, err, out)
		}
	}
	return nil
}

func (c *gpioCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *gpioCollector) Collect(ch chan<- prometheus.Metric) {
	levels, err := readPinLevels(c.pins)
	if err != nil {
		log.Printf("gpio_pins: %v", err)
		ch <- prometheus.NewInvalidMetric(c.desc, err)
		return
	}
	for _, cp := range c.pins {
		val := 0.0
		if levels[cp.offset] {
			val = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			c.desc, prometheus.GaugeValue, val,
			cp.name, cp.pullStr,
		)
	}
}

// readPinLevels runs "pinctrl get <n>,<n>,..." once and parses the output.
// Returns a map of pin offset → true (high) / false (low).
//
// pinctrl output format per line:
//
//	 5: ip    pu | hi // GPIO5 = input
func readPinLevels(pins []configuredPin) (map[int]bool, error) {
	parts := make([]string, len(pins))
	for i, cp := range pins {
		parts[i] = strconv.Itoa(cp.offset)
	}
	pinList := strings.Join(parts, ",")

	out, err := exec.Command("pinctrl", "get", pinList).Output()
	if err != nil {
		return nil, fmt.Errorf("pinctrl get %s: %w", pinList, err)
	}

	levels := make(map[int]bool, len(pins))
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()

		// " 5: ip    pu | hi // GPIO5 = input"
		// Split on ":" to get pin offset, then "|" to get level.
		pinPart, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		offset, err := strconv.Atoi(strings.TrimSpace(pinPart))
		if err != nil {
			continue
		}
		_, levelPart, ok := strings.Cut(rest, "|")
		if !ok {
			continue
		}
		fields := strings.Fields(levelPart)
		if len(fields) == 0 {
			continue
		}
		levels[offset] = fields[0] == "hi"
	}
	return levels, nil
}

// parsePinOffset accepts "GPIO17", "gpio17", or bare "17" and returns the integer offset.
func parsePinOffset(s string) (int, error) {
	s = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(s)), "GPIO")
	return strconv.Atoi(s)
}
