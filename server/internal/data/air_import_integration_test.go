package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	airlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderimporthandlingevent"
)

func TestAirImportHandlingLifecyclePostgres(t *testing.T) {
	for _, mode := range []string{"DIRECT", "HOUSE"} {
		t.Run(mode, func(t *testing.T) {
			d, cleanup := getIntegrationData(t)
			t.Cleanup(cleanup)
			f := newAirExportFixture(t, d)
			p := airPrincipalForBusiness(f, biz.OrderBusinessAI)
			ctx := biz.WithPrincipal(context.Background(), p)
			in := airInputForBusiness(f, biz.OrderBusinessAI, "78112345675", "AI-HANDLING")
			if mode == "DIRECT" {
				in.AirExportInput.DocumentMode = mode
				in.AirExportInput.House = nil
			}
			o := f.create(t, in)
			uc := f.newUsecase()
			air := biz.NewAirExportUsecase(NewAirExportRepo(d))
			h, err := uc.GetImportHandling(ctx, f.organizationID, o.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(h.Steps) != 5 || h.OrderVersion != o.Version || biz.ImportHandlingFlow(h) != biz.OrderFlowImportNotApplicable || o.FlowStatus != biz.OrderFlowImportNotApplicable {
				t.Fatal("进口建票未创建真实五项空服务")
			}
			save := func(next *biz.OrderImportHandling, reason string) *biz.OrderImportHandling {
				t.Helper()
				out, err := uc.SaveImportHandling(ctx, f.organizationID, f.actorID, o.ID, h.OrderVersion, next, reason)
				if err != nil {
					t.Fatal(err)
				}
				return out
			}
			h.ExchangeAgentID = &f.partnerID
			h.Steps[1].Undertaken = true
			h.Steps[4].Undertaken = true
			h = save(h, "")
			if biz.ImportHandlingFlow(h) != biz.OrderFlowImportNotStarted {
				t.Fatal("承接摘要错误")
			}
			// 派送先完成，未承接项目不强制完成；小数秒必须完整保留。
			at := time.Date(2026, 11, 16, 3, 4, 5, 123456789, time.UTC)
			h.Steps[4].OccurredAt = &at
			h.Steps[4].ResponsibleUserID = &f.actorID
			h = save(h, "")
			if biz.ImportHandlingFlow(h) != biz.OrderFlowImportInProgress || !h.Steps[4].OccurredAt.Equal(at) || h.Steps[4].ResponsibleUserName == "" {
				t.Fatal("并行服务或真实完成时间错误")
			}
			if _, err = uc.TransitionClosure(ctx, f.organizationID, f.actorID, o.ID, h.OrderVersion, biz.OrderClosureClosed, ""); !errors.Is(err, biz.ErrOrderClosureBlocked) {
				t.Fatalf("未完成承接不应结案: %v", err)
			}
			if _, err = uc.SaveImportHandling(ctx, f.organizationID, f.actorID, o.ID, h.OrderVersion-1, cloneHandling(t, h), ""); !errors.Is(err, biz.ErrOrderStatusConflict) {
				t.Fatalf("过期服务版本未拒绝: %v", err)
			}
			next := cloneHandling(t, h)
			next.Steps[4].OccurredAt = nil
			if _, err = uc.SaveImportHandling(ctx, f.organizationID, f.actorID, o.ID, h.OrderVersion, next, ""); err == nil {
				t.Fatal("撤回未要求原因")
			}
			h = save(next, "更正派送完成时间")
			h.Steps[1].OccurredAt = &at
			h.Steps[4].OccurredAt = &at
			h = save(h, "")
			closed, err := uc.TransitionClosure(ctx, f.organizationID, f.actorID, o.ID, h.OrderVersion, biz.OrderClosureClosed, "")
			if err != nil || closed.FlowStatus != biz.OrderFlowImportCompleted {
				t.Fatalf("适用项目完成后结案: %v", err)
			}
			if _, err = uc.SaveImportHandling(ctx, f.organizationID, f.actorID, o.ID, closed.Version, cloneHandling(t, h), ""); !errors.Is(err, biz.ErrOrderClosureBlocked) {
				t.Fatal("完结后服务未要求显式重开")
			}
			reopened, err := uc.TransitionClosure(ctx, f.organizationID, f.actorID, o.ID, closed.Version, biz.OrderClosureOpen, "")
			if err != nil {
				t.Fatal(err)
			}
			h.OrderVersion = reopened.Version
			before := d.db.Order.GetX(ctx, o.ID)
			events := d.db.OrderImportHandlingEvent.Query().Where(eventent.OrderIDEQ(o.ID)).CountX(ctx)
			// 真实审计错误不能留下新服务、摘要、版本或事件。
			if _, err = d.sqlDB.ExecContext(ctx, `CREATE FUNCTION reject_ai_handling_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='order.import_handling.save' THEN RAISE EXCEPTION '测试审计失败'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_ai_handling_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION reject_ai_handling_audit()`); err != nil {
				t.Fatal(err)
			}
			if _, err = uc.SaveImportHandling(ctx, f.organizationID, f.actorID, o.ID, h.OrderVersion, cloneHandling(t, h), ""); err == nil {
				t.Fatal("审计失败未外传")
			}
			if current := d.db.Order.GetX(ctx, o.ID); current.Version != before.Version || current.FlowStatus != before.FlowStatus || d.db.OrderImportHandlingEvent.Query().Where(eventent.OrderIDEQ(o.ID)).CountX(ctx) != events {
				t.Fatal("服务审计失败未整体回滚")
			}
			if _, err = d.sqlDB.ExecContext(ctx, `DROP TRIGGER reject_ai_handling_audit ON audit_logs; DROP FUNCTION reject_ai_handling_audit()`); err != nil {
				t.Fatal(err)
			}
			lock := NewOrderLockRepo(d, nil)
			state, err := lock.GetOrderLockState(ctx, f.organizationID, o.ID, p)
			if err != nil {
				t.Fatal(err)
			}
			review := biz.OrderLockReviewInput{ExpectedFeeContentVersion: state.FeeContentVersion, FeeContentFingerprint: state.FeeContentFingerprint, AirDocumentFingerprint: state.AirDocumentFingerprint}
			result, err := lock.LockOrder(ctx, p, o.ID, state.OrderVersion, uuid.NewString(), review, nil)
			if err != nil {
				t.Fatal(err)
			}
			record := d.db.OrderLockRecord.GetX(ctx, result.LockRecord.ID)
			if record.BusinessType != "AI" || record.AirDocumentSnapshot["document_mode"] != mode || record.MasterBillID != nil || record.TransportExecutionID != nil {
				t.Fatal("AI审核未固化真实航空单证")
			}
			current := d.db.Order.GetX(ctx, o.ID)
			if _, err = uc.SaveImportHandling(ctx, f.organizationID, f.actorID, o.ID, current.Version, cloneHandling(t, h), ""); err == nil {
				t.Fatal("审核锁未阻断服务写入")
			}
			if _, err = lock.RequestOrderUnlock(ctx, p, o.ID, current.Version, uuid.NewString(), nil, nil); err != nil {
				t.Fatal(err)
			}
			current = d.db.Order.GetX(ctx, o.ID)
			typ := biz.OrderTerminationOther
			terminated, err := uc.TransitionTermination(ctx, f.organizationID, f.actorID, o.ID, current.Version, biz.OrderTerminationTerminating, &typ, "终止测试")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = uc.SaveImportHandling(ctx, f.organizationID, f.actorID, o.ID, terminated.Version, cloneHandling(t, h), ""); err == nil {
				t.Fatal("终止中服务写入被放行")
			}
			terminated, err = uc.TransitionTermination(ctx, f.organizationID, f.actorID, o.ID, terminated.Version, biz.OrderTerminationTerminated, &typ, "完成终止")
			if err != nil {
				t.Fatal(err)
			}
			old := d.db.AirMasterWaybillOrderLink.GetX(ctx, terminated.AirExport.LinkID)
			house := ""
			if terminated.AirExport.House != nil {
				house = terminated.AirExport.House.HouseNo
			}
			if mode == "HOUSE" {
				houseID := terminated.AirExport.House.ID
				d.db.AirHouseWaybill.UpdateOneID(houseID).SetOrganizationID(f.headquartersID).ExecX(ctx)
				if err = air.Link(ctx, f.organizationID, f.actorID, terminated.AirExport.Master.ID, o.ID, terminated.AirExport.Master.Version, terminated.Version, house, "拒绝外公司分单"); err == nil {
					t.Fatal("恢复接受外公司历史HAWB")
				}
				if current := d.db.Order.GetX(ctx, o.ID); current.Version != terminated.Version || current.TerminationStatus != orderent.TerminationStatusTERMINATED {
					t.Fatal("跨公司分单恢复留下部分写入")
				}
				d.db.AirHouseWaybill.UpdateOneID(houseID).SetOrganizationID(f.organizationID).ExecX(ctx)
			}
			if err = air.Link(ctx, f.organizationID, f.actorID, terminated.AirExport.Master.ID, o.ID, terminated.AirExport.Master.Version, terminated.Version, house, "恢复原主单"); err != nil {
				t.Fatal(err)
			}
			restored, err := uc.GetImportHandling(ctx, f.organizationID, o.ID)
			if err != nil || restored.Steps[1].OccurredAt == nil || restored.Steps[4].OccurredAt == nil {
				t.Fatalf("恢复丢失本票服务: %v", err)
			}
			link := d.db.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(o.ID), airlink.StatusEQ("ACTIVE")).OnlyX(ctx)
			if link.ID == old.ID || d.db.AirMasterWaybillOrderLink.GetX(ctx, old.ID).Status != "ENDED" {
				t.Fatal("恢复覆盖历史Link")
			}
			if _, err = uc.TransitionStatus(ctx, f.organizationID, f.actorID, o.ID, restored.OrderVersion, biz.OrderFlowBooked, ""); err == nil {
				t.Fatal("进口接受出口操作流程")
			}
		})
	}
}

func TestAirImportSharedETAMidnightRateAndRollbackPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	cleanupRateRefresh(t, d, f.organizationID)
	ctx := context.Background()
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	e := &rateRefreshEnv{t: t, data: d, orgID: f.organizationID, partnerID: f.partnerID, actorID: f.actorID, suffix: f.suffix}
	for _, rate := range []struct {
		day                 int
		receivable, payable string
	}{{19, "6.8", "6.7"}, {26, "7.0", "6.9"}} {
		createWeeklyRate(t, d, &f.organizationID, biz.ExchangeRateSettingSourceManual, "USD", time.Date(2026, 10, rate.day, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation()), rate.receivable, rate.payable)
	}
	const oldETA = "2026-10-25T15:59:00Z"
	const newETA = "2026-10-25T16:01:00Z"
	in := airInputForBusiness(f, biz.OrderBusinessAI, "78112345675", "AI-RATE-A")
	in.AirExportInput.Master.ETA = oldETA
	a := f.create(t, in)
	next := airInputForBusiness(f, biz.OrderBusinessAI, "78112345675", "AI-RATE-B")
	next.AirExportInput.MasterWaybillID = &a.AirExport.Master.ID
	next.AirExportInput.ExpectedMasterVersion = a.AirExport.Master.Version
	next.AirExportInput.Master = nil
	b := f.create(t, next)
	feeA := e.addFee(a.ID, "ai-midnight", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-10-25")
	feeB := e.addFee(b.ID, "ai-missing", orderfeeent.DirectionPAYABLE, "EUR", "50", "8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-10-25")
	run := func() error {
		detail, err := air.Get(ctx, f.organizationID, a.AirExport.Master.ID, biz.SelectorListOptions{Page: 1, PageSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		content := detail.Master.Content
		content.ETA = newETA
		preview, err := air.Preview(ctx, f.organizationID, detail.Master.ID, detail.Master.Version, &content)
		if err != nil {
			t.Fatal(err)
		}
		return air.Update(ctx, f.organizationID, f.actorID, detail.Master.ID, detail.Master.Version, &content, preview.InputFingerprint, "中国自然日跨周")
	}
	originalMaster := d.db.AirMasterWaybill.GetX(ctx, a.AirExport.Master.ID)
	originalHistory := d.db.AirMasterWaybillHistory.Query().CountX(ctx)
	if err := run(); !errors.Is(err, biz.ErrExchangeRateMissing) {
		t.Fatalf("另一成员缺目标周汇率须回滚: %v", err)
	}
	if master := d.db.AirMasterWaybill.GetX(ctx, originalMaster.ID); master.Version != originalMaster.Version || master.Eta != oldETA || d.db.AirMasterWaybillHistory.Query().CountX(ctx) != originalHistory {
		t.Fatal("缺汇率留下主单或历史写入")
	}
	for _, id := range []uuid.UUID{a.ID, b.ID} {
		if e.order(id).Eta != oldETA {
			t.Fatal("缺汇率留下成员日期写入")
		}
	}
	e.expectFee("缺率回滚", feeA.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-10-25", "680", 1)
	d.db.OrderFee.UpdateOneID(feeB.ID).SetExchangeRateSource(orderfeeent.ExchangeRateSourceMANUAL).ExecX(ctx)
	if err := run(); err != nil {
		t.Fatal(err)
	}
	e.expectFee("进口ETA跨中国午夜", feeA.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-10-26", "700", 2)
	e.expectFee("手工率保留", feeB.ID, "8.0", orderfeeent.ExchangeRateSourceMANUAL, "2026-10-25", "400", 1)
	for _, id := range []uuid.UUID{a.ID, b.ID} {
		o, err := uc.Get(ctx, f.organizationID, id)
		if err != nil || o.ETA != newETA || o.ETD != a.ETD {
			t.Fatalf("权威主单/详情投影错误: %v", err)
		}
		if day, err := biz.OrderRateDay(o); err != nil || day != "2026-10-26" {
			t.Fatalf("经营日未按中国ETA: %s %v", day, err)
		}
	}
	from := time.Date(2026, 10, 26, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())
	to := from.AddDate(0, 0, 1)
	listed, err := uc.List(ctx, []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessAI, OrganizationIDs: []uuid.UUID{f.organizationID}}}, biz.OrderListOptions{BusinessType: biz.OrderBusinessAI, Page: 1, PageSize: 20, ETARange: biz.OrderDateRange{From: &from, ToExclusive: &to}})
	if err != nil || listed.Total != 2 {
		t.Fatalf("跨午夜ETA列表筛选未覆盖完整成员: %+v %v", listed, err)
	}
	report, err := NewFinanceReportRepo(d).Profit(ctx, f.organizationID, biz.FinanceReportFilter{BusinessType: "AI", DateFrom: "2026-10-26", DateTo: "2026-10-26", Page: 1, PageSize: 20}, false)
	if err != nil || report.Total != 2 {
		t.Fatalf("AI经营报表未使用权威ETA: %+v %v", report, err)
	}
	for _, item := range report.Items {
		if item.BusinessDate == nil || *item.BusinessDate != "2026-10-26" {
			t.Fatal("AI报表业务日不一致")
		}
	}
	// 锁定任一成员后共享改期原子拒绝。
	before := d.db.AirMasterWaybill.GetX(ctx, originalMaster.ID)
	d.db.Order.UpdateOneID(b.ID).SetLockedAt(time.Now()).SetLockedBy(f.actorID).SetLockSource(orderent.LockSourceMANUAL).ExecX(ctx)
	if err := run(); !errors.Is(err, biz.ErrAirWaybillBlocked) {
		t.Fatalf("锁成员仍可改期: %v", err)
	}
	if current := d.db.AirMasterWaybill.GetX(ctx, before.ID); current.Version != before.Version || current.Eta != before.Eta {
		t.Fatal("锁冲突留下部分改期")
	}
}

func TestAirImportRejectsCrossIdentityAndIncompleteMembersPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	a := f.create(t, airInputForBusiness(f, biz.OrderBusinessAI, "78112345675", "AI-A"))
	exported := f.create(t, f.input("78112345675", "AE-A"))
	if a.AirExport.Master.ID == exported.AirExport.Master.ID {
		t.Fatal("进出口同号未隔离")
	}
	other := d.db.Organization.Create().SetCode(uuid.NewString()).SetName("其他公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	foreign := d.db.AirMasterWaybill.Create().SetBusinessType("AI").SetOrganizationID(other.ID).SetMasterNo("78112345675").SetAirlineID(f.airlineID).SetOriginAirportID(f.originID).SetDestinationAirportID(f.destinationID).SaveX(ctx)
	for _, id := range []uuid.UUID{exported.AirExport.Master.ID, foreign.ID} {
		in := airInputForBusiness(f, biz.OrderBusinessAI, "78112345675", "AI-FORBIDDEN")
		in.AirExportInput.MasterWaybillID = &id
		in.AirExportInput.ExpectedMasterVersion = 1
		in.AirExportInput.Master = nil
		if _, err := uc.Create(ctx, f.organizationID, f.actorID, in); err == nil {
			t.Fatal("创建接受另一方向或公司主单ID")
		}
	}
	if d.db.Order.Query().CountX(ctx) != 2 {
		t.Fatal("失败创建留下部分订单")
	}
	in := airInputForBusiness(f, biz.OrderBusinessAI, "78112345675", "AI-B")
	in.AirExportInput.MasterWaybillID = &a.AirExport.Master.ID
	in.AirExportInput.ExpectedMasterVersion = a.AirExport.Master.Version
	in.AirExportInput.Master = nil
	b := f.create(t, in)
	p := airPrincipalForBusiness(f, biz.OrderBusinessAI)
	p.IsBootstrapAdmin = false
	ctx = biz.WithPrincipal(ctx, p)
	weight := biz.NewAirWeightUsecase(NewAirWeightRepo(d))
	aePrincipal := airPrincipalForBusiness(f, biz.OrderBusinessAE)
	aePrincipal.IsBootstrapAdmin = false
	for _, scope := range []string{"ORDER", "MASTER"} {
		target := a.ID
		if scope == "MASTER" {
			target = a.AirExport.Master.ID
		}
		input := &biz.AirWeightInput{Scope: scope, TargetID: target, Source: "PLANNED"}
		if _, err := weight.Preview(biz.WithPrincipal(ctx, aePrincipal), f.organizationID, input); !errors.Is(err, biz.ErrPermissionDenied) {
			t.Fatalf("AI重量借AE权限 %s: %v", scope, err)
		}
	}
	d.db.Order.UpdateOneID(b.ID).SetBusinessType(orderent.BusinessTypeAE).SetTradeDirection(orderent.TradeDirectionExport).SetFlowStatus(orderent.FlowStatusDRAFT).ExecX(ctx)
	if _, err := air.Get(ctx, f.organizationID, a.AirExport.Master.ID, biz.SelectorListOptions{Page: 1, PageSize: 1}); err == nil {
		t.Fatal("读取共享主单静默遗漏错误方向成员")
	}
	for _, source := range []string{"PLANNED", "ACTUAL"} {
		if _, err := weight.Preview(ctx, f.organizationID, &biz.AirWeightInput{Scope: "MASTER", TargetID: a.AirExport.Master.ID, Source: source}); err == nil {
			t.Fatalf("%s重量忽略错误方向成员", source)
		}
	}
	// 审核必须重验完整成员；不能只锁目标票后接受错误方向共享资料。
	p.IsBootstrapAdmin = true
	state, err := NewOrderLockRepo(d, nil).GetOrderLockState(ctx, f.organizationID, a.ID, p)
	if err != nil {
		t.Fatal(err)
	}
	review := biz.OrderLockReviewInput{ExpectedFeeContentVersion: state.FeeContentVersion, FeeContentFingerprint: state.FeeContentFingerprint, AirDocumentFingerprint: state.AirDocumentFingerprint}
	if _, err := NewOrderLockRepo(d, nil).LockOrder(ctx, p, a.ID, state.OrderVersion, uuid.NewString(), review, nil); !errors.Is(err, biz.ErrAirWaybillConflict) {
		t.Fatalf("审核未重验完整方向集合: %v", err)
	}
	if d.db.Order.GetX(ctx, a.ID).LockedAt != nil || d.db.OrderLockRecord.Query().CountX(ctx) != 0 {
		t.Fatal("失败审核产生部分锁定事实")
	}
	// 真实AI权限仅来自AI清单，不需要AE。
	if p.HasPermission(access.OrderPermission(access.OrderBusinessAE, access.OrderRead)) {
		t.Fatal("测试身份意外具有AE权限")
	}
}
