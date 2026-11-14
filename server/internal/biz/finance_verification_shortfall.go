package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateWithCashflow 将财务明确填写的差额流水及其核销合并为一个原子提交。
// 空 CashflowID 仅在此入口表示本次新增流水；金额必须完整分配，不按实时余额扩大差额。
func (u *VerificationUsecase) CreateWithCashflow(ctx context.Context, org, actor uuid.UUID, in CreateVerificationInput, cashflowInput CreateFinanceCashflowInput, cashflowUsecase *FinanceCashflowUsecase, canOverrideExchangeRate bool) (*FinanceVerification, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if u.transactor == nil || cashflowUsecase == nil || org == uuid.Nil || actor == uuid.Nil || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 || len(in.Allocations) == 0 || len(in.Allocations) > 500 {
		return nil, ErrVerificationInvalid
	}
	allocated := decimal.Zero
	for _, allocation := range in.Allocations {
		if allocation == nil || allocation.BillID == uuid.Nil || !allocation.Amount.IsPositive() || allocation.Amount.Exponent() < -8 {
			return nil, ErrVerificationInvalid
		}
		if allocation.CashflowID == uuid.Nil {
			allocated = allocated.Add(allocation.Amount)
		}
	}
	if !allocated.IsPositive() || !allocated.Equal(cashflowInput.Amount) {
		return nil, ErrVerificationBalance
	}
	fingerprint := sha256.Sum256([]byte("verification-shortfall:" + in.IdempotencyKey))
	cashflowInput.IdempotencyKey = "verification-shortfall:" + hex.EncodeToString(fingerprint[:])
	var result *FinanceVerification
	err := u.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Create 优先按意图重放原流水，不依赖当下汇率配置。已有成功核销重试不再次确认流水。
		cashflow, err := cashflowUsecase.Create(txCtx, org, actor, cashflowInput, canOverrideExchangeRate)
		if err != nil {
			return err
		}
		mapped := in
		mapped.Allocations = make([]*VerificationAllocation, 0, len(in.Allocations))
		for _, allocation := range in.Allocations {
			copy := *allocation
			if copy.CashflowID == uuid.Nil {
				copy.CashflowID = cashflow.ID
			}
			mapped.Allocations = append(mapped.Allocations, &copy)
		}
		old, err := u.repo.GetByKey(txCtx, org, in.IdempotencyKey)
		if err != nil {
			return err
		}
		if old != nil {
			mapped.Note = normalizedOptionalFinanceString(mapped.Note)
			if !sameVerificationIntent(old, mapped) {
				return ErrVerificationIdempotency
			}
			result = old
			return nil
		}
		if cashflow.Status != FinanceCashflowDraft {
			return ErrFinanceCashflowInvalidTransition
		}
		if _, err := cashflowUsecase.Confirm(txCtx, org, actor, cashflow.ID, cashflow.Version); err != nil {
			return err
		}
		result, err = u.create(txCtx, org, actor, mapped, false)
		return err
	})
	if err != nil {
		// 并发同键插入的唯一约束失败会使事务回滚；只在原组合已完整提交时回放。
		old, lookupErr := u.repo.GetByKey(ctx, org, in.IdempotencyKey)
		if lookupErr == nil && old != nil {
			cashflow, cashErr := cashflowUsecase.repo.GetByIdempotencyKey(ctx, org, cashflowInput.IdempotencyKey)
			normalized := normalizeCreateFinanceCashflowInput(cashflowInput)
			if cashErr == nil && cashflow != nil {
				normalized.BaseCurrency = cashflow.BaseCurrency
				mapped := in
				mapped.Note = normalizedOptionalFinanceString(mapped.Note)
				mapped.Allocations = make([]*VerificationAllocation, 0, len(in.Allocations))
				for _, allocation := range in.Allocations {
					copy := *allocation
					if copy.CashflowID == uuid.Nil {
						copy.CashflowID = cashflow.ID
					}
					mapped.Allocations = append(mapped.Allocations, &copy)
				}
				if sameFinanceCashflowIntent(cashflow, normalized) && sameVerificationIntent(old, mapped) {
					return old, nil
				}
			}
		}
		return nil, err
	}
	return u.repo.Get(ctx, org, result.ID)
}
