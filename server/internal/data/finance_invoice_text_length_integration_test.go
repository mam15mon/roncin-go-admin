package data

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// 业务备注和反向原因的500字符契约不能在Ent落库时被UTF-8字节数收紧。
func TestFinanceInvoiceChineseTextLengthPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newRegistrationFixture(t, d)
	scope := []uuid.UUID{f.organizationID}
	bill := f.createConfirmedBill("long-note")
	longNote := strings.Repeat("登", 500)
	record := f.record(bill, "LONG-NOTE", "20", "1")
	record.Note = &longNote
	created, err := f.register("long-note", record)
	if err != nil || created[0].Note == nil || *created[0].Note != longNote {
		t.Fatalf("500中文登记备注须原样落库: %v", err)
	}
	invoice := created[0]
	bill = d.db.FinanceBill.GetX(ctx, bill.ID)
	overlong := longNote + "登"
	record = f.record(bill, "LONG-NOTE-REJECTED", "10", "0")
	record.Note = &overlong
	if _, err := f.register("overlong-note", record); !errors.Is(err, biz.ErrFinanceInvoiceInvalidArgument) {
		t.Fatalf("501中文登记备注须在领域层拒绝: %v", err)
	}
	if d.db.FinanceBill.GetX(ctx, bill.ID).Version != bill.Version {
		t.Fatal("超长备注不得推进账单版本")
	}
	longRevisionNote := strings.Repeat("更", 500)
	record = f.record(bill, "LONG-NOTE", "30", "2")
	record.Note = &longRevisionNote
	revised, err := f.uc.ReviseRegistrations(ctx, scope, f.actorID, biz.ReviseFinanceInvoiceRegistrationsInput{IdempotencyKey: "long-note-revise", Reason: strings.Repeat("修", 500), BillVersions: []biz.FinanceInvoiceBillVersion{{BillID: bill.ID, ExpectedVersion: bill.Version}}, Entries: []biz.FinanceInvoiceRevisionEntry{{ID: invoice.ID, ExpectedVersion: invoice.Version, Registration: &record}}})
	if err != nil || revised[0].Note == nil || *revised[0].Note != longRevisionNote {
		t.Fatalf("500中文修订备注及原因须原样落库: %v", err)
	}
	invoice = revised[0]
	latestBill := d.db.FinanceBill.GetX(ctx, bill.ID)
	invalidRevision := f.record(latestBill, "LONG-NOTE", "30", "2")
	invalidRevision.Note = &overlong
	if _, err := f.uc.ReviseRegistrations(ctx, scope, f.actorID, biz.ReviseFinanceInvoiceRegistrationsInput{IdempotencyKey: "overlong-revision-note", Reason: "备注超长拒绝", BillVersions: []biz.FinanceInvoiceBillVersion{{BillID: bill.ID, ExpectedVersion: latestBill.Version}}, Entries: []biz.FinanceInvoiceRevisionEntry{{ID: invoice.ID, ExpectedVersion: invoice.Version, Registration: &invalidRevision}}}); !errors.Is(err, biz.ErrFinanceInvoiceInvalidArgument) {
		t.Fatalf("501中文修订备注须在领域层拒绝: %v", err)
	}
	if d.db.FinanceInvoice.GetX(ctx, invoice.ID).Version != invoice.Version || d.db.FinanceBill.GetX(ctx, bill.ID).Version != latestBill.Version {
		t.Fatal("超长修订备注不得改变票据或账单版本")
	}
	longCancelReason := strings.Repeat("撤", 500)
	if _, err := f.uc.Cancel(ctx, f.organizationID, f.actorID, invoice.ID, invoice.Version, longCancelReason+"撤"); !errors.Is(err, biz.ErrFinanceInvoiceInvalidArgument) {
		t.Fatalf("501中文作废原因须在领域层拒绝: %v", err)
	}
	if d.db.FinanceInvoice.GetX(ctx, invoice.ID).Version != invoice.Version {
		t.Fatal("超长作废原因不得推进版本或释放关联")
	}
	cancelled, err := f.uc.Cancel(ctx, f.organizationID, f.actorID, invoice.ID, invoice.Version, longCancelReason)
	if err != nil || cancelled.CancellationReason == nil || *cancelled.CancellationReason != longCancelReason || cancelled.Status != biz.FinanceInvoiceCancelled {
		t.Fatalf("500中文作废原因须原样保存并完成反向: %v", err)
	}

	redBill := f.createConfirmedBill("long-red-reason")
	redCreated, err := f.register("long-red", f.record(redBill, "LONG-RED", "40", "3"))
	if err != nil {
		t.Fatal(err)
	}
	redInvoice := redCreated[0]
	longRedReason := strings.Repeat("红", 500)
	if _, err := f.uc.RedFlush(ctx, f.organizationID, f.actorID, redInvoice.ID, redInvoice.Version, "LONG-RED-NO", financeInvoiceIntegrationDate, longRedReason+"红"); !errors.Is(err, biz.ErrFinanceInvoiceInvalidArgument) {
		t.Fatalf("501中文红冲原因须在领域层拒绝: %v", err)
	}
	if d.db.FinanceInvoice.GetX(ctx, redInvoice.ID).Version != redInvoice.Version {
		t.Fatal("超长红冲原因不得推进版本或释放关联")
	}
	redFlushed, err := f.uc.RedFlush(ctx, f.organizationID, f.actorID, redInvoice.ID, redInvoice.Version, "LONG-RED-NO", financeInvoiceIntegrationDate, longRedReason)
	if err != nil || redFlushed.RedFlushReason == nil || *redFlushed.RedFlushReason != longRedReason || redFlushed.Status != biz.FinanceInvoiceRedFlushed {
		t.Fatalf("500中文红冲原因须原样保存并完成反向: %v", err)
	}
}
