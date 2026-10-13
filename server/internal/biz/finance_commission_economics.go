package biz

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CommissionEconomicTrigger struct {
	Kind    string
	ID      uuid.UUID
	Version uint64
}
type CommissionSourceQuota struct {
	SourceType                                                            string
	SourceID, AllocationID, BillLineID, EconomicRootID, SettlementPartyID uuid.UUID
	Currency, BaseCurrency                                                string
	EffectiveAt                                                           time.Time
	NativeNetAmount, NativeQuota, BaseQuota                               decimal.Decimal
}
type CommissionEconomicOrderProjection struct {
	OrderID             uuid.UUID
	FeeContentVersion   uint64
	BaseCurrency        string
	Receivable, Payable decimal.Decimal
	Quotas              []CommissionSourceQuota
	Fingerprint         string
}

// ProjectCommissionEconomics 只消费统一清偿投影；旧款沿经济根继承，内部清偿没有收入。
// 退款缓存其原正贡献分摊，反转不能重新分配到后来的真实回款。
func ProjectCommissionEconomics(orderID uuid.UUID, version uint64, allFees, currentFees []*OrderFee, bases map[uuid.UUID]decimal.Decimal, settlements []*FinanceBillLineSettlement) (*CommissionEconomicOrderProjection, error) {
	p := &CommissionEconomicOrderProjection{OrderID: orderID, FeeContentVersion: version}
	fees := make(map[uuid.UUID]*OrderFee, len(allFees))
	for _, f := range allFees {
		if f != nil {
			fees[f.ID] = f
			if p.BaseCurrency == "" {
				p.BaseCurrency = f.BaseCurrency
			}
		}
	}
	for _, f := range currentFees {
		if f != nil {
			fees[f.ID] = f
		}
	}
	roots := make(map[uuid.UUID]uuid.UUID)
	var rootOf func(uuid.UUID, map[uuid.UUID]bool) (uuid.UUID, error)
	rootOf = func(id uuid.UUID, path map[uuid.UUID]bool) (uuid.UUID, error) {
		if r, ok := roots[id]; ok {
			return r, nil
		}
		f := fees[id]
		if f == nil || f.OrderID != orderID || path[id] {
			return uuid.Nil, ErrCommissionSnapshotUnavailable
		}
		path[id] = true
		r := id
		var err error
		if f.SourceFeeID != nil {
			r, err = rootOf(*f.SourceFeeID, path)
			if err != nil {
				return uuid.Nil, err
			}
		}
		delete(path, id)
		roots[id] = r
		return r, nil
	}
	type bucket struct {
		root, party uuid.UUID
		currency    string
	}
	current := make(map[bucket]*OrderFee)
	for _, f := range currentFees {
		if !IsCurrentFee(f) {
			continue
		}
		if p.BaseCurrency == "" {
			p.BaseCurrency = f.BaseCurrency
		}
		if f.BaseCurrency != p.BaseCurrency {
			return nil, ErrCommissionBaseCurrencyMismatch
		}
		base := f.BaseCurrencyAmount
		if b, ok := bases[f.ID]; ok && f.CorrectionRequestID == nil && f.SupplementRequestID == nil {
			base = b
		}
		if f.Direction == OrderFeeReceivable {
			p.Receivable = p.Receivable.Add(base)
			r, err := rootOf(f.ID, map[uuid.UUID]bool{})
			if err != nil {
				return nil, err
			}
			key := bucket{r, f.SettlementPartyID, f.Currency}
			if current[key] != nil {
				return nil, ErrCommissionSnapshotUnavailable
			}
			current[key] = f
		} else {
			p.Payable = p.Payable.Add(base)
		}
	}
	type cash struct {
		key        bucket
		q          CommissionSourceQuota
		amount     decimal.Decimal
		refund     bool
		reversedAt *time.Time
	}
	events := make([]cash, 0)
	for _, line := range settlements {
		f := fees[line.FeeID]
		if f == nil {
			continue
		}
		if f.Direction != OrderFeeReceivable {
			continue
		}
		r, err := rootOf(f.ID, map[uuid.UUID]bool{})
		if err != nil {
			return nil, err
		}
		refund := f.EconomicEffect == OrderFeeEconomicDecrease
		if (!refund && line.Direction != OrderFeeReceivable) || (refund && line.Direction != OrderFeePayable) {
			return nil, ErrCommissionSnapshotUnavailable
		}
		for _, c := range line.Contributions {
			if c.IsCorrectionSettlement {
				continue
			}
			events = append(events, cash{key: bucket{r, f.SettlementPartyID, f.Currency}, q: CommissionSourceQuota{SourceType: c.SourceType, SourceID: c.SourceID, AllocationID: c.AllocationID, BillLineID: line.BillLineID, EconomicRootID: r, SettlementPartyID: f.SettlementPartyID, Currency: f.Currency, BaseCurrency: f.BaseCurrency, EffectiveAt: c.EffectiveAt}, amount: c.Amount, refund: refund, reversedAt: c.ReversedAt})
		}
	}
	less := func(a, b CommissionSourceQuota) bool {
		if !a.EffectiveAt.Equal(b.EffectiveAt) {
			return a.EffectiveAt.Before(b.EffectiveAt)
		}
		if a.SourceType != b.SourceType {
			return a.SourceType < b.SourceType
		}
		if a.SourceID != b.SourceID {
			return a.SourceID.String() < b.SourceID.String()
		}
		if a.AllocationID != b.AllocationID {
			return a.AllocationID.String() < b.AllocationID.String()
		}
		return a.BillLineID.String() < b.BillLineID.String()
	}
	sort.Slice(events, func(i, j int) bool { return less(events[i].q, events[j].q) })
	type action struct {
		idx     int
		at      time.Time
		reverse bool
	}
	actions := make([]action, 0, len(events)*2)
	for i, e := range events {
		if !e.amount.IsPositive() {
			return nil, ErrFinanceSettlementProjection
		}
		actions = append(actions, action{i, e.q.EffectiveAt, false})
		if e.reversedAt != nil {
			actions = append(actions, action{i, *e.reversedAt, true})
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
		return less(events[a.idx].q, events[b.idx].q)
	})
	balances := make(map[int]decimal.Decimal)
	byBucket := make(map[bucket][]int)
	refundShares := make(map[int]map[int]decimal.Decimal)
	for _, a := range actions {
		e := events[a.idx]
		if a.reverse {
			if e.refund {
				for idx, amount := range refundShares[a.idx] {
					balances[idx] = balances[idx].Add(amount)
				}
			} else {
				balances[a.idx] = balances[a.idx].Sub(e.amount)
				if balances[a.idx].IsNegative() {
					return nil, ErrFinanceSettlementProjection
				}
			}
			continue
		}
		if !e.refund {
			balances[a.idx] = e.amount
			byBucket[e.key] = append(byBucket[e.key], a.idx)
			continue
		}
		indices := make([]int, 0)
		total := decimal.Zero
		for _, idx := range byBucket[e.key] {
			if balances[idx].IsPositive() {
				indices = append(indices, idx)
				total = total.Add(balances[idx])
			}
		}
		if e.amount.GreaterThan(total) || len(indices) == 0 {
			return nil, ErrFinanceSettlementProjection
		}
		shares := make(map[int]decimal.Decimal)
		weights := make([]decimal.Decimal, len(indices))
		for i, idx := range indices {
			weights[i] = balances[idx]
		}
		amounts, err := FinanceSettlementDistribute(e.amount, weights)
		if err != nil {
			return nil, err
		}
		for i, idx := range indices {
			balances[idx] = balances[idx].Sub(amounts[i])
			shares[idx] = amounts[i]
		}
		refundShares[a.idx] = shares
	}
	// 当前有效桶按来源发生顺序占额，多付和新来源均不能抢占早先收入。
	keys := make([]bucket, 0, len(byBucket))
	for k := range byBucket {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
	for _, key := range keys {
		f := current[key]
		if f == nil {
			continue
		}
		indices := byBucket[key]
		total := decimal.Zero
		for _, idx := range indices {
			total = total.Add(balances[idx])
		}
		quota := decimal.Min(f.TotalAmount, total)
		if !quota.IsPositive() {
			continue
		}
		base := f.BaseCurrencyAmount
		if b, ok := bases[f.ID]; ok && f.CorrectionRequestID == nil && f.SupplementRequestID == nil {
			base = b
		}
		baseTotal := quota.Mul(base).Div(f.TotalAmount).RoundBank(8)
		remaining := quota
		quotas := make([]CommissionSourceQuota, 0, len(indices))
		weights := make([]decimal.Decimal, 0, len(indices))
		// 原币和本位币量纲不同，统一放大权重以满足分摊器的容量条件。
		// 所有权重乘同一正整数，比例和稳定尾差顺序不变。
		scale := decimal.Max(decimal.NewFromInt(1), baseTotal.Div(quota).Ceil())
		for _, idx := range indices {
			q := events[idx].q
			q.NativeNetAmount = balances[idx]
			q.NativeQuota = decimal.Min(balances[idx], remaining)
			remaining = remaining.Sub(q.NativeQuota)
			quotas = append(quotas, q)
			weights = append(weights, q.NativeQuota.Mul(scale))
		}
		baseShares, err := FinanceSettlementDistribute(baseTotal, weights)
		if err != nil {
			return nil, err
		}
		for i, q := range quotas {
			q.BaseQuota = baseShares[i]
			p.Quotas = append(p.Quotas, q)
		}
	}
	sort.Slice(p.Quotas, func(i, j int) bool {
		a, b := p.Quotas[i], p.Quotas[j]
		if a.EconomicRootID != b.EconomicRootID {
			return a.EconomicRootID.String() < b.EconomicRootID.String()
		}
		return less(a, b)
	})
	raw, err := json.Marshal(struct {
		Fees        []*OrderFee
		Current     []*OrderFee
		Projection  *CommissionEconomicOrderProjection
		Settlements []*FinanceBillLineSettlement
	}{allFees, currentFees, p, settlements})
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	p.Fingerprint = hex.EncodeToString(digest[:])
	return p, nil
}

func (p *CommissionEconomicOrderProjection) SourceRevenue(kind string, id uuid.UUID) decimal.Decimal {
	v := decimal.Zero
	for _, q := range p.Quotas {
		if q.SourceType == kind && q.SourceID == id {
			v = v.Add(q.BaseQuota)
		}
	}
	return v.Round(8)
}

// CommissionEconomicTarget 保留原计算版本和逐步Round(8)；零额度无需除零。
func CommissionEconomicTarget(version string, basis CommissionCalculationBasis, realized, receivable, payable, rate decimal.Decimal) (decimal.Decimal, error) {
	if version != CommissionCalculationVersion {
		return decimal.Zero, ErrCommissionSnapshotUnavailable
	}
	if realized.IsNegative() || receivable.IsNegative() || payable.IsNegative() {
		return decimal.Zero, ErrCommissionSource
	}
	if !realized.IsPositive() {
		return decimal.Zero, nil
	}
	_, _, _, amount, err := CalculateCommissionLine(realized, receivable, payable, rate, basis)
	return amount, err
}

// PlanCommissionSystemAdjustment 只对费用系统累计金额对账，人工奖罚不被恢复抹去。
func PlanCommissionSystemAdjustment(original, target, systemPaid, systemPending, lineOther, parentOther, lineDraft, parentDraft decimal.Decimal) (desired, rebuild, excess decimal.Decimal) {
	desired = target.Sub(original)
	if desired.IsNegative() {
		cap := decimal.Max(decimal.Zero, decimal.Min(original.Add(lineOther).Sub(lineDraft), parentOther.Sub(parentDraft)))
		needed := desired.Abs()
		desired = decimal.Min(needed, cap).Neg()
		excess = needed.Sub(desired.Abs())
	}
	if desired.Equal(systemPaid.Add(systemPending)) {
		return desired, decimal.Zero, excess
	}
	return desired, desired.Sub(systemPaid), excess
}
