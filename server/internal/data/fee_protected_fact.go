package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceexemption"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// feeHasProtectedFinanceFact 判断费用是否受后续财务事实保护（此时禁止按订单日期变化自动重取汇率）。
// 判定是账单级的：费用所在有效账单（活动账单行且账单未取消）上只要存在下列任一有效事实即受保护，
// 不要求事实逐行落到该费用上：
//
//   - 发票：活动开票关联且发票状态为 DRAFT 或 ISSUED（CANCELLED、RED_FLUSHED 不算）；
//
//   - 核销：活动核销分摊且核销单为 ACTIVE；
//
//   - 对冲：活动对冲分摊且对冲单为 CONFIRMED（草稿对冲不算）。
//
//   - 免票：存在未撤销的无需发票决定。
//
// 口径分层（同一组事实，按用途取舍）：
//   - 受保护事实（本谓词）：费用汇率联动与账单字段修改共用，草稿对冲不计入；
//   - 费用锁定事实（loadFinanceBillFeeLockFacts）：增删账单费用使用，在受保护事实之上追加草稿对冲，
//     因为草稿对冲确认时按账单当前快照重算分摊，费用集合变化会使已配对的抵销失真。
//
// 调用方须已按「订单 → 费用」固定锁序持有目标订单行锁：发票创建/开具、核销创建/反转、
// 对冲确认/反转、建账等写路径均先按主键升序锁 Order，故锁内重读即为最新事实。
// 必须通过 Data.client(ctx) 或事务内 client 传入，禁止直连连接池。
func feeHasProtectedFinanceFact(ctx context.Context, client *ent.Client, feeID uuid.UUID) (bool, error) {
	return client.FinanceBill.Query().
		Where(
			financebill.StatusNEQ(financebill.StatusCANCELLED),
			financebill.HasLinesWith(
				financebillline.OrderFeeIDEQ(feeID),
				financebillline.ActiveEQ(true),
			),
			protectedFinanceFactPredicate(),
		).
		Exist(ctx)
}

// billHasProtectedFinanceFact 判断指定账单自身是否存在受保护财务事实，口径与
// feeHasProtectedFinanceFact 相同（同一谓词，费用侧只多一步「费用 → 有效账单」）。
// 调用方须已按「订单 → 账单」锁序持有账单行锁。
func billHasProtectedFinanceFact(ctx context.Context, client *ent.Client, billID uuid.UUID) (bool, error) {
	return client.FinanceBill.Query().
		Where(financebill.IDEQ(billID), protectedFinanceFactPredicate()).
		Exist(ctx)
}

// HasProtectedFinanceFact 供账单更新用例在共享事务内读取受保护事实。
func (r *financeBillRepo) HasProtectedFinanceFact(ctx context.Context, billID uuid.UUID) (bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return false, err
	}
	return billHasProtectedFinanceFact(ctx, client, billID)
}

// protectedFinanceFactPredicate 是账单级「受保护财务事实」谓词，见 feeHasProtectedFinanceFact。
func protectedFinanceFactPredicate() predicate.FinanceBill {
	return financebill.Or(
		financebill.HasInvoiceExemptionsWith(financeinvoiceexemption.RevokedAtIsNil()),
		financebill.HasInvoiceLinksWith(
			financeinvoicebill.ActiveEQ(true),
			financeinvoicebill.HasInvoiceWith(financeinvoice.StatusIn(financeinvoice.StatusDRAFT, financeinvoice.StatusISSUED)),
		),
		financebill.HasVerificationAllocationsWith(
			financeverificationallocation.ActiveEQ(true),
			financeverificationallocation.HasVerificationWith(financeverification.StatusEQ(financeverification.StatusACTIVE)),
		),
		financebill.HasNettingAllocationsWith(
			financenettingallocation.ActiveEQ(true),
			financenettingallocation.HasNettingWith(financenetting.StatusEQ(financenetting.StatusCONFIRMED)),
		),
	)
}

// billFeeLockFacts 返回单张账单阻止增删费用的有效财务事实，口径见 loadFinanceBillFeeLockFacts。
// 写路径调用方须已按「订单 → 账单」锁序持有账单行锁，锁内读取即为最新事实。
func billFeeLockFacts(ctx context.Context, client *ent.Client, billID uuid.UUID) ([]biz.FinanceBillFeeLockFact, error) {
	facts, _, err := loadFinanceBillFeeLockFacts(ctx, client, []uuid.UUID{billID})
	if err != nil {
		return nil, err
	}
	return facts[billID], nil
}

// loadFinanceBillFeeLockFacts 批量读取账单的费用锁定事实与有效发票登记额，固定 5 次查询、无逐行 N+1：
//   - INVOICE：活动开票关联且发票为 DRAFT/ISSUED；同一查询累计登记额（开票进度的 invoiced）；
//   - VERIFICATION：活动核销分摊且核销单 ACTIVE；
//   - NETTING：活动对冲分摊且对冲单 CONFIRMED；
//   - NETTING_DRAFT：对冲单为 DRAFT 的分摊（草稿分摊未激活，与取消账单的依赖口径一致）。
//
// 前三类与 protectedFinanceFactPredicate 同源；事实按 biz.FinanceBillFeeLockFactOrder 排序。
func loadFinanceBillFeeLockFacts(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) (map[uuid.UUID][]biz.FinanceBillFeeLockFact, map[uuid.UUID]decimal.Decimal, error) {
	facts := make(map[uuid.UUID][]biz.FinanceBillFeeLockFact, len(billIDs))
	invoiced := make(map[uuid.UUID]decimal.Decimal, len(billIDs))
	if len(billIDs) == 0 {
		return facts, invoiced, nil
	}
	present := make(map[uuid.UUID]map[biz.FinanceBillFeeLockFact]bool, len(billIDs))
	mark := func(billID uuid.UUID, fact biz.FinanceBillFeeLockFact) {
		if present[billID] == nil {
			present[billID] = make(map[biz.FinanceBillFeeLockFact]bool, 1)
		}
		present[billID][fact] = true
	}
	links, err := client.FinanceInvoiceBill.Query().
		Where(
			financeinvoicebill.BillIDIn(billIDs...),
			financeinvoicebill.ActiveEQ(true),
			financeinvoicebill.HasInvoiceWith(financeinvoice.StatusIn(financeinvoice.StatusDRAFT, financeinvoice.StatusISSUED)),
		).
		Select(financeinvoicebill.FieldBillID, financeinvoicebill.FieldAmount).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, link := range links {
		amount, parseErr := decimalOf(link.Amount)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		invoiced[link.BillID] = invoiced[link.BillID].Add(amount)
		mark(link.BillID, biz.FinanceBillFeeLockInvoice)
	}
	factQueries := []struct {
		fact      biz.FinanceBillFeeLockFact
		predicate predicate.FinanceBill
	}{
		{biz.FinanceBillFeeLockInvoiceExemption, financebill.HasInvoiceExemptionsWith(financeinvoiceexemption.RevokedAtIsNil())},
		{biz.FinanceBillFeeLockVerification, financebill.HasVerificationAllocationsWith(
			financeverificationallocation.ActiveEQ(true),
			financeverificationallocation.HasVerificationWith(financeverification.StatusEQ(financeverification.StatusACTIVE)),
		)},
		{biz.FinanceBillFeeLockNetting, financebill.HasNettingAllocationsWith(
			financenettingallocation.ActiveEQ(true),
			financenettingallocation.HasNettingWith(financenetting.StatusEQ(financenetting.StatusCONFIRMED)),
		)},
		{biz.FinanceBillFeeLockNettingDraft, financebill.HasNettingAllocationsWith(
			financenettingallocation.HasNettingWith(financenetting.StatusEQ(financenetting.StatusDRAFT)),
		)},
	}
	for _, item := range factQueries {
		ids, queryErr := client.FinanceBill.Query().Where(financebill.IDIn(billIDs...), item.predicate).IDs(ctx)
		if queryErr != nil {
			return nil, nil, queryErr
		}
		for _, id := range ids {
			mark(id, item.fact)
		}
	}
	for billID, set := range present {
		ordered := make([]biz.FinanceBillFeeLockFact, 0, len(set))
		for _, fact := range biz.FinanceBillFeeLockFactOrder {
			if set[fact] {
				ordered = append(ordered, fact)
			}
		}
		facts[billID] = ordered
	}
	return facts, invoiced, nil
}
