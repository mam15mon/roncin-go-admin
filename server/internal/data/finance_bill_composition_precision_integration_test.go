package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/shopspring/decimal"
)

func TestFinanceBillCompositionVerificationPrecisionPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ensureFinanceBillBatchCurrencies(t, data, "CNY")
	ctx := context.Background()
	for _, amount := range []string{"0.12345678", "0.123456789", "0"} {
		t.Run(amount, func(t *testing.T) {
			f, operations, profile := compositionFixture(t, data)
			fee := f.createBatchUnbilledFee("precision")
			ledger := &feeLedgerPostgresFixture{financeBillPostgresFixture: f.financeBillPostgresFixture}
			cash := ledger.createConfirmedCashflow("precision", "100", financeBillIntegrationDate)
			uc := f.newUsecase(NewFinanceBillRepo(data))
			input := f.buildBatchInput(t, uc, "precision-"+f.suffix, []uuid.UUID{fee}, biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}, nil)
			input.Groups[0].Invoices = []biz.FinanceBillBatchInvoiceInput{compositionInvoice(profile, fee, "PRECISION-"+f.suffix, 100)}
			input.Groups[0].Verifications = []biz.FinanceBillBatchVerificationInput{{VerificationDate: financeBillIntegrationDate, Allocations: []biz.FinanceBillBatchVerificationAllocation{{CashflowID: cash.ID, Amount: decimal.RequireFromString(amount)}}}}
			result, err := uc.CreateBatchWithOperations(ctx, f.organizationID, f.actorUserID, input, operations)
			if err != nil {
				t.Fatal(err)
			}
			group := result.Groups[0]
			if amount == "0.12345678" {
				if group.Err != nil || group.Bill == nil || !group.Bill.VerifiedAmount.Equal(decimal.RequireFromString(amount)) {
					t.Fatalf("合法8位核销金额未完整保存: %+v", group)
				}
				return
			}
			if !errors.Is(group.Err, biz.ErrVerificationInvalid) {
				t.Fatalf("非法金额没有按组拒绝: %v", group.Err)
			}
			if count, err := data.db.FinanceBill.Query().Where(financebill.OrganizationIDEQ(f.organizationID)).Count(ctx); err != nil || count != 0 {
				t.Fatalf("非法核销留下账单: %d %v", count, err)
			}
			if count, err := data.db.FinanceVerification.Query().Where(financeverification.OrganizationIDEQ(f.organizationID)).Count(ctx); err != nil || count != 0 {
				t.Fatalf("非法核销留下分配: %d %v", count, err)
			}
		})
	}
}
