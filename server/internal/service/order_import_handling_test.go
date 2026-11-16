package service

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestImportHandlingWireProjection(t *testing.T) {
	id := uuid.New()
	actor := uuid.New()
	h := biz.EmptyImportHandling(id, 4)
	h.Steps[1].Undertaken = true
	now := time.Date(2026, 11, 16, 1, 2, 3, 0, time.FixedZone("offset", 8*3600))
	h.Steps[1].OccurredAt = &now
	h.Steps[1].ResponsibleUserID = &actor
	h.Steps[1].ResponsibleUserName = "进口操作员"
	wire, e := protojson.Marshal(importHandlingToAPI(h))
	if e != nil {
		t.Fatal(e)
	}
	var decoded v1.OrderImportHandling
	if e = protojson.Unmarshal(wire, &decoded); e != nil {
		t.Fatal(e)
	}
	if decoded.Steps[0].Undertaken || decoded.Steps[0].OccurredAt != nil || decoded.Steps[1].GetResponsibleUserName() != "进口操作员" || decoded.Steps[1].GetOccurredAt() != "2026-11-15T17:02:03Z" {
		t.Fatalf("wire零值/可选时间/姓名: %s", wire)
	}
	for _, flow := range []biz.OrderFlowStatus{biz.OrderFlowImportNotApplicable, biz.OrderFlowImportNotStarted, biz.OrderFlowImportInProgress, biz.OrderFlowImportCompleted} {
		if actual := orderFlowStatusFromAPI(orderFlowStatusToAPI(flow)); !reflect.DeepEqual(actual, flow) {
			t.Fatalf("flow双向映射: %s %s", flow, actual)
		}
	}
}

func TestImportHandlingCompletedTimeRoundTrip(t *testing.T) {
	for _, timestamp := range []string{"2026-11-16T01:02:03Z", "2026-11-16T09:02:03.123456789+08:00"} {
		t.Run(timestamp, func(t *testing.T) {
			occurredAt, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				t.Fatal(err)
			}
			occurredAt = occurredAt.UTC()
			before := biz.EmptyImportHandling(uuid.New(), 4)
			before.Steps[1].Undertaken = true
			before.Steps[1].OccurredAt = &occurredAt
			encoded, err := protojson.Marshal(importHandlingToAPI(before))
			if err != nil {
				t.Fatal(err)
			}
			var wire v1.OrderImportHandling
			if err := protojson.Unmarshal(encoded, &wire); err != nil {
				t.Fatal(err)
			}
			roundTripTime, err := time.Parse(time.RFC3339, wire.Steps[1].GetOccurredAt())
			if err != nil {
				t.Fatal(err)
			}
			roundTripTime = roundTripTime.UTC()
			after := biz.EmptyImportHandling(before.OrderID, before.OrderVersion)
			after.Steps[1].Undertaken = true
			after.Steps[1].OccurredAt = &roundTripTime
			if err := biz.ValidateImportHandling(after, before, ""); err != nil {
				t.Fatalf("原样保存完成时间不应要求更正原因，响应时间=%s: %v", wire.Steps[1].GetOccurredAt(), err)
			}
		})
	}
}
