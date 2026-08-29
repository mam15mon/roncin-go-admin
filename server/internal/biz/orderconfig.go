package biz

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrNumberRuleNotFound      = errors.NotFound("NUMBER_RULE_NOT_FOUND", "编号规则不存在")
	ErrNumberRuleExists        = errors.Conflict("NUMBER_RULE_EXISTS", "单据类型的编号规则已存在")
	ErrNumberSequenceExhausted = errors.Conflict("NUMBER_SEQUENCE_EXHAUSTED", "当前编号周期的序列已耗尽")
)

type DocumentType string

const (
	DocumentTypeOrder             DocumentType = "order"
	DocumentTypeBill              DocumentType = "bill"
	DocumentTypeBillBatch         DocumentType = "bill_batch"
	DocumentTypeQuotation         DocumentType = "quotation"
	DocumentTypeWriteOff          DocumentType = "write_off"
	DocumentTypeReceiptPayment    DocumentType = "receipt_payment"
	DocumentTypeContract          DocumentType = "contract"
	DocumentTypeInternalReference DocumentType = "internal_reference"
	DocumentTypeCustomerReference DocumentType = "customer_reference"
	DocumentTypeHouseBill         DocumentType = "house_bill"
	DocumentTypeInvoice           DocumentType = "invoice"
	DocumentTypeFreightRate       DocumentType = "freight_rate"
	DocumentTypeCommission        DocumentType = "commission"
)

func (value DocumentType) Valid() bool {
	switch value {
	case DocumentTypeOrder, DocumentTypeBill, DocumentTypeBillBatch, DocumentTypeQuotation, DocumentTypeWriteOff,
		DocumentTypeReceiptPayment, DocumentTypeContract, DocumentTypeInternalReference,
		DocumentTypeCustomerReference, DocumentTypeHouseBill, DocumentTypeInvoice,
		DocumentTypeFreightRate, DocumentTypeCommission:
		return true
	default:
		return false
	}
}

type DateFormat string

const (
	DateFormatYYYYMMDD DateFormat = "yyyyMMdd"
	DateFormatYYYYMM   DateFormat = "yyyyMM"
	DateFormatYYYY     DateFormat = "yyyy"
	DateFormatNone     DateFormat = "none"
)

func (value DateFormat) Valid() bool {
	return value == DateFormatYYYYMMDD || value == DateFormatYYYYMM || value == DateFormatYYYY || value == DateFormatNone
}

type ResetPolicy string

const (
	ResetPolicyDaily   ResetPolicy = "daily"
	ResetPolicyMonthly ResetPolicy = "monthly"
	ResetPolicyYearly  ResetPolicy = "yearly"
	ResetPolicyNever   ResetPolicy = "never"
)

func (value ResetPolicy) Valid() bool {
	return value == ResetPolicyDaily || value == ResetPolicyMonthly || value == ResetPolicyYearly || value == ResetPolicyNever
}

type BusinessType string

const (
	BusinessTypeSE   BusinessType = "SE"
	BusinessTypeSI   BusinessType = "SI"
	BusinessTypeAE   BusinessType = "AE"
	BusinessTypeAI   BusinessType = "AI"
	BusinessTypeLand BusinessType = "LAND"
	BusinessTypeRail BusinessType = "RAIL"
)

func (value BusinessType) Valid() bool {
	switch value {
	case BusinessTypeSE, BusinessTypeSI, BusinessTypeAE, BusinessTypeAI, BusinessTypeLand, BusinessTypeRail:
		return true
	default:
		return false
	}
}

type NumberRule struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	DocumentType   DocumentType
	Prefix         string
	DateFormat     DateFormat
	SequenceLength int
	ResetPolicy    ResetPolicy
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func DefaultNumberRules() []NumberRule {
	return []NumberRule{
		{DocumentType: DocumentTypeOrder, DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeBill, Prefix: "BI", DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeBillBatch, Prefix: "BG", DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeQuotation, Prefix: "QO", DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeWriteOff, Prefix: "WO", DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeReceiptPayment, Prefix: "PR", DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeContract, Prefix: "CT", DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeInternalReference, DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: false},
		{DocumentType: DocumentTypeCustomerReference, DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: false},
		{DocumentType: DocumentTypeHouseBill, DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: false},
		{DocumentType: DocumentTypeInvoice, DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: false},
		{DocumentType: DocumentTypeFreightRate, Prefix: "FR", DateFormat: DateFormatYYYYMM, SequenceLength: 3, ResetPolicy: ResetPolicyMonthly, Enabled: true},
		{DocumentType: DocumentTypeCommission, Prefix: "CM", DateFormat: DateFormatYYYYMMDD, SequenceLength: 5, ResetPolicy: ResetPolicyDaily, Enabled: true},
	}
}

type OrderConfigRepo interface {
	ListNumberRules(context.Context, uuid.UUID) ([]*NumberRule, error)
	CreateNumberRule(context.Context, uuid.UUID, *NumberRule, *AuditEvent) (*NumberRule, error)
	UpdateNumberRule(context.Context, uuid.UUID, uuid.UUID, *NumberRule, *AuditEvent) (*NumberRule, error)
	AllocateNumber(context.Context, uuid.UUID, DocumentType, time.Time) (*NumberRule, int64, error)
}

type OrderConfigUsecase struct {
	repo OrderConfigRepo
	now  func() time.Time
}

func NewOrderConfigUsecase(repo OrderConfigRepo) *OrderConfigUsecase {
	return &OrderConfigUsecase{repo: repo, now: time.Now}
}

func (uc *OrderConfigUsecase) ListNumberRules(ctx context.Context, organizationID uuid.UUID) ([]*NumberRule, error) {
	if organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	return uc.repo.ListNumberRules(ctx, organizationID)
}

func (uc *OrderConfigUsecase) CreateNumberRule(ctx context.Context, organizationID, actorID uuid.UUID, input *NumberRule) (*NumberRule, error) {
	normalized, err := normalizeNumberRule(input, true)
	if err != nil {
		return nil, err
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "number_rule.create",
		Result:         "success",
		Details:        map[string]string{"document_type": string(normalized.DocumentType)},
	}
	created, err := uc.repo.CreateNumberRule(ctx, organizationID, normalized, audit)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *OrderConfigUsecase) UpdateNumberRule(ctx context.Context, organizationID, actorID, id uuid.UUID, input *NumberRule) (*NumberRule, error) {
	if id == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	normalized, err := normalizeNumberRule(input, false)
	if err != nil {
		return nil, err
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "number_rule.update",
		Result:         "success",
		Details:        map[string]string{"number_rule.id": id.String()},
	}
	updated, err := uc.repo.UpdateNumberRule(ctx, organizationID, id, normalized, audit)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (uc *OrderConfigUsecase) NextNumber(ctx context.Context, organizationID uuid.UUID, documentType DocumentType) (string, error) {
	return uc.nextNumber(ctx, organizationID, documentType, "")
}

func (uc *OrderConfigUsecase) nextNumber(ctx context.Context, organizationID uuid.UUID, documentType DocumentType, businessCode string) (string, error) {
	if organizationID == uuid.Nil || !documentType.Valid() {
		return "", ErrMasterDataInvalidArgument
	}
	now := uc.now().UTC()
	rule, sequence, err := uc.repo.AllocateNumber(ctx, organizationID, documentType, now)
	if err != nil {
		return "", err
	}
	return FormatAllocatedNumber(now, rule, sequence, businessCode)
}

// FormatAllocatedNumber 使用已锁定分配的序列值生成最终单据编号。
func FormatAllocatedNumber(at time.Time, rule *NumberRule, sequence int64, businessCode string) (string, error) {
	if rule == nil || sequence < 1 || rule.SequenceLength < 1 {
		return "", ErrMasterDataInvalidArgument
	}
	sequenceText := strconv.FormatInt(sequence, 10)
	if len(sequenceText) > rule.SequenceLength {
		return "", ErrNumberSequenceExhausted
	}
	return rule.Prefix + businessCode + formatNumberDate(at.UTC(), rule.DateFormat) + fmt.Sprintf("%0*d", rule.SequenceLength, sequence), nil
}

func (uc *OrderConfigUsecase) NextOrderNumber(ctx context.Context, organizationID uuid.UUID, businessType OrderBusinessType) (string, error) {
	if businessType != OrderBusinessSE {
		return "", ErrMasterDataInvalidArgument
	}
	return uc.nextNumber(ctx, organizationID, DocumentTypeOrder, string(businessType))
}

func normalizeNumberRule(input *NumberRule, creating bool) (*NumberRule, error) {
	if input == nil {
		return nil, ErrMasterDataInvalidArgument
	}
	output := *input
	output.Prefix = strings.ToUpper(strings.TrimSpace(output.Prefix))
	if (creating && !output.DocumentType.Valid()) || !output.DateFormat.Valid() || !output.ResetPolicy.Valid() || output.SequenceLength < 1 || output.SequenceLength > 12 || utf8.RuneCountInString(output.Prefix) > 32 {
		return nil, ErrMasterDataInvalidArgument
	}
	return &output, nil
}

func NumberPeriodKey(now time.Time, policy ResetPolicy) string {
	switch policy {
	case ResetPolicyMonthly:
		return now.Format("200601")
	case ResetPolicyYearly:
		return now.Format("2006")
	case ResetPolicyNever:
		return "all"
	default:
		return now.Format("20060102")
	}
}

func formatNumberDate(now time.Time, format DateFormat) string {
	switch format {
	case DateFormatYYYYMM:
		return now.Format("200601")
	case DateFormatYYYY:
		return now.Format("2006")
	case DateFormatNone:
		return ""
	default:
		return now.Format("20060102")
	}
}

var _ OrderConfigRepo = (OrderConfigRepo)(nil)
