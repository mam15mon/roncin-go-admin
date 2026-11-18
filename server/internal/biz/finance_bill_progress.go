package biz

import "github.com/shopspring/decimal"

// FinanceBillBusinessStatus 是有效账单的当前业务待办，不是持久化实体状态。
type FinanceBillBusinessStatus string

const (
	FinanceBillBusinessUninvoiced        FinanceBillBusinessStatus = "UNINVOICED"
	FinanceBillBusinessPartiallyInvoiced FinanceBillBusinessStatus = "PARTIALLY_INVOICED"
	FinanceBillBusinessUnverified        FinanceBillBusinessStatus = "UNVERIFIED"
	FinanceBillBusinessPartiallyVerified FinanceBillBusinessStatus = "PARTIALLY_VERIFIED"
	FinanceBillBusinessCompleted         FinanceBillBusinessStatus = "COMPLETED"
)

func ValidFinanceBillBusinessStatus(value FinanceBillBusinessStatus) bool {
	switch value {
	case FinanceBillBusinessUninvoiced, FinanceBillBusinessPartiallyInvoiced, FinanceBillBusinessUnverified,
		FinanceBillBusinessPartiallyVerified, FinanceBillBusinessCompleted:
		return true
	default:
		return false
	}
}

// ResolveFinanceBillBusinessStatus 与 data.billBusinessStatusPredicate 同源。
// 开票待办优先；纯对冲不冒充实际核销，完成条件由完整账务投影决定。
func ResolveFinanceBillBusinessStatus(status FinanceBillStatus, settlement FinanceBillSettlementProgress, invoice FinanceBillInvoiceProgress, completed bool, verified decimal.Decimal) FinanceBillBusinessStatus {
	if status != FinanceBillConfirmed || !ValidFinanceBillSettlementProgress(settlement) || !ValidFinanceBillInvoiceProgress(invoice) {
		return ""
	}
	if completed {
		return FinanceBillBusinessCompleted
	}
	switch invoice {
	case FinanceBillInvoiceNone:
		return FinanceBillBusinessUninvoiced
	case FinanceBillInvoicePartial:
		return FinanceBillBusinessPartiallyInvoiced
	}
	switch settlement {
	case FinanceBillSettlementUnsettled:
		return FinanceBillBusinessUnverified
	case FinanceBillSettlementPartial:
		if verified.IsZero() {
			return FinanceBillBusinessUnverified
		}
		return FinanceBillBusinessPartiallyVerified
	default:
		return ""
	}
}

// CanVerifyFinanceBillInvoiceProgress 只检查开票前置；余额、权限及公司仍由核销用例复核。
func CanVerifyFinanceBillInvoiceProgress(invoice FinanceBillInvoiceProgress) bool {
	return invoice == FinanceBillInvoicePartial || invoice == FinanceBillInvoiceFull || invoice == FinanceBillInvoiceNotRequired
}

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
	// FinanceBillFeeLockInvoiceExemption 表示当前有效的整账单免票决定。
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
//     关联有效）对本账单登记额的合计，需开票额取账单总额。
//   - 零额账单：无需核销、无需开票，结算进度为已结清、开票进度为无需开票（读取时推导，不冻结）。
//   - 负数账单：带符号净额等于账单总额即结清；开票仍按需票规则判定，不因负数自动免票。
//   - invoiceExempt 表示当前存在未撤销的「无需发票」决定。
//   - 只有已确认账单可以完成；取消账单的进度仍按事实投影，但不算完成。
//
// SQL 侧开票、结算及完成筛选谓词必须与本函数同源，修改时同步。
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
