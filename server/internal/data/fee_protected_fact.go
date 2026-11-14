package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// feeHasProtectedFinanceFact 判断费用是否受后续财务事实保护（此时禁止按订单日期变化自动重取汇率）。
// 判定是账单级的：费用所在有效账单（活动账单行且账单未取消）上只要存在下列任一有效事实即受保护，
// 不要求事实逐行落到该费用上：
//   - 发票：活动开票关联且发票状态为 DRAFT 或 ISSUED（CANCELLED、RED_FLUSHED 不算）；
//   - 核销：活动核销分摊且核销单为 ACTIVE；
//   - 对冲：活动对冲分摊且对冲单为 CONFIRMED（草稿对冲不算）。
//
// 扩展点：账单任务引入「无需发票」标记后，在 protectedFinanceFactPredicate 追加一项
// 「账单已标记无需发票」条件即可；本任务不创建该字段。
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
// feeHasProtectedFinanceFact 相同（同一谓词）。调用方须已按「订单 → 账单」锁序持有账单行锁。
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
