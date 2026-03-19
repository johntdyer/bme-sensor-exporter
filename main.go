package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/quhar/bme280"
	"github.com/warthog618/go-gpiocdev"
	"golang.org/x/exp/io/i2c"
)

func main() {
	addr         := flag.String("listen-address", ":9100", "Address to listen on for HTTP requests")
	dev          := flag.String("i2c-device", "/dev/i2c-4", "I2C device path")
	gpioChip     := flag.String("gpio-chip", "gpiochip0", "GPIO chip for fan tachometer (gpiochip0 on Pi 5, check ls /dev/gpiochip*)")
	tachPin      := flag.String("tach-pin", "GPIO13", "GPIO pin for fan tachometer (BCM name e.g. GPIO13, or bare offset e.g. 13)")
	pulsesPerRev := flag.Int("tach-pulses", 2, "Tachometer pulses per fan revolution (2 for most Noctua fans)")
	gpioPins     := flag.String("gpio-pins", "", "Comma-separated GPIO pins to monitor via pinctrl (e.g. GPIO17:up,GPIO18:down,GPIO27)")
	flag.Parse()

	d, err := i2c.Open(&i2c.Devfs{Dev: *dev}, bme280.I2CAddr)
	if err != nil {
		log.Fatalf("failed to open I2C device %s: %v", *dev, err)
	}

	b := bme280.New(d)
	if err := b.Init(); err != nil {
		log.Fatalf("failed to initialize BME280: %v", err)
	}

	chip, err := gpiocdev.NewChip(*gpioChip, gpiocdev.WithConsumer("bme-sensor-exporter"))
	if err != nil {
		log.Fatalf("failed to open GPIO chip %s: %v\n(run 'ls -la /dev/gpiochip*' on the Pi to find the right chip)", *gpioChip, err)
	}
	defer chip.Close()

	tachOffset, err := parsePinOffset(*tachPin)
	if err != nil {
		log.Fatalf("invalid --tach-pin %q: %v", *tachPin, err)
	}

	tach, err := newFanTach(chip, tachOffset, *pulsesPerRev)
	if err != nil {
		log.Fatalf("failed to set up fan tachometer on %s: %v", *tachPin, err)
	}
	defer tach.Close()

	reg := prometheus.NewRegistry()
	reg.MustRegister(newCollector(b))
	reg.MustRegister(newFanCollector(tach))

	if *gpioPins != "" {
		gpioCol, err := newGPIOCollector(*gpioPins)
		if err != nil {
			log.Fatalf("failed to set up GPIO pin monitor: %v", err)
		}
		reg.MustRegister(gpioCol)
	}

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Printf("Listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
