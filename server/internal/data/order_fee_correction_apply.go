package data

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	correctionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
)

func createCorrectionFee(ctx context.Context, client *ent.Client, fee *biz.OrderFee) error {
	builder := client.OrderFee.Create().
		SetID(fee.ID).
		SetOrderID(fee.OrderID).
		SetIdempotencyKey(fee.IdempotencyKey).
		SetDirection(feeent.Direction(fee.Direction)).
		SetFeeCode(fee.FeeCode).
		SetFeeName(fee.FeeName).
		SetSettlementPartyID(fee.SettlementPartyID).
		SetBillingUnit(fee.BillingUnit).
		SetQuantitySource(map[string]string{"mode": "MANUAL", "adopted_quantity": fee.Quantity.String(), "unit": fee.BillingUnit, "reason": "受控补录或更正人工确定数量"}).
		SetQuantity(fee.Quantity.StringFixed(4)).
		SetUnitPrice(fee.UnitPrice.StringFixed(4)).
		SetTotalAmount(fee.TotalAmount.StringFixed(8)).
		SetTaxInclusive(fee.TaxInclusive).
		SetNetAmount(fee.NetAmount.StringFixed(8)).
		SetTaxAmount(fee.TaxAmount.StringFixed(8)).
		SetCurrency(fee.Currency).
		SetExchangeRate(fee.ExchangeRate.StringFixed(8)).
		SetExchangeRateSource(feeent.ExchangeRateSource(fee.ExchangeRateSource)).
		SetExchangeRateDate(fee.ExchangeRateDate).
		SetBaseCurrency(fee.BaseCurrency).
		SetBaseCurrencyAmount(fee.BaseCurrencyAmount.StringFixed(8)).
		SetExpenseDate(fee.ExpenseDate).
		SetEconomicEffect(feeent.EconomicEffect(fee.EconomicEffect)).
		SetAccountingRequired(fee.AccountingRequired).
		SetNillableSourceFeeID(fee.SourceFeeID).
		SetNillableCorrectionRequestID(fee.CorrectionRequestID).
		SetVersion(1)
	if fee.FeeSettingID != nil {
		builder.SetFeeSettingID(*fee.FeeSettingID)
	}
	if fee.FeeNameEN != nil {
		builder.SetFeeNameEn(*fee.FeeNameEN)
	}
	if fee.BillingUnitID != nil {
		builder.SetBillingUnitID(*fee.BillingUnitID)
	}
	if fee.TaxRate != nil {
		builder.SetTaxRate(fee.TaxRate.StringFixed(2))
	}
	if fee.TaxableServiceName != nil {
		builder.SetTaxableServiceName(*fee.TaxableServiceName)
	}
	if fee.ExchangeRateSettingID != nil {
		builder.SetExchangeRateSettingID(*fee.ExchangeRateSettingID)
	}
	if fee.Note != nil {
		builder.SetNote(*fee.Note)
	}
	_, err := builder.Save(ctx)
	return err
}

// Apply 只写来源、更正费用和新批准快照，原账单/核销/开票不变。
func (r *orderFeeCorrectionRepo) Apply(ctx context.Context, request *biz.OrderFeeCorrectionRequest, actor uuid.UUID, comment string) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	if request.SourceFeeID != nil {
		source, err := client.OrderFee.Query().Where(feeent.IDEQ(*request.SourceFeeID), feeent.OrderIDEQ(request.OrderID)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFeeCorrectionSource, nil)
		}
		if source.SupersededByRequestID != nil || source.EconomicEffect != feeent.EconomicEffectINCREASE || source.Version != request.Before.Version {
			return biz.ErrFeeCorrectionConflict
		}
		if len(source.Edges.FinanceBillLines) > 0 {
			return biz.ErrOrderFeeBillOccupied
		}
		occupied := false
		if _, err = client.OrderFee.UpdateOne(source).SetSupersededByRequestID(request.ID).SetAccountingRequired(occupied).SetVersion(source.Version + 1).Save(ctx); err != nil {
			return err
		}
		reversal := *request.Before
		reversal.ID = uuid.Must(uuid.NewV7())
		reversal.IdempotencyKey = "fee-correction:reversal:" + request.ID.String()
		reversal.EconomicEffect = biz.OrderFeeEconomicDecrease
		reversal.AccountingRequired = occupied
		reversal.SourceFeeID = request.SourceFeeID
		reversal.CorrectionRequestID = &request.ID
		reversal.SupplementRequestID = nil
		reversal.Version = 1
		if err = createCorrectionFee(ctx, client, &reversal); err != nil {
			return err
		}
	}
	if request.After != nil {
		replacement := *request.After
		replacement.IdempotencyKey = "fee-correction:replacement:" + request.ID.String()
		replacement.EconomicEffect = biz.OrderFeeEconomicIncrease
		replacement.AccountingRequired = true
		replacement.SourceFeeID = request.SourceFeeID
		replacement.CorrectionRequestID = &request.ID
		replacement.SupplementRequestID = nil
		replacement.Version = 1
		if err = createCorrectionFee(ctx, client, &replacement); err != nil {
			return err
		}
	}
	if err := client.Order.UpdateOneID(request.OrderID).AddFeeContentVersion(1).Exec(ctx); err != nil {
		return err
	}

	payload, err := json.Marshal(request.ApprovalResult)
	if err != nil {
		return err
	}
	return client.OrderFeeCorrectionRequest.Update().Where(correctionent.IDEQ(request.ID)).SetApprovalResultSnapshot(payload).Exec(ctx)
}
