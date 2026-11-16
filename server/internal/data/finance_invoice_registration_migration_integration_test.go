package data

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicefeeallocation"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

// oldRegistrationInvoice 只使用迁移前真实列，不借现行Ent创建不存在的revision列。
func oldRegistrationInvoice(t *testing.T, d *Data, f *financeInvoicePostgresFixture, status string) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	bill := f.createConfirmedBill("legacy-" + status)
	line := d.db.FinanceBillLine.Query().Where(financebillline.BillIDEQ(bill.ID)).OnlyX(ctx)
	invoiceID, linkID := uuid.New(), uuid.New()
	sourceIDs, err := json.Marshal([]uuid.UUID{line.OrderFeeID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.sqlDB.ExecContext(ctx, `INSERT INTO finance_invoices(id,created_at,updated_at,organization_id,record_no,idempotency_key,direction,status,invoice_type,settlement_party_id,settlement_party_name,currency,base_currency,total_amount,net_amount,tax_amount,bill_count,version,tax_invoice_no,invoice_date,issued_at,issued_by,exchange_rate,exchange_rate_source,exchange_rate_date,base_currency_amount,cancelled_at,cancelled_by,cancellation_reason)
 VALUES($1,NOW(),NOW(),$2,$3::text,$3::text,'RECEIVABLE',$4::text,'NORMAL',$5,'迁移测试客商','CNY','CNY',100,100,0,1,1,CASE WHEN $4::text='ISSUED' THEN $3::text ELSE NULL END,CASE WHEN $4::text='ISSUED' THEN '2026-09-10' ELSE NULL END,CASE WHEN $4::text='ISSUED' THEN NOW() ELSE NULL END,CASE WHEN $4::text='ISSUED' THEN $6::uuid ELSE NULL END,CASE WHEN $4::text='ISSUED' THEN 1 ELSE NULL END,CASE WHEN $4::text='ISSUED' THEN 'SYSTEM' ELSE NULL END,CASE WHEN $4::text='ISSUED' THEN '2026-09-10' ELSE NULL END,CASE WHEN $4::text='ISSUED' THEN 100 ELSE NULL END,CASE WHEN $4::text='CANCELLED' THEN NOW() ELSE NULL END,CASE WHEN $4::text='CANCELLED' THEN $6::uuid ELSE NULL END,CASE WHEN $4::text='CANCELLED' THEN '原草稿已取消' ELSE NULL END)`, invoiceID, f.organizationID, invoiceID.String(), status, f.partnerID, f.actorID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.sqlDB.ExecContext(ctx, `INSERT INTO finance_invoice_bills(id,created_at,updated_at,invoice_id,bill_id,bill_no,amount,tax_amount,active,source_fee_ids)VALUES($1,NOW(),NOW(),$2,$3,$4,100,0,$5,$6)`, linkID, invoiceID, bill.ID, bill.BillNo, status != "CANCELLED", sourceIDs)
	if err != nil {
		t.Fatal(err)
	}
	return invoiceID, line.ID, line.OrderFeeID
}
func TestFinanceInvoiceRegistrationMigrationPostgres(t *testing.T) {
	t.Run("完整已登记与取消历史通过并保留税率存在性", func(t *testing.T) {
		d, cleanup := getIntegrationDataBeforeMigration(t, "20261114140000")
		defer cleanup()
		f := newFinanceInvoiceMigrationFixture(t, d)
		ctx := context.Background()
		issued, _, _ := oldRegistrationInvoice(t, d, f, "ISSUED")
		cancelled, zeroLine, _ := oldRegistrationInvoice(t, d, f, "CANCELLED")
		if err := d.db.FinanceBillLine.UpdateOneID(zeroLine).SetTaxRate("0").Exec(ctx); err != nil {
			t.Fatal(err)
		}
		if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatal(err)
		}
		rows := d.db.FinanceInvoiceFeeAllocation.Query().Where(allocation.InvoiceIDIn(issued, cancelled)).AllX(ctx)
		if len(rows) != 2 {
			t.Fatalf("原始两条分配都须保留: %d", len(rows))
		}
		for _, row := range rows {
			if row.InvoiceID == issued && row.SourceTaxRate != nil {
				t.Fatal("未设来源税率不能补0")
			}
			if row.InvoiceID == cancelled && (row.SourceTaxRate == nil || *row.SourceTaxRate != "0.0000") {
				t.Fatal("显式零税率应保留")
			}
		}
		if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatal(err)
		}
		if d.db.FinanceInvoiceFeeAllocation.Query().CountX(ctx) != 2 {
			t.Fatal("重跑正式迁移不得重复分配")
		}
	})
	for _, scenario := range []string{"不完备草稿", "已删除来源无法确定历史分配"} {
		t.Run(scenario, func(t *testing.T) {
			d, cleanup := getIntegrationDataBeforeMigration(t, "20261114140000")
			defer cleanup()
			f := newFinanceInvoiceMigrationFixture(t, d)
			ctx := context.Background()
			status := "DRAFT"
			if scenario != "不完备草稿" {
				status = "CANCELLED"
			}
			invoice, line, fee := oldRegistrationInvoice(t, d, f, status)
			if status == "CANCELLED" {
				if err := d.db.FinanceBillLine.UpdateOneID(line).SetActive(false).Exec(ctx); err != nil {
					t.Fatal(err)
				}
				if err := d.db.OrderFee.DeleteOneID(fee).Exec(ctx); err != nil {
					t.Fatal(err)
				}
			}
			err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations"))
			if err == nil || !strings.Contains(err.Error(), "迁移受阻") {
				t.Fatalf("历史事实不完整必须显式拒绝迁移: %v", err)
			}
			var count int
			if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM finance_invoices WHERE id=$1`, invoice).Scan(&count); err != nil || count != 1 {
				t.Fatalf("阻断不得删除原票: %d %v", count, err)
			}
			var exists bool
			if err := d.sqlDB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='finance_invoices' AND column_name='revision_number')`).Scan(&exists); err != nil || exists {
				t.Fatalf("失败迁移应回滚新结构: %v %v", exists, err)
			}
		})
	}
}
