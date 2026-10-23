package data

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestComputeTransportExecutionContentHashIncludesAllAuthoritativeRouteFields(t *testing.T) {
	shippingLineID := uuid.New()
	originID := uuid.New()
	dischargeID := uuid.New()
	transitID := uuid.New()
	etd := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	eta := time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC)
	base := ent.SeaTransportExecution{
		ShippingLineID:      shippingLineID,
		OriginLocationID:    &originID,
		DischargeLocationID: &dischargeID,
		TransitLocationID:   &transitID,
		VesselName:          "VESSEL A",
		VoyageNo:            "V001",
		Etd:                 &etd,
		Eta:                 &eta,
	}
	baseHash := computeTransportExecutionContentHash(&base)

	assertChangesHash := func(name string, mutate func(*ent.SeaTransportExecution)) {
		t.Helper()
		candidate := base
		mutate(&candidate)
		if got := computeTransportExecutionContentHash(&candidate); got == baseHash {
			t.Fatalf("权威航程字段 %s 变化后 content_hash 未变化", name)
		}
	}

	assertChangesHash("origin_location_id", func(v *ent.SeaTransportExecution) { id := uuid.New(); v.OriginLocationID = &id })
	assertChangesHash("discharge_location_id", func(v *ent.SeaTransportExecution) { id := uuid.New(); v.DischargeLocationID = &id })
	assertChangesHash("transit_location_id", func(v *ent.SeaTransportExecution) { id := uuid.New(); v.TransitLocationID = &id })
	assertChangesHash("vessel_name", func(v *ent.SeaTransportExecution) { v.VesselName = "VESSEL B" })
	assertChangesHash("voyage_no", func(v *ent.SeaTransportExecution) { v.VoyageNo = "V002" })
	assertChangesHash("etd", func(v *ent.SeaTransportExecution) { changed := etd.Add(time.Hour); v.Etd = &changed })
	assertChangesHash("eta", func(v *ent.SeaTransportExecution) { changed := eta.Add(time.Hour); v.Eta = &changed })
}

func TestComputeMBLContentHashIncludesAuthoritativeFields(t *testing.T) {
	shippingLineID := uuid.New()
	mbl := &ent.SeaMasterBill{
		MasterNo:           "MBL-ROUTE-HASH",
		NormalizedMasterNo: "MBL-ROUTE-HASH",
		ShippingLineID:     shippingLineID,
	}
	baseHash := computeMBLContentHash(mbl)

	mbl2 := &ent.SeaMasterBill{
		MasterNo:           "MBL-ROUTE-HASH-2",
		NormalizedMasterNo: "MBL-ROUTE-HASH-2",
		ShippingLineID:     shippingLineID,
	}
	if computeMBLContentHash(mbl2) == baseHash {
		t.Fatalf("MBL号码变化后 hash 未变化")
	}
}

func TestLockOrderFingerprintBindsReviewEvidence(t *testing.T) {
	org, order, caller := uuid.New(), uuid.New(), uuid.New()
	review := biz.OrderLockReviewInput{ExpectedFeeContentVersion: 1, FeeContentFingerprint: "snapshot-a", Comment: "审核意见"}
	original := computeLockOrderFingerprint(org, order, 1, caller, review)
	changed := review
	changed.ExpectedFeeContentVersion++
	if computeLockOrderFingerprint(org, order, 1, caller, changed) == original {
		t.Fatal("费用版本必须进入幂等指纹")
	}
	changed = review
	changed.FeeContentFingerprint = "snapshot-b"
	if computeLockOrderFingerprint(org, order, 1, caller, changed) == original {
		t.Fatal("费用内容必须进入幂等指纹")
	}
	changed = review
	changed.Comment = "另一意见"
	if computeLockOrderFingerprint(org, order, 1, caller, changed) == original {
		t.Fatal("审核意见必须进入幂等指纹")
	}
}
