package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	abnormal "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// workbenchTodoOrderPredicate 让摘要和明细共用订单读取与分类读取的交集。
func workbenchTodoOrderPredicate(scope biz.WorkbenchScope, ids []uuid.UUID, readable []biz.OrderBusinessType) predicate.Order {
	types := make([]orderent.BusinessType, 0, len(readable))
	for _, businessType := range readable {
		for _, orderType := range scope.ReadableOrderBusinessTypes {
			if businessType == orderType && (businessType == biz.OrderBusinessSE || businessType == biz.OrderBusinessAE) {
				types = append(types, orderent.BusinessType(businessType))
				break
			}
		}
	}
	return orderent.And(orderent.IDIn(ids...), orderent.OrganizationIDEQ(scope.OrganizationID), orderent.BusinessTypeIn(types...))
}

func workbenchUnbilledFeeQuery(client *ent.Client, scope biz.WorkbenchScope, ids []uuid.UUID) *ent.OrderFeeQuery {
	return client.OrderFee.Query().Where(fee.OrderIDIn(ids...), accountingOrderFeePredicate(), fee.Not(fee.HasFinanceBillLinesWith(effectiveBillLinePredicate())), fee.HasOrderWith(workbenchTodoOrderPredicate(scope, ids, scope.ReadableFeeBusinessTypes)))
}

func workbenchOpenAbnormalQuery(client *ent.Client, scope biz.WorkbenchScope, ids []uuid.UUID) *ent.OrderAbnormalCaseQuery {
	return client.OrderAbnormalCase.Query().Where(abnormal.OrderIDIn(ids...), abnormal.StatusEQ(abnormal.StatusACTIVE), abnormal.HasOrderWith(workbenchTodoOrderPredicate(scope, ids, scope.ReadableAbnormalBusinessTypes)))
}

func workbenchTodoOrderToBiz(o *ent.Order) biz.WorkbenchTodoOrder {
	customer := ""
	if o.Edges.Customer != nil {
		customer = o.Edges.Customer.LegalName
	}
	return biz.WorkbenchTodoOrder{
		OrderID:           o.ID,
		OrderNo:           o.OrderNo,
		BusinessType:      string(o.BusinessType),
		CustomerName:      customer,
		OrderLocked:       o.LockedAt != nil,
		TerminationStatus: string(o.TerminationStatus),
		ClosureStatus:     string(o.ClosureStatus),
	}
}

func (r *workbenchRepo) ListMyUnbilledFees(ctx context.Context, scope biz.WorkbenchScope, f biz.WorkbenchTodoFilter) (*biz.PagedList[*biz.WorkbenchUnbilledFee], error) {
	empty := &biz.PagedList[*biz.WorkbenchUnbilledFee]{Items: []*biz.WorkbenchUnbilledFee{}, Page: f.Page, PageSize: f.PageSize}
	if len(scope.ReadableFeeBusinessTypes) == 0 {
		return empty, nil
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	ids, err := r.workbenchMyOrderIDs(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return empty, nil
	}
	query := workbenchUnbilledFeeQuery(client, scope, ids)
	pending := map[uuid.UUID]bool{}
	return paginate(ctx, func(ctx context.Context) (int, error) { return query.Clone().Count(ctx) }, func(ctx context.Context, offset, limit int) ([]*ent.OrderFee, error) {
		rows, err := query.Clone().WithOrder(func(q *ent.OrderQuery) { q.WithCustomer() }).WithSettlementParty().Order(fee.ByCreatedAt(), fee.ByID()).Offset(offset).Limit(limit).All(ctx)
		if err != nil {
			return nil, err
		}
		feeIDs := make([]uuid.UUID, 0, len(rows))
		for _, row := range rows {
			feeIDs = append(feeIDs, row.ID)
		}
		pending, err = pendingCorrectionSourceFees(ctx, client, feeIDs)
		return rows, err
	}, f.Page, f.PageSize, func(row *ent.OrderFee) (*biz.WorkbenchUnbilledFee, error) {
		amount, err := decimalOf(row.TotalAmount)
		if err != nil {
			return nil, err
		}
		o, err := row.Edges.OrderOrErr()
		if err != nil {
			return nil, err
		}
		party, err := row.Edges.SettlementPartyOrErr()
		if err != nil {
			return nil, err
		}
		item := &biz.WorkbenchUnbilledFee{
			WorkbenchTodoOrder:  workbenchTodoOrderToBiz(o),
			FeeID:               row.ID,
			Direction:           entFeeSettlementDirection(row),
			FeeName:             row.FeeName,
			SettlementPartyName: party.LegalName,
			Currency:            row.Currency,
			TotalAmount:         amount,
			CreatedAt:           row.CreatedAt,
			HasPendingChange:    pending[row.ID],
		}
		if item.HasPendingChange {
			item.BlockedReasons = []string{"原费用存在待补审申请，暂不能建账"}
		}
		return item, nil
	})
}

func (r *workbenchRepo) ListMyOpenAbnormalCases(ctx context.Context, scope biz.WorkbenchScope, f biz.WorkbenchTodoFilter) (*biz.PagedList[*biz.WorkbenchOpenAbnormalCase], error) {
	empty := &biz.PagedList[*biz.WorkbenchOpenAbnormalCase]{Items: []*biz.WorkbenchOpenAbnormalCase{}, Page: f.Page, PageSize: f.PageSize}
	if len(scope.ReadableAbnormalBusinessTypes) == 0 {
		return empty, nil
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	ids, err := r.workbenchMyOrderIDs(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return empty, nil
	}
	query := workbenchOpenAbnormalQuery(client, scope, ids)
	names := map[uuid.UUID]string{}
	return paginate(ctx, func(ctx context.Context) (int, error) { return query.Clone().Count(ctx) }, func(ctx context.Context, offset, limit int) ([]*ent.OrderAbnormalCase, error) {
		rows, err := query.Clone().WithOrder(func(q *ent.OrderQuery) { q.WithCustomer() }).Order(abnormal.ByMarkedAt(), abnormal.ByID()).Offset(offset).Limit(limit).All(ctx)
		if err != nil {
			return nil, err
		}
		typeIDs := make([]uuid.UUID, 0, len(rows))
		for _, row := range rows {
			typeIDs = append(typeIDs, row.AbnormalCaseID)
		}
		names, err = abnormalCaseNames(ctx, client, typeIDs)
		return rows, err
	}, f.Page, f.PageSize, func(row *ent.OrderAbnormalCase) (*biz.WorkbenchOpenAbnormalCase, error) {
		o, err := row.Edges.OrderOrErr()
		if err != nil {
			return nil, err
		}
		// 使用既有内容门禁；无需查询锁定人即可给出同口径原因。
		reason := orderBusinessEditBlockReason(ensureOrderBusinessContentEditable(ctx, nil, o))
		return &biz.WorkbenchOpenAbnormalCase{
			WorkbenchTodoOrder:         workbenchTodoOrderToBiz(o),
			ID:                         row.ID,
			AbnormalCaseID:             row.AbnormalCaseID,
			AbnormalCaseName:           names[row.AbnormalCaseID],
			MarkedAt:                   row.MarkedAt,
			BusinessWriteBlockedReason: reason,
		}, nil
	})
}
