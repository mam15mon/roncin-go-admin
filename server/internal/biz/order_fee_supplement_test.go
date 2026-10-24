package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestOrderFeeSupplementStatusValidAndTerminal(t *testing.T) {
	if OrderFeeSupplementPending.Terminal() {
		t.Fatalf("PENDING 不应是终态")
	}
	for _, terminal := range []OrderFeeSupplementStatus{OrderFeeSupplementApproved, OrderFeeSupplementRejected, OrderFeeSupplementWithdrawn} {
		if !terminal.Terminal() {
			t.Fatalf("%s 应是终态", terminal)
		}
	}
	var invalid OrderFeeSupplementStatus
	if invalid.Valid() {
		t.Fatalf("未登记状态不应通过取值校验")
	}
	if !OrderFeeSupplementPending.Valid() || !OrderFeeSupplementWithdrawn.Valid() {
		t.Fatalf("已登记状态应通过取值校验")
	}
}

func TestOrderFeeSupplementLockBasisValid(t *testing.T) {
	for _, basis := range []OrderFeeSupplementLockBasis{SupplementLockBasisBusiness, SupplementLockBasisFinancial, SupplementLockBasisBoth, SupplementLockBasisApproval} {
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
		ID:                 uuid.New(),
		OrganizationID:     uuid.New(),
		OrderID:            uuid.New(),
		LockBasis:          basis,
		IdempotencyKey:     "supp-20260918-0001",
		RequestFingerprint: "fp-v1:demo",
		Reason:             "漏录拖车成本",
		RequestedBy:        uuid.New(),
		Status:             OrderFeeSupplementPending,
		Fee: OrderFeeSupplementFeeSnapshot{
			Direction:          OrderFeePayable,
			FeeCode:            "TRUCK",
			FeeName:            "拖车费",
			SettlementPartyID:  uuid.New(),
			BillingUnit:        "CTR",
			Quantity:           decimal.RequireFromString("1.0000"),
			UnitPrice:          decimal.RequireFromString("120.0000"),
			TotalAmount:        decimal.RequireFromString("120.00000000"),
			NetAmount:          decimal.RequireFromString("120.00000000"),
			TaxAmount:          decimal.RequireFromString("0.00000000"),
			Currency:           "CNY",
			ExchangeRate:       decimal.RequireFromString("1.00000000"),
			BaseCurrency:       "CNY",
			BaseCurrencyAmount: decimal.RequireFromString("120.00000000"),
		},
	}
	switch basis {
	case SupplementLockBasisApproval:
		eventID := uuid.New()
		contentVersion := uint64(1)
		request.ApprovedFeeReviewEventID = &eventID
		request.ApprovedFeeContentVersion = &contentVersion
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
	valid := newOrderFeeSupplementRequestFixture(SupplementLockBasisApproval)
	if err := valid.ValidateLockBasisEvidence(); err != nil {
		t.Fatal(err)
	}
	for _, basis := range []OrderFeeSupplementLockBasis{SupplementLockBasisBusiness, SupplementLockBasisFinancial, SupplementLockBasisBoth} {
		if err := newOrderFeeSupplementRequestFixture(basis).ValidateLockBasisEvidence(); err == nil {
			t.Fatalf("历史锁依据%s不能授权新申请", basis)
		}
	}
	for _, scenario := range []string{"无批准事件", "空批准ID", "无内容版本", "零内容版本"} {
		t.Run(scenario, func(t *testing.T) {
			request := newOrderFeeSupplementRequestFixture(SupplementLockBasisApproval)
			switch scenario {
			case "无批准事件":
				request.ApprovedFeeReviewEventID = nil
			case "空批准ID":
				empty := uuid.Nil
				request.ApprovedFeeReviewEventID = &empty
			case "无内容版本":
				request.ApprovedFeeContentVersion = nil
			case "零内容版本":
				zero := uint64(0)
				request.ApprovedFeeContentVersion = &zero
			}
			if err := request.ValidateLockBasisEvidence(); err == nil {
				t.Fatal("不完整批准依据应拒绝")
			}
		})
	}
}

func TestOrderFeeSupplementValidate(t *testing.T) {
	request := newOrderFeeSupplementRequestFixture(SupplementLockBasisApproval)
	if err := request.Validate(); err != nil {
		t.Fatalf("合法申请不应报错: %v", err)
	}

	receivable := newOrderFeeSupplementRequestFixture(SupplementLockBasisApproval)
	receivable.Fee.Direction = OrderFeeReceivable
	if err := receivable.Validate(); err == nil {
		t.Fatalf("应收方向补录必须在领域边界拒绝")
	}

	nonPositive := newOrderFeeSupplementRequestFixture(SupplementLockBasisApproval)
	nonPositive.Fee.TotalAmount = decimal.Zero
	if err := nonPositive.Validate(); err == nil {
		t.Fatalf("非正数应付金额必须拒绝")
	}

	badStatus := newOrderFeeSupplementRequestFixture(SupplementLockBasisApproval)
	badStatus.Status = OrderFeeSupplementApproved
	if err := badStatus.Validate(); err == nil {
		t.Fatalf("创建时申请状态必须是 PENDING")
	}

	noReason := newOrderFeeSupplementRequestFixture(SupplementLockBasisApproval)
	noReason.Reason = ""
	if err := noReason.Validate(); err == nil {
		t.Fatalf("补录原因必填")
	}
}
