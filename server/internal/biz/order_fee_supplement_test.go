package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestOrderFeeSupplementStatusStateMachine(t *testing.T) {
	if OrderFeeSupplementPending.Terminal() {
		t.Fatalf("PENDING 不应是终态")
	}
	for _, terminal := range []OrderFeeSupplementStatus{OrderFeeSupplementApproved, OrderFeeSupplementRejected, OrderFeeSupplementWithdrawn} {
		if !terminal.Terminal() {
			t.Fatalf("%s 应是终态", terminal)
		}
		if !OrderFeeSupplementPending.CanTransitionTo(terminal) {
			t.Fatalf("PENDING 应允许进入终态 %s", terminal)
		}
		if terminal.CanTransitionTo(terminal) || terminal.CanTransitionTo(OrderFeeSupplementPending) {
			t.Fatalf("终态 %s 不允许任何后续流转", terminal)
		}
		for _, other := range []OrderFeeSupplementStatus{OrderFeeSupplementApproved, OrderFeeSupplementRejected, OrderFeeSupplementWithdrawn} {
			if terminal.CanTransitionTo(other) {
				t.Fatalf("终态 %s 不允许流转到 %s", terminal, other)
			}
		}
	}
	if OrderFeeSupplementPending.CanTransitionTo(OrderFeeSupplementPending) {
		t.Fatalf("PENDING 不允许自流转")
	}
	var invalid OrderFeeSupplementStatus
	if invalid.CanTransitionTo(OrderFeeSupplementApproved) || invalid.Valid() {
		t.Fatalf("未登记状态不允许流转且不应通过取值校验")
	}
	if !OrderFeeSupplementPending.Valid() || !OrderFeeSupplementWithdrawn.Valid() {
		t.Fatalf("已登记状态应通过取值校验")
	}
}

func TestOrderFeeSupplementLockBasisValid(t *testing.T) {
	for _, basis := range []OrderFeeSupplementLockBasis{SupplementLockBasisBusiness, SupplementLockBasisFinancial, SupplementLockBasisBoth} {
		if !basis.Valid() {
			t.Fatalf("锁依据 %s 应通过取值校验", basis)
		}
	}
	if OrderFeeSupplementLockBasis("RECEIVABLE").Valid() {
		t.Fatalf("未登记锁依据不应通过取值校验")
	}
}

func newOrderFeeSupplementRequestFixture(basis OrderFeeSupplementLockBasis) *OrderFeeSupplementRequest {
	generation := uint64(3)
	netAmount := decimal.RequireFromString("120.00000000")
	version := "FINANCIAL_LOCK_EVIDENCE_V1"
	hash := "a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8"
	request := &OrderFeeSupplementRequest{
		ID:                uuid.New(),
		OrganizationID:    uuid.New(),
		OrderID:           uuid.New(),
		LockBasis:         basis,
		IdempotencyKey:    "supp-20260918-0001",
		RequestFingerprint: "fp-v1:demo",
		Reason:            "漏录拖车成本",
		RequestedBy:       uuid.New(),
		Status:            OrderFeeSupplementPending,
		Fee: OrderFeeSupplementFeeSnapshot{
			Direction:   OrderFeePayable,
			FeeCode:     "TRUCK",
			FeeName:     "拖车费",
			SettlementPartyID: uuid.New(),
			BillingUnit: "CTR",
			Quantity:    decimal.RequireFromString("1.0000"),
			UnitPrice:   decimal.RequireFromString("120.0000"),
			TotalAmount: decimal.RequireFromString("120.00000000"),
			NetAmount:   decimal.RequireFromString("120.00000000"),
			TaxAmount:   decimal.RequireFromString("0.00000000"),
			Currency:    "CNY",
			ExchangeRate: decimal.RequireFromString("1.00000000"),
			BaseCurrency: "CNY",
			BaseCurrencyAmount: decimal.RequireFromString("120.00000000"),
			ExpenseDate: "2026-09-18",
		},
	}
	switch basis {
	case SupplementLockBasisBusiness:
		request.BusinessLockGeneration = &generation
	case SupplementLockBasisFinancial:
		request.FinancialLockEvidenceVersion = &version
		request.FinancialLockEvidenceHash = &hash
		request.FinancialLockNetAmount = &netAmount
	case SupplementLockBasisBoth:
		request.BusinessLockGeneration = &generation
		request.FinancialLockEvidenceVersion = &version
		request.FinancialLockEvidenceHash = &hash
		request.FinancialLockNetAmount = &netAmount
	}
	return request
}

func TestOrderFeeSupplementValidateLockBasisEvidence(t *testing.T) {
	for _, basis := range []OrderFeeSupplementLockBasis{SupplementLockBasisBusiness, SupplementLockBasisFinancial, SupplementLockBasisBoth} {
		if err := newOrderFeeSupplementRequestFixture(basis).ValidateLockBasisEvidence(); err != nil {
			t.Fatalf("%s 合法组合不应报错: %v", basis, err)
		}
	}

	zeroNet := decimal.RequireFromString("0.00000000")
	negativeNet := decimal.RequireFromString("-1.00000000")
	cases := []struct {
		name   string
		basis  OrderFeeSupplementLockBasis
		mutate func(*OrderFeeSupplementRequest)
	}{
		{"BUSINESS 缺少业务锁代次", SupplementLockBasisBusiness, func(r *OrderFeeSupplementRequest) {
			r.BusinessLockGeneration = nil
		}},
		{"BUSINESS 业务锁代次为零", SupplementLockBasisBusiness, func(r *OrderFeeSupplementRequest) {
			zero := uint64(0)
			r.BusinessLockGeneration = &zero
		}},
		{"BUSINESS 不得携带财务证据", SupplementLockBasisBusiness, func(r *OrderFeeSupplementRequest) {
			net := decimal.RequireFromString("10.00000000")
			version := "FINANCIAL_LOCK_EVIDENCE_V1"
			r.FinancialLockEvidenceVersion = &version
			r.FinancialLockEvidenceHash = &version
			r.FinancialLockNetAmount = &net
		}},
		{"FINANCIAL 缺少证据哈希", SupplementLockBasisFinancial, func(r *OrderFeeSupplementRequest) {
			r.FinancialLockEvidenceHash = nil
		}},
		{"FINANCIAL 缺少证据版本", SupplementLockBasisFinancial, func(r *OrderFeeSupplementRequest) {
			r.FinancialLockEvidenceVersion = nil
		}},
		{"FINANCIAL 净额为零", SupplementLockBasisFinancial, func(r *OrderFeeSupplementRequest) {
			r.FinancialLockNetAmount = &zeroNet
		}},
		{"FINANCIAL 净额为负", SupplementLockBasisFinancial, func(r *OrderFeeSupplementRequest) {
			r.FinancialLockNetAmount = &negativeNet
		}},
		{"FINANCIAL 不得携带业务锁代次", SupplementLockBasisFinancial, func(r *OrderFeeSupplementRequest) {
			generation := uint64(1)
			r.BusinessLockGeneration = &generation
		}},
		{"BOTH 缺少业务锁代次", SupplementLockBasisBoth, func(r *OrderFeeSupplementRequest) {
			r.BusinessLockGeneration = nil
		}},
		{"BOTH 缺少财务证据版本", SupplementLockBasisBoth, func(r *OrderFeeSupplementRequest) {
			r.FinancialLockEvidenceVersion = nil
		}},
		{"BOTH 缺少财务证据哈希", SupplementLockBasisBoth, func(r *OrderFeeSupplementRequest) {
			r.FinancialLockEvidenceHash = nil
		}},
		{"BOTH 缺少净额快照", SupplementLockBasisBoth, func(r *OrderFeeSupplementRequest) {
			r.FinancialLockNetAmount = nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := newOrderFeeSupplementRequestFixture(tc.basis)
			tc.mutate(request)
			if err := request.ValidateLockBasisEvidence(); err == nil {
				t.Fatalf("%s 应校验失败", tc.name)
			}
		})
	}
}

func TestOrderFeeSupplementValidate(t *testing.T) {
	request := newOrderFeeSupplementRequestFixture(SupplementLockBasisBoth)
	if err := request.Validate(); err != nil {
		t.Fatalf("合法申请不应报错: %v", err)
	}

	receivable := newOrderFeeSupplementRequestFixture(SupplementLockBasisBusiness)
	receivable.Fee.Direction = OrderFeeReceivable
	if err := receivable.Validate(); err == nil {
		t.Fatalf("应收方向补录必须在领域边界拒绝")
	}

	nonPositive := newOrderFeeSupplementRequestFixture(SupplementLockBasisBusiness)
	nonPositive.Fee.TotalAmount = decimal.Zero
	if err := nonPositive.Validate(); err == nil {
		t.Fatalf("非正数应付金额必须拒绝")
	}

	badStatus := newOrderFeeSupplementRequestFixture(SupplementLockBasisBusiness)
	badStatus.Status = OrderFeeSupplementApproved
	if err := badStatus.Validate(); err == nil {
		t.Fatalf("创建时申请状态必须是 PENDING")
	}

	noReason := newOrderFeeSupplementRequestFixture(SupplementLockBasisBusiness)
	noReason.Reason = ""
	if err := noReason.Validate(); err == nil {
		t.Fatalf("补录原因必填")
	}
}
