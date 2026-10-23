package data

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func supplementSnapshotFromEnt(item *ent.OrderFeeSupplementRequest) (biz.OrderFeeSupplementFeeSnapshot, error) {
	quantity, err := decimalOf(item.Quantity)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	unitPrice, err := decimalOf(item.UnitPrice)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	totalAmount, err := decimalOf(item.TotalAmount)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	netAmount, err := decimalOf(item.NetAmount)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	taxAmount, err := decimalOf(item.TaxAmount)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	exchangeRate, err := decimalOf(item.ExchangeRate)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	baseCurrencyAmount, err := decimalOf(item.BaseCurrencyAmount)
	if err != nil {
		return biz.OrderFeeSupplementFeeSnapshot{}, err
	}
	snapshot := biz.OrderFeeSupplementFeeSnapshot{
		Direction:             biz.OrderFeeDirection(item.Direction),
		FeeSettingID:          item.FeeSettingID,
		FeeCode:               item.FeeCode,
		FeeName:               item.FeeName,
		FeeNameEN:             item.FeeNameEn,
		SettlementPartyID:     item.SettlementPartyID,
		BillingUnitID:         item.BillingUnitID,
		BillingUnit:           item.BillingUnit,
		TaxableServiceName:    item.TaxableServiceName,
		Quantity:              quantity,
		UnitPrice:             unitPrice,
		TotalAmount:           totalAmount,
		TaxInclusive:          item.TaxInclusive,
		NetAmount:             netAmount,
		TaxAmount:             taxAmount,
		Currency:              item.Currency,
		ExchangeRate:          exchangeRate,
		ExchangeRateSource:    string(item.ExchangeRateSource),
		ExchangeRateDate:      item.ExchangeRateDate,
		ExchangeRateSettingID: item.ExchangeRateSettingID,
		BaseCurrency:          item.BaseCurrency,
		BaseCurrencyAmount:    baseCurrencyAmount,
		ExpenseDate:           item.ExpenseDate,
	}
	if item.TaxRate != nil {
		taxRate, parseErr := decimalOf(*item.TaxRate)
		if parseErr != nil {
			return biz.OrderFeeSupplementFeeSnapshot{}, parseErr
		}
		snapshot.TaxRate = &taxRate
	}
	if item.Note != "" {
		note := item.Note
		snapshot.Note = &note
	}
	return snapshot, nil
}

func supplementRequestToBiz(item *ent.OrderFeeSupplementRequest) (*biz.OrderFeeSupplementRequest, error) {
	snapshot, err := supplementSnapshotFromEnt(item)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFeeSupplementRequest{
		ApprovedFeeReviewEventID:  item.ApprovedFeeReviewEventID,
		ApprovedFeeContentVersion: item.ApprovedFeeContentVersion,
		ID:                        item.ID,
		OrganizationID:            item.OrganizationID,
		OrderID:                   item.OrderID,
		LockBasis:                 biz.OrderFeeSupplementLockBasis(item.LockBasis),
		IdempotencyKey:            item.IdempotencyKey,
		RequestFingerprint:        item.RequestFingerprint,
		Fee:                       snapshot,
		Reason:                    item.Reason,
		RequestedBy:               item.RequestedBy,
		RequestedAt:               item.RequestedAt,
		Status:                    biz.OrderFeeSupplementStatus(item.Status),
		Version:                   item.Version,
		DecidedBy:                 item.DecidedBy,
		DecidedAt:                 item.DecidedAt,
		DecisionReason:            item.DecisionReason,
		CreatedAt:                 item.CreatedAt,
		UpdatedAt:                 item.UpdatedAt,
	}
	if user := item.Edges.RequestedByUser; user != nil {
		result.RequestedByName = user.DisplayName
	}
	if user := item.Edges.DecidedByUser; user != nil {
		name := user.DisplayName
		result.DecidedByName = &name
	}
	if item.BusinessLockGeneration != nil {
		generation := *item.BusinessLockGeneration
		result.BusinessLockGeneration = &generation
	}
	if item.FinancialLockEvidenceVersion != nil {
		version := *item.FinancialLockEvidenceVersion
		result.FinancialLockEvidenceVersion = &version
	}
	if item.FinancialLockEvidenceHash != nil {
		hash := *item.FinancialLockEvidenceHash
		result.FinancialLockEvidenceHash = &hash
	}
	if item.FinancialLockNetAmountSnapshot != nil {
		net, parseErr := decimalOf(*item.FinancialLockNetAmountSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.FinancialLockNetAmount = &net
	}
	return result, nil
}
