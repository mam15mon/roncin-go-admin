package data

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestSortUUIDs(t *testing.T) {
	u1 := uuid.MustParse("018f0000-0000-7000-8000-000000000001")
	u2 := uuid.MustParse("018f0000-0000-7000-8000-000000000002")
	u3 := uuid.MustParse("018f0000-0000-7000-8000-000000000003")

	// 乱序与重复
	input := []uuid.UUID{u3, u1, u2, u1, u3}
	sorted := sortAndDeduplicateUUIDs(input)

	if len(sorted) != 3 {
		t.Fatalf("expected 3 unique sorted UUIDs, got %d", len(sorted))
	}
	if sorted[0] != u1 || sorted[1] != u2 || sorted[2] != u3 {
		t.Fatalf("expected sorted order u1, u2, u3, got %v", sorted)
	}
}

func TestSeaOrderChange_MakeDiff(t *testing.T) {
	diff1 := makeDiff("vessel_name", "船名", "SHIP A", "SHIP B")
	if !diff1.IsDifferent || diff1.FieldName != "vessel_name" || diff1.CurrentValue != "SHIP A" || diff1.TargetValue != "SHIP B" {
		t.Errorf("makeDiff unexpected for different values: %+v", diff1)
	}

	diff2 := makeDiff("voyage_no", "航次", "001W", "001W")
	if diff2.IsDifferent {
		t.Errorf("makeDiff should be false for same values: %+v", diff2)
	}
}

func TestSeaOrderChange_ParseOptionalTime(t *testing.T) {
	if parseOptionalTime("") != nil {
		t.Error("expected nil for empty string")
	}

	if parseOptionalTime("invalid-time-format") != nil {
		t.Error("expected nil for invalid format")
	}

	rfcTm := parseOptionalTime("2026-09-03T14:00:00Z")
	if rfcTm == nil || rfcTm.Year() != 2026 || rfcTm.Month() != time.September || rfcTm.Day() != 3 || rfcTm.Hour() != 14 {
		t.Errorf("expected parsed RFC3339 time, got %v", rfcTm)
	}
}

func TestSeaOrderChange_DecimalZeroTolerance(t *testing.T) {
	d1 := decimal.RequireFromString("100.123")
	d2 := decimal.RequireFromString("50.001")
	d3 := decimal.RequireFromString("50.122")

	sum := d2.Add(d3)
	if !sum.Equal(d1) {
		t.Fatalf("expected sum %v to equal %v", sum, d1)
	}

	diff := d1.Sub(sum)
	if !diff.IsZero() {
		t.Fatalf("expected diff to be zero, got %v", diff)
	}
}

func TestDecodeSplitResultSnapshotSummaryRejectsDamagedHistory(t *testing.T) {
	packageCount, grossWeight, volume, err := decodeSplitResultSnapshotSummary([]byte(
		`{"schema_version":1,"package_count":8,"gross_weight_kg":"10.125","volume_cbm":"2.500001"}`,
	))
	if err != nil {
		t.Fatalf("valid snapshot returned error: %v", err)
	}
	if packageCount != 8 || !grossWeight.Equal(decimal.RequireFromString("10.125")) || !volume.Equal(decimal.RequireFromString("2.500001")) {
		t.Fatalf("valid snapshot decoded incorrectly: package=%d weight=%s volume=%s", packageCount, grossWeight, volume)
	}

	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "malformed json", raw: `{"schema_version":`},
		{name: "unsupported version", raw: `{"schema_version":2,"package_count":8,"gross_weight_kg":"10.125","volume_cbm":"2.500001"}`},
		{name: "missing quantity", raw: `{"schema_version":1}`},
		{name: "invalid decimal", raw: `{"schema_version":1,"package_count":8,"gross_weight_kg":"bad","volume_cbm":"2.500001"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := decodeSplitResultSnapshotSummary([]byte(tc.raw)); err == nil {
				t.Fatal("damaged split result snapshot must return an error")
			}
		})
	}
}

func TestSeaOrderChange_Fingerprint(t *testing.T) {
	s1 := "test-content-123"
	fp1 := biz.ComputeFingerprint(s1)
	fp2 := biz.ComputeFingerprint(s1)

	if fp1 == "" {
		t.Fatal("fingerprint should not be empty")
	}
	if fp1 != fp2 {
		t.Fatalf("fingerprint should be deterministic: %s vs %s", fp1, fp2)
	}

	s2 := "test-content-456"
	fp3 := biz.ComputeFingerprint(s2)
	if fp1 == fp3 {
		t.Fatal("different content must yield different fingerprint")
	}
}
