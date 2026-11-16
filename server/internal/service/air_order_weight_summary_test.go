package service

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
	"strings"
	"testing"
	"time"
)

func TestAirOrderWeightSummaryWirePresence(t *testing.T) {
	empty, err := protojson.Marshal(airOrderToAPI(&biz.AirOrderInfo{}))
	if err != nil || strings.Contains(string(empty), "weightSummaries") {
		t.Fatalf("无确认应省略集合：%s %v", empty, err)
	}
	wire, err := protojson.Marshal(airOrderToAPI(&biz.AirOrderInfo{WeightSummaries: []*biz.AirOrderWeightSummary{
		{Source: "PLANNED", ConfirmedWeightKG: "0", ConfirmedAt: time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)},
		{Source: "ACTUAL", ConfirmedWeightKG: "125.5", ConfirmationStale: true},
	}}))
	if err != nil || !strings.Contains(string(wire), `"confirmedWeightKg":"0"`) || !strings.Contains(string(wire), `"confirmationStale":true`) {
		t.Fatalf("确认零及过期状态丢失：%s %v", wire, err)
	}
}
