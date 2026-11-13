package data

import (
	"context"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

// 两类主单的关联表结构一致。集中构造关联子查询，使列表、详情和搜索遵循同一
// 读取规则：活动订单取活动关系，终态仅取本次终止保留关系，不取旧改配单号。
func feeLedgerReadableMasterLink(link *entsql.Selector, air bool) *entsql.Predicate {
	business, endedReason := order.BusinessTypeSE, orderTerminationLinkEndedReason
	if air {
		business, endedReason = order.BusinessTypeAE, "订单终止"
	}
	o := entsql.Table(order.Table).As("ledger_master_order")
	return entsql.Exists(entsql.Select(o.C(order.FieldID)).From(o).Where(entsql.And(
		entsql.ColumnsEQ(o.C(order.FieldID), link.C("order_id")),
		entsql.ColumnsEQ(o.C(order.FieldOrganizationID), link.C("organization_id")),
		entsql.EQ(o.C(order.FieldBusinessType), business),
		entsql.Or(
			entsql.And(entsql.NEQ(o.C(order.FieldTerminationStatus), order.TerminationStatusTERMINATED), entsql.EQ(link.C("status"), "ACTIVE")),
			entsql.And(entsql.EQ(o.C(order.FieldTerminationStatus), order.TerminationStatusTERMINATED), entsql.EQ(link.C("status"), "ENDED"),
				entsql.EQ(link.C("ended_reason"), endedReason), entsql.ColumnsEQ(link.C("ended_at"), o.C(order.FieldTerminatedAt))),
		),
	)))
}

func feeLedgerMasterNoPredicate(keyword string) predicate.Order {
	return func(s *entsql.Selector) {
		sea := entsql.Table(seamasterbillorderlink.Table)
		mbl := entsql.Table(seamasterbill.Table)
		seaQuery := entsql.Select(sea.C(seamasterbillorderlink.FieldOrderID)).From(sea).
			Join(mbl).On(sea.C(seamasterbillorderlink.FieldMasterBillID), mbl.C(seamasterbill.FieldID))
		seaQuery.Where(entsql.And(feeLedgerReadableMasterLink(seaQuery, false),
			entsql.ColumnsEQ(sea.C(seamasterbillorderlink.FieldOrganizationID), mbl.C(seamasterbill.FieldOrganizationID)),
			entsql.ContainsFold(mbl.C(seamasterbill.FieldMasterNo), keyword)))
		airKeyword := strings.ReplaceAll(keyword, "-", "")
		if airKeyword == "" {
			s.Where(entsql.In(s.C(order.FieldID), seaQuery))
			return
		}
		air := entsql.Table(airmasterwaybillorderlink.Table)
		mawb := entsql.Table(airmasterwaybill.Table)
		airQuery := entsql.Select(air.C(airmasterwaybillorderlink.FieldOrderID)).From(air).
			Join(mawb).On(air.C(airmasterwaybillorderlink.FieldMasterWaybillID), mawb.C(airmasterwaybill.FieldID))
		airQuery.Where(entsql.And(feeLedgerReadableMasterLink(airQuery, true),
			entsql.ColumnsEQ(air.C(airmasterwaybillorderlink.FieldOrganizationID), mawb.C(airmasterwaybill.FieldOrganizationID)),
			entsql.ContainsFold(mawb.C(airmasterwaybill.FieldMasterNo), airKeyword)))
		s.Where(entsql.Or(entsql.In(s.C(order.FieldID), seaQuery), entsql.In(s.C(order.FieldID), airQuery)))
	}
}

func enrichFeeLedgerMasterNos(ctx context.Context, client *ent.Client, items []*biz.FeeLedgerItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.Fee.OrderID)
	}
	sea, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlink.OrderIDIn(ids...), func(s *entsql.Selector) {
		s.Where(feeLedgerReadableMasterLink(s, false))
	}).WithMasterBill().All(ctx)
	if err != nil {
		return err
	}
	air, err := client.AirMasterWaybillOrderLink.Query().Where(airmasterwaybillorderlink.OrderIDIn(ids...), func(s *entsql.Selector) {
		s.Where(feeLedgerReadableMasterLink(s, true))
	}).WithMasterWaybill().All(ctx)
	if err != nil {
		return err
	}
	numbers := make(map[uuid.UUID]string)
	for _, link := range sea {
		master, err := link.Edges.MasterBillOrErr()
		if err != nil {
			return err
		}
		if master.OrganizationID == link.OrganizationID {
			numbers[link.OrderID] = master.MasterNo
		}
	}
	for _, link := range air {
		master, err := link.Edges.MasterWaybillOrErr()
		if err != nil {
			return err
		}
		if master.OrganizationID == link.OrganizationID {
			numbers[link.OrderID] = master.MasterNo
		}
	}
	for _, item := range items {
		item.MasterNo = numbers[item.Fee.OrderID]
	}
	return nil
}
