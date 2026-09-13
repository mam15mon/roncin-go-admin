package main

import (
	"testing"
)

func TestParseAirlinesValid(t *testing.T) {
	raw := []byte(`1,"Private flight",\N,"-","N/A","","","Y"
24,"American Airlines",\N,"AA","AAL","AMERICAN","United States","Y"
135,"Air China",\N,"CA","CCA","AIR CHINA","China","Y"
784,"China Southern Airlines",\N,"CZ","CSN","CHINA SOUTHERN","China","Y"
999,"Defunct Airline",\N,"ZZ","ZZZ","DEFUNCT","Germany","N"
1000,"Cargolux",\N,"CV","CLX","CARGOLUX","Luxembourg","Y"
`)

	records, summary, err := parseAirlines(raw, "mock-hash")
	if err != nil {
		t.Fatalf("parseAirlines failed: %v", err)
	}

	if summary.ValidAirlines < 4 {
		t.Errorf("expected at least 4 valid airlines, got %d", summary.ValidAirlines)
	}
	if summary.SkippedInactive < 1 {
		t.Errorf("expected at least 1 inactive skipped, got %d", summary.SkippedInactive)
	}
	if summary.CargoOnly < 1 {
		t.Errorf("expected at least 1 cargo only, got %d", summary.CargoOnly)
	}

	recordMap := make(map[string]string)
	awbMap := make(map[string]string)
	for _, r := range records {
		if r.NameZH != nil {
			recordMap[r.IATACode] = *r.NameZH
		}
		if r.AWBPrefix != nil {
			awbMap[r.IATACode] = *r.AWBPrefix
		}
	}

	if recordMap["CA"] != "中国国际航空" {
		t.Errorf("expected CA to be 中国国际航空, got %s", recordMap["CA"])
	}
	if awbMap["CA"] != "999" {
		t.Errorf("expected CA AWB to be 999, got %s", awbMap["CA"])
	}
	if recordMap["CZ"] != "中国南方航空" {
		t.Errorf("expected CZ to be 中国南方航空, got %s", recordMap["CZ"])
	}
	if awbMap["CZ"] != "784" {
		t.Errorf("expected CZ AWB to be 784, got %s", awbMap["CZ"])
	}
	if recordMap["AA"] != "美国航空" {
		t.Errorf("expected AA to be 美国航空, got %s", recordMap["AA"])
	}
}

func TestMapOpenFlightsCountry(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"China", "CN"},
		{"United States", "US"},
		{"Hong Kong", "HK"},
		{"Japan", "JP"},
		{"Germany", "DE"},
		{"Unknown Country XYZ", ""},
	}
	for _, tt := range tests {
		got := mapOpenFlightsCountry(tt.input)
		if got != tt.want {
			t.Errorf("mapOpenFlightsCountry(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestKnownAirlinesIntegrity(t *testing.T) {
	seenAWB := make(map[string]string)
	seenICAO := make(map[string]string)
	for iata, meta := range knownAirlines {
		if meta.AWBPrefix != "" {
			if prev, ok := seenAWB[meta.AWBPrefix]; ok {
				t.Errorf("duplicate AWB prefix %q for IATA %s and %s", meta.AWBPrefix, prev, iata)
			}
			seenAWB[meta.AWBPrefix] = iata
		}
		if meta.ICAOCode != "" {
			if prev, ok := seenICAO[meta.ICAOCode]; ok {
				t.Errorf("duplicate ICAO %q for IATA %s and %s", meta.ICAOCode, prev, iata)
			}
			seenICAO[meta.ICAOCode] = iata
		}
		if len(meta.CountryCode) != 2 {
			t.Errorf("invalid country code %q for IATA %s", meta.CountryCode, iata)
		}
	}
}
