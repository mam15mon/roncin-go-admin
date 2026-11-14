package data

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceexemption"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
)

func TestFinanceInvoiceExemptionPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	t.Run("数据库拒绝缺原因的撤销且保留原决定", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, data)
		bill := f.createConfirmedBill("invalid-revoke")
		newDecision := func() *ent.FinanceInvoiceExemptionCreate {
			return data.db.FinanceInvoiceExemption.Create().SetOrganizationID(f.organizationID).SetBillID(bill.ID).SetMarkedBy(f.actorID).SetReason("无需发票")
		}
		if _, err := newDecision().SetRevokedAt(time.Now()).SetRevokedBy(f.actorID).Save(ctx); !ent.IsConstraintError(err) {
			t.Fatalf("撤销原因NULL必须被CHECK拒绝: %v", err)
		}
		decision, err := newDecision().Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := decision.Update().SetRevokedAt(time.Now()).SetRevokedBy(f.actorID).Exec(ctx); !ent.IsConstraintError(err) {
			t.Fatalf("更新撤销同样不得遗漏原因: %v", err)
		}
		current := data.db.FinanceInvoiceExemption.GetX(ctx, decision.ID)
		if current.RevokedAt != nil || current.RevokedBy != nil {
			t.Fatal("CHECK拒绝后原有效决定应完整保留")
		}
		if err := current.Update().SetRevokedAt(time.Now()).SetRevokedBy(f.actorID).SetRevocationReason("补齐实际发票").Exec(ctx); err != nil {
			t.Fatalf("完整撤销事实应允许: %v", err)
		}
		if _, err := newDecision().Save(ctx); err != nil {
			t.Fatalf("有效决定撤销后允许形成新决定: %v", err)
		}
	})

	t.Run("决定撤销留痕且结清免票才完成", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, data)
		bill := f.createConfirmedBill("exempt")
		scope := []uuid.UUID{f.organizationID}
		invoiceRepo := NewFinanceInvoiceRepo(data)
		uc := biz.NewFinanceInvoiceUsecase(invoiceRepo, nil, nil)
		billRepo := NewFinanceBillRepo(data)
		if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, 1, " 境外服务无需发票 ", false); err != nil {
			t.Fatal(err)
		}
		get := func() *biz.FinanceBill {
			t.Helper()
			item, err := billRepo.Get(ctx, scope, bill.ID)
			if err != nil {
				t.Fatal(err)
			}
			return item
		}
		got := get()
		if got.Version != 2 || got.InvoiceProgress != biz.FinanceBillInvoiceNotRequired || got.Completed || !got.FeesLocked() || len(got.InvoiceExemptions) != 1 || got.InvoiceExemptions[0].Reason != "境外服务无需发票" || got.InvoiceExemptions[0].MarkedByName == "" {
			t.Fatalf("免票投影或审计历史不完整: %+v", got)
		}
		lines := data.db.FinanceBillLine.Query().Where(financebillline.BillIDEQ(bill.ID)).AllX(ctx)
		protected, err := feeHasProtectedFinanceFact(ctx, data.db, lines[0].OrderFeeID)
		if err != nil || !protected {
			t.Fatalf("免票费用未受汇率保护: %v/%v", protected, err)
		}
		if err := billCancellationDependencies(ctx, data.db, bill); err == nil {
			t.Fatal("免票决定未撤销时不得取消账单")
		}
		if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, 2, "重复免票", false); !errors.Is(err, biz.ErrFinanceInvoiceExemptionConflict) {
			t.Fatalf("不能覆盖现有有效决定: %v", err)
		}
		profile, err := data.db.PartnerInvoiceProfile.Create().SetOrganizationID(f.organizationID).SetPartnerID(f.partnerID).SetInvoiceTitle("免票互斥测试抬头").SetTaxpayerIdentificationNo("EXEMPT-TEST").Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, err = invoiceRepo.Create(ctx, &biz.FinanceInvoice{ID: uuid.New(), OrganizationID: f.organizationID, InvoiceProfileID: &profile.ID, SettlementPartyID: f.partnerID, InvoiceTitle: profile.InvoiceTitle, TaxpayerIdentificationNo: profile.TaxpayerIdentificationNo, Links: []*biz.FinanceInvoiceBill{{BillID: bill.ID}}}, &biz.AuditEvent{Action: "test.exempt.invoice", Result: "success"})
		if !errors.Is(err, biz.ErrFinanceInvoiceExemptionConflict) {
			t.Fatalf("有免票时服务端必须拒绝开票: %v", err)
		}
		if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, 1, "旧版本撤销", true); !errors.Is(err, biz.ErrFinanceBillVersionConflict) {
			t.Fatalf("陈旧版本应冲突: %v", err)
		}

		if _, err := data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(numberrule.DocumentTypeWriteOff).SetPrefix("EX").SetDateFormat(numberrule.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberrule.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
		cash := correctionFinanceCash(t, f, "exempt-cash-"+f.suffix, "100", "RECEIVABLE")
		verification, err := correctionFinanceVerification(t, f, cash, bill, "100")
		if err != nil {
			t.Fatal(err)
		}
		got = get()
		if !got.Completed || got.SettlementProgress != biz.FinanceBillSettlementSettled {
			t.Fatal("已结清免票账单应完成")
		}
		completed := true
		listed, err := billRepo.List(ctx, scope, biz.FinanceBillFilter{Page: 1, PageSize: 200, Completed: &completed, InvoiceProgress: biz.FinanceBillInvoiceNotRequired})
		if err != nil || listed.Total != 1 || listed.Items[0].ID != bill.ID {
			t.Fatalf("免票完成筛选与展示不同源: %+v/%v", listed, err)
		}

		if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, got.Version, "改为需要正式发票", true); err != nil {
			t.Fatal(err)
		}
		got = get()
		if got.Completed || got.InvoiceProgress != biz.FinanceBillInvoiceNone || !got.FeesLocked() || got.InvoiceExemptions[0].RevokedAt == nil || got.InvoiceExemptions[0].RevokedByName == "" || got.InvoiceExemptions[0].Version != 2 {
			t.Fatalf("撤销必须保留历史且仍有核销就保持锁定: %+v", got)
		}
		if _, err := NewVerificationRepo(data).Reverse(ctx, f.organizationID, verification.ID, f.actorID, verification.Version, "撤销结算", &biz.AuditEvent{Action: "test.exempt.reverse", Result: "success"}); err != nil {
			t.Fatal(err)
		}
		got = get()
		if got.FeesLocked() {
			t.Fatal("全部有效事实解除后应恢复费用增删")
		}
		protected, err = feeHasProtectedFinanceFact(ctx, data.db, lines[0].OrderFeeID)
		if err != nil || protected {
			t.Fatalf("撤销后的历史决定不应继续保护汇率: %v/%v", protected, err)
		}
		if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, got.Version, "再次确认无需发票", false); err != nil {
			t.Fatal(err)
		}
		if got := get(); len(got.InvoiceExemptions) != 2 || got.InvoiceExemptions[0].RevokedAt == nil || got.InvoiceExemptions[1].RevokedAt != nil {
			t.Fatal("再次决定应新建一条历史而非覆盖原决定")
		}
	})

	t.Run("已有发票不能免票且跨公司不可写", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, data)
		bill := f.createConfirmedBill("invoiced")
		f.createInvoiceWithStatus("invoiced", financeinvoice.StatusDRAFT, bill)
		uc := biz.NewFinanceInvoiceUsecase(NewFinanceInvoiceRepo(data), nil, nil)
		if err := uc.SetBillInvoiceExemption(ctx, []uuid.UUID{f.organizationID}, f.actorID, bill.ID, 1, "免票", false); !errors.Is(err, biz.ErrFinanceInvoiceExemptionConflict) {
			t.Fatalf("已有部分或全部有效发票应拒绝免票: %v", err)
		}
		if err := uc.SetBillInvoiceExemption(ctx, []uuid.UUID{uuid.New()}, f.actorID, bill.ID, 1, "跨公司", false); !errors.Is(err, biz.ErrFinanceBillNotFound) {
			t.Fatalf("范围外账单应不可见: %v", err)
		}
		if count := data.db.FinanceInvoiceExemption.Query().Where(financeinvoiceexemption.BillIDEQ(bill.ID)).CountX(ctx); count != 0 {
			t.Fatal("失败请求留下免票事实")
		}
	})

	t.Run("空原因拒绝且审计失败整笔回滚", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, data)
		bill := f.createConfirmedBill("rollback")
		repo := NewFinanceInvoiceRepo(data)
		uc := biz.NewFinanceInvoiceUsecase(repo, nil, nil)
		if err := uc.SetBillInvoiceExemption(ctx, []uuid.UUID{f.organizationID}, f.actorID, bill.ID, 1, " \n ", false); !errors.Is(err, biz.ErrFinanceInvoiceInvalidArgument) {
			t.Fatalf("空原因应拒绝: %v", err)
		}
		if err := repo.SetBillInvoiceExemption(ctx, []uuid.UUID{f.organizationID}, f.actorID, bill.ID, 1, "回滚用例", false, &biz.AuditEvent{Result: "success"}); err == nil {
			t.Fatal("缺失审计动作应失败")
		}
		if count := data.db.FinanceInvoiceExemption.Query().Where(financeinvoiceexemption.BillIDEQ(bill.ID)).CountX(ctx); count != 0 {
			t.Fatal("审计失败留下标记")
		}
		if version := data.db.FinanceBill.Query().Where(financebill.IDEQ(bill.ID)).OnlyX(ctx).Version; version != 1 {
			t.Fatalf("回滚后版本变化: %d", version)
		}
	})

	t.Run("同版本并发只能形成一个有效决定", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, data)
		bill := f.createConfirmedBill("race")
		uc := biz.NewFinanceInvoiceUsecase(NewFinanceInvoiceRepo(data), nil, nil)
		start := make(chan struct{})
		results := make(chan error, 2)
		var workers sync.WaitGroup
		for i := 0; i < 2; i++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				results <- uc.SetBillInvoiceExemption(ctx, []uuid.UUID{f.organizationID}, f.actorID, bill.ID, 1, "并发免票", false)
			}()
		}
		close(start)
		workers.Wait()
		close(results)
		success, conflicts := 0, 0
		for err := range results {
			if err == nil {
				success++
			} else if errors.Is(err, biz.ErrFinanceBillVersionConflict) {
				conflicts++
			} else {
				t.Fatalf("意外并发错误: %v", err)
			}
		}
		if success != 1 || conflicts != 1 || data.db.FinanceInvoiceExemption.Query().Where(financeinvoiceexemption.BillIDEQ(bill.ID), financeinvoiceexemption.RevokedAtIsNil()).CountX(ctx) != 1 {
			t.Fatalf("并发结果错误 success=%d conflict=%d", success, conflicts)
		}
	})
}
