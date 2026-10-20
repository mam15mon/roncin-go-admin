package biz

import (
	"github.com/google/uuid"
	"testing"
)

func TestAirMasterAndHouseNumberContract(t *testing.T) {
	for _, v := range []string{"781-12345675", " 78112345675 "} {
		got, e := NormalizeAirMasterNo(v)
		if e != nil || got != "78112345675" {
			t.Fatalf("规范主单号 %q: %q %v", v, got, e)
		}
	}
	for _, v := range []string{"", "78112345678", "1234567", "abc"} {
		if _, e := NormalizeAirMasterNo(v); e == nil {
			t.Fatalf("非法主单放行 %q", v)
		}
	}
	got, e := NormalizeAirHouseNo(" h001 ")
	if e != nil || got != "H001" {
		t.Fatalf("分单规范化 %q %v", got, e)
	}
	for _, v := range []string{"", "\n", "<script>"} {
		if _, e := NormalizeAirHouseNo(v); e == nil {
			t.Fatalf("非法分单放行 %q", v)
		}
	}
}
func TestAirMasterTimeAndAirportValidation(t *testing.T) {
	in := &AirMasterWaybillInput{MasterNo: "78112345675", AirlineID: uuid.New(), OriginAirportID: uuid.New(), DestinationAirportID: uuid.New(), ETD: "2026-10-22T08:00:00+08:00", ETA: "2026-10-22T15:00:00+08:00"}
	out, e := NormalizeAirMasterInput(in)
	if e != nil || out.ETD != "2026-10-22T00:00:00Z" {
		t.Fatalf("时区归一 %v %v", out, e)
	}
	bad := *in
	bad.ETA = "2026-10-21T12:00:00Z"
	if _, e := NormalizeAirMasterInput(&bad); e == nil {
		t.Fatal("ETA 早于 ETD 放行")
	}
	bad = *in
	bad.DestinationAirportID = bad.OriginAirportID
	if _, e := NormalizeAirMasterInput(&bad); e == nil {
		t.Fatal("相同机场放行")
	}
}
