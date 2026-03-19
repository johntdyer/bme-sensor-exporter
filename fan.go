package main

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"periph.io/x/conn/v3/gpio"
)

const (
	minPulseInterval = 5 * time.Millisecond // reject spuriously short pulses
	staleDuration    = 2 * time.Second      // report 0 RPM if no pulse seen within this window
)

// fanTach monitors a GPIO tachometer pin and tracks the current fan RPM.
// It counts falling-edge pulses and derives RPM from inter-pulse timing,
// matching the behaviour of the reference Noctua Python snippet.
type fanTach struct {
	pin           gpio.PinIn
	pulsesPerRev  int
	mu            sync.RWMutex
	rpm           float64
	lastPulse     time.Time
	done          chan struct{}
	wg            sync.WaitGroup
}

func newFanTach(pin gpio.PinIn, pulsesPerRev int) (*fanTach, error) {
	if err := pin.In(gpio.PullUp, gpio.FallingEdge); err != nil {
		return nil, err
	}
	f := &fanTach{
		pin:          pin,
		pulsesPerRev: pulsesPerRev,
		done:         make(chan struct{}),
	}
	f.wg.Add(1)
	go f.count()
	return f, nil
}

// count runs in a goroutine, waiting for falling edges and computing RPM
// from the time elapsed between consecutive pulses.
func (f *fanTach) count() {
	defer f.wg.Done()
	last := time.Now()
	for {
		select {
		case <-f.done:
			return
		default:
		}

		if !f.pin.WaitForEdge(100 * time.Millisecond) {
			continue // timeout — loop and check done channel
		}

		now := time.Now()
		dt := now.Sub(last)
		if dt < minPulseInterval {
			continue // reject spurious pulse
		}

		freq := 1.0 / dt.Seconds()
		rpm := (freq / float64(f.pulsesPerRev)) * 60

		f.mu.Lock()
		f.rpm = rpm
		f.lastPulse = now
		f.mu.Unlock()

		last = now
	}
}

// RPM returns the most recently computed fan speed in RPM.
// Returns 0 if no pulse has been seen within staleDuration.
func (f *fanTach) RPM() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.lastPulse.IsZero() || time.Since(f.lastPulse) > staleDuration {
		return 0
	}
	return f.rpm
}

// Close stops the background goroutine.
func (f *fanTach) Close() {
	close(f.done)
	f.wg.Wait()
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
