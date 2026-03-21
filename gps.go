package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type gpsSatCollector struct {
	desc       *prometheus.Desc
	gpspipeBin string
	seconds    int

	mu     sync.RWMutex
	cached int
	err    error
}

func newGPSSatCollector(gpspipeBin string, seconds int) *gpsSatCollector {
	return &gpsSatCollector{
		desc: prometheus.NewDesc(
			"gps_used_satellites",
			"Number of satellites currently used in GPS solution (gpsd SKY.uSat)",
			nil,
			nil,
		),
		gpspipeBin: gpspipeBin,
		seconds:    seconds,
		err:        fmt.Errorf("GPS data not yet available"),
	}
}

// start performs an initial fetch (blocking) then polls every 30 seconds in a goroutine.
func (c *gpsSatCollector) start() {
	c.poll()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			c.poll()
		}
	}()
}

func (c *gpsSatCollector) poll() {
	uSat, err := readUsedSatellites(c.gpspipeBin, c.seconds)
	c.mu.Lock()
	c.cached = uSat
	c.err = err
	c.mu.Unlock()
}

func (c *gpsSatCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *gpsSatCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	cached, err := c.cached, c.err
	c.mu.RUnlock()

	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.desc, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(cached))
}

func readUsedSatellites(gpspipeBin string, seconds int) (int, error) {
	if seconds <= 0 {
		return 0, fmt.Errorf("seconds must be positive, got %d", seconds)
	}

	var stderr bytes.Buffer
	cmd := exec.Command(gpspipeBin, "-w", "-x", strconv.Itoa(seconds))
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("gpspipe stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("gpspipe start: %w", err)
	}

	uSat, found, parseErr := parseUsedSatellites(stdout)
	if waitErr := cmd.Wait(); waitErr != nil {
		if stderr.Len() > 0 {
			return 0, fmt.Errorf("gpspipe failed: %w: %s", waitErr, stderr.String())
		}
		return 0, fmt.Errorf("gpspipe failed: %w", waitErr)
	}
	if parseErr != nil {
		return 0, parseErr
	}
	if !found {
		return 0, fmt.Errorf("gpspipe output did not include SKY.uSat")
	}

	return uSat, nil
}

func parseUsedSatellites(r io.Reader) (int, bool, error) {
	type skyMessage struct {
		Class string `json:"class"`
		USat  *int   `json:"uSat"`
	}

	sc := bufio.NewScanner(r)
	found := false
	uSat := 0
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}

		var msg skyMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.Class == "SKY" && msg.USat != nil {
			uSat = *msg.USat
			found = true
		}
	}
	if err := sc.Err(); err != nil {
		return 0, false, fmt.Errorf("read gpspipe output: %w", err)
	}
	return uSat, found, nil
}
