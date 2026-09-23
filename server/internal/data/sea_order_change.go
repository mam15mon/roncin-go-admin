package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecommissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type seaOrderChangeRepo struct {
	data *Data
}

func NewSeaOrderChangeRepo(data *Data) biz.SeaOrderChangeRepo {
	return &seaOrderChangeRepo{data: data}
}

// ---------------------------------------------------------------------------
// 1. 动作摘要
// ---------------------------------------------------------------------------

func (r *seaOrderChangeRepo) GetChangeActions(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.SeaOrderChangeActions, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	order, err := client.Order.Query().
		Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}

	actions := &biz.SeaOrderChangeActions{
		CanSplit:               true,
		CanReassign:            true,
		SplitBlockedReasons:    []string{},
		ReassignBlockedReasons: []string{},
	}

	// 1. 业务类型门禁：仅海运出口 (SE)
	if order.BusinessType != orderent.BusinessTypeSE {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "非海运出口订单不支持拆票与改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
		return actions, nil
	}

	// 2. 流程与生命周期门禁：仅未终止 (ACTIVE) 且未关单 (OPEN)
	if order.TerminationStatus != orderent.TerminationStatusACTIVE {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单已终止，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}
	if order.ClosureStatus != orderent.ClosureStatusOPEN {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单已关单，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 2.5 统一业务内容门禁：终止流程与业务锁定同样阻断拆票/改配按钮，原因文案
	// 与写入侧门禁（ensureOrderBusinessContentEditable）同源。
	if gateErr := ensureOrderBusinessContentEditable(ctx, client.User, order); gateErr != nil {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单 " + order.OrderNo + " " + orderBusinessEditBlockReason(gateErr)
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 3. 唯一步调关系门禁
	activeLink, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrderIDEQ(orderID),
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		WithMasterBill().
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, err
		}
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单缺少活动的母单关联，无法拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
		return actions, nil
	}
	if activeLink.Edges.MasterBill == nil || activeLink.Edges.MasterBill.Status != seamasterbillent.StatusDRAFT {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "当前母单已确认或不可变，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 4. 下游财务与单证门禁检查
	// 仅检查唯一当前 HBL 是否仍为草稿；历史失效（VOIDED）HBL 不参与当前结构门禁
	nonDraftCurrentHblCount, err := client.SeaHouseBill.Query().
		Where(
			seahousebillent.OrderIDEQ(orderID),
			seahousebillent.StatusIn(seahousebillent.StatusCONFIRMED, seahousebillent.StatusRELEASED),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if nonDraftCurrentHblCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "当前分单(HBL)已确认或不可变，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 费用是否全为未建账或已作废（已建账费用构成不可改写的财务事实）
	billedFeeCount, err := client.OrderFee.Query().
		Where(
			orderfeeent.OrderIDEQ(orderID),
			orderfeeent.StatusNEQ(orderfeeent.StatusUNBILLED),
			orderfeeent.StatusNEQ(orderfeeent.StatusCANCELLED),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if billedFeeCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "存在已建账的费用，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 账单明细门禁
	activeBillLineCount, err := client.FinanceBillLine.Query().
		Where(financebilllineent.OrderIDEQ(orderID), financebilllineent.ActiveEQ(true)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if activeBillLineCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单费用已进入账单，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 提成事实门禁
	commissionCount, err := client.FinanceCommissionLine.Query().
		Where(financecommissionlineent.OrderIDEQ(orderID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if commissionCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单已产生提成计算事实，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 5. 拆票专属门禁：HOUSE 订单唯一当前 HBL 即可拆票，不要求多张 HBL
	if activeLink.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
		actions.CanSplit = false
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, "DIRECT 当前没有 HBL 箱货分配，暂不支持部分拆票，可执行整票改配")
	} else {
		currentHblCount, err := client.SeaHouseBill.Query().
			Where(
				seahousebillent.OrderIDEQ(orderID),
				seahousebillent.StatusIn(seahousebillent.StatusDRAFT, seahousebillent.StatusCONFIRMED, seahousebillent.StatusRELEASED),
			).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		if currentHblCount != 1 {
			actions.CanSplit = false
			actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, "HOUSE 订单必须恰有一张当前分单(HBL)才能拆票")
		}
	}

	return actions, nil
}

// ---------------------------------------------------------------------------
// 2. 拆票上下文
// ---------------------------------------------------------------------------

var _ biz.SeaOrderChangeRepo = (*seaOrderChangeRepo)(nil)
