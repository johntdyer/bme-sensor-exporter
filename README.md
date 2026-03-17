# bme-sensor-exporter

Prometheus metrics exporter for the BME280 temperature, humidity, and pressure sensor on a Raspberry Pi.

## Metrics

| Metric | Description |
|---|---|
| `bme280_temperature_celsius` | Temperature in Celsius |
| `bme280_pressure_hpa` | Atmospheric pressure in hPa |
| `bme280_humidity_percent` | Relative humidity in percent |

## Flags

| Flag | Default | Description |
|---|---|---|
| `--listen-address` | `:9100` | Address to listen on for HTTP requests |
| `--i2c-device` | `/dev/i2c-4` | I2C device path |

## Collecting Metrics

### Prometheus

Add a scrape job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: bme280
    static_configs:
      - targets:
          - raspberrypi.local:9100
        labels:
          location: living_room
```

### Grafana Alloy

Add a scrape component to your Alloy config:

```alloy
discovery.relabel "bme280" {
  targets = [
    {
      __address__ = "raspberrypi.local:9100",
      location    = "living_room",
    },
  ]

  rule {
    target_label = "instance"
    replacement  = constants.hostname
  }
}

prometheus.scrape "bme280" {
  targets    = discovery.relabel.bme280.output
  forward_to = [prometheus.remote_write.default.receiver]
}

prometheus.remote_write "default" {
  endpoint {
    url = "http://your-mimir-or-prometheus:9090/api/v1/write"
  }
}
```

## Deployment

See [`examples/bme-sensor-exporter.service`](examples/bme-sensor-exporter.service) for a systemd unit file.

```bash
sudo cp bme-sensor-exporter /usr/local/bin/
sudo cp examples/bme-sensor-exporter.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now bme-sensor-exporter
```
