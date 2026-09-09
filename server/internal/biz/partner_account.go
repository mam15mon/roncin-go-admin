package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerAccountNotFound        = errors.NotFound("PARTNER_ACCOUNT_NOT_FOUND", "结算账户不存在")
	ErrPartnerAccountInvalidArgument = errors.BadRequest("PARTNER_ACCOUNT_INVALID_ARGUMENT", "结算账户字段不合法")
	ErrPartnerAccountDefaultConflict = errors.Conflict("PARTNER_ACCOUNT_DEFAULT_CONFLICT", "同一往来单位、币种和用途只能有一个默认结算账户")
)

type PartnerAccountUsage string

const (
	PartnerAccountUsageReceivable PartnerAccountUsage = "RECEIVABLE"
	PartnerAccountUsagePayable    PartnerAccountUsage = "PAYABLE"
	PartnerAccountUsageBoth       PartnerAccountUsage = "BOTH"
)

func (u PartnerAccountUsage) Valid() bool {
	return u == PartnerAccountUsageReceivable || u == PartnerAccountUsagePayable || u == PartnerAccountUsageBoth
}

func (u PartnerAccountUsage) Supports(direction OrderFeeDirection) bool {
	return (direction == OrderFeeReceivable && (u == PartnerAccountUsageReceivable || u == PartnerAccountUsageBoth)) ||
		(direction == OrderFeePayable && (u == PartnerAccountUsagePayable || u == PartnerAccountUsageBoth))
}

type PartnerAccount struct {
	ID                  uuid.UUID
	PartnerID           uuid.UUID
	Name                string
	AccountHolder       string
	Currency            string
	BankName            string
	AccountNo           string
	SwiftCode           string
	Usage               PartnerAccountUsage
	IsDefaultReceivable bool
	IsDefaultPayable    bool
	Enabled             bool
	Remark              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type PartnerAccountFilter struct {
	Enabled  *bool
	Usage    PartnerAccountUsage
	Currency string
}

type PartnerAccountRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID, PartnerAccountFilter) ([]*PartnerAccount, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *PartnerAccount, *AuditEvent) (*PartnerAccount, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *PartnerAccount, *AuditEvent) (*PartnerAccount, error)
}

type PartnerAccountUsecase struct {
	repo PartnerAccountRepo
}

func NewPartnerAccountUsecase(repo PartnerAccountRepo) *PartnerAccountUsecase {
	return &PartnerAccountUsecase{repo: repo}
}

func (uc *PartnerAccountUsecase) List(ctx context.Context, organizationID, partnerID uuid.UUID, filter PartnerAccountFilter) ([]*PartnerAccount, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil {
		return nil, ErrPartnerAccountInvalidArgument
	}
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	if (filter.Usage != "" && !filter.Usage.Valid()) || (filter.Currency != "" && len(filter.Currency) != 3) {
		return nil, ErrPartnerAccountInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, partnerID, filter)
}

func (uc *PartnerAccountUsecase) Create(ctx context.Context, organizationID, actorID, partnerID uuid.UUID, input *PartnerAccount) (*PartnerAccount, error) {
	normalized, err := normalizePartnerAccount(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Create(ctx, organizationID, partnerID, normalized, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.account.create", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"partner.id": partnerID.String()}})
}

func (uc *PartnerAccountUsecase) Update(ctx context.Context, organizationID, actorID, partnerID, id uuid.UUID, input *PartnerAccount) (*PartnerAccount, error) {
	if id == uuid.Nil {
		return nil, ErrPartnerAccountNotFound
	}
	normalized, err := normalizePartnerAccount(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, partnerID, id, normalized, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.account.update", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"account.id": id.String(), "partner.id": partnerID.String()}})
}

func normalizePartnerAccount(input *PartnerAccount) (*PartnerAccount, error) {
	if input == nil {
		return nil, ErrPartnerAccountInvalidArgument
	}
	output := *input
	output.Name = strings.TrimSpace(output.Name)
	output.AccountHolder = strings.TrimSpace(output.AccountHolder)
	output.Currency = strings.ToUpper(strings.TrimSpace(output.Currency))
	output.BankName = strings.TrimSpace(output.BankName)
	output.AccountNo = strings.TrimSpace(output.AccountNo)
	output.SwiftCode = strings.ToUpper(strings.TrimSpace(output.SwiftCode))
	output.Remark = strings.TrimSpace(output.Remark)
	if output.Name == "" || output.AccountHolder == "" || output.BankName == "" || output.AccountNo == "" || len(output.Currency) != 3 || !output.Usage.Valid() || utf8.RuneCountInString(output.Name) > 200 || utf8.RuneCountInString(output.AccountHolder) > 200 || utf8.RuneCountInString(output.BankName) > 200 || utf8.RuneCountInString(output.AccountNo) > 100 || utf8.RuneCountInString(output.SwiftCode) > 32 || utf8.RuneCountInString(output.Remark) > 500 {
		return nil, ErrPartnerAccountInvalidArgument
	}
	if (!output.Enabled && (output.IsDefaultReceivable || output.IsDefaultPayable)) || (output.IsDefaultReceivable && !output.Usage.Supports(OrderFeeReceivable)) || (output.IsDefaultPayable && !output.Usage.Supports(OrderFeePayable)) {
		return nil, ErrPartnerAccountInvalidArgument
	}
	return &output, nil
}
