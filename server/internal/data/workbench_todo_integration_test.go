package data

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	master "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	correction "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
	personnel "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
)

func workbenchTodoFee(t *testing.T, f *workbenchIntegrationFixture) *ent.OrderFeeCreate {
	t.Helper()
	return f.data.db.OrderFee.Create().SetOrderID(f.orderID).SetIdempotencyKey(uuid.NewString()).SetDirection(fee.DirectionRECEIVABLE).SetFeeCode("DOC").SetFeeName("单证费").SetSettlementPartyID(f.customerID).SetBillingUnit("票").SetQuantity("1").SetUnitPrice("1234.50").SetTotalAmount("1234.50").SetNetAmount("1234.50").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).SetExchangeRateDate(workbenchIntegrationDate).SetBaseCurrency("CNY").SetBaseCurrencyAmount("1234.50")
}

func workbenchTodoType(t *testing.T, f *workbenchIntegrationFixture, name string, enabled bool) *ent.MasterDataItem {
	t.Helper()
	return f.data.db.MasterDataItem.Create().SetKind(master.KindAbnormalCase).SetCode("WB-" + uuid.NewString()[:8]).SetName(name).SetEnabled(enabled).SaveX(t.Context())
}

func assertWorkbenchTodoCounts(t *testing.T, f *workbenchIntegrationFixture, scope biz.WorkbenchScope, wantFees, wantAbnormal int) {
	t.Helper()
	ctx := t.Context()
	overview, err := f.repo().GetOverview(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	fees, err := f.repo().ListMyUnbilledFees(ctx, scope, biz.WorkbenchTodoFilter{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatal(err)
	}
	abnormal, err := f.repo().ListMyOpenAbnormalCases(ctx, scope, biz.WorkbenchTodoFilter{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatal(err)
	}
	if overview.Todos.UnbilledFeeCount != wantFees || fees.Total != wantFees || len(fees.Items) != wantFees || overview.Todos.OpenAbnormalCount != wantAbnormal || abnormal.Total != wantAbnormal || len(abnormal.Items) != wantAbnormal {
		t.Fatalf("摘要/明细不一致: summary=%+v fees=%+v abnormal=%+v, want=%d/%d", overview.Todos, fees, abnormal, wantFees, wantAbnormal)
	}
}

func TestWorkbenchTodoScopePostgres(t *testing.T) {
	f := newWorkbenchIntegrationFixture(t)
	ctx := t.Context()
	employee := f.newEmployee("todo")
	f.addPersonnel(employee, personnel.RoleOPERATOR)
	kind := workbenchTodoType(t, f, "海关查验", false)
	f.data.db.OrderAbnormalCase.Create().SetOrderID(f.orderID).SetAbnormalCaseID(kind.ID).SetMarkedBy(f.actorID).SaveX(ctx)
	scope := f.scope(employee, nil)
	assertWorkbenchTodoCounts(t, f, scope, 1, 1)
	for _, scenario := range []struct {
		name           string
		mutate         func(*biz.WorkbenchScope)
		fees, abnormal int
	}{
		{"他人", func(s *biz.WorkbenchScope) { s.UserID = uuid.New() }, 0, 0},
		{"管理员也仅本人协作", func(s *biz.WorkbenchScope) { s.UserID = uuid.New(); s.IsBootstrapAdmin = true }, 0, 0},
		{"其他公司", func(s *biz.WorkbenchScope) { s.OrganizationID = uuid.New() }, 0, 0},
		{"无订单读取", func(s *biz.WorkbenchScope) { s.ReadableOrderBusinessTypes = nil }, 0, 0},
		{"仅空运订单读取", func(s *biz.WorkbenchScope) {
			s.ReadableOrderBusinessTypes = []biz.OrderBusinessType{biz.OrderBusinessAE}
		}, 0, 0},
		{"无费用读取", func(s *biz.WorkbenchScope) { s.ReadableFeeBusinessTypes = nil }, 0, 1},
		{"仅空运费用读取", func(s *biz.WorkbenchScope) { s.ReadableFeeBusinessTypes = []biz.OrderBusinessType{biz.OrderBusinessAE} }, 0, 1},
		{"无异常读取", func(s *biz.WorkbenchScope) { s.ReadableAbnormalBusinessTypes = nil }, 1, 0},
		{"仅空运异常读取", func(s *biz.WorkbenchScope) {
			s.ReadableAbnormalBusinessTypes = []biz.OrderBusinessType{biz.OrderBusinessAE}
		}, 1, 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			s := scope
			scenario.mutate(&s)
			if scenario.name == "其他公司" {
				// Overview 需要实际工作区组织；不能用不存在的组织掩盖跨公司过滤。
				org := f.data.db.Organization.Create().SetCode("TODO-" + uuid.NewString()[:8]).SetName("另一个公司").SetKind("company").SetBaseCurrency("CNY").SetEnabled(true).SaveX(ctx)
				s.OrganizationID = org.ID
			}
			assertWorkbenchTodoCounts(t, f, s, scenario.fees, scenario.abnormal)
		})
	}
	f.data.db.Order.UpdateOneID(f.orderID).SetBusinessType(orderent.BusinessTypeAE).ExecX(ctx)
	assertWorkbenchTodoCounts(t, f, scope, 1, 1)
	fees, err := f.repo().ListMyUnbilledFees(ctx, scope, biz.WorkbenchTodoFilter{Page: 1, PageSize: 20})
	if err != nil || fees.Items[0].BusinessType != "AE" || fees.Items[0].OrderNo != f.orderNo || fees.Items[0].CustomerName == "" || fees.Items[0].SettlementPartyName == "" {
		t.Fatalf("空运字段缺失: %+v %v", fees, err)
	}
}

func TestWorkbenchTodoFeeAccountingPostgres(t *testing.T) {
	f := newWorkbenchIntegrationFixture(t)
	ctx := t.Context()
	employee := f.newEmployee("accounting")
	f.addPersonnel(employee, personnel.RoleOPERATOR)
	scope := f.scope(employee, nil)
	assertWorkbenchTodoCounts(t, f, scope, 1, 0)
	// 有效草稿与已确认账单均占用费用；只有取消账单或非活动行才释放。
	f.data.db.FinanceBill.UpdateOneID(f.receivableBillID).SetStatus(bill.StatusDRAFT).ClearConfirmedAt().ClearConfirmedBy().ClearCancelledAt().ClearCancelledBy().ClearCancellationReason().ExecX(ctx)
	assertWorkbenchTodoCounts(t, f, scope, 1, 0)
	f.data.db.FinanceBill.UpdateOneID(f.receivableBillID).SetStatus(bill.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(f.actorID).SetCancellationReason("工作台测试取消").ExecX(ctx)
	assertWorkbenchTodoCounts(t, f, scope, 2, 0)
	f.data.db.FinanceBill.UpdateOneID(f.receivableBillID).SetStatus(bill.StatusDRAFT).ClearConfirmedAt().ClearConfirmedBy().ClearCancelledAt().ClearCancelledBy().ClearCancellationReason().ExecX(ctx)
	f.data.db.FinanceBillLine.Update().Where(billline.BillIDEQ(f.receivableBillID)).SetActive(false).ExecX(ctx)
	assertWorkbenchTodoCounts(t, f, scope, 2, 0)
	f.data.db.FinanceBillLine.Update().Where(billline.BillIDEQ(f.receivableBillID)).SetActive(true).ExecX(ctx)
	assertWorkbenchTodoCounts(t, f, scope, 1, 0)

	// 同一时间的两行按主键稳定跨页；无需记账、已替代及无需记账冲回均排除。
	sameTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first := workbenchTodoFee(t, f).SetCreatedAt(sameTime).SaveX(ctx)
	second := workbenchTodoFee(t, f).SetCreatedAt(sameTime).SaveX(ctx)
	workbenchTodoFee(t, f).SetAccountingRequired(false).SaveX(ctx)
	request := f.data.db.OrderFeeCorrectionRequest.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderID).SetOperation(correction.OperationVOID).SetSourceFeeID(first.ID).SetSourceFeeVersion(first.Version).SetBeforeSnapshot(json.RawMessage(`{}`)).SetApprovedFeeContentVersion(1).SetApprovedFeeReviewVersion(1).SetReason("金额录错").SetIdempotencyKey(uuid.NewString()).SetRequestFingerprint("todo-test").SetRequestedBy(f.actorID).SetRequestedAt(time.Now()).SetStatus(correction.StatusAPPROVED).SaveX(ctx)
	old := workbenchTodoFee(t, f).SetSupersededByRequestID(request.ID).SaveX(ctx)
	workbenchTodoFee(t, f).SetEconomicEffect(fee.EconomicEffectDECREASE).SetAccountingRequired(false).SetSourceFeeID(old.ID).SetCorrectionRequestID(request.ID).SaveX(ctx)
	request2 := f.data.db.OrderFeeCorrectionRequest.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderID).SetOperation(correction.OperationVOID).SetSourceFeeID(first.ID).SetSourceFeeVersion(first.Version).SetBeforeSnapshot(json.RawMessage(`{}`)).SetApprovedFeeContentVersion(1).SetApprovedFeeReviewVersion(1).SetReason("冲回记账").SetIdempotencyKey(uuid.NewString()).SetRequestFingerprint("todo-reversal").SetRequestedBy(f.actorID).SetRequestedAt(time.Now()).SetStatus(correction.StatusAPPROVED).SaveX(ctx)
	reversal := workbenchTodoFee(t, f).SetEconomicEffect(fee.EconomicEffectDECREASE).SetSourceFeeID(first.ID).SetCorrectionRequestID(request2.ID).SaveX(ctx)
	expectedIDs := []uuid.UUID{first.ID, second.ID}
	sort.Slice(expectedIDs, func(i, j int) bool { return expectedIDs[i].String() < expectedIDs[j].String() })
	for i, id := range expectedIDs {
		page, err := f.repo().ListMyUnbilledFees(ctx, scope, biz.WorkbenchTodoFilter{Page: i + 1, PageSize: 1})
		if err != nil || page.Total != 4 || len(page.Items) != 1 || page.Items[0].FeeID != id {
			t.Fatalf("稳定分页错误: %+v %v", page, err)
		}
	}
	all, err := f.repo().ListMyUnbilledFees(ctx, scope, biz.WorkbenchTodoFilter{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range all.Items {
		if row.FeeID == reversal.ID && row.Direction != biz.OrderFeePayable {
			t.Fatalf("冲回应投影实际应付: %+v", row)
		}
	}
	assertWorkbenchTodoCounts(t, f, scope, 4, 0)
}

func TestWorkbenchTodoAbnormalAndBlockedPostgres(t *testing.T) {
	f := newWorkbenchIntegrationFixture(t)
	ctx := t.Context()
	employee := f.newEmployee("blocked")
	f.addPersonnel(employee, personnel.RoleOPERATOR)
	scope := f.scope(employee, nil)
	source := workbenchTodoFee(t, f).SaveX(ctx)
	f.data.db.OrderFeeCorrectionRequest.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderID).SetOperation(correction.OperationVOID).SetSourceFeeID(source.ID).SetSourceFeeVersion(source.Version).SetBeforeSnapshot(json.RawMessage(`{}`)).SetApprovedFeeContentVersion(1).SetApprovedFeeReviewVersion(1).SetReason("待补审").SetIdempotencyKey(uuid.NewString()).SetRequestFingerprint("todo-pending").SetRequestedBy(f.actorID).SetRequestedAt(time.Now()).SetStatus(correction.StatusPENDING).SaveX(ctx)
	sameTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	kindA := workbenchTodoType(t, f, "查验（已停用）", false)
	kindB := workbenchTodoType(t, f, "货损", true)
	a := f.data.db.OrderAbnormalCase.Create().SetOrderID(f.orderID).SetAbnormalCaseID(kindA.ID).SetMarkedAt(sameTime).SetMarkedBy(f.actorID).SaveX(ctx)
	b := f.data.db.OrderAbnormalCase.Create().SetOrderID(f.orderID).SetAbnormalCaseID(kindB.ID).SetMarkedAt(sameTime).SetMarkedBy(f.actorID).SaveX(ctx)
	ids := []uuid.UUID{a.ID, b.ID}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	for i, id := range ids {
		page, err := f.repo().ListMyOpenAbnormalCases(ctx, scope, biz.WorkbenchTodoFilter{Page: i + 1, PageSize: 1})
		if err != nil || page.Total != 2 || len(page.Items) != 1 || page.Items[0].ID != id || page.Items[0].AbnormalCaseName == "" {
			t.Fatalf("异常同时间分页/名称错误: %+v %v", page, err)
		}
	}
	usecase := biz.NewOrderAbnormalCaseUsecase(NewOrderAbnormalCaseRepo(f.data))
	for _, scenario := range []struct {
		name   string
		apply  func()
		reason string
	}{
		{"审核锁", func() {
			f.data.db.Order.UpdateOneID(f.orderID).SetLockedAt(time.Now()).SetLockedBy(f.actorID).SetLockGeneration(1).ExecX(ctx)
		}, "锁定"},
		{"退关中", func() {
			f.data.db.Order.UpdateOneID(f.orderID).ClearLockedAt().ClearLockedBy().SetTerminationStatus(orderent.TerminationStatusTERMINATING).SetTerminationType(orderent.TerminationTypeCUSTOMS_RETURN).SetTerminationReason("退关测试").ExecX(ctx)
		}, "终止"},
		{"已退关", func() {
			f.data.db.Order.UpdateOneID(f.orderID).SetTerminationStatus(orderent.TerminationStatusTERMINATED).SetTerminatedAt(time.Now()).SetTerminatedBy(f.actorID).ExecX(ctx)
		}, "终止"},
		{"已结案", func() {
			f.data.db.Order.UpdateOneID(f.orderID).SetTerminationStatus(orderent.TerminationStatusACTIVE).ClearTerminationType().ClearTerminationReason().ClearTerminatedAt().ClearTerminatedBy().SetClosureStatus(orderent.ClosureStatusCLOSED).SetClosureReason("结案测试").SetClosedAt(time.Now()).SetClosedBy(f.actorID).ExecX(ctx)
		}, "结案"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			scenario.apply()
			if _, err := usecase.Resolve(ctx, f.organizationID, f.actorID, f.orderID, b.ID); err == nil {
				t.Fatal("业务只读状态允许解决异常")
			}
			if err := usecase.Remove(ctx, f.organizationID, f.actorID, f.orderID, b.ID); err == nil {
				t.Fatal("业务只读状态允许删除异常")
			}
			if _, err := usecase.Mark(ctx, f.organizationID, f.actorID, f.orderID, kindB.ID); err == nil {
				t.Fatal("业务只读状态允许登记异常")
			}
			assertWorkbenchTodoCounts(t, f, scope, 2, 2)
			fees, err := f.repo().ListMyUnbilledFees(ctx, scope, biz.WorkbenchTodoFilter{Page: 1, PageSize: 20})
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range fees.Items {
				if row.FeeID == source.ID {
					if !row.HasPendingChange || len(row.BlockedReasons) != 1 || !strings.Contains(row.BlockedReasons[0], "待补审") {
						t.Fatalf("源费补审原因丢失: %+v", row)
					}
				} else if row.HasPendingChange || len(row.BlockedReasons) > 0 {
					t.Fatalf("无关费用被补审/锁误阻: %+v", row)
				}
			}
			cases, err := f.repo().ListMyOpenAbnormalCases(ctx, scope, biz.WorkbenchTodoFilter{Page: 1, PageSize: 20})
			if err != nil || !strings.Contains(cases.Items[0].BusinessWriteBlockedReason, scenario.reason) {
				t.Fatalf("业务只读原因错误: %+v %v", cases, err)
			}
		})
	}
	// 原面板同样可以解析停用类型；解锁后解决/移除会刷新待办事实。
	original, err := NewOrderAbnormalCaseRepo(f.data).List(ctx, f.organizationID, f.orderID)
	if err != nil || len(original) != 2 || original[0].AbnormalCaseName == "" {
		t.Fatalf("原异常名称缺失: %+v %v", original, err)
	}
	f.data.db.Order.UpdateOneID(f.orderID).SetClosureStatus(orderent.ClosureStatusOPEN).ClearClosureReason().ClearClosedAt().ClearClosedBy().ExecX(ctx)
	resolved, err := usecase.Resolve(ctx, f.organizationID, f.actorID, f.orderID, a.ID)
	if err != nil || resolved.AbnormalCaseName != kindA.Name {
		t.Fatalf("解决停用类型异常失败: %+v %v", resolved, err)
	}
	assertWorkbenchTodoCounts(t, f, scope, 2, 1)
	if err := usecase.Remove(ctx, f.organizationID, f.actorID, f.orderID, b.ID); err != nil {
		t.Fatal(err)
	}
	assertWorkbenchTodoCounts(t, f, scope, 2, 0)
	marked, err := usecase.Mark(ctx, f.organizationID, f.actorID, f.orderID, kindB.ID)
	if err != nil || marked.AbnormalCaseName != kindB.Name {
		t.Fatalf("重新登记名称缺失: %+v %v", marked, err)
	}
	assertWorkbenchTodoCounts(t, f, scope, 2, 1)
}
