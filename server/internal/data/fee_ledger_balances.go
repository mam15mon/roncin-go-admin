package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

// enrichFeeLedgerBalances 批量重放相关账单的完整行集合，避免按当前页费用分摊整张账单金额。
// 核销、对冲和反转复用资金结算的唯一投影；列表与整单详情保持相同余额口径。
func enrichFeeLedgerBalances(ctx context.Context, client *ent.Client, items []*biz.FeeLedgerItem, fees []*ent.OrderFee) error {
	billIDs := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]bool)
	for _, fee := range fees {
		for _, line := range fee.Edges.FinanceBillLines {
			if !seen[line.BillID] {
				seen[line.BillID] = true
				billIDs = append(billIDs, line.BillID)
			}
		}
	}
	settlements, err := loadFinanceBillLineSettlements(ctx, client, billIDs)
	if err != nil {
		return err
	}
	byFee := make(map[uuid.UUID]*biz.FinanceBillLineSettlement, len(settlements))
	for _, settlement := range settlements {
		byFee[settlement.FeeID] = settlement
	}
	for _, item := range items {
		item.UnverifiedAmount = item.Fee.TotalAmount
		settlement := byFee[item.Fee.ID]
		if settlement == nil {
			if item.BillNo != "" {
				return biz.ErrFinanceSettlementProjection
			}
			continue
		}
		item.UnverifiedAmount = settlement.RemainingAmount
		for _, contribution := range settlement.Contributions {
			if contribution.ReversedAt != nil {
				continue
			}
			if contribution.SourceType == "VERIFICATION" {
				item.VerifiedAmount = item.VerifiedAmount.Add(contribution.Amount)
			} else if contribution.SourceType == "NETTING" {
				item.NettedAmount = item.NettedAmount.Add(contribution.Amount)
			}
		}
	}
	return nil
}
