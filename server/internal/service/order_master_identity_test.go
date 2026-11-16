package service

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestMasterCandidatesRequireExplicitBusinessIdentity(t *testing.T) {
	for _, businessType := range []string{"", "SI", "AI", "SE", "AE", "UNKNOWN"} {
		t.Run(businessType, func(t *testing.T) {
			if businessType != "SE" && businessType != "SI" {
				_, err := NewOrderService(nil).MatchSeaMasterBillCandidate(context.Background(), &v1.MatchSeaMasterBillCandidateRequest{BusinessType: businessType})
				if !errors.Is(err, biz.ErrSeaMasterBillInvalidArgument) {
					t.Fatalf("海运候选未拒绝不适用类型 %q: %v", businessType, err)
				}
			}
			if businessType != "AE" {
				_, err := NewAirExportService(nil).ListAirMasterWaybills(context.Background(), &v1.ListAirMasterWaybillsRequest{BusinessType: businessType})
				if !errors.Is(err, biz.ErrAirWaybillInvalid) {
					t.Fatalf("空运候选未拒绝不适用类型 %q: %v", businessType, err)
				}
			}
		})
	}
}
