package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	netting "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingallocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	verificationallocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// loadFinanceBillLineSettlements 与提成共用完整事件重放，保留已反转事件的原分摊证据。
func loadFinanceBillLineSettlements(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) ([]*biz.FinanceBillLineSettlement, error) {
	if len(billIDs) == 0 {
		return nil, nil
	}
	rawLines, err := client.FinanceBillLine.Query().Where(billline.BillIDIn(billIDs...), billline.ActiveEQ(true)).WithBill().WithOrderFee(func(q *ent.OrderFeeQuery) { q.WithSupersededByRequest() }).All(ctx)
	if err != nil {
		return nil, err
	}
	lines := make([]biz.FinanceSettlementLine, 0, len(rawLines))
	for _, line := range rawLines {
		amount, err := decimalOf(line.TotalAmount)
		if err != nil {
			return nil, err
		}
		b, err := line.Edges.BillOrErr()
		if err != nil {
			return nil, err
		}
		fee, err := line.Edges.OrderFeeOrErr()
		if err != nil {
			return nil, err
		}
		item := biz.FinanceSettlementLine{BillID: line.BillID, ID: line.ID, FeeID: line.OrderFeeID, Direction: biz.OrderFeeDirection(b.Direction), TotalAmount: amount}
		if fee.SupersededByRequestID != nil {
			request, err := fee.Edges.SupersededByRequestOrErr()
			if err != nil {
				return nil, err
			}
			item.SupersededAt = request.DecidedAt
		}
		lines = append(lines, item)
	}
	events := make([]biz.FinanceSettlementEvent, 0)
	verifications, err := client.FinanceVerificationAllocation.Query().Where(verificationallocation.BillIDIn(billIDs...), verificationallocation.HasVerificationWith(verification.StatusIn(verification.StatusACTIVE, verification.StatusREVERSED))).WithVerification().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, allocation := range verifications {
		parent, err := allocation.Edges.VerificationOrErr()
		if err != nil {
			return nil, err
		}
		// 失效但没有真实反转的孤立分摊不属于已发生结算。
		if !allocation.Active && parent.Status != verification.StatusREVERSED {
			continue
		}
		amount, err := decimalOf(allocation.Amount)
		if err != nil {
			return nil, err
		}
		events = append(events, biz.FinanceSettlementEvent{SourceType: "VERIFICATION", SourceID: parent.ID, AllocationID: allocation.ID, BillID: allocation.BillID, Amount: amount, EffectiveAt: parent.CreatedAt, ReversedAt: parent.ReversedAt})
	}
	nettings, err := client.FinanceNettingAllocation.Query().Where(nettingallocation.BillIDIn(billIDs...), nettingallocation.HasNettingWith(netting.StatusIn(netting.StatusCONFIRMED, netting.StatusREVERSED))).WithNetting().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, allocation := range nettings {
		parent, err := allocation.Edges.NettingOrErr()
		if err != nil {
			return nil, err
		}
		if !allocation.Active && parent.Status != netting.StatusREVERSED {
			continue
		}
		if parent.ConfirmedAt == nil {
			return nil, biz.ErrFinanceSettlementProjection
		}
		amount, err := decimalOf(allocation.Amount)
		if err != nil {
			return nil, err
		}
		events = append(events, biz.FinanceSettlementEvent{SourceType: "NETTING", SourceID: parent.ID, AllocationID: allocation.ID, BillID: allocation.BillID, SourceFeeID: allocation.SourceFeeID, Amount: amount, EffectiveAt: *parent.ConfirmedAt, ReversedAt: parent.ReversedAt, CorrectionRequestID: parent.CorrectionRequestID})
	}
	return biz.ProjectFinanceBillLineSettlements(lines, events)
}

// validateOrdinaryFinanceSettlement 被替代原行清偿完成后，普通收款只分摊剩余合法行。
func validateOrdinaryFinanceSettlement(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) error {
	projection, err := loadFinanceBillLineSettlements(ctx, client, billIDs)
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(projection))
	for _, p := range projection {
		if p.RemainingAmount.IsPositive() {
			ids = append(ids, p.FeeID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	exists, err := client.OrderFee.Query().Where(orderfee.IDIn(ids...), orderfee.SupersededByRequestIDNotNil()).Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return biz.ErrFinanceCorrectionSettlementRequired
	}
	return nil
}

func lockFinanceEconomicOrders(ctx context.Context, client *ent.Client, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := client.Order.Query().Where(order.IDIn(ids...)).Order(order.ByID()).ForUpdate().All(ctx)
	return err
}

// validateFinanceEconomicOrderSet 拿齐预读订单锁后再次展开父提成订单集合。
// 等锁期间若新的跨票父单已经提交，应当整体重试，不能在资金父锁后补锁其他订单。
func validateFinanceEconomicOrderSet(ctx context.Context, client *ent.Client, org uuid.UUID, billIDs, lockedOrderIDs []uuid.UUID) error {
	fresh, err := commissionEconomicAffectedOrderIDs(ctx, client, org, billIDs)
	if err != nil {
		return err
	}
	if len(fresh) != len(lockedOrderIDs) {
		return biz.ErrOrderStatusConflict
	}
	for i, id := range fresh {
		if lockedOrderIDs[i] != id {
			return biz.ErrOrderStatusConflict
		}
	}
	return nil
}
