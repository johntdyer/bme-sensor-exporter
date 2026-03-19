package main

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/warthog618/go-gpiocdev"
)

const (
	minPulseInterval = 5 * time.Millisecond // reject spuriously short pulses
	staleDuration    = 2 * time.Second      // report 0 RPM if no pulse seen within this window
)

// fanTach monitors a tachometer GPIO pin and tracks the current fan RPM.
// gpiocdev delivers edge events via the kernel's GPIO character device ioctl
// interface, which handles hundreds of edges per second — unlike periph.io's
// sysfs backend which is capped at ~4 Hz.
type fanTach struct {
	line         *gpiocdev.Line
	pulsesPerRev int
	mu           sync.RWMutex
	rpm          float64
	lastPulse    time.Time
	prevEdge     time.Time
}

func newFanTach(chip *gpiocdev.Chip, pinOffset, pulsesPerRev int) (*fanTach, error) {
	f := &fanTach{pulsesPerRev: pulsesPerRev}

	line, err := chip.RequestLine(pinOffset,
		gpiocdev.AsInput,
		gpiocdev.WithPullUp,
		gpiocdev.WithFallingEdge,
		gpiocdev.WithEventHandler(f.handleEdge),
	)
	if err != nil {
		return nil, err
	}
	f.line = line
	return f, nil
}

func (f *fanTach) handleEdge(_ gpiocdev.LineEvent) {
	now := time.Now()
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.prevEdge.IsZero() {
		f.prevEdge = now
		return
	}

	dt := now.Sub(f.prevEdge)
	if dt < minPulseInterval {
		return // reject spurious pulse
	}

	freq := 1.0 / dt.Seconds()
	f.rpm = (freq / float64(f.pulsesPerRev)) * 60
	f.lastPulse = now
	f.prevEdge = now
}

// RPM returns the most recently computed fan speed.
// Returns 0 if no pulse has been seen within staleDuration (fan stopped).
func (f *fanTach) RPM() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.lastPulse.IsZero() || time.Since(f.lastPulse) > staleDuration {
		return 0
	}
	return f.rpm
}

func (f *fanTach) Close() {
	f.line.Close()
}

// fanCollector is a prometheus.Collector that exposes fan RPM.
type fanCollector struct {
	tach *fanTach
	desc *prometheus.Desc
}

func newFanCollector(tach *fanTach) *fanCollector {
	return &fanCollector{
		tach: tach,
		desc: prometheus.NewDesc(
			"fan_rpm",
			"Fan speed in RPM derived from tachometer pulse counting",
			nil, nil,
		),
	}
}

func (c *fanCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *fanCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, c.tach.RPM())
}
