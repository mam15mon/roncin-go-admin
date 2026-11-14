package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var ErrFinanceInvoiceExemptionConflict = errors.Conflict("FINANCE_INVOICE_EXEMPTION_CONFLICT", "账单已有有效发票或无需发票标记，请刷新后重试")
var ErrFinanceInvoiceExemptionNotFound = errors.Conflict("FINANCE_INVOICE_EXEMPTION_NOT_FOUND", "账单没有有效的无需发票标记，请刷新后重试")

// FinanceInvoiceExemption 保存免票决定和撤销记录，Active 由 RevokedAt 是否为空推导。
type FinanceInvoiceExemption struct {
	ID, OrganizationID, BillID, MarkedBy uuid.UUID
	Reason                               string
	MarkedByName, RevokedByName          string
	Version                              uint64
	CreatedAt                            time.Time
	RevokedAt                            *time.Time
	RevokedBy                            *uuid.UUID
	RevocationReason                     *string
}

func (uc *FinanceInvoiceUsecase) SetBillInvoiceExemption(ctx context.Context, organizationIDs []uuid.UUID, actorID, billID uuid.UUID, expectedVersion uint64, reason string, revoke bool) error {
	reason = strings.TrimSpace(reason)
	if !validFinanceOrganizationIDs(organizationIDs) || actorID == uuid.Nil || billID == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return ErrFinanceInvoiceInvalidArgument
	}
	action := "finance.invoice.exempt"
	if revoke {
		action = "finance.invoice.exemption_revoke"
	}
	audit := &AuditEvent{UserID: &actorID, Action: action, Result: "success", ResourceType: "finance_bill", ResourceID: billID.String(), Details: map[string]string{"finance_bill.id": billID.String(), "reason": reason}}
	return uc.repo.SetBillInvoiceExemption(ctx, organizationIDs, actorID, billID, expectedVersion, reason, revoke, audit)
}
