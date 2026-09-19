package data

import (
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func financeBillToBiz(item *ent.FinanceBill) (*biz.FinanceBill, error) {
	totalAmount, err := decimalOf(item.TotalAmount)
	if err != nil {
		return nil, err
	}
	netAmount, err := decimalOf(item.NetAmount)
	if err != nil {
		return nil, err
	}
	taxAmount, err := decimalOf(item.TaxAmount)
	if err != nil {
		return nil, err
	}
	baseAmount, err := decimalOf(item.BaseCurrencyAmount)
	if err != nil {
		return nil, err
	}
	exchangeRate, err := decimalOf(item.ExchangeRate)
	if err != nil {
		return nil, err
	}
	var estimatedRate, estimatedAmount *decimal.Decimal
	if item.EstimatedInvoiceRate != nil {
		value, parseErr := decimalOf(*item.EstimatedInvoiceRate)
		if parseErr != nil {
			return nil, parseErr
		}
		estimatedRate = &value
	}
	if item.EstimatedInvoiceAmount != nil {
		value, parseErr := decimalOf(*item.EstimatedInvoiceAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		estimatedAmount = &value
	}
	result := &biz.FinanceBill{
		ID: item.ID, OrganizationID: item.OrganizationID, BatchID: item.BatchID, BillNo: item.BillNo, IdempotencyKey: item.IdempotencyKey,
		Direction: biz.OrderFeeDirection(item.Direction), Status: biz.FinanceBillStatus(item.Status),
		SettlementPartyID: item.SettlementPartyID, SettlementPartyName: item.SettlementPartyName, SettlementAccountID: item.SettlementAccountID, SettlementAccountName: item.SettlementAccountName, SettlementAccountHolder: item.SettlementAccountHolder, SettlementBankName: item.SettlementBankName, SettlementBankAccount: item.SettlementBankAccount, SettlementAccountCurrency: item.SettlementAccountCurrency, SettlementSwiftCode: item.SettlementSwiftCode, EstimatedInvoiceCurrency: item.EstimatedInvoiceCurrency, EstimatedInvoiceRate: estimatedRate, EstimatedInvoiceAmount: estimatedAmount,
		Currency: item.Currency, BaseCurrency: item.BaseCurrency, TotalAmount: totalAmount, NetAmount: netAmount, TaxAmount: taxAmount,
		ExchangeRate: exchangeRate, ExchangeRateSource: string(item.ExchangeRateSource), ExchangeRateDate: item.ExchangeRateDate, ExchangeRateSettingID: item.ExchangeRateSettingID,
		BaseCurrencyAmount: baseAmount, FeeCount: item.FeeCount, BillDate: item.BillDate, StatementTitle: item.StatementTitle, PaymentTermsDays: item.PaymentTermsDays, DueDate: item.DueDate, Note: item.Note,
		Version: item.Version, ConfirmedAt: item.ConfirmedAt, ConfirmedBy: item.ConfirmedBy, CancelledAt: item.CancelledAt,
		CancelledBy: item.CancelledBy, CancellationReason: item.CancellationReason, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		Lines: make([]*biz.FinanceBillLine, 0, len(item.Edges.Lines)),
	}
	if organization, edgeErr := item.Edges.OrganizationOrErr(); edgeErr == nil {
		result.OrganizationName = organization.Name
	}
	if batchItem, edgeErr := item.Edges.BatchOrErr(); edgeErr == nil {
		result.BatchNo = batchItem.BatchNo
	}
	for _, line := range item.Edges.Lines {
		converted, convertErr := financeBillLineToBiz(line)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Lines = append(result.Lines, converted)
	}
	return result, nil
}

func financeBillLineToBiz(item *ent.FinanceBillLine) (*biz.FinanceBillLine, error) {
	quantity, err := decimalOf(item.Quantity)
	if err != nil {
		return nil, err
	}
	unitPrice, err := decimalOf(item.UnitPrice)
	if err != nil {
		return nil, err
	}
	totalAmount, err := decimalOf(item.TotalAmount)
	if err != nil {
		return nil, err
	}
	netAmount, err := decimalOf(item.NetAmount)
	if err != nil {
		return nil, err
	}
	taxAmount, err := decimalOf(item.TaxAmount)
	if err != nil {
		return nil, err
	}
	exchangeRate, err := decimalOf(item.ExchangeRate)
	if err != nil {
		return nil, err
	}
	baseAmount, err := decimalOf(item.BaseCurrencyAmount)
	if err != nil {
		return nil, err
	}
	var taxRate *decimal.Decimal
	if item.TaxRate != nil {
		value, parseErr := decimalOf(*item.TaxRate)
		if parseErr != nil {
			return nil, parseErr
		}
		taxRate = &value
	}
	businessType := ""
	if businessOrder, edgeErr := item.Edges.OrderOrErr(); edgeErr == nil {
		businessType = string(businessOrder.BusinessType)
	}
	return &biz.FinanceBillLine{
		ID: item.ID, BillID: item.BillID, OrderFeeID: item.OrderFeeID, OrderID: item.OrderID, OrderNo: item.OrderNo, BusinessType: businessType,
		FeeCode: item.FeeCode, FeeName: item.FeeName, Quantity: quantity, UnitPrice: unitPrice, TotalAmount: totalAmount, NetAmount: netAmount, TaxAmount: taxAmount, TaxRate: taxRate,
		Currency: item.Currency, ExchangeRate: exchangeRate, BaseCurrency: item.BaseCurrency, BaseCurrencyAmount: baseAmount,
		Active: item.Active, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}, nil
}

func financeDecimalString(value *decimal.Decimal, scale int32) *string {
	if value == nil {
		return nil
	}
	formatted := value.StringFixed(scale)
	return &formatted
}

func financeDecimalStringEqual(stored *string, expected *decimal.Decimal, scale int32) bool {
	if stored == nil || expected == nil {
		return stored == nil && expected == nil
	}
	parsed, err := decimal.NewFromString(*stored)
	return err == nil && parsed.StringFixed(scale) == expected.StringFixed(scale)
}
