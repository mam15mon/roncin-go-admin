package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// correctionReversalCashCoverage 只转移原费已经真实实现的结算覆盖，不能先消费尚未清掉的原债。
func correctionReversalCashCoverage(ctx context.Context, client *ent.Client, requestID uuid.UUID) (decimal.Decimal, error) {
	rows, err := client.OrderFee.Query().Where(fee.CorrectionRequestIDEQ(requestID), fee.EconomicEffectEQ(fee.EconomicEffectDECREASE)).All(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	if len(rows) != 1 || rows[0].SourceFeeID == nil {
		return decimal.Zero, biz.ErrFinanceNettingMismatch
	}
	reversal := rows[0]
	sourceID := *reversal.SourceFeeID
	links, err := client.FinanceBillLine.Query().Where(billline.OrderFeeIDIn(sourceID, reversal.ID), effectiveBillLinePredicate()).All(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	ids := make([]uuid.UUID, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.BillID)
	}
	projection, err := loadFinanceBillLineSettlements(ctx, client, ids)
	if err != nil {
		return decimal.Zero, err
	}
	cash := decimal.Zero
	originalOffsets := make(map[uuid.UUID]bool)
	for _, p := range projection {
		if p.FeeID != sourceID {
			continue
		}
		for _, c := range p.Contributions {
			if c.ReversedAt != nil {
				continue
			}
			if c.IsCorrectionSettlement && c.CorrectionRequestID != nil && *c.CorrectionRequestID == requestID {
				originalOffsets[c.SourceID] = true
			} else {
				cash = cash.Add(c.Amount)
			}
		}
	}
	for _, p := range projection {
		if p.FeeID != reversal.ID {
			continue
		}
		for _, c := range p.Contributions {
			if c.ReversedAt == nil && !originalOffsets[c.SourceID] {
				cash = cash.Sub(c.Amount)
			}
		}
	}
	if cash.IsNegative() {
		return decimal.Zero, biz.ErrFinanceSettlementProjection
	}
	return cash, nil
}

func validateFinanceCorrectionCoverages(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) error {
	lines, err := client.FinanceBillLine.Query().Where(billline.BillIDIn(billIDs...)).All(ctx)
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(lines))
	for _, l := range lines {
		ids = append(ids, l.OrderFeeID)
	}
	reversals, err := client.OrderFee.Query().Where(fee.EconomicEffectEQ(fee.EconomicEffectDECREASE), fee.Or(fee.IDIn(ids...), fee.SourceFeeIDIn(ids...))).All(ctx)
	if err != nil {
		return err
	}
	for _, r := range reversals {
		if r.CorrectionRequestID == nil {
			return biz.ErrFinanceNettingMismatch
		}
		if _, err := correctionReversalCashCoverage(ctx, client, *r.CorrectionRequestID); err != nil {
			return err
		}
	}
	return nil
}
