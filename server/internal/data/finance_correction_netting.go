package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// applyFinanceCorrectionNettingEligibility 只允许同一步骤的原费/冲回/替代清偿，不能占用合并账单其他行。
func applyFinanceCorrectionNettingEligibility(ctx context.Context, client *ent.Client, bills []*ent.FinanceBill, result []*biz.FinanceNettingBill) error {
	ids := make([]uuid.UUID, 0, len(bills))
	var requestID, sourceID *uuid.UUID
	for _, b := range bills {
		ids = append(ids, b.ID)
		if b.Purpose != bill.PurposeCORRECTION_REVERSAL {
			continue
		}
		for _, l := range b.Edges.Lines {
			f, err := l.Edges.OrderFeeOrErr()
			if err != nil {
				return err
			}
			if f.EconomicEffect != fee.EconomicEffectDECREASE || f.CorrectionRequestID == nil || f.SourceFeeID == nil {
				return biz.ErrFinanceNettingMismatch
			}
			if requestID != nil && *requestID != *f.CorrectionRequestID {
				return biz.ErrFinanceNettingMismatch
			}
			requestID, sourceID = f.CorrectionRequestID, f.SourceFeeID
		}
	}
	projection, err := loadFinanceBillLineSettlements(ctx, client, ids)
	if err != nil {
		return err
	}
	balances := make(map[uuid.UUID]decimal.Decimal)
	for _, p := range projection {
		balances[p.FeeID] = balances[p.FeeID].Add(p.RemainingAmount)
	}
	hasOriginal := false
	if sourceID != nil {
		for _, b := range bills {
			for _, l := range b.Edges.Lines {
				if l.OrderFeeID == *sourceID {
					hasOriginal = true
				}
			}
		}
	}
	var inherited decimal.Decimal
	if requestID != nil {
		inherited, err = correctionReversalCashCoverage(ctx, client, *requestID)
		if err != nil {
			return err
		}
	}
	for i, b := range bills {
		if requestID == nil {
			for _, l := range b.Edges.Lines {
				f, err := l.Edges.OrderFeeOrErr()
				if err != nil {
					return err
				}
				if f.SupersededByRequestID != nil && balances[f.ID].IsPositive() {
					return biz.ErrFinanceCorrectionSettlementRequired
				}
			}
			continue
		}
		var selected *ent.OrderFee
		for _, l := range b.Edges.Lines {
			f, err := l.Edges.OrderFeeOrErr()
			if err != nil {
				return err
			}
			if f.ID == *sourceID || (f.CorrectionRequestID != nil && *f.CorrectionRequestID == *requestID) {
				if selected != nil {
					return biz.ErrFinanceNettingMismatch
				}
				selected = f
			}
		}
		if selected != nil && selected.SupersededByRequestID != nil && selected.ID != *sourceID {
			return biz.ErrFinanceNettingMismatch
		}
		if selected == nil {
			return biz.ErrFinanceNettingMismatch
		}
		amount := balances[selected.ID]
		if selected.ID != *sourceID && selected.EconomicEffect != fee.EconomicEffectDECREASE {
			amount = decimal.Min(amount, inherited)
		}
		if selected.EconomicEffect == fee.EconomicEffectDECREASE && !hasOriginal {
			amount = decimal.Min(amount, inherited)
		}
		id := selected.ID
		result[i].SourceFeeID = &id
		result[i].CorrectionRequestID = requestID
		result[i].EligibleAmount = &amount
	}
	return nil
}
