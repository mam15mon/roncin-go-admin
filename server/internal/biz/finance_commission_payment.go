package biz

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CommissionPaymentInput 记录财务确认的实际净发放金额及其预览身份。
type CommissionPaymentInput struct {
	PreviewToken, IdempotencyKey string
	ActualAmount                 decimal.Decimal
}
type CommissionPaymentAdjustment struct {
	ID           uuid.UUID
	Version      uint64
	Direction    CommissionAdjustmentDirection
	Amount       decimal.Decimal
	AdjustmentNo string
}
type CommissionPaymentPreview struct {
	CommissionID                                   uuid.UUID
	Version                                        uint64
	BaseCurrency                                   string
	OriginalAmount, AdjustmentAmount, ActualAmount decimal.Decimal
	PreviewToken                                   string
	Adjustments                                    []CommissionPaymentAdjustment
	OffsetSettlement                               bool
}

// CalculateCommissionPayment 区分未执行权益与历史付款，父单未发时不允许已有独立执行事实。
func CalculateCommissionPayment(parent *FinanceCommission, adjustments []*FinanceCommissionAdjustment) (*CommissionPaymentPreview, error) {
	if parent == nil || parent.Status != CommissionConfirmed {
		return nil, ErrCommissionTransition
	}
	result := &CommissionPaymentPreview{CommissionID: parent.ID, Version: parent.Version, BaseCurrency: parent.BaseCurrency, OriginalAmount: parent.CommissionAmount, Adjustments: make([]CommissionPaymentAdjustment, 0)}
	for _, a := range adjustments {
		if a.Status == CommissionPaid {
			return nil, ErrCommissionPaymentEvidence
		}
		if a.Status != CommissionConfirmed {
			continue
		}
		if a.BaseCurrency != parent.BaseCurrency || !a.Amount.IsPositive() {
			return nil, ErrCommissionAdjustmentInvalid
		}
		result.Adjustments = append(result.Adjustments, CommissionPaymentAdjustment{ID: a.ID, Version: a.Version, Direction: a.Direction, Amount: a.Amount, AdjustmentNo: a.AdjustmentNo})
		if a.Direction == CommissionAdjustmentDecrease {
			result.AdjustmentAmount = result.AdjustmentAmount.Sub(a.Amount)
		} else if a.Direction == CommissionAdjustmentIncrease {
			result.AdjustmentAmount = result.AdjustmentAmount.Add(a.Amount)
		} else {
			return nil, ErrCommissionAdjustmentInvalid
		}
	}
	result.ActualAmount = result.OriginalAmount.Add(result.AdjustmentAmount).Round(8)
	if result.ActualAmount.IsNegative() {
		return nil, ErrCommissionAdjustmentExceeds
	}
	result.OffsetSettlement = result.ActualAmount.IsZero()
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(encoded)
	result.PreviewToken = hex.EncodeToString(sum[:])
	return result, nil
}
