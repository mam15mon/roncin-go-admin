package biz

import (
	"sort"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrFinanceSettlementProjection = errors.Conflict("FINANCE_SETTLEMENT_PROJECTION_INVALID", "结算分摊与账单行余额不一致，请先处理关联结算")
var ErrFinanceCorrectionSettlementRequired = errors.Conflict("FINANCE_CORRECTION_SETTLEMENT_REQUIRED", "已更正费用仍有原账余额，请先完成更正内部对冲")

type FinanceSettlementLine struct {
	BillID, ID, FeeID uuid.UUID
	Direction         OrderFeeDirection
	TotalAmount       decimal.Decimal
	SupersededAt      *time.Time
}

type FinanceSettlementEvent struct {
	SourceType                     string
	SourceID, AllocationID, BillID uuid.UUID
	SourceFeeID                    *uuid.UUID
	Amount                         decimal.Decimal
	EffectiveAt                    time.Time
	ReversedAt                     *time.Time
	CorrectionRequestID            *uuid.UUID
}

type FinanceSettlementContribution struct {
	CorrectionRequestID    *uuid.UUID
	SourceType             string
	SourceID, AllocationID uuid.UUID
	EffectiveAt            time.Time
	Amount                 decimal.Decimal
	IsCorrectionSettlement bool
	ReversedAt             *time.Time
}

type FinanceBillLineSettlement struct {
	BillID, BillLineID, FeeID                   uuid.UUID
	Direction                                   OrderFeeDirection
	TotalAmount, SettledAmount, RemainingAmount decimal.Decimal
	Contributions                               []FinanceSettlementContribution
}

// ProjectFinanceBillLineSettlements 按真实发生时间重放分摊及反转。
// 反转释放原分摊缓存，不改变随后事件已取得的行归属。
func ProjectFinanceBillLineSettlements(lines []FinanceSettlementLine, events []FinanceSettlementEvent) ([]*FinanceBillLineSettlement, error) {
	lines = append([]FinanceSettlementLine(nil), lines...)
	result := make([]*FinanceBillLineSettlement, 0, len(lines))
	byBill := make(map[uuid.UUID][]int)
	seen := make(map[uuid.UUID]bool)
	sort.Slice(lines, func(i, j int) bool { return lines[i].ID.String() < lines[j].ID.String() })
	for _, line := range lines {
		if line.ID == uuid.Nil || seen[line.ID] || !line.TotalAmount.IsPositive() {
			return nil, ErrFinanceSettlementProjection
		}
		seen[line.ID] = true
		byBill[line.BillID] = append(byBill[line.BillID], len(result))
		result = append(result, &FinanceBillLineSettlement{BillID: line.BillID, BillLineID: line.ID, FeeID: line.FeeID, Direction: line.Direction, TotalAmount: line.TotalAmount, RemainingAmount: line.TotalAmount})
	}
	type action struct {
		event   int
		at      time.Time
		reverse bool
	}
	actions := make([]action, 0, len(events)*2)
	for i, e := range events {
		if e.AllocationID == uuid.Nil || !e.Amount.IsPositive() || e.EffectiveAt.IsZero() || (e.SourceType != "VERIFICATION" && e.SourceType != "NETTING") {
			return nil, ErrFinanceSettlementProjection
		}
		actions = append(actions, action{i, e.EffectiveAt, false})
		if e.ReversedAt != nil {
			if e.ReversedAt.Before(e.EffectiveAt) {
				return nil, ErrFinanceSettlementProjection
			}
			actions = append(actions, action{i, *e.ReversedAt, true})
		}
	}
	sort.SliceStable(actions, func(i, j int) bool {
		a, b := actions[i], actions[j]
		if !a.at.Equal(b.at) {
			return a.at.Before(b.at)
		}
		if a.reverse != b.reverse {
			return !a.reverse
		}
		x, y := events[a.event], events[b.event]
		if x.SourceType != y.SourceType {
			return x.SourceType < y.SourceType
		}
		if x.SourceID != y.SourceID {
			return x.SourceID.String() < y.SourceID.String()
		}
		return x.AllocationID.String() < y.AllocationID.String()
	})
	type share struct {
		line, contribution int
		amount             decimal.Decimal
	}
	cache := make(map[int][]share)
	for _, a := range actions {
		e := events[a.event]
		if a.reverse {
			shares, ok := cache[a.event]
			if !ok {
				return nil, ErrFinanceSettlementProjection
			}
			for _, s := range shares {
				r := result[s.line]
				r.SettledAmount = r.SettledAmount.Sub(s.amount)
				r.RemainingAmount = r.RemainingAmount.Add(s.amount)
				r.Contributions[s.contribution].ReversedAt = e.ReversedAt
			}
			continue
		}
		indices := make([]int, 0)
		total := decimal.Zero
		for _, idx := range byBill[e.BillID] {
			r := result[idx]
			if e.SourceFeeID != nil && r.FeeID != *e.SourceFeeID {
				continue
			}
			if e.SourceFeeID == nil && lines[idx].SupersededAt != nil && !a.at.Before(*lines[idx].SupersededAt) && r.RemainingAmount.IsPositive() {
				return nil, ErrFinanceCorrectionSettlementRequired
			}
			if r.RemainingAmount.IsPositive() {
				indices = append(indices, idx)
				total = total.Add(r.RemainingAmount)
			}
		}
		if len(indices) == 0 || e.Amount.GreaterThan(total) {
			return nil, ErrFinanceSettlementProjection
		}

		balances := make([]decimal.Decimal, 0, len(indices))
		for _, idx := range indices {
			balances = append(balances, result[idx].RemainingAmount)
		}
		amounts, err := FinanceSettlementDistribute(e.Amount, balances)
		if err != nil {
			return nil, err
		}
		remaining := e.Amount
		for i, amount := range amounts {
			if !amount.IsPositive() {
				continue
			}
			idx := indices[i]
			r := result[idx]
			r.SettledAmount = r.SettledAmount.Add(amount)
			r.RemainingAmount = r.RemainingAmount.Sub(amount)
			cache[a.event] = append(cache[a.event], share{idx, len(r.Contributions), amount})
			r.Contributions = append(r.Contributions, FinanceSettlementContribution{SourceType: e.SourceType, SourceID: e.SourceID, AllocationID: e.AllocationID, EffectiveAt: e.EffectiveAt, Amount: amount, CorrectionRequestID: e.CorrectionRequestID, IsCorrectionSettlement: e.CorrectionRequestID != nil})
			remaining = remaining.Sub(amount)
		}

		if !remaining.IsZero() {
			return nil, ErrFinanceSettlementProjection
		}
	}
	return result, nil
}

// FinanceSettlementDistribute 用八位最大余数分摊，调用方先固定输入顺序作为同余数的尾差归属。
func FinanceSettlementDistribute(amount decimal.Decimal, balances []decimal.Decimal) ([]decimal.Decimal, error) {
	total := decimal.Zero
	for _, b := range balances {
		if b.IsNegative() {
			return nil, ErrFinanceSettlementProjection
		}
		total = total.Add(b)
	}
	if amount.IsNegative() || amount.GreaterThan(total) || !amount.Equal(amount.Truncate(8)) {
		return nil, ErrFinanceSettlementProjection
	}
	result := make([]decimal.Decimal, len(balances))
	if amount.IsZero() {
		return result, nil
	}
	if !total.IsPositive() {
		return nil, ErrFinanceSettlementProjection
	}
	type part struct {
		index     int
		remainder decimal.Decimal
	}
	parts := make([]part, 0, len(balances))
	remaining := amount
	for i, b := range balances {
		exact := amount.Mul(b).DivRound(total, 24)
		result[i] = exact.Truncate(8)
		parts = append(parts, part{i, exact.Sub(result[i])})
		remaining = remaining.Sub(result[i])
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].remainder.GreaterThan(parts[j].remainder) })
	tick := decimal.RequireFromString("0.00000001")
	for _, p := range parts {
		if remaining.IsZero() {
			break
		}
		if remaining.LessThan(tick) {
			return nil, ErrFinanceSettlementProjection
		}
		if result[p.index].Add(tick).GreaterThan(balances[p.index]) {
			continue
		}
		result[p.index] = result[p.index].Add(tick)
		remaining = remaining.Sub(tick)
	}
	if !remaining.IsZero() {
		return nil, ErrFinanceSettlementProjection
	}
	return result, nil
}
