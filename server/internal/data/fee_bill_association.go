package data

import (
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// effectiveBillLinePredicate 是「有效账单关联」的统一谓词：活动账单行且所属账单
// 未取消（含草稿账单）。费用是否已建账由该关联事实决定，不是一套独立状态机；
// 占用复核、候选查询、统计与台账投影必须复用同一口径。
func effectiveBillLinePredicate() predicate.FinanceBillLine {
	return financebillline.And(
		financebillline.ActiveEQ(true),
		financebillline.HasBillWith(financebill.StatusNEQ(financebill.StatusCANCELLED)),
	)
}

// effectiveBillLineFilter 是费→账单行的预加载修饰：按有效账单关联口径批量装配
// HasActiveBill 投影，禁止逐费用 N+1 查询，也禁止把未加载关联默认为未建账。
func effectiveBillLineFilter(lineQuery *ent.FinanceBillLineQuery) {
	lineQuery.Where(effectiveBillLinePredicate())
}
