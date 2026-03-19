package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/quhar/bme280"
	"golang.org/x/exp/io/i2c"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

func main() {
	addr         := flag.String("listen-address", ":9100", "Address to listen on for HTTP requests")
	dev          := flag.String("i2c-device", "/dev/i2c-4", "I2C device path")
	tachPin      := flag.String("tach-pin", "GPIO13", "GPIO pin name for fan tachometer (BCM numbering, e.g. GPIO13)")
	pulsesPerRev := flag.Int("tach-pulses", 2, "Tachometer pulses per fan revolution (2 for most Noctua fans)")
	flag.Parse()

	// Initialise periph.io host drivers (GPIO, I2C, SPI, etc.)
	if _, err := host.Init(); err != nil {
		log.Fatalf("failed to initialise periph.io host: %v", err)
	}

	d, err := i2c.Open(&i2c.Devfs{Dev: *dev}, bme280.I2CAddr)
	if err != nil {
		log.Fatalf("failed to open I2C device %s: %v", *dev, err)
	}

	b := bme280.New(d)
	if err := b.Init(); err != nil {
		log.Fatalf("failed to initialize BME280: %v", err)
	}

	pin := gpioreg.ByName(*tachPin)
	if pin == nil {
		log.Fatalf("GPIO pin %q not found — check --tach-pin and ensure I2C/GPIO are enabled", *tachPin)
	}

	tach, err := newFanTach(pin, *pulsesPerRev)
	if err != nil {
		log.Fatalf("failed to set up fan tachometer on %s: %v", *tachPin, err)
	}
	defer tach.Close()

	reg := prometheus.NewRegistry()
	reg.MustRegister(newCollector(b))
	reg.MustRegister(newFanCollector(tach))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Printf("Listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
