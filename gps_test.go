package main

import (
	"strings"
	"testing"
)

func TestParseUsedSatellites(t *testing.T) {
	in := strings.NewReader(`
{"class":"TPV","time":"2026-03-20T15:23:00.000Z"}
{"class":"SKY","uSat":9}
{"class":"SKY","uSat":11}
`)

	got, found, err := parseUsedSatellites(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected SKY.uSat to be found")
	}
	if got != 11 {
		t.Fatalf("uSat = %d, want 11", got)
	}
}

func TestParseUsedSatellitesMissingUSat(t *testing.T) {
	in := strings.NewReader(`
{"class":"TPV","time":"2026-03-20T15:23:00.000Z"}
{"class":"SKY"}
`)

	_, found, err := parseUsedSatellites(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected SKY.uSat to be missing")
	}
}

func TestParseUsedSatellitesIgnoresInvalidJSON(t *testing.T) {
	in := strings.NewReader("not-json\n{\"class\":\"SKY\",\"uSat\":6}\n")

	got, found, err := parseUsedSatellites(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected SKY.uSat to be found")
	}
	if got != 6 {
		t.Fatalf("uSat = %d, want 6", got)
	}
}
