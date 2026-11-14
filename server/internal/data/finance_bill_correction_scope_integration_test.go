package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

func TestFinanceBillCorrectionScopeAfterReloadPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newFinanceInvoicePostgresFixture(t, data)
	ctx := context.Background()
	scope := []uuid.UUID{f.organizationID}
	repo := NewFinanceBillRepo(data)
	uc := biz.NewFinanceBillUsecase(repo, nil, data, nil)

	original := correctionFinanceFee(t, f, "scope-original", "100", nil, nil, orderfee.EconomicEffectINCREASE)
	request := correctionFinanceApprove(t, f, original)
	replacement := correctionFinanceFee(t, f, "scope-replacement", "120", &original.ID, &request.ID, orderfee.EconomicEffectINCREASE)
	correctedBill := correctionFinanceBill(t, f, "scope-corrected", financebill.PurposeNORMAL, replacement)
	normalFee := correctionFinanceFee(t, f, "scope-normal", "20", nil, nil, orderfee.EconomicEffectINCREASE)

	loaded, err := repo.Get(ctx, scope, correctedBill.ID)
	if err != nil || loaded.CorrectionRequestID == nil || *loaded.CorrectionRequestID != request.ID {
		t.Fatalf("重读须保留活动来源的更正归属: bill=%#v err=%v", loaded, err)
	}
	candidates, err := uc.ListFeeCandidates(ctx, scope, correctedBill.ID, biz.FinanceBillFeeCandidateFilter{Page: 1, PageSize: 200})
	if err != nil || candidates.Total != 0 || len(candidates.Items) != 0 {
		t.Fatalf("更正账单候选不得混入普通费用: result=%#v err=%v", candidates, err)
	}
	if _, err := uc.UpdateFees(ctx, scope, f.actorID, biz.UpdateFinanceBillFeesInput{ID: correctedBill.ID, ExpectedVersion: loaded.Version, AddFeeIDs: []uuid.UUID{normalFee.ID}}); !errors.Is(err, biz.ErrFinanceBillFeeMismatch) {
		t.Fatalf("直接提交普通费用也须拒绝: %v", err)
	}
	after, err := repo.Get(ctx, scope, correctedBill.ID)
	if err != nil || after.Version != loaded.Version || after.FeeCount != 1 || !after.TotalAmount.Equal(loaded.TotalAmount) {
		t.Fatalf("拒绝后账单必须零写入: bill=%#v err=%v", after, err)
	}

	// 相邻普通路径仍允许添加同范围的普通费用，不把更正限制扩大为普通账单永久冻结。
	normalBill := correctionFinanceBill(t, f, "scope-normal-bill", financebill.PurposeNORMAL, normalFee)
	additional := correctionFinanceFee(t, f, "scope-additional", "5", nil, nil, orderfee.EconomicEffectINCREASE)
	updated, err := uc.UpdateFees(ctx, scope, f.actorID, biz.UpdateFinanceBillFeesInput{ID: normalBill.ID, ExpectedVersion: normalBill.Version, AddFeeIDs: []uuid.UUID{additional.ID}})
	if err != nil || updated.FeeCount != 2 || updated.CorrectionRequestID != nil || updated.TotalAmount.String() != "25" {
		t.Fatalf("普通账单同源追加应继续成功: bill=%#v err=%v", updated, err)
	}
}

func TestFinanceBillEditAfterFeeRemovalPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newBillFeeUpdateFixture(t, data)
	f.createExchangeRateSetting("7.20000000")
	removedID := f.fee(billFeeSpec{key: "edit-removed", total: "100", currency: "USD", rate: "7"})
	keptID := f.fee(billFeeSpec{key: "edit-kept", total: "50", currency: "USD", rate: "7"})
	bill := f.bill("edit-history", f.usdAccountID, removedID, keptID)
	updated, err := f.update(bill.ID, bill.Version, nil, []uuid.UUID{removedID})
	if err != nil {
		t.Fatal(err)
	}
	history := data.db.FinanceBillLine.Query().Where(financebillline.BillIDEQ(bill.ID), financebillline.OrderFeeIDEQ(removedID)).OnlyX(f.ctx)
	if err := data.db.OrderFee.DeleteOneID(removedID).Exec(f.ctx); err != nil {
		t.Fatalf("已释放的普通费用应允许删除: %v", err)
	}
	note := "费用移除后继续编辑账单"
	result, err := f.usecase.Update(f.ctx, f.scope, f.actorID, biz.UpdateFinanceBillInput{
		ID: bill.ID, ExpectedVersion: updated.Version, BillDate: updated.BillDate, SettlementAccountID: updated.SettlementAccountID,
		PaymentTermsDays: updated.PaymentTermsDays, DueDate: updated.DueDate, EstimatedInvoiceCurrency: updated.EstimatedInvoiceCurrency, EstimatedInvoiceRate: updated.EstimatedInvoiceRate, Note: &note,
	})
	if err != nil || result.Version != updated.Version+1 || result.TotalAmount.String() != "50" || result.Note == nil || *result.Note != note {
		t.Fatalf("编辑应只依赖当前活动费用: result=%#v err=%v", result, err)
	}
	f.requireBillConsistent(bill.ID, keptID)
	storedHistory := data.db.FinanceBillLine.GetX(f.ctx, history.ID)
	if storedHistory.Active || storedHistory.TotalAmount != history.TotalAmount || storedHistory.BaseCurrencyAmount != history.BaseCurrencyAmount || storedHistory.OrderFeeID != uuid.Nil {
		t.Fatal("历史释放行应保留原金额且不参与重算，来源删除后仅清除引用")
	}
}
