package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/quhar/bme280"
	"golang.org/x/exp/io/i2c"
)

func main() {
	addr := flag.String("listen-address", ":9100", "Address to listen on for HTTP requests")
	dev := flag.String("i2c-device", "/dev/i2c-4", "I2C device path")
	flag.Parse()

	d, err := i2c.Open(&i2c.Devfs{Dev: *dev}, bme280.I2CAddr)
	if err != nil {
		log.Fatalf("failed to open I2C device %s: %v", *dev, err)
	}

	b := bme280.New(d)
	if err := b.Init(); err != nil {
		log.Fatalf("failed to initialize BME280: %v", err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(newCollector(b))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Printf("Listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
