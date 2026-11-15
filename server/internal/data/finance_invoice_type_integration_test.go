package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

func TestFinanceInvoiceTypeRegistrationAndRevisionPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newRegistrationFixture(t, d)
	types := []biz.FinanceInvoiceType{
		biz.FinanceInvoiceDigitalNormal, biz.FinanceInvoiceDigitalSpecial,
		biz.FinanceInvoicePaperNormal, biz.FinanceInvoicePaperSpecial,
		biz.FinanceInvoiceElectronicNormal, biz.FinanceInvoiceElectronicSpecial,
	}
	for index, kind := range types {
		t.Run(string(kind), func(t *testing.T) {
			bill := f.createConfirmedBill(fmt.Sprintf("type-%d", index))
			record := f.record(bill, "TYPE-"+string(kind), "40", "4")
			record.InvoiceType = kind
			if kind.Special() {
				f.profile = d.db.PartnerInvoiceProfile.UpdateOneID(f.profile.ID).SetRegisteredAddress("").SetRegisteredPhone("").SetBankName("").SetBankAccount("").SaveX(ctx)
				if _, err := f.register("incomplete-"+string(kind), record); !errors.Is(err, biz.ErrFinanceInvoiceProfileRequired) {
					t.Fatalf("三类专票均须拒绝不完整抬头: %v", err)
				}
				if d.db.FinanceBill.GetX(ctx, bill.ID).Version != bill.Version {
					t.Fatal("拒绝专票不能消耗账单版本")
				}
			}
			f.profile = d.db.PartnerInvoiceProfile.UpdateOneID(f.profile.ID).SetRegisteredAddress("测试地址").SetRegisteredPhone("01012345678").SetBankName("测试银行").SetBankAccount("12345678").SaveX(ctx)
			created, err := f.register("type-"+string(kind), record)
			if err != nil || len(created) != 1 || created[0].InvoiceType != kind {
				t.Fatalf("实际登记须保留票种: %+v %v", created, err)
			}
			original := created[0]
			bill = d.db.FinanceBill.GetX(ctx, bill.ID)
			replacement := f.record(bill, "REVISED-"+string(kind), "50", "5")
			replacement.InvoiceType = kind
			revised, err := f.uc.ReviseRegistrations(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.ReviseFinanceInvoiceRegistrationsInput{
				IdempotencyKey: "revision-type-" + string(kind), Reason: "核对实际票种及票面金额",
				BillVersions: []biz.FinanceInvoiceBillVersion{{BillID: bill.ID, ExpectedVersion: bill.Version}},
				Entries:      []biz.FinanceInvoiceRevisionEntry{{ID: original.ID, ExpectedVersion: original.Version, Registration: &replacement}},
			})
			if err != nil || len(revised) != 1 || revised[0].InvoiceType != kind || len(revised[0].Revisions) != 1 {
				t.Fatalf("实际修订须保留票种和历史: %+v %v", revised, err)
			}
		})
	}
}

func TestFinanceInvoiceTypeMigrationPostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, "20261115120000")
	defer cleanup()
	ctx := context.Background()
	expectedInvoices := map[uuid.UUID]string{}
	expectedProfiles := map[uuid.UUID]string{}
	for _, legacy := range []string{"NORMAL", "SPECIAL"} {
		f := newFinanceInvoicePostgresFixture(t, d)
		invoiceID, _, _ := oldRegistrationInvoice(t, d, f, "ISSUED")
		if _, err := d.sqlDB.ExecContext(ctx, `UPDATE finance_invoices SET invoice_type=$1 WHERE id=$2`, legacy, invoiceID); err != nil {
			t.Fatal(err)
		}
		profileID := uuid.New()
		expectedInvoices[invoiceID] = "DIGITAL_" + legacy
		expectedProfiles[profileID] = "DIGITAL_" + legacy
		if _, err := d.sqlDB.ExecContext(ctx, `INSERT INTO partner_invoice_profiles(id,created_at,updated_at,organization_id,partner_id,invoice_title,taxpayer_identification_no,default_invoice_type,is_default,enabled,version) VALUES($1,NOW(),NOW(),$2,$3,$4,$4,$4,false,true,1)`, profileID, f.organizationID, f.partnerID, legacy); err != nil {
			t.Fatal(err)
		}
		before := json.RawMessage(`{"invoiceType":"` + legacy + `","totalAmount":"40.00000000"}`)
		after := json.RawMessage(`{"invoiceType":"` + legacy + `","totalAmount":"50.00000000"}`)
		d.db.FinanceInvoiceRevision.Create().SetInvoiceID(invoiceID).SetActorID(f.actorID).SetCommandID(uuid.New()).SetRevisionNumber(2).SetReason("迁移前修订").SetBeforeSnapshot(before).SetAfterSnapshot(after).SaveX(ctx)
	}
	if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	for id, expected := range expectedInvoices {
		if got := d.db.FinanceInvoice.GetX(ctx, id).InvoiceType; string(got) != expected {
			t.Fatalf("原票种须按普专分别映射: %s != %s", got, expected)
		}
	}
	for id, expected := range expectedProfiles {
		if got := d.db.PartnerInvoiceProfile.GetX(ctx, id).DefaultInvoiceType; string(got) != expected {
			t.Fatalf("抬头默认票种须按普专分别映射: %s != %s", got, expected)
		}
	}
	var invalid int
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM finance_invoices WHERE invoice_type NOT IN ('DIGITAL_NORMAL','DIGITAL_SPECIAL')) + (SELECT count(*) FROM partner_invoice_profiles WHERE default_invoice_type NOT IN ('DIGITAL_NORMAL','DIGITAL_SPECIAL')) + (SELECT count(*) FROM finance_invoice_revisions WHERE before_snapshot->>'invoiceType' NOT IN ('DIGITAL_NORMAL','DIGITAL_SPECIAL') OR after_snapshot->>'invoiceType' NOT IN ('DIGITAL_NORMAL','DIGITAL_SPECIAL'))`).Scan(&invalid); err != nil || invalid != 0 {
		t.Fatalf("票据、抬头及修订历史必须一次性迁移为数电票种: %d %v", invalid, err)
	}
	for _, revision := range d.db.FinanceInvoiceRevision.Query().AllX(ctx) {
		var before, after map[string]string
		if err := json.Unmarshal(revision.BeforeSnapshot, &before); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(revision.AfterSnapshot, &after); err != nil {
			t.Fatal(err)
		}
		if before["invoiceType"] != expectedInvoices[revision.InvoiceID] || after["invoiceType"] != expectedInvoices[revision.InvoiceID] || before["totalAmount"] != "40.00000000" || after["totalAmount"] != "50.00000000" {
			t.Fatal("票种迁移不得改动历史经济事实")
		}
	}
	if _, err := d.sqlDB.ExecContext(ctx, `UPDATE finance_invoices SET invoice_type='NORMAL'`); err == nil {
		t.Fatal("迁移后数据库应拒绝旧票种")
	}
}
