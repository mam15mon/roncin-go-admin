package service

import (
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestParseContractDatesRequiresRFC3339(t *testing.T) {
	if _, _, err := parseContractDates("2026-08-20", "2027-08-20"); err != biz.ErrPartnerContractInvalidArgument {
		t.Fatalf("parse date error = %v, want ErrPartnerContractInvalidArgument", err)
	}
	start, end, err := parseContractDates("2026-08-20T00:00:00Z", "2027-08-20T00:00:00Z")
	if err != nil {
		t.Fatalf("parse RFC3339 dates error = %v", err)
	}
	if start.Format(time.RFC3339) != "2026-08-20T00:00:00Z" || end.Format(time.RFC3339) != "2027-08-20T00:00:00Z" {
		t.Fatalf("parsed dates = %s - %s", start, end)
	}
}
