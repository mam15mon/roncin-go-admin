package biz

import "github.com/shopspring/decimal"

// FinanceBillSettlementProgress 是账单结算进度投影。
type FinanceBillSettlementProgress string

const (
	FinanceBillSettlementUnsettled FinanceBillSettlementProgress = "UNSETTLED"
	FinanceBillSettlementPartial   FinanceBillSettlementProgress = "PARTIAL"
	FinanceBillSettlementSettled   FinanceBillSettlementProgress = "SETTLED"
)

// FinanceBillInvoiceProgress 是账单开票进度投影。
type FinanceBillInvoiceProgress string

const (
	FinanceBillInvoiceNotRequired FinanceBillInvoiceProgress = "NOT_REQUIRED"
	FinanceBillInvoiceNone        FinanceBillInvoiceProgress = "NONE"
	FinanceBillInvoicePartial     FinanceBillInvoiceProgress = "PARTIAL"
	FinanceBillInvoiceFull        FinanceBillInvoiceProgress = "FULL"
)

// FinanceBillFeeLockFact 是阻止账单增删费用的有效财务事实类型。
type FinanceBillFeeLockFact string

const (
	FinanceBillFeeLockInvoice      FinanceBillFeeLockFact = "INVOICE"
	FinanceBillFeeLockVerification FinanceBillFeeLockFact = "VERIFICATION"
	// FinanceBillFeeLockNetting 为已确认对冲；同时属于账单字段与费用汇率联动的受保护事实。
	FinanceBillFeeLockNetting FinanceBillFeeLockFact = "NETTING"
	// FinanceBillFeeLockNettingDraft 为草稿对冲：只阻止增删费用（草稿确认时按账单快照重算分摊，
	// 费用集合变化会使配对失真），不限制账单字段修改，也不阻止费用汇率联动。
	FinanceBillFeeLockNettingDraft FinanceBillFeeLockFact = "NETTING_DRAFT"
	// FinanceBillFeeLockInvoiceExemption 为「无需发票」标记的扩展位；标记模型由发票任务落地，
	// 落地前不会产生该事实。
	FinanceBillFeeLockInvoiceExemption FinanceBillFeeLockFact = "INVOICE_EXEMPTION"
)

// FinanceBillFeeLockFactOrder 是事实类型的固定输出顺序，保证投影与提示稳定。
var FinanceBillFeeLockFactOrder = []FinanceBillFeeLockFact{
	FinanceBillFeeLockInvoice,
	FinanceBillFeeLockVerification,
	FinanceBillFeeLockNetting,
	FinanceBillFeeLockNettingDraft,
	FinanceBillFeeLockInvoiceExemption,
}

// ValidFinanceBillSettlementProgress 校验结算进度筛选值。
func ValidFinanceBillSettlementProgress(value FinanceBillSettlementProgress) bool {
	switch value {
	case FinanceBillSettlementUnsettled, FinanceBillSettlementPartial, FinanceBillSettlementSettled:
		return true
	default:
		return false
	}
}

// ValidFinanceBillInvoiceProgress 校验开票进度筛选值。
func ValidFinanceBillInvoiceProgress(value FinanceBillInvoiceProgress) bool {
	switch value {
	case FinanceBillInvoiceNotRequired, FinanceBillInvoiceNone, FinanceBillInvoicePartial, FinanceBillInvoiceFull:
		return true
	default:
		return false
	}
}

// ResolveFinanceBillProgress 按账单币种、带符号金额推导结算进度、开票进度与是否完成。
//
//   - settled 为有效核销分摊与有效对冲分摊的带符号合计；invoiced 为有效发票（草稿或已开具、
//     关联有效）对本账单登记额的合计；需开票额当前取账单总额，发票任务确定分配事实后再替换。
//   - 零额账单：无需核销、无需开票，结算进度为已结清、开票进度为无需开票（读取时推导，不冻结）。
//   - 负数账单：带符号净额等于账单总额即结清；开票仍按需票规则判定，不因负数自动免票。
//   - invoiceExempt 为「无需发票」标记扩展位，发票任务落地前调用方恒传 false。
//   - 只有已确认账单可以完成；取消账单的进度仍按事实投影，但不算完成。
//
// SQL 侧筛选（data.financeBillProgressPredicate）必须与本函数同源，修改时同步。
func ResolveFinanceBillProgress(status FinanceBillStatus, total, settled, invoiced decimal.Decimal, invoiceExempt bool) (FinanceBillSettlementProgress, FinanceBillInvoiceProgress, bool) {
	var settlement FinanceBillSettlementProgress
	switch {
	case total.IsZero() || settled.Equal(total):
		settlement = FinanceBillSettlementSettled
	case settled.IsZero():
		settlement = FinanceBillSettlementUnsettled
	default:
		settlement = FinanceBillSettlementPartial
	}
	var invoice FinanceBillInvoiceProgress
	switch {
	case total.IsZero() || invoiceExempt:
		invoice = FinanceBillInvoiceNotRequired
	case invoiced.IsZero():
		invoice = FinanceBillInvoiceNone
	case invoiced.Equal(total):
		invoice = FinanceBillInvoiceFull
	default:
		invoice = FinanceBillInvoicePartial
	}
	completed := status == FinanceBillConfirmed && settlement == FinanceBillSettlementSettled &&
		(invoice == FinanceBillInvoiceFull || invoice == FinanceBillInvoiceNotRequired)
	return settlement, invoice, completed
}
