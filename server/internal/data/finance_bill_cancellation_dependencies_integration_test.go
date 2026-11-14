package data

import (
	"context"
	"strings"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	lineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	cashent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	numberent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
)

// requireBlockedBillCancellation 验证具体回退指引与失败零写入，而非只检查状态错误码。
func requireBlockedBillCancellation(t *testing.T, data *Data, bill *ent.FinanceBill, actorID uuid.UUID, documentNo, action, permission string) {
	t.Helper()
	ctx := context.Background()
	_, err := NewFinanceBillRepo(data).Cancel(ctx, []uuid.UUID{bill.OrganizationID}, bill.ID, actorID, bill.Version, "退回未建账", &biz.AuditEvent{Action: "finance.bill.cancel", Result: "success"})
	problem := kratoserrors.FromError(err)
	if err == nil || problem.Reason != "FINANCE_BILL_CANCELLATION_BLOCKED" || !strings.Contains(problem.Message, documentNo) || !strings.Contains(problem.Message, action) || !strings.Contains(problem.Message, bill.BillNo) || !strings.Contains(problem.Message, "整张") {
		t.Fatalf("缺少实际依赖及回退指引: %v", err)
	}
	if problem.Metadata["rollback_scope"] != "WHOLE_BILL" || !strings.Contains(problem.Metadata["required_permissions"], permission) || problem.Metadata["dependencies"] == "" {
		t.Fatalf("缺少影响范围或权限: %+v", problem.Metadata)
	}
	latest, err := data.db.FinanceBill.Get(ctx, bill.ID)
	if err != nil || latest.Status != bill.Status || latest.Version != bill.Version {
		t.Fatalf("阻断后账单发生变化: %+v %v", latest, err)
	}
	count, err := data.db.FinanceBillLine.Query().Where(lineent.BillIDEQ(bill.ID), lineent.ActiveEQ(true)).Count(ctx)
	if err != nil || count != bill.FeeCount {
		t.Fatalf("阻断后费用关联变化: %d %v", count, err)
	}
}

func cancelReleasedBillForTest(t *testing.T, data *Data, organizationID, billID, actorID uuid.UUID) {
	t.Helper()
	bill, err := data.db.FinanceBill.Get(context.Background(), billID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewFinanceBillRepo(data).Cancel(context.Background(), []uuid.UUID{organizationID}, billID, actorID, bill.Version, "依赖已回退，整张取消", &biz.AuditEvent{Action: "finance.bill.cancel", Result: "success"})
	if err != nil || result.Status != biz.FinanceBillCancelled {
		t.Fatalf("完成回退后整账取消失败: %+v %v", result, err)
	}
	count, err := data.db.FinanceBillLine.Query().Where(lineent.BillIDEQ(billID), lineent.ActiveEQ(true)).Count(context.Background())
	if err != nil || count != 0 {
		t.Fatalf("整账取消未释放全部费用: %d %v", count, err)
	}
}

func TestFinanceBillCancellationDependenciesPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	// 草稿开票已退役：改为当前直登记的部分作废、足额红冲两个相邻回退路径。
	for _, tc := range []struct {
		name, amount string
		cancel       bool
	}{{"部分登记后作废释放", "30", true}, {"足额登记后红冲释放", "100", false}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRegistrationFixture(t, data)
			bill := f.createConfirmedBill("rollback-registration")
			invoices, err := f.register("rollback-registration", f.record(bill, "TAX-"+f.suffix, tc.amount, "0"))
			if err != nil {
				t.Fatal(err)
			}
			invoice := invoices[0]
			bill = data.db.FinanceBill.GetX(ctx, bill.ID)
			requireBlockedBillCancellation(t, data, bill, f.actorID, invoice.RecordNo, "红冲", access.FinanceInvoiceUpdate)
			repo := NewFinanceInvoiceRepo(data)
			if tc.cancel {
				_, err = repo.Cancel(ctx, f.organizationID, invoice.ID, f.actorID, invoice.Version, "作废关联登记", financeInvoiceTestAudit("finance.invoice.cancel", f.organizationID, f.actorID, invoice.ID))
			} else {
				_, err = repo.RedFlush(ctx, f.organizationID, invoice.ID, f.actorID, invoice.Version, "RED-"+f.suffix, financeInvoiceIntegrationDate, "红冲关联登记", financeInvoiceTestAudit("finance.invoice.red_flush", f.organizationID, f.actorID, invoice.ID))
			}
			if err != nil {
				t.Fatal(err)
			}
			// 票据与逐费用分配仍保留，只有活动性释放，才能继续整账取消。
			original, err := repo.Get(ctx, f.organizationID, invoice.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(original.Links) != 1 || original.Links[0].Active || len(original.Links[0].Fees) != 1 || !original.TotalAmount.Equal(invoice.TotalAmount) || !original.Links[0].Amount.Equal(invoice.TotalAmount) {
				t.Fatalf("反向操作不得删除或重算原始分配: %+v", original)
			}
			cancelReleasedBillForTest(t, data, f.organizationID, bill.ID, f.actorID)
		})
	}

	t.Run("核销", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, data)
		if _, err := data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(numberent.DocumentTypeWriteOff).SetPrefix("WO").SetDateFormat(numberent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberent.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
		bill := f.createConfirmedBill("rollback-verification")
		cash := correctionFinanceCash(t, f, "cash-"+f.suffix, "40", cashent.DirectionRECEIVABLE)
		verification, err := correctionFinanceVerification(t, f, cash, bill, "40")
		if err != nil {
			t.Fatal(err)
		}
		bill, err = data.db.FinanceBill.Get(ctx, bill.ID)
		if err != nil {
			t.Fatal(err)
		}
		requireBlockedBillCancellation(t, data, bill, f.actorID, verification.VerificationNo, "反核销", access.FinanceVerificationReverse)
		if _, err = NewVerificationRepo(data).Reverse(ctx, f.organizationID, verification.ID, f.actorID, verification.Version, "反核销后改单", &biz.AuditEvent{Action: "finance.verification.reverse", Result: "success"}); err != nil {
			t.Fatal(err)
		}
		cancelReleasedBillForTest(t, data, f.organizationID, bill.ID, f.actorID)
	})
	for _, confirmed := range []bool{false, true} {
		name := "草稿对冲"
		if confirmed {
			name = "已确认对冲"
		}
		t.Run(name, func(t *testing.T) {
			// 账单取消人需晚于对冲夹具和账单本体清理，避免外键反序。
			billActor := data.db.User.Create().SetDisplayName("账单取消测试人员").SaveX(ctx)
			t.Cleanup(func() {
				if err := data.db.User.DeleteOneID(billActor.ID).Exec(context.Background()); err != nil {
					t.Errorf("清理账单取消人: %v", err)
				}
			})
			f := newNettingPostgresFixture(t, data)
			uc := f.newUsecase()
			netting, err := uc.Create(ctx, f.organizationID, f.actorUserID, f.input("rollback-dependency"))
			if err != nil {
				t.Fatal(err)
			}
			action := "取消草稿对冲"
			if confirmed {
				netting, err = uc.Confirm(ctx, []uuid.UUID{f.organizationID}, f.actorUserID, netting.ID, netting.Version)
				if err != nil {
					t.Fatal(err)
				}
				action = "反对冲"
			}
			bill, err := data.db.FinanceBill.Query().Where(billent.IDEQ(f.receivableBillID)).Only(ctx)
			if err != nil {
				t.Fatal(err)
			}
			requireBlockedBillCancellation(t, data, bill, f.actorUserID, netting.NettingNo, action, access.FinanceNettingReverse)
			repo := NewFinanceNettingRepo(data)
			audit := &biz.AuditEvent{Action: "finance.netting.reverse", Result: "success"}
			if confirmed {
				_, err = repo.Reverse(ctx, f.organizationID, f.actorUserID, netting.ID, netting.Version, "回退改单", audit)
			} else {
				_, err = repo.Cancel(ctx, f.organizationID, f.actorUserID, netting.ID, netting.Version, "取消草稿后改单", audit)
			}
			if err != nil {
				t.Fatal(err)
			}
			cancelReleasedBillForTest(t, data, f.organizationID, bill.ID, billActor.ID)
		})
	}
}
