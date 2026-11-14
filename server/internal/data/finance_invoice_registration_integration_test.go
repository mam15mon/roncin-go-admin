package data

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	invoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	commandent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicecommand"
	allocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicefeeallocation"
	number "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/shopspring/decimal"
)

type registrationFixture struct {
	*financeInvoicePostgresFixture
	uc      *biz.FinanceInvoiceUsecase
	profile *ent.PartnerInvoiceProfile
}

func newRegistrationFixture(t *testing.T, d *Data) *registrationFixture {
	t.Helper()
	f := newFinanceInvoicePostgresFixture(t, d)
	ctx := context.Background()
	_, err := d.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(number.DocumentTypeInvoice).SetPrefix("RI").SetDateFormat(number.DateFormatNone).SetSequenceLength(6).SetResetPolicy(number.ResetPolicyNever).SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := d.db.PartnerInvoiceProfile.Create().SetOrganizationID(f.organizationID).SetPartnerID(f.partnerID).SetInvoiceTitle("手税测试抬头").SetTaxpayerIdentificationNo("TAX-MANUAL").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return &registrationFixture{f, biz.NewFinanceInvoiceUsecase(NewFinanceInvoiceRepo(d), biz.NewOrderConfigUsecase(NewOrderConfigRepo(d)), biz.NewExchangeRateUsecase(NewExchangeRateRepo(d), nil), d), profile}
}
func (f *registrationFixture) record(b *ent.FinanceBill, no, amount, tax string) biz.FinanceInvoiceRegistrationInput {
	f.t.Helper()
	lines := f.data.db.FinanceBillLine.Query().Where(billline.BillIDEQ(b.ID), billline.ActiveEQ(true)).AllX(context.Background())
	if len(lines) != 1 {
		f.t.Fatal("夹具必须一条来源费用")
	}
	return biz.FinanceInvoiceRegistrationInput{InvoiceProfileID: f.profile.ID, InvoiceType: biz.FinanceInvoiceNormal, TaxInvoiceNo: no, InvoiceDate: financeInvoiceIntegrationDate, TotalAmount: decimal.RequireFromString(amount), Allocations: []biz.FinanceInvoiceAllocationInput{{BillID: b.ID, ExpectedVersion: b.Version, Amount: decimal.RequireFromString(amount), Fees: []biz.FinanceInvoiceFeeInput{{OrderFeeID: lines[0].OrderFeeID, Amount: decimal.RequireFromString(amount), TaxAmount: decimal.RequireFromString(tax)}}}}}
}
func (f *registrationFixture) register(key string, records ...biz.FinanceInvoiceRegistrationInput) ([]*biz.FinanceInvoice, error) {
	return f.uc.RegisterScoped(context.Background(), []uuid.UUID{f.organizationID}, f.actorID, biz.RegisterFinanceInvoicesInput{IdempotencyKey: key, Records: records})
}

func TestFinanceInvoiceRegistrationPartialManualTaxPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	f := newRegistrationFixture(t, d)
	ctx := context.Background()
	b := f.createConfirmedBill("partial")
	records := []biz.FinanceInvoiceRegistrationInput{f.record(b, "P-1", "30", "3.14159265"), f.record(b, "P-2", "20", "0")}
	invoices, err := f.register("partial", records...)
	if err != nil {
		t.Fatal(err)
	}
	if len(invoices) != 2 || invoices[0].Status != biz.FinanceInvoiceIssued || !invoices[0].TaxAmount.Equal(decimal.RequireFromString("3.14159265")) || !invoices[0].NetAmount.Equal(decimal.RequireFromString("26.85840735")) {
		t.Fatalf("须直登记并保留手填税额: %+v", invoices)
	}
	got, err := NewFinanceBillRepo(d).Get(ctx, []uuid.UUID{f.organizationID}, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 2 || got.InvoiceProgress != biz.FinanceBillInvoicePartial || !got.InvoiceableAmount.Equal(decimal.NewFromInt(50)) || !got.Lines[0].InvoicedAmount.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("部分投影/版本不正确: %+v", got)
	}
	source := d.db.OrderFee.GetX(ctx, got.Lines[0].OrderFeeID)
	if source.TaxAmount != "0.00000000" || source.NetAmount != "100.00000000" {
		t.Fatalf("登记不得改源费用税基: %+v", source)
	}
	candidates, err := f.uc.ListCreationBills(ctx, f.organizationID, biz.FinanceInvoiceCreationBillFilter{Page: 1, PageSize: 1, BillID: &b.ID})
	if err != nil || len(candidates.Items) != 1 || !candidates.Items[0].InvoiceableAmount.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("部分登记仍须可继续: %+v %v", candidates, err)
	}
	replay, err := f.register("partial", records...)
	if err != nil || replay[0].ID != invoices[0].ID || replay[1].ID != invoices[1].ID {
		t.Fatalf("同键基线旧也应重放原结果: %+v %v", replay, err)
	}
	changed := f.record(b, "P-1", "30", "2")
	if _, err = f.register("partial", changed, records[1]); !errors.Is(err, biz.ErrFinanceInvoiceIdempotencyConflict) {
		t.Fatalf("同键不同税额必须冲突: %v", err)
	}
	if d.db.FinanceInvoice.Query().Where(invoiceent.OrganizationIDEQ(f.organizationID)).CountX(ctx) != 2 {
		t.Fatal("重放不能重复计数")
	}
}

func TestFinanceInvoiceRegistrationAtomicAndConcurrentPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	t.Run("第二条超额整次回滚", func(t *testing.T) {
		f := newRegistrationFixture(t, d)
		b := f.createConfirmedBill("rollback")
		_, err := f.register("rollback", f.record(b, "R-1", "60", "1"), f.record(b, "R-2", "50", "1"))
		if !errors.Is(err, biz.ErrFinanceInvoiceAllocationExceeded) {
			t.Fatalf("应拒超额: %v", err)
		}
		if d.db.FinanceInvoice.Query().Where(invoiceent.OrganizationIDEQ(f.organizationID)).CountX(ctx) != 0 || d.db.FinanceInvoiceCommand.Query().Where(commandent.OrganizationIDEQ(f.organizationID)).CountX(ctx) != 0 || d.db.FinanceBill.GetX(ctx, b.ID).Version != 1 {
			t.Fatal("任一发票失败必须无命令/发票/版本残留")
		}
	})
	t.Run("并发不同命令不能超开", func(t *testing.T) {
		f := newRegistrationFixture(t, d)
		b := f.createConfirmedBill("concurrent")
		start := make(chan struct{})
		results := make(chan error, 2)
		var wg sync.WaitGroup
		for _, no := range []string{"C-1", "C-2"} {
			record := f.record(b, no, "80", "0")
			wg.Add(1)
			go func(no string, r biz.FinanceInvoiceRegistrationInput) {
				defer wg.Done()
				<-start
				_, err := f.register(no, r)
				results <- err
			}(no, record)
		}
		close(start)
		wg.Wait()
		close(results)
		success, conflict := 0, 0
		for err := range results {
			if err == nil {
				success++
			} else if errors.Is(err, biz.ErrFinanceBillVersionConflict) {
				conflict++
			} else {
				t.Fatalf("并发非预期: %v", err)
			}
		}
		if success != 1 || conflict != 1 || d.db.FinanceInvoice.Query().Where(invoiceent.OrganizationIDEQ(f.organizationID)).CountX(ctx) != 1 {
			t.Fatal("只能一条成功，另一条版本冲突")
		}
	})
}

func TestFinanceInvoiceRegistrationRevisionHistoryPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	f := newRegistrationFixture(t, d)
	ctx := context.Background()
	b := f.createConfirmedBill("revision")
	created, err := f.register("original", f.record(b, "OLD-1", "40", "4"))
	if err != nil {
		t.Fatal(err)
	}
	original := created[0]
	b = d.db.FinanceBill.GetX(ctx, b.ID)
	replacement := f.record(b, "NEW-1", "70", "12.00000001")
	input := biz.ReviseFinanceInvoiceRegistrationsInput{IdempotencyKey: "revision", Reason: "修正实际票据金额", BillVersions: []biz.FinanceInvoiceBillVersion{{BillID: b.ID, ExpectedVersion: b.Version}}, Entries: []biz.FinanceInvoiceRevisionEntry{{ID: original.ID, ExpectedVersion: original.Version, Registration: &replacement}}}
	revised, err := f.uc.ReviseRegistrations(ctx, []uuid.UUID{f.organizationID}, f.actorID, input)
	if err != nil {
		t.Fatal(err)
	}
	current := revised[0]
	if current.ID != original.ID || current.RecordNo != original.RecordNo || current.Version != 2 || current.RevisionNumber != 2 || len(current.Revisions) != 1 || current.Revisions[0].ActorName == "" || current.Revisions[0].Reason != input.Reason || !current.TotalAmount.Equal(decimal.NewFromInt(70)) {
		t.Fatalf("修订身份/历史不完整: %+v", current)
	}
	var before, after map[string]any
	if err = json.Unmarshal(current.Revisions[0].Before, &before); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(current.Revisions[0].After, &after); err != nil {
		t.Fatal(err)
	}
	if before["totalAmount"] != "40.00000000" || after["totalAmount"] != "70.00000000" || before["IdempotencyKey"] != nil {
		t.Fatalf("业务前后快照不正确: %s %s", current.Revisions[0].Before, current.Revisions[0].After)
	}
	if d.db.FinanceInvoiceFeeAllocation.Query().Where(allocationent.InvoiceIDEQ(original.ID)).CountX(ctx) != 2 {
		t.Fatal("旧逐费事实须保留并追加新事实")
	}
	got, err := NewFinanceBillRepo(d).Get(ctx, []uuid.UUID{f.organizationID}, b.ID)
	if err != nil || !got.InvoiceableAmount.Equal(decimal.NewFromInt(30)) || got.Version != 3 {
		t.Fatalf("修订余额按新分配而非重复扣减: %+v %v", got, err)
	}
	replay, err := f.uc.ReviseRegistrations(ctx, []uuid.UUID{f.organizationID}, f.actorID, input)
	if err != nil || replay[0].Version != 2 {
		t.Fatalf("修订同键必须重放: %+v %v", replay, err)
	}
	remove := biz.ReviseFinanceInvoiceRegistrationsInput{IdempotencyKey: "remove", Reason: "撤除错误记录", BillVersions: []biz.FinanceInvoiceBillVersion{{BillID: b.ID, ExpectedVersion: got.Version}}, Entries: []biz.FinanceInvoiceRevisionEntry{{ID: current.ID, ExpectedVersion: current.Version, Remove: true}}}
	removed, err := f.uc.ReviseRegistrations(ctx, []uuid.UUID{f.organizationID}, f.actorID, remove)
	if err != nil || removed[0].Status != biz.FinanceInvoiceCancelled {
		t.Fatalf("移除须留原票: %+v %v", removed, err)
	}
	got, err = NewFinanceBillRepo(d).Get(ctx, []uuid.UUID{f.organizationID}, b.ID)
	if err != nil || got.FeesLocked() || !got.InvoiceableAmount.Equal(decimal.NewFromInt(100)) || got.Version != 4 {
		t.Fatalf("末关联移除才解除保护并释放一次: %+v %v", got, err)
	}
	if _, err = f.uc.Cancel(ctx, f.organizationID, f.actorID, current.ID, removed[0].Version, "重复移除"); !errors.Is(err, biz.ErrFinanceInvoiceInvalidTransition) {
		t.Fatalf("不可重复释放余额: %v", err)
	}
	if d.db.FinanceBill.GetX(ctx, b.ID).Version != 4 {
		t.Fatal("重复移除不推进账单版本")
	}
}

func TestFinanceInvoiceRegistrationCrossBillAndSettlementPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	f := newRegistrationFixture(t, d)
	ctx := context.Background()
	_, err := d.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(number.DocumentTypeWriteOff).SetPrefix("WO").SetDateFormat(number.DateFormatNone).SetSequenceLength(6).SetResetPolicy(number.ResetPolicyNever).SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, b := f.createConfirmedBill("cross-a"), f.createConfirmedBill("cross-b")
	r := f.record(a, "CROSS-1", "30", "1")
	second := f.record(b, "ignored", "20", "2")
	r.TotalAmount = decimal.NewFromInt(50)
	r.Allocations = append(r.Allocations, second.Allocations[0])
	created, err := f.register("cross", r)
	if err != nil {
		t.Fatal(err)
	}
	inv := created[0]
	impact, err := f.uc.GetRevisionImpact(ctx, []uuid.UUID{f.organizationID}, inv.ID)
	if err != nil || len(impact.Bills) != 2 || !impact.CanRevise {
		t.Fatalf("跨账单必须返回全部影响: %+v %v", impact, err)
	}
	missing := biz.ReviseFinanceInvoiceRegistrationsInput{IdempotencyKey: "missing-cross", Reason: "遗漏另一张账单", BillVersions: []biz.FinanceInvoiceBillVersion{{BillID: a.ID, ExpectedVersion: 2}}, Entries: []biz.FinanceInvoiceRevisionEntry{{ID: inv.ID, ExpectedVersion: 1, Remove: true}}}
	if _, err = f.uc.ReviseRegistrations(ctx, []uuid.UUID{f.organizationID}, f.actorID, missing); !errors.Is(err, biz.ErrFinanceInvoiceBillInvalid) {
		t.Fatalf("跨账单漏基线必须拒绝: %v", err)
	}
	flow := correctionFinanceCash(t, f.financeInvoicePostgresFixture, "到账", "10", cash.DirectionRECEIVABLE)
	if _, err = correctionFinanceVerification(t, f.financeInvoicePostgresFixture, flow, a, "10"); err != nil {
		t.Fatal(err)
	}
	impact, err = f.uc.GetRevisionImpact(ctx, []uuid.UUID{f.organizationID}, inv.ID)
	if err != nil || impact.CanRevise || len(impact.Dependencies) != 1 || impact.Dependencies[0].ID == uuid.Nil {
		t.Fatalf("应展示已结算影响并禁止直改: %+v %v", impact, err)
	}
	versions := []biz.FinanceInvoiceBillVersion{}
	for _, bill := range impact.Bills {
		versions = append(versions, biz.FinanceInvoiceBillVersion{BillID: bill.ID, ExpectedVersion: bill.Version})
	}
	missing.BillVersions = versions
	missing.IdempotencyKey = "blocked-settlement"
	if _, err = f.uc.ReviseRegistrations(ctx, []uuid.UUID{f.organizationID}, f.actorID, missing); !errors.Is(err, biz.ErrFinanceInvoiceSettlementDependency) {
		t.Fatalf("已有结算必须受控处理: %v", err)
	}
	for _, operation := range []func() error{func() error { _, err := f.uc.Cancel(ctx, f.organizationID, f.actorID, inv.ID, 1, "作废"); return err }, func() error {
		_, err := f.uc.RedFlush(ctx, f.organizationID, f.actorID, inv.ID, 1, "RED-1", financeInvoiceIntegrationDate, "红冲")
		return err
	}} {
		if err := operation(); !errors.Is(err, biz.ErrFinanceInvoiceSettlementDependency) {
			t.Fatalf("作废/红冲不可绕过结算依赖: %v", err)
		}
	}
	if d.db.FinanceInvoice.GetX(ctx, inv.ID).Status != invoiceent.StatusISSUED {
		t.Fatal("阻断后原发票事实不变")
	}
}
