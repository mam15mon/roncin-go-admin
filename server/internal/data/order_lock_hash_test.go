package data

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestComputeMBLContentHashIncludesAllAuthoritativeRouteFields(t *testing.T) {
	carrierID := uuid.New()
	originID := uuid.New()
	dischargeID := uuid.New()
	transitID := uuid.New()
	etd := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	eta := time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC)
	mbl := &ent.SeaMasterBill{
		MasterNo:             "MBL-ROUTE-HASH",
		NormalizedMasterNo:   "MBL-ROUTE-HASH",
		IssuerPartnerID:      uuid.New(),
		TransportExecutionID: uuid.New(),
	}
	base := ent.SeaTransportExecution{
		CarrierID:           &carrierID,
		OriginLocationID:    &originID,
		DischargeLocationID: &dischargeID,
		TransitLocationID:   &transitID,
		VesselName:          "VESSEL A",
		VoyageNo:            "V001",
		Etd:                 &etd,
		Eta:                 &eta,
	}
	baseHash := computeMBLContentHash(mbl, &base)

	assertChangesHash := func(name string, mutate func(*ent.SeaTransportExecution)) {
		t.Helper()
		candidate := base
		mutate(&candidate)
		if got := computeMBLContentHash(mbl, &candidate); got == baseHash {
			t.Fatalf("权威航程字段 %s 变化后 content_hash 未变化", name)
		}
	}

	assertChangesHash("carrier_id", func(v *ent.SeaTransportExecution) { id := uuid.New(); v.CarrierID = &id })
	assertChangesHash("origin_location_id", func(v *ent.SeaTransportExecution) { id := uuid.New(); v.OriginLocationID = &id })
	assertChangesHash("discharge_location_id", func(v *ent.SeaTransportExecution) { id := uuid.New(); v.DischargeLocationID = &id })
	assertChangesHash("transit_location_id", func(v *ent.SeaTransportExecution) { id := uuid.New(); v.TransitLocationID = &id })
	assertChangesHash("vessel_name", func(v *ent.SeaTransportExecution) { v.VesselName = "VESSEL B" })
	assertChangesHash("voyage_no", func(v *ent.SeaTransportExecution) { v.VoyageNo = "V002" })
	assertChangesHash("etd", func(v *ent.SeaTransportExecution) { changed := etd.Add(time.Hour); v.Etd = &changed })
	assertChangesHash("eta", func(v *ent.SeaTransportExecution) { changed := eta.Add(time.Hour); v.Eta = &changed })
}
