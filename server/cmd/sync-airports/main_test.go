package main

import (
	"strings"
	"testing"
)

func TestParseAirports(t *testing.T) {
	t.Parallel()
	raw := []byte(strings.Join([]string{
		"iata_code,gps_code,name,municipality,iso_country,type",
		"PVG,ZSPD,Shanghai Pudong International Airport,Shanghai,CN,large_airport",
		"OLD,,Closed Airport,Old City,US,closed",
		",ZZZZ,No IATA,Nowhere,US,small_airport",
	}, "\n"))
	rows, summary, err := parseAirports(raw)
	if err != nil {
		t.Fatalf("parseAirports() error = %v", err)
	}
	if len(rows) != 2 || summary.SkippedIATA != 1 || summary.Closed != 1 || len(summary.FatalProblems) != 0 {
		t.Fatalf("unexpected summary: rows=%d summary=%+v", len(rows), summary)
	}
	if rows[0].IATACode != "OLD" || rows[0].Enabled || rows[1].IATACode != "PVG" || !rows[1].Enabled {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestParseAirportsRejectsDuplicateCodes(t *testing.T) {
	t.Parallel()
	raw := []byte(strings.Join([]string{
		"iata_code,gps_code,name,municipality,iso_country,type",
		"PVG,ZSPD,Shanghai Pudong International Airport,Shanghai,CN,large_airport",
		"PVG,ZSSS,Duplicate IATA,Shanghai,CN,large_airport",
		"SHA,ZSPD,Duplicate ICAO,Shanghai,CN,large_airport",
	}, "\n"))
	_, summary, err := parseAirports(raw)
	if err != nil {
		t.Fatalf("parseAirports() error = %v", err)
	}
	if len(summary.FatalProblems) != 2 {
		t.Fatalf("fatal problems = %v, want 2", summary.FatalProblems)
	}
}

func TestParseAirportsRequiresHeaders(t *testing.T) {
	t.Parallel()
	_, _, err := parseAirports([]byte("iata_code,name\nPVG,Airport\n"))
	if err == nil {
		t.Fatal("parseAirports() error = nil, want missing header error")
	}
}
