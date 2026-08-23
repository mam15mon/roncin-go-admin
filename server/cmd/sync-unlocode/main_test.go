package main

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestParseUNLocodeFiltersPorts(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		partNames[0]: []byte(strings.Join([]string{
			",CN,,.CHINA,,,,,,,,",
			",CN,SHG,Shanghai,Shanghai,SH,1234----,AI,9501,,3114N 12129E,",
			",CN,BJS,Beijing,Beijing,BJ,--34----,AI,9501,,3955N 11625E,",
			"=,CN,SHA,Shanghai Alias,Shanghai Alias,SH,1-------,AI,9501,,,",
		}, "\n")),
		partNames[1]: []byte(",US,LAX,Los Angeles,Los Angeles,CA,1-------,AI,9501,,3356N 11824W,\n"),
		partNames[2]: []byte(",GB,OLD,Old Port,Old Port,,1-------,XX,9501,,,\n"),
	}
	rows, summary, err := parseUNLocode(files, []byte("zip"))
	if err != nil {
		t.Fatalf("parseUNLocode() error = %v", err)
	}
	if len(rows) != 3 || summary.NonPorts != 1 || summary.CountryHeaders != 1 || summary.Aliases != 1 || summary.Withdrawn != 1 {
		t.Fatalf("unexpected summary: rows=%d summary=%+v", len(rows), summary)
	}
	if strings.Join(rows[0].TransportModes, ",") != "SEA,RAIL,ROAD" {
		t.Fatalf("transport modes = %v", rows[0].TransportModes)
	}
}

func TestParseUNLocodeRejectsDuplicatePort(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		partNames[0]: []byte(",CN,SHG,Shanghai,Shanghai,SH,1-------,AI,9501,,,\n"),
		partNames[1]: []byte(",CN,SHG,Duplicate,Duplicate,SH,1-------,AI,9501,,,\n"),
		partNames[2]: []byte(",US,LAX,Los Angeles,Los Angeles,CA,1-------,AI,9501,,,\n"),
	}
	_, summary, err := parseUNLocode(files, []byte("zip"))
	if err != nil {
		t.Fatalf("parseUNLocode() error = %v", err)
	}
	if len(summary.FatalProblems) != 1 {
		t.Fatalf("fatal problems = %v, want 1", summary.FatalProblems)
	}
}

func TestParseUNLocodePrefersChangedRow(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		partNames[0]: []byte(strings.Join([]string{
			",BE,BRU,Bruxelles,Bruxelles,BRU,1234----,AI,1101,,,",
			"#,BE,BRU,Brussel,Brussel,BRU,1234----,AI,2501,,,",
		}, "\n")),
		partNames[1]: []byte(",US,LAX,Los Angeles,Los Angeles,CA,1-------,AI,9501,,,\n"),
		partNames[2]: []byte(",CN,SHG,Shanghai,Shanghai,SH,1-------,AI,9501,,,\n"),
	}
	rows, summary, err := parseUNLocode(files, []byte("zip"))
	if err != nil {
		t.Fatalf("parseUNLocode() error = %v", err)
	}
	if summary.Superseded != 1 || len(summary.FatalProblems) != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	for _, row := range rows {
		if row.UNLocode == "BEBRU" && row.NameEN != "Brussel" {
			t.Fatalf("BEBRU name = %q, want changed row", row.NameEN)
		}
	}
}

func TestReadCodeListFilesRequiresAllParts(t *testing.T) {
	t.Parallel()
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)
	entry, err := writer.Create("release/csv/2025-1 UNLOCODE CodeListPart1.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("data")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	_, _, err = readCodeListFiles(buffer.Bytes())
	if err == nil {
		t.Fatal("readCodeListFiles() error = nil, want missing parts error")
	}
}
