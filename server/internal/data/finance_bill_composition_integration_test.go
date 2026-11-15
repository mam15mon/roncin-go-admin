package data

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicecommand"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicefeeallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicerevision"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerinvoiceprofile"
	"github.com/shopspring/decimal"
)

func compositionFixture(t *testing.T, data *Data) (*financeBillBatchPostgresFixture, biz.FinanceBillBatchOperations, uuid.UUID) {
	t.Helper()
	f := newFinanceBillBatchPostgresFixture(t, data)
	ctx := context.Background()
	for _, documentType := range []numberrule.DocumentType{numberrule.DocumentTypeInvoice, numberrule.DocumentTypeReceiptPayment, numberrule.DocumentTypeWriteOff, numberrule.DocumentTypeNetting} {
		if _, err := data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(documentType).SetPrefix(string(documentType) + "-").SetDateFormat(numberrule.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberrule.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
	}
	profile, err := data.db.PartnerInvoiceProfile.Create().SetOrganizationID(f.organizationID).SetPartnerID(f.partnerID).SetInvoiceTitle("组合登记抬头").SetTaxpayerIdentificationNo("COMBO-" + f.suffix).SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		steps := []func() error{
			func() error {
				_, e := data.db.FinanceVerificationAllocation.Delete().Where(financeverificationallocation.HasVerificationWith(financeverification.OrganizationIDEQ(f.organizationID))).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.FinanceVerification.Delete().Where(financeverification.OrganizationIDEQ(f.organizationID)).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.FinanceCashflow.Delete().Where(financecashflow.OrganizationIDEQ(f.organizationID)).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.FinanceInvoiceFeeAllocation.Delete().Where(financeinvoicefeeallocation.HasInvoiceWith(financeinvoice.OrganizationIDEQ(f.organizationID))).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.FinanceInvoiceRevision.Delete().Where(financeinvoicerevision.HasInvoiceWith(financeinvoice.OrganizationIDEQ(f.organizationID))).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.FinanceInvoiceBill.Delete().Where(financeinvoicebill.HasInvoiceWith(financeinvoice.OrganizationIDEQ(f.organizationID))).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.FinanceInvoiceLine.Delete().Where(financeinvoiceline.HasInvoiceWith(financeinvoice.OrganizationIDEQ(f.organizationID))).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.FinanceInvoice.Delete().Where(financeinvoice.OrganizationIDEQ(f.organizationID)).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.FinanceInvoiceCommand.Delete().Where(financeinvoicecommand.OrganizationIDEQ(f.organizationID)).Exec(ctx)
				return e
			},
			func() error {
				_, e := data.db.PartnerInvoiceProfile.Delete().Where(partnerinvoiceprofile.OrganizationIDEQ(f.organizationID)).Exec(ctx)
				return e
			},
		}
		for _, step := range steps {
			if err := step(); err != nil {
				t.Error(err)
			}
		}
	})
	t.Cleanup(f.cleanupNettingRows)
	exchange := biz.NewExchangeRateUsecase(NewExchangeRateRepo(data), nil)
	operations := biz.FinanceBillBatchOperations{Invoices: biz.NewFinanceInvoiceUsecase(NewFinanceInvoiceRepo(data), biz.NewOrderConfigUsecase(NewOrderConfigRepo(data)), exchange, data),
		Verifications: biz.NewVerificationUsecase(NewVerificationRepo(data), exchange, data, nil), Cashflows: biz.NewFinanceCashflowUsecase(NewFinanceCashflowRepo(data), exchange),
		Permissions: biz.FinanceBillBatchOperationPermissions{RegisterInvoices: true, CreateVerifications: true, CreateCashflows: true, ConfirmCashflows: true, OverrideCashflowRate: true}}
	return f, operations, profile.ID
}

func compositionInvoice(profile, fee uuid.UUID, number string, amount int64) biz.FinanceBillBatchInvoiceInput {
	return biz.FinanceBillBatchInvoiceInput{InvoiceProfileID: profile, InvoiceType: biz.FinanceInvoiceDigitalNormal, TaxInvoiceNo: number, InvoiceDate: financeBillIntegrationDate, TotalAmount: decimal.NewFromInt(amount), Fees: []biz.FinanceInvoiceFeeInput{{OrderFeeID: fee, Amount: decimal.NewFromInt(amount), TaxAmount: decimal.Zero}}}
}
func compositionShortfall(amount int64) biz.FinanceBillBatchVerificationInput {
	return biz.FinanceBillBatchVerificationInput{VerificationDate: financeBillIntegrationDate, Allocations: []biz.FinanceBillBatchVerificationAllocation{{Amount: decimal.NewFromInt(amount)}}, ShortfallCashflow: &biz.FinanceBillBatchCashflowInput{Amount: decimal.NewFromInt(amount), TransactionDate: financeBillIntegrationDate, OurAccount: "组合收款账户", PaymentMethod: "BANK_TRANSFER"}}
}

func TestFinanceBillCompositionAtomicPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ensureFinanceBillBatchCurrencies(t, data, "CNY")
	ctx := context.Background()
	for _, scenario := range []string{"多票多次核销及同键重试", "不登记发票先核销", "既有资金加差额", "多笔既有资金锁序", "同键并发不重复组合", "发票第二条失败整组回滚", "核销第二条失败整组回滚", "缺附带权限整组拒绝", "某组失败不影响其他组", "NETTING任一步失败整批回滚"} {
		t.Run(scenario, func(t *testing.T) {
			f, operations, profile := compositionFixture(t, data)
			fee := f.createBatchUnbilledFee("composite")
			policy := biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}
			fees := []uuid.UUID{fee}
			if scenario == "某组失败不影响其他组" || scenario == "NETTING任一步失败整批回滚" {
				fees = append(fees, f.createBatchUnbilledPayableFee("other"))
			}
			if scenario == "NETTING任一步失败整批回滚" {
				policy.Mode = "NETTING"
			}
			usecase := f.newUsecase(NewFinanceBillRepo(data))
			input := f.buildBatchInput(t, usecase, "composition-"+f.suffix, fees, policy, nil)
			target := 0
			for index, group := range input.Groups {
				if group.FeeIDs[0] == fee {
					target = index
				}
			}
			group := &input.Groups[target]
			group.Invoices = []biz.FinanceBillBatchInvoiceInput{compositionInvoice(profile, fee, "INVOICE-A-"+f.suffix, 40), compositionInvoice(profile, fee, "INVOICE-B-"+f.suffix, 60)}
			group.Verifications = []biz.FinanceBillBatchVerificationInput{compositionShortfall(40), compositionShortfall(60)}
			expectFailure := false
			switch scenario {
			case "不登记发票先核销":
				group.Invoices = nil
			case "既有资金加差额":
				ledger := &feeLedgerPostgresFixture{financeBillPostgresFixture: f.financeBillPostgresFixture}
				cash := ledger.createConfirmedCashflow("existing", "60", financeBillIntegrationDate)
				group.Verifications = []biz.FinanceBillBatchVerificationInput{{VerificationDate: financeBillIntegrationDate, Allocations: []biz.FinanceBillBatchVerificationAllocation{{CashflowID: cash.ID, Amount: decimal.NewFromInt(60)}}}, compositionShortfall(40)}
			case "多笔既有资金锁序":
				ledger := &feeLedgerPostgresFixture{financeBillPostgresFixture: f.financeBillPostgresFixture}
				cashA, cashB := ledger.createConfirmedCashflow("first", "40", financeBillIntegrationDate), ledger.createConfirmedCashflow("second", "60", financeBillIntegrationDate)
				group.Verifications = []biz.FinanceBillBatchVerificationInput{{VerificationDate: financeBillIntegrationDate, Allocations: []biz.FinanceBillBatchVerificationAllocation{{CashflowID: cashB.ID, Amount: decimal.NewFromInt(60)}}}, {VerificationDate: financeBillIntegrationDate, Allocations: []biz.FinanceBillBatchVerificationAllocation{{CashflowID: cashA.ID, Amount: decimal.NewFromInt(40)}}}}
			case "发票第二条失败整组回滚":
				group.Invoices[1].Fees[0].Amount = decimal.NewFromInt(61)
				expectFailure = true
			case "核销第二条失败整组回滚", "某组失败不影响其他组", "NETTING任一步失败整批回滚":
				group.Verifications[1] = compositionShortfall(61)
				expectFailure = true
			case "缺附带权限整组拒绝":
				operations.Permissions.ConfirmCashflows = false
				expectFailure = true
			}
			if scenario == "同键并发不重复组合" {
				results := make(chan financeBillBatchCreateResult, 2)
				start := make(chan struct{})
				for range 2 {
					go func() {
						<-start
						result, err := usecase.CreateBatchWithOperations(ctx, f.organizationID, f.actorUserID, input, operations)
						results <- financeBillBatchCreateResult{result: result, err: err}
					}()
				}
				close(start)
				var billID uuid.UUID
				for range 2 {
					item := <-results
					if item.err != nil || item.result.Groups[0].Err != nil {
						t.Fatalf("同键并发组合失败: %+v", item)
					}
					id := item.result.Groups[0].Bill.ID
					if billID != uuid.Nil && billID != id {
						t.Fatal("并发生成了不同账单")
					}
					billID = id
				}
			}
			result, err := usecase.CreateBatchWithOperations(ctx, f.organizationID, f.actorUserID, input, operations)
			if scenario == "NETTING任一步失败整批回滚" {
				if err == nil || errors.FromError(err).Metadata["step"] != "VERIFICATION" {
					t.Fatalf("NETTING应整体失败且定位核销: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("普通组处理失败: %v", err)
				}
				byKey := groupResultsByKey(t, input, result)
				if expectFailure {
					failure := byKey[group.GroupKey].Err
					if failure == nil || errors.FromError(failure).Metadata["step"] == "" {
						t.Fatalf("应返回逐步骤失败: %v", failure)
					}
					if scenario == "某组失败不影响其他组" {
						for key, item := range byKey {
							if key != group.GroupKey && (item.Bill == nil || item.Err != nil) {
								t.Fatalf("其他组未成功: %v", item.Err)
							}
						}
					}
				} else if byKey[group.GroupKey].Err != nil {
					t.Fatal(byKey[group.GroupKey].Err)
				}
			}
			invoiceCount, err := data.db.FinanceInvoice.Query().Where(financeinvoice.OrganizationIDEQ(f.organizationID)).Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			cashCount, err := data.db.FinanceCashflow.Query().Where(financecashflow.OrganizationIDEQ(f.organizationID)).Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			verificationCount, err := data.db.FinanceVerification.Query().Where(financeverification.OrganizationIDEQ(f.organizationID)).Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if expectFailure {
				if invoiceCount != 0 || cashCount != 0 || verificationCount != 0 {
					t.Fatalf("失败留下部分事实: invoice=%d cash=%d verification=%d", invoiceCount, cashCount, verificationCount)
				}
				billCount, err := data.db.FinanceBill.Query().Where(financebill.OrganizationIDEQ(f.organizationID)).Count(ctx)
				wantBills := 0
				if scenario == "某组失败不影响其他组" {
					wantBills = 1
				}
				if err != nil || billCount != wantBills {
					t.Fatalf("失败留下账单: count=%d want=%d err=%v", billCount, wantBills, err)
				}
				sequenceCount, err := data.db.NumberSequence.Query().Where(numbersequence.HasRuleWith(numberrule.OrganizationIDEQ(f.organizationID), numberrule.DocumentTypeIn(numberrule.DocumentTypeInvoice, numberrule.DocumentTypeReceiptPayment, numberrule.DocumentTypeWriteOff))).Count(ctx)
				if err != nil || sequenceCount != 0 {
					t.Fatalf("失败留下组合编号序列: %d %v", sequenceCount, err)
				}
				audits, err := data.db.AuditLog.Query().Where(auditlog.OrganizationIDEQ(f.organizationID), auditlog.ActionIn("finance.invoice.register", "finance.cashflow.create", "finance.cashflow.confirm", "finance.verification.create")).Count(ctx)
				if err != nil || audits != 0 {
					t.Fatalf("失败留下组合审计: %d %v", audits, err)
				}
				return
			}
			wantInvoices := 2
			if scenario == "不登记发票先核销" {
				wantInvoices = 0
			}
			if invoiceCount != wantInvoices || cashCount != 2 || verificationCount != 2 {
				t.Fatalf("组合事实数量不符: %d/%d/%d", invoiceCount, cashCount, verificationCount)
			}
			full := result.Groups[target].Bill
			if !full.VerifiedAmount.Equal(decimal.NewFromInt(100)) || !full.UnverifiedAmount.IsZero() || (wantInvoices > 0 && full.InvoiceProgress != biz.FinanceBillInvoiceFull) {
				t.Fatalf("组合结果未返回完整投影: %+v", full)
			}
			replay, err := usecase.CreateBatchWithOperations(ctx, f.organizationID, f.actorUserID, input, operations)
			if err != nil || replay.Groups[target].Bill.ID != result.Groups[target].Bill.ID {
				t.Fatalf("重试没有回放原组: %v", err)
			}
			differentNote := "不同组合内容"
			group.Verifications[0].Note = &differentNote
			if _, err := usecase.CreateBatchWithOperations(ctx, f.organizationID, f.actorUserID, input, operations); !stderrors.Is(err, biz.ErrFinanceBillBatchConflict) {
				t.Fatalf("同键不同组合意图应冲突: %v", err)
			}
		})
	}
}
