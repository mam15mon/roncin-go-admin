package data

import (
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
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
		ID:                 item.ID,
		OrganizationID:     item.OrganizationID,
		OrderID:            item.OrderID,
		LockBasis:          biz.OrderFeeSupplementLockBasis(item.LockBasis),
		IdempotencyKey:     item.IdempotencyKey,
		RequestFingerprint: item.RequestFingerprint,
		Fee:                snapshot,
		Reason:             item.Reason,
		RequestedBy:        item.RequestedBy,
		RequestedAt:        item.RequestedAt,
		Status:             biz.OrderFeeSupplementStatus(item.Status),
		Version:            item.Version,
		DecidedBy:          item.DecidedBy,
		DecidedAt:          item.DecidedAt,
		DecisionReason:     item.DecisionReason,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
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

// commissionImpactLineToBiz 把锁定的父单、订单行与调整聚合为边际计算上下文行：
// 双层余额按「CONFIRMED/PAID 符号化合计 − 其他 DRAFT DECREASE 预留」统计，
// DRAFT INCREASE 不提前扩张余额。
func commissionImpactLineToBiz(parent *ent.FinanceCommission, line *ent.FinanceCommissionLine, adjustments []*ent.FinanceCommissionAdjustment) (*biz.OrderFeeSupplementCommissionLine, error) {
	parentAmount, err := decimalOf(parent.CommissionAmount)
	if err != nil {
		return nil, err
	}
	ratePercent, err := decimalOf(line.RatePercent)
	if err != nil {
		return nil, err
	}
	lineAmount, err := decimalOf(line.CommissionAmount)
	if err != nil {
		return nil, err
	}
	realizedRevenue, err := decimalOf(line.RealizedRevenue)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFeeSupplementCommissionLine{
		CommissionID:           parent.ID,
		CommissionNo:           parent.CommissionNo,
		CommissionStatus:       string(parent.Status),
		CommissionAmount:       parentAmount,
		AdjustmentSequence:     parent.AdjustmentSequence,
		EmployeeID:             parent.EmployeeID,
		EmployeeName:           parent.EmployeeName,
		ParentBaseCurrency:     parent.BaseCurrency,
		LineID:                 line.ID,
		OrderID:                line.OrderID,
		OrderNo:                line.OrderNo,
		CalculationVersion:     parent.CalculationVersion,
		CalculationBasis:       biz.CommissionCalculationBasis(line.CalculationBasis),
		CalculationRatePercent: ratePercent,
		RealizedRevenue:        realizedRevenue,
		LineCommissionAmount:   lineAmount,
		LineBaseCurrency:       line.BaseCurrency,
		SnapshotStatus:         snapshotStatusOrEmpty(line.SnapshotStatus),
	}
	if line.TotalReceivableSnapshot != nil {
		receivable, parseErr := decimalOf(*line.TotalReceivableSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.TotalReceivableSnapshot = &receivable
	}
	if line.TotalPayableSnapshot != nil {
		payable, parseErr := decimalOf(*line.TotalPayableSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.TotalPayableSnapshot = &payable
	}
	parentEffective := parentAmount
	lineEffective := lineAmount
	parentDraftDecrease := decimal.Zero
	lineDraftDecrease := decimal.Zero
	for _, adjustment := range adjustments {
		amount, parseErr := decimalOf(adjustment.Amount)
		if parseErr != nil {
			return nil, parseErr
		}
		isDecrease := adjustment.Direction == commissionadjustmentent.DirectionDECREASE
		switch adjustment.Status {
		case commissionadjustmentent.StatusCONFIRMED, commissionadjustmentent.StatusPAID:
			if isDecrease {
				parentEffective = parentEffective.Sub(amount)
				if adjustment.OrderID == line.OrderID {
					lineEffective = lineEffective.Sub(amount)
				}
			} else {
				parentEffective = parentEffective.Add(amount)
				if adjustment.OrderID == line.OrderID {
					lineEffective = lineEffective.Add(amount)
				}
			}
		case commissionadjustmentent.StatusDRAFT:
			if isDecrease {
				parentDraftDecrease = parentDraftDecrease.Add(amount)
				if adjustment.OrderID == line.OrderID {
					lineDraftDecrease = lineDraftDecrease.Add(amount)
				}
			}
		}
	}
	result.LineEffective = lineEffective.Round(8)
	result.ParentEffective = parentEffective.Round(8)
	result.LineDraftDecrease = lineDraftDecrease.Round(8)
	result.ParentDraftDecrease = parentDraftDecrease.Round(8)
	return result, nil
}
