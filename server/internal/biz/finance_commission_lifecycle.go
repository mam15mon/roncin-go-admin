package biz

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CalculateCommissionAmount 根据规则口径计算提成，亏损时按零基数计提。
func CalculateCommissionAmount(realizedRevenue, realizedProfit, ratePercent decimal.Decimal, basis CommissionCalculationBasis) (decimal.Decimal, decimal.Decimal, error) {
	base := realizedProfit
	if basis == CommissionBasisRealizedRevenue {
		base = realizedRevenue
	} else if basis != CommissionBasisRealizedProfit {
		return decimal.Zero, decimal.Zero, ErrCommissionRuleInvalid
	}
	if base.IsNegative() {
		base = decimal.Zero
	}
	return base.Round(8), base.Mul(ratePercent).Div(decimal.NewFromInt(100)).Round(8), nil
}

// CalculateCommissionLine 按单个订单计算本次核销收入对应的成本、已实现毛利和提成。
func CalculateCommissionLine(realizedRevenue, totalReceivable, totalPayable, ratePercent decimal.Decimal, basis CommissionCalculationBasis) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	if !realizedRevenue.IsPositive() || !totalReceivable.IsPositive() || totalPayable.IsNegative() {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, ErrCommissionSource
	}
	allocatedCost := realizedRevenue.Mul(totalPayable).Div(totalReceivable).Round(8)
	realizedProfit := realizedRevenue.Sub(allocatedCost).Round(8)
	base, amount, err := CalculateCommissionAmount(realizedRevenue, realizedProfit, ratePercent, basis)
	if err != nil {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	return allocatedCost, realizedProfit, base, amount, nil
}

// sameCommissionCreateIntent 判断幂等重放请求与已存在提成是否语义一致：
// 来源（核销/对冲二选一）、员工、人员身份与备注都必须相同；规则由服务端按
// 来源日期解析，不属于客户端意图的一部分。
func sameCommissionCreateIntent(old *FinanceCommission, in CreateCommissionInput) bool {
	return old.VerificationID == in.VerificationID && old.NettingID == in.NettingID && old.EmployeeID == in.EmployeeID && old.PersonnelRole == in.PersonnelRole && stringPointersEqual(old.Note, in.Note)
}

func (u *CommissionUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateCommissionInput) (*FinanceCommission, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || !validCommissionSource(in.VerificationID, in.NettingID) || in.EmployeeID == uuid.Nil || !validCommissionPersonnelRole(in.PersonnelRole) || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return nil, ErrCommissionInvalid
	}
	if u.transactor == nil {
		return nil, ErrCommissionInvalid
	}
	if old, err := u.repo.GetByKey(ctx, org, in.IdempotencyKey); err != nil {
		return nil, err
	} else if old != nil {
		if !sameCommissionCreateIntent(old, in) {
			return nil, ErrCommissionDuplicate
		}
		return old, nil
	}
	id := uuid.Must(uuid.NewV7())
	commissionNo, err := u.config.NextNumber(ctx, org, DocumentTypeCommission)
	if err != nil {
		return nil, err
	}
	c := &FinanceCommission{ID: id, OrganizationID: org, CommissionNo: commissionNo, IdempotencyKey: in.IdempotencyKey, VerificationID: in.VerificationID, NettingID: in.NettingID, EmployeeID: in.EmployeeID, PersonnelRole: in.PersonnelRole, Status: CommissionDraft, Note: in.Note, Version: 1}
	// 生成上下文读取与提成写入在同一共享事务内完成；CNY 快照不依赖预览结果，
	// 按事务内固化（原币记账恒等口径，无需再解析外部汇率）。
	err = u.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		generation, transactionErr := u.repo.GetGenerationContext(txCtx, org, in.VerificationID, in.NettingID)
		if transactionErr != nil {
			return transactionErr
		}
		snapshot, transactionErr := ResolveCommissionCNYRate(generation.BaseCurrency, generation.CommissionDate, decimal.Zero)
		if transactionErr != nil {
			return transactionErr
		}
		return u.repo.Create(txCtx, org, c, snapshot, commissionCreateAudit(org, actor, id, snapshot))
	})
	if err == nil {
		return u.repo.Get(ctx, org, c.ID)
	}
	// 并发重试可能在预查后命中幂等唯一索引；仅在请求语义一致时重放原结果。
	old, lookupErr := u.repo.GetByKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && sameCommissionCreateIntent(old, in) {
		return old, nil
	}
	return nil, err
}

func commissionCreateAudit(org, actor, id uuid.UUID, snapshot *CommissionCNYSnapshot) *AuditEvent {
	event := commissionAudit(org, actor, id, "finance.commission.create")
	details := map[string]string{
		"commission_date":   snapshot.CommissionDate,
		"cny.exchange_rate": snapshot.ExchangeRate.StringFixed(8),
		"cny.rate_date":     snapshot.ExchangeRateDate,
		"cny.source":        snapshot.ExchangeRateSource,
	}
	if snapshot.ExchangeRateSettingID != nil {
		details["cny.setting_id"] = snapshot.ExchangeRateSettingID.String()
	}
	event.Details = details
	return event
}

func (u *CommissionUsecase) Confirm(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionConfirmed, "")
}
func (u *CommissionUsecase) MarkPaid(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionPaid, "")
}
func (u *CommissionUsecase) Cancel(ctx context.Context, org, actor, id uuid.UUID, version uint64, reason string) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionCancelled, strings.TrimSpace(reason))
}

func (u *CommissionUsecase) CreateAdjustment(ctx context.Context, org, actor uuid.UUID, in CreateCommissionAdjustmentInput) (*FinanceCommissionAdjustment, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || in.CommissionID == uuid.Nil || in.OrderID == uuid.Nil ||
		(in.Direction != CommissionAdjustmentIncrease && in.Direction != CommissionAdjustmentDecrease) ||
		!in.Amount.IsPositive() || !totalAmountPattern.MatchString(in.Amount.String()) || in.Reason == "" ||
		utf8.RuneCountInString(in.Reason) > 500 || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 ||
		(in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return nil, ErrCommissionAdjustmentInvalid
	}
	if old, err := u.repo.GetAdjustmentByKey(ctx, org, in.IdempotencyKey); err != nil {
		return nil, err
	} else if old != nil {
		if !sameCommissionAdjustmentIntent(old, in) {
			return nil, ErrCommissionAdjustmentInvalid
		}
		return old, nil
	}
	item := &FinanceCommissionAdjustment{
		ID: uuid.Must(uuid.NewV7()), OrganizationID: org, CommissionID: in.CommissionID, OrderID: in.OrderID,
		IdempotencyKey: in.IdempotencyKey, Direction: in.Direction, SourceType: CommissionAdjustmentSourceManual, Status: CommissionDraft,
		Amount: in.Amount.Round(8), Reason: in.Reason, Note: in.Note, Version: 1,
	}
	created, err := u.repo.CreateAdjustment(ctx, org, actor, item, commissionAdjustmentAudit(org, actor, item.ID, "finance.commission_adjustment.create"))
	if err == nil {
		return created, nil
	}
	old, lookupErr := u.repo.GetAdjustmentByKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && sameCommissionAdjustmentIntent(old, in) {
		return old, nil
	}
	return nil, err
}

func sameCommissionAdjustmentIntent(old *FinanceCommissionAdjustment, in CreateCommissionAdjustmentInput) bool {
	return old.CommissionID == in.CommissionID && old.OrderID == in.OrderID && old.Direction == in.Direction &&
		old.Amount.Equal(in.Amount.Round(8)) && old.Reason == in.Reason && stringPointersEqual(old.Note, in.Note)
}

func (u *CommissionUsecase) ConfirmAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionConfirmed, "")
}

func (u *CommissionUsecase) MarkAdjustmentPaid(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionPaid, "")
}

func (u *CommissionUsecase) CancelAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64, reason string) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionCancelled, strings.TrimSpace(reason))
}

func (u *CommissionUsecase) transitionAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64, target CommissionStatus, reason string) (*FinanceCommissionAdjustment, error) {
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || version == 0 ||
		(target == CommissionCancelled && (reason == "" || utf8.RuneCountInString(reason) > 500)) {
		return nil, ErrCommissionAdjustmentInvalid
	}
	return u.repo.TransitionAdjustment(ctx, org, id, actor, version, target, reason, commissionAdjustmentAudit(org, actor, id, "finance.commission_adjustment."+strings.ToLower(string(target))))
}
func (u *CommissionUsecase) transition(ctx context.Context, org, actor, id uuid.UUID, version uint64, target CommissionStatus, reason string) (*FinanceCommission, error) {
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || version == 0 || (target == CommissionCancelled && (reason == "" || utf8.RuneCountInString(reason) > 500)) {
		return nil, ErrCommissionInvalid
	}
	return u.repo.Transition(ctx, org, id, actor, version, target, reason, commissionAudit(org, actor, id, "finance.commission."+strings.ToLower(string(target))))
}
