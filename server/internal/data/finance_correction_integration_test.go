package data

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	number "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	request "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
	"github.com/shopspring/decimal"
)

func correctionFinanceFee(t *testing.T, f *financeInvoicePostgresFixture, key, amount string, source, correction *uuid.UUID, effect fee.EconomicEffect) *ent.OrderFee {
	t.Helper()
	builder := f.data.db.OrderFee.Create().SetOrderID(f.orderID).SetIdempotencyKey(key).SetDirection(fee.DirectionRECEIVABLE).SetEconomicEffect(effect).SetFeeCode(key).SetFeeName(key).SetSettlementPartyID(f.partnerID).SetBillingUnit("票").SetQuantity("1").SetUnitPrice(amount).SetTotalAmount(amount).SetNetAmount(amount).SetTaxAmount("0").SetTaxRate("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeInvoiceIntegrationDate).SetBaseCurrency("CNY").SetBaseCurrencyAmount(amount).SetNillableSourceFeeID(source).SetNillableCorrectionRequestID(correction)
	result, err := builder.Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func correctionFinanceBill(t *testing.T, f *financeInvoicePostgresFixture, key string, purpose bill.Purpose, rows ...*ent.OrderFee) *ent.FinanceBill {
	t.Helper()
	ctx := context.Background()
	total := decimal.Zero
	for _, r := range rows {
		total = total.Add(decimal.RequireFromString(r.TotalAmount))
	}
	direction := bill.DirectionRECEIVABLE
	if purpose == bill.PurposeCORRECTION_REVERSAL {
		direction = bill.DirectionPAYABLE
	}
	builder := f.data.db.FinanceBill.Create().SetOrganizationID(f.organizationID).SetBillNo(key).SetIdempotencyKey(key).SetDirection(direction).SetPurpose(purpose).SetStatus(bill.StatusCONFIRMED).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("测试客户").SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(bill.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeInvoiceIntegrationDate).SetTotalAmount(total.String()).SetNetAmount(total.String()).SetTaxAmount("0").SetBaseCurrencyAmount(total.String()).SetFeeCount(len(rows)).SetBillDate(financeInvoiceIntegrationDate)
	result, err := withTestFinanceBillSettlementAccountSnapshot(builder, uuid.New(), "CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range rows {
		id := uuid.New()
		if key == "old" {
			id = uuid.MustParse([]string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"}[i])
		}
		_, err = f.data.db.FinanceBillLine.Create().SetID(id).SetBillID(result.ID).SetOrderFeeID(r.ID).SetOrderID(r.OrderID).SetOrderNo("SE" + f.suffix).SetFeeCode(r.FeeCode).SetFeeName(r.FeeName).SetQuantity("1").SetUnitPrice(r.UnitPrice).SetTotalAmount(r.TotalAmount).SetNetAmount(r.NetAmount).SetTaxAmount("0").SetTaxRate("0").SetCurrency("CNY").SetExchangeRate("1").SetBaseCurrency("CNY").SetBaseCurrencyAmount(r.TotalAmount).SetActive(true).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}
func correctionFinanceApprove(t *testing.T, f *financeInvoicePostgresFixture, original *ent.OrderFee) *ent.OrderFeeCorrectionRequest {
	t.Helper()
	now := time.Now()
	v := original.Version
	r, err := f.data.db.OrderFeeCorrectionRequest.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderID).SetOperation(request.OperationREPLACE).SetSourceFeeID(original.ID).SetSourceFeeVersion(v).SetBeforeSnapshot(json.RawMessage(`{}`)).SetAfterSnapshot(json.RawMessage(`{}`)).SetApprovedFeeReviewEventID(uuid.New()).SetApprovedFeeContentVersion(1).SetApprovedFeeReviewVersion(1).SetReason("金额录错").SetIdempotencyKey(uuid.NewString()).SetRequestFingerprint("test").SetRequestedBy(f.actorID).SetRequestedAt(now).SetStatus(request.StatusAPPROVED).SetDecidedAt(now).SetDecidedBy(f.actorID).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err = f.data.db.OrderFee.UpdateOne(original).SetSupersededByRequestID(r.ID).Exec(context.Background()); err != nil {
		t.Fatal(err)
	}
	return r
}
func correctionFinanceCash(t *testing.T, f *financeInvoicePostgresFixture, key, amount string, direction cash.Direction) *ent.FinanceCashflow {
	t.Helper()
	r, err := f.data.db.FinanceCashflow.Create().SetOrganizationID(f.organizationID).SetFlowNo(key).SetIdempotencyKey(key).SetDirection(direction).SetStatus(cash.StatusCONFIRMED).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("客户").SetCurrency("CNY").SetAmount(amount).SetExchangeRate("1").SetExchangeRateSource(cash.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeInvoiceIntegrationDate).SetBaseCurrency("CNY").SetBaseAmount(amount).SetTransactionDate(financeInvoiceIntegrationDate).SetOurAccount("测试").SetPaymentMethod("银行").Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func correctionFinanceVerification(t *testing.T, f *financeInvoicePostgresFixture, c *ent.FinanceCashflow, b *ent.FinanceBill, amount string) (*biz.FinanceVerification, error) {
	t.Helper()
	v := &biz.FinanceVerification{ID: uuid.New(), Direction: biz.OrderFeeDirection(c.Direction), BaseCurrency: "CNY", Currency: "CNY", Amount: decimal.RequireFromString(amount), IdempotencyKey: uuid.NewString(), VerificationDate: financeInvoiceIntegrationDate, Allocations: []*biz.VerificationAllocation{{ID: uuid.New(), CashflowID: c.ID, BillID: b.ID, Amount: decimal.RequireFromString(amount)}}}
	return NewVerificationRepo(f.data).Create(context.Background(), f.organizationID, f.actorID, v, &biz.AuditEvent{Action: "test.verification", Result: "success"})
}

func TestFinanceCorrectionMergedSettlementPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newFinanceInvoicePostgresFixture(t, data)
	ctx := context.Background()
	for _, kind := range []number.DocumentType{number.DocumentTypeNetting, number.DocumentTypeWriteOff} {
		if _, err := data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(kind).SetPrefix("T").SetDateFormat(number.DateFormatNone).SetSequenceLength(4).SetResetPolicy(number.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
	}
	a := correctionFinanceFee(t, f, "A", "1000", nil, nil, fee.EconomicEffectINCREASE)
	b := correctionFinanceFee(t, f, "B", "200", nil, nil, fee.EconomicEffectINCREASE)
	otherOrder, err := data.db.Order.Create().SetIdempotencyKey(uuid.NewString()).SetOrganizationID(f.organizationID).SetOrderNo("SE-B" + f.suffix).SetCustomerID(f.partnerID).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetTradeTerm(orderent.TradeTermFOB).SetPaymentTerm(orderent.PaymentTermPREPAID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err = data.db.OrderFee.UpdateOne(b).SetOrderID(otherOrder.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	old := correctionFinanceBill(t, f, "old", bill.PurposeNORMAL, a, b)
	addTestBillInvoiceExemption(t, data, old, f.actorID)
	c := correctionFinanceCash(t, f, "收400", "400", cash.DirectionRECEIVABLE)
	v, err := correctionFinanceVerification(t, f, c, old, "400")
	if err != nil {
		t.Fatal(err)
	}
	r := correctionFinanceApprove(t, f, a)
	rev := correctionFinanceFee(t, f, "冲回A", "1000", &a.ID, &r.ID, fee.EconomicEffectDECREASE)
	replacement := correctionFinanceFee(t, f, "替代A", "100", &a.ID, &r.ID, fee.EconomicEffectINCREASE)
	reverseBill := correctionFinanceBill(t, f, "reverse", bill.PurposeCORRECTION_REVERSAL, rev)
	newBill := correctionFinanceBill(t, f, "replacement", bill.PurposeNORMAL, replacement)
	addTestBillInvoiceExemption(t, data, reverseBill, f.actorID)
	if _, err := NewFinanceBillRepo(data).Cancel(ctx, []uuid.UUID{f.organizationID}, old.ID, f.actorID, old.Version, "取消", &biz.AuditEvent{Action: "test", Result: "success"}); err == nil {
		t.Fatal("受保护原账不可取消")
	}
	c2 := correctionFinanceCash(t, f, "先收B", "133.33333333", cash.DirectionRECEIVABLE)
	if _, err := correctionFinanceVerification(t, f, c2, old, "133.33333333"); err != biz.ErrFinanceCorrectionSettlementRequired {
		t.Fatalf("原费未清需阻断普通核销: %v", err)
	}
	uc := biz.NewFinanceNettingUsecase(NewFinanceNettingRepo(data), data, nil)
	create := func(first, second *ent.FinanceBill, key string) *biz.FinanceNetting {
		n, err := uc.Create(ctx, f.organizationID, f.actorID, biz.CreateFinanceNettingInput{IdempotencyKey: key, Bills: []biz.FinanceNettingBillVersion{{BillID: first.ID, ExpectedVersion: 1}, {BillID: second.ID, ExpectedVersion: 1}}})
		if err != nil {
			t.Fatal(err)
		}
		if n.CorrectionRequestID == nil || *n.CorrectionRequestID != r.ID {
			t.Fatal("缺更正来源")
		}
		for _, a := range n.Allocations {
			if a.SourceFeeID == nil {
				t.Fatal("缺精确行来源")
			}
		}
		return n
	}
	n := create(old, reverseBill, "原冲回")
	if !n.Amount.Equal(decimal.RequireFromString("666.66666667")) {
		t.Fatalf("原A抵销只能666.66666667: %s", n.Amount)
	}
	if _, err = uc.Confirm(ctx, []uuid.UUID{f.organizationID}, f.actorID, n.ID, n.Version); err != nil {
		t.Fatal(err)
	}
	n = create(reverseBill, newBill, "替代覆盖")
	if _, err = uc.Confirm(ctx, []uuid.UUID{f.organizationID}, f.actorID, n.ID, n.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := NewVerificationRepo(data).Reverse(ctx, f.organizationID, v.ID, f.actorID, v.Version, "反核销", &biz.AuditEvent{Action: "test", Result: "success"}); err == nil {
		t.Fatal("已转移覆盖必须阻断来源反转")
	}
	if _, err = correctionFinanceVerification(t, f, c2, old, "133.33333333"); err != nil {
		t.Fatal(err)
	}
	projection, err := loadFinanceBillLineSettlements(ctx, data.db, []uuid.UUID{old.ID, reverseBill.ID, newBill.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range projection {
		expected := decimal.Zero
		if p.FeeID == rev.ID {
			expected = decimal.RequireFromString("233.33333333")
		}
		if !p.RemainingAmount.Equal(expected) {
			t.Fatalf("fee=%s剩余%s期望%s", p.FeeID, p.RemainingAmount, expected)
		}
	}
	refund := correctionFinanceCash(t, f, "退233", "233.33333333", cash.DirectionPAYABLE)
	if _, err = correctionFinanceVerification(t, f, refund, reverseBill, "233.33333333"); err != nil {
		t.Fatal(err)
	}
	projection, err = loadFinanceBillLineSettlements(ctx, data.db, []uuid.UUID{old.ID, reverseBill.ID, newBill.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range projection {
		if !p.RemainingAmount.IsZero() {
			t.Fatal("退款后未结", p.RemainingAmount)
		}
	}
	original, err := data.db.FinanceBill.Get(ctx, old.ID)
	if err != nil || original.TotalAmount != "1200.00000000" {
		t.Fatalf("原账被改写 %+v %v", original, err)
	}
}

func TestFinanceCorrectionUnpaidTransferBlockedPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	f := newFinanceInvoicePostgresFixture(t, data)
	ctx := context.Background()
	a := correctionFinanceFee(t, f, "A", "1000", nil, nil, fee.EconomicEffectINCREASE)
	old := correctionFinanceBill(t, f, "old", bill.PurposeNORMAL, a)
	_ = old
	r := correctionFinanceApprove(t, f, a)
	rev := correctionFinanceFee(t, f, "冲回", "1000", &a.ID, &r.ID, fee.EconomicEffectDECREASE)
	newFee := correctionFinanceFee(t, f, "新", "100", &a.ID, &r.ID, fee.EconomicEffectINCREASE)
	reverseBill := correctionFinanceBill(t, f, "reverse", bill.PurposeCORRECTION_REVERSAL, rev)
	newBill := correctionFinanceBill(t, f, "new", bill.PurposeNORMAL, newFee)
	uc := biz.NewFinanceNettingUsecase(NewFinanceNettingRepo(data), data, nil)
	_, err := uc.Create(ctx, f.organizationID, f.actorID, biz.CreateFinanceNettingInput{IdempotencyKey: "先抵替代", Bills: []biz.FinanceNettingBillVersion{{BillID: reverseBill.ID, ExpectedVersion: 1}, {BillID: newBill.ID, ExpectedVersion: 1}}})
	if err != biz.ErrFinanceNettingBalance {
		t.Fatalf("未收不能转移覆盖: %v", err)
	}
}

func correctionFinanceInvoice(t *testing.T, f *financeInvoicePostgresFixture, key string, billIDs ...uuid.UUID) (*biz.FinanceInvoice, error) {
	t.Helper()
	ctx := context.Background()
	profiles, err := f.data.db.PartnerInvoiceProfile.Query().All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var profile *ent.PartnerInvoiceProfile
	for _, p := range profiles {
		if p.PartnerID == f.partnerID {
			profile = p
		}
	}
	if profile == nil {
		profile, err = f.data.db.PartnerInvoiceProfile.Create().SetOrganizationID(f.organizationID).SetPartnerID(f.partnerID).SetInvoiceTitle("测试抬头").SetTaxpayerIdentificationNo("TEST").Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
	bills, err := NewFinanceInvoiceRepo(f.data).LoadBills(ctx, f.organizationID, billIDs)
	if err != nil {
		return nil, err
	}
	rule, err := f.data.db.NumberRule.Query().Where(number.OrganizationIDEQ(f.organizationID), number.DocumentTypeEQ(number.DocumentTypeInvoice)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !rule {
		if _, err = f.data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(number.DocumentTypeInvoice).SetPrefix("IN").SetDateFormat(number.DateFormatNone).SetSequenceLength(6).SetResetPolicy(number.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
			return nil, err
		}
	}
	record := biz.FinanceInvoiceRegistrationInput{InvoiceProfileID: profile.ID, InvoiceType: biz.FinanceInvoiceDigitalNormal, TaxInvoiceNo: key, InvoiceDate: financeInvoiceIntegrationDate}
	for _, b := range bills {
		if b.Purpose != biz.FinanceBillNormal || len(b.Lines) == 0 {
			return nil, biz.ErrFinanceInvoiceBillInvalid
		}
		allocation := biz.FinanceInvoiceAllocationInput{BillID: b.ID, ExpectedVersion: b.Version, Amount: *b.InvoiceableAmount}
		for _, line := range b.Lines {
			allocation.Fees = append(allocation.Fees, biz.FinanceInvoiceFeeInput{OrderFeeID: line.OrderFeeID, Amount: line.TotalAmount, TaxAmount: line.TaxAmount})
		}
		record.TotalAmount = record.TotalAmount.Add(allocation.Amount)
		record.Allocations = append(record.Allocations, allocation)
	}
	uc := biz.NewFinanceInvoiceUsecase(NewFinanceInvoiceRepo(f.data), biz.NewOrderConfigUsecase(NewOrderConfigRepo(f.data)), biz.NewExchangeRateUsecase(NewExchangeRateRepo(f.data), nil), f.data)
	invoices, err := uc.RegisterScoped(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.RegisterFinanceInvoicesInput{IdempotencyKey: key, Records: []biz.FinanceInvoiceRegistrationInput{record}})
	if err != nil {
		return nil, err
	}
	return invoices[0], nil
}

func TestFinanceCorrectionInvoiceEconomicContentPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	t.Run("合并发票整红后只重开有效经济行", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, data)
		a := correctionFinanceFee(t, f, "A", "1000", nil, nil, fee.EconomicEffectINCREASE)
		b := correctionFinanceFee(t, f, "B", "200", nil, nil, fee.EconomicEffectINCREASE)
		old := correctionFinanceBill(t, f, "old-invoice", bill.PurposeNORMAL, a, b)
		inv, err := correctionFinanceInvoice(t, f, "原发票", old.ID)
		if err != nil {
			t.Fatal(err)
		}
		repo := NewFinanceInvoiceRepo(data)

		r := correctionFinanceApprove(t, f, a)
		rev := correctionFinanceFee(t, f, "冲回A", "1000", &a.ID, &r.ID, fee.EconomicEffectDECREASE)
		newFee := correctionFinanceFee(t, f, "新A", "100", &a.ID, &r.ID, fee.EconomicEffectINCREASE)
		reverseBill := correctionFinanceBill(t, f, "reverse-invoice", bill.PurposeCORRECTION_REVERSAL, rev)
		newBill := correctionFinanceBill(t, f, "new-invoice", bill.PurposeNORMAL, newFee)
		if _, err := correctionFinanceInvoice(t, f, "重复替代票", newBill.ID); !errors.Is(err, biz.ErrFinanceInvoiceBillInvalid) {
			t.Fatalf("原票未红冲应阻止替代重复票: %v", err)
		}
		if _, err := correctionFinanceInvoice(t, f, "冲回普通票", reverseBill.ID); !errors.Is(err, biz.ErrFinanceInvoiceBillInvalid) {
			t.Fatalf("反向账单不可普通开票: %v", err)
		}
		if _, err = repo.RedFlush(ctx, f.organizationID, inv.ID, f.actorID, inv.Version, "红票", financeInvoiceIntegrationDate, "更正重开", &biz.AuditEvent{Action: "test.red", Result: "success"}); err != nil {
			t.Fatal(err)
		}
		reopened, err := correctionFinanceInvoice(t, f, "旧账重开", old.ID)
		if err != nil {
			t.Fatal(err)
		}
		newInvoice, err := correctionFinanceInvoice(t, f, "新账开票", newBill.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reopened.TotalAmount.Equal(decimal.NewFromInt(200)) || !newInvoice.TotalAmount.Equal(decimal.NewFromInt(100)) {
			t.Fatalf("重开应200+100: %s %s", reopened.TotalAmount, newInvoice.TotalAmount)
		}
		current, err := data.db.FinanceBill.Get(ctx, old.ID)
		if err != nil || current.TotalAmount != "1200.00000000" {
			t.Fatal("原账历史额丢失", err)
		}
	})
	t.Run("更正后来源不可重复登记", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, data)
		a := correctionFinanceFee(t, f, "A", "1000", nil, nil, fee.EconomicEffectINCREASE)
		old := correctionFinanceBill(t, f, "registered-original", bill.PurposeNORMAL, a)
		inv, err := correctionFinanceInvoice(t, f, "原登记", old.ID)
		if err != nil {
			t.Fatal(err)
		}
		correctionFinanceApprove(t, f, a)
		if _, err := correctionFinanceInvoice(t, f, "更正后重复登记", old.ID); err != biz.ErrFinanceInvoiceInvalidArgument && !errors.Is(err, biz.ErrFinanceInvoiceBillInvalid) {
			t.Fatalf("失效经济来源不得再次登记: %v", err)
		}
		current, err := NewFinanceInvoiceRepo(data).Get(ctx, f.organizationID, inv.ID)
		if err != nil || !current.TotalAmount.Equal(decimal.NewFromInt(1000)) || current.Status != biz.FinanceInvoiceIssued {
			t.Fatalf("原票历史必须保留: %+v %v", current, err)
		}
	})
}

func TestFinanceCorrectionSuccessiveCashCoveragePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newFinanceInvoicePostgresFixture(t, data)
	ctx := context.Background()
	for _, kind := range []number.DocumentType{number.DocumentTypeNetting, number.DocumentTypeWriteOff} {
		if _, err := data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(kind).SetPrefix("T").SetDateFormat(number.DateFormatNone).SetSequenceLength(4).SetResetPolicy(number.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
	}
	a := correctionFinanceFee(t, f, "A", "1000", nil, nil, fee.EconomicEffectINCREASE)
	old := correctionFinanceBill(t, f, "root", bill.PurposeNORMAL, a)
	addTestBillInvoiceExemption(t, data, old, f.actorID)
	cash500 := correctionFinanceCash(t, f, "收500", "500", cash.DirectionRECEIVABLE)
	if _, err := correctionFinanceVerification(t, f, cash500, old, "500"); err != nil {
		t.Fatal(err)
	}
	uc := biz.NewFinanceNettingUsecase(NewFinanceNettingRepo(data), data, nil)
	offset := func(first, second *ent.FinanceBill, key string, expected string) {
		n, err := uc.Create(ctx, f.organizationID, f.actorID, biz.CreateFinanceNettingInput{IdempotencyKey: key, Bills: []biz.FinanceNettingBillVersion{{BillID: first.ID, ExpectedVersion: 1}, {BillID: second.ID, ExpectedVersion: 1}}})
		if err != nil {
			t.Fatal(err)
		}
		if !n.Amount.Equal(decimal.RequireFromString(expected)) {
			t.Fatalf("%s抵销=%s期望%s", key, n.Amount, expected)
		}
		if _, err = uc.Confirm(ctx, []uuid.UUID{f.organizationID}, f.actorID, n.ID, n.Version); err != nil {
			t.Fatal(err)
		}
	}
	r1 := correctionFinanceApprove(t, f, a)
	rev1 := correctionFinanceFee(t, f, "冲回1", "1000", &a.ID, &r1.ID, fee.EconomicEffectDECREASE)
	a800 := correctionFinanceFee(t, f, "A800", "800", &a.ID, &r1.ID, fee.EconomicEffectINCREASE)
	revBill1 := correctionFinanceBill(t, f, "reverse1", bill.PurposeCORRECTION_REVERSAL, rev1)
	bill800 := correctionFinanceBill(t, f, "replacement800", bill.PurposeNORMAL, a800)
	offset(old, revBill1, "旧未结清偿", "500")
	offset(revBill1, bill800, "覆盖500继承", "500")
	r2 := correctionFinanceApprove(t, f, a800)
	rev2 := correctionFinanceFee(t, f, "冲回2", "800", &a800.ID, &r2.ID, fee.EconomicEffectDECREASE)
	a700 := correctionFinanceFee(t, f, "A700", "700", &a800.ID, &r2.ID, fee.EconomicEffectINCREASE)
	revBill2 := correctionFinanceBill(t, f, "reverse2", bill.PurposeCORRECTION_REVERSAL, rev2)
	bill700 := correctionFinanceBill(t, f, "replacement700", bill.PurposeNORMAL, a700)
	offset(bill800, revBill2, "第二旧未结清偿", "300")
	offset(revBill2, bill700, "第二继承现金", "500")
	projection, err := loadFinanceBillLineSettlements(ctx, data.db, []uuid.UUID{old.ID, revBill1.ID, bill800.ID, revBill2.ID, bill700.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range projection {
		expected := decimal.Zero
		if p.FeeID == a700.ID {
			expected = decimal.NewFromInt(200)
		}
		if !p.RemainingAmount.Equal(expected) {
			t.Fatalf("连续更正fee=%s remaining=%s期望%s", p.FeeID, p.RemainingAmount, expected)
		}
	}
}
