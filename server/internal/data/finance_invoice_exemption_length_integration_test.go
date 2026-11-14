package data

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceexemption"
)

// 免票原因的接口上限是字符数，不能在 Ent 持久化时错误地按 UTF-8 字节收紧。
func TestFinanceInvoiceExemptionChineseReasonLengthPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newFinanceInvoicePostgresFixture(t, data)
	bill := f.createConfirmedBill("chinese-reason-length")
	ctx := context.Background()
	uc := biz.NewFinanceInvoiceUsecase(NewFinanceInvoiceRepo(data), nil, nil, data)
	scope := []uuid.UUID{f.organizationID}
	longReason := strings.Repeat("免", 500)
	if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, bill.Version, longReason, false); err != nil {
		t.Fatalf("500个中文标记原因必须可落库: %v", err)
	}
	marked := data.db.FinanceInvoiceExemption.Query().Where(financeinvoiceexemption.BillIDEQ(bill.ID)).OnlyX(ctx)
	if marked.Reason != longReason {
		t.Fatal("标记原因不应截断或改写")
	}
	current := data.db.FinanceBill.Query().Where(financebill.IDEQ(bill.ID)).OnlyX(ctx)
	if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, current.Version, longReason+"免", true); !errors.Is(err, biz.ErrFinanceInvoiceInvalidArgument) {
		t.Fatalf("501个中文撤销原因必须在领域层拒绝: %v", err)
	}
	if data.db.FinanceInvoiceExemption.GetX(ctx, marked.ID).RevokedAt != nil || data.db.FinanceBill.GetX(ctx, bill.ID).Version != current.Version {
		t.Fatal("超长撤销请求不应推进版本或写入撤销事实")
	}
	revocation := strings.Repeat("撤", 500)
	if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, current.Version, revocation, true); err != nil {
		t.Fatalf("500个中文撤销原因必须可落库: %v", err)
	}
	revoked := data.db.FinanceInvoiceExemption.GetX(ctx, marked.ID)
	if revoked.RevocationReason == nil || *revoked.RevocationReason != revocation || revoked.RevokedAt == nil {
		t.Fatal("撤销原因不应截断，原标记历史必须保留")
	}
	version := data.db.FinanceBill.GetX(ctx, bill.ID).Version
	if err := uc.SetBillInvoiceExemption(ctx, scope, f.actorID, bill.ID, version, longReason+"免", false); !errors.Is(err, biz.ErrFinanceInvoiceInvalidArgument) {
		t.Fatalf("501个中文标记原因必须在领域层拒绝: %v", err)
	}
	if data.db.FinanceInvoiceExemption.Query().Where(financeinvoiceexemption.BillIDEQ(bill.ID)).CountX(ctx) != 1 || data.db.FinanceBill.GetX(ctx, bill.ID).Version != version {
		t.Fatal("超长标记请求不应新建记录或推进版本")
	}
}
