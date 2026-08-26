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
	ErrNumberRuleNotFound            = errors.NotFound("NUMBER_RULE_NOT_FOUND", "编号规则不存在")
	ErrNumberRuleExists              = errors.Conflict("NUMBER_RULE_EXISTS", "单据类型的编号规则已存在")
	ErrStatusTemplateNotFound        = errors.NotFound("STATUS_TEMPLATE_NOT_FOUND", "状态模板不存在")
	ErrStatusTemplateExists          = errors.Conflict("STATUS_TEMPLATE_EXISTS", "状态模板版本已存在")
	ErrStatusTemplateInvalid         = errors.BadRequest("STATUS_TEMPLATE_INVALID", "状态模板不合法")
	ErrStatusTemplateReadOnly        = errors.Forbidden("STATUS_TEMPLATE_READ_ONLY", "状态流转模板为系统内置配置，不允许修改")
	ErrNumberSequenceExhausted       = errors.Conflict("NUMBER_SEQUENCE_EXHAUSTED", "当前编号周期的序列已耗尽")
	ErrStatusTemplateDefaultConflict = errors.Conflict("STATUS_TEMPLATE_DEFAULT_CONFLICT", "默认状态模板设置冲突")
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

type StatusTemplateItem struct {
	ID         uuid.UUID
	Code       string
	Label      string
	SortOrder  int
	Enabled    bool
	ColorToken *string
	System     bool
}

type StatusTemplate struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Name           string
	BusinessType   BusinessType
	Version        int
	IsDefault      bool
	PublishedAt    *time.Time
	Enabled        bool
	Items          []*StatusTemplateItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func DefaultStatusTemplates() []StatusTemplate {
	return []StatusTemplate{
		defaultStatusTemplate("SE_SYSTEM", "海运出口内置状态流转", BusinessTypeSE, []StatusTemplateItem{
			{Code: "BOOKED", Label: "已订舱", ColorToken: statusColorPointer("blue")},
			{Code: "SPACE_ALLOCATED", Label: "已配舱", ColorToken: statusColorPointer("cyan")},
			{Code: "TRUCKING_ARRANGED", Label: "拖车已安排", ColorToken: statusColorPointer("amber")},
			{Code: "WAREHOUSE_DELIVERED", Label: "已送仓", ColorToken: statusColorPointer("amber")},
			{Code: "INVOICE_ISSUED", Label: "已开票", ColorToken: statusColorPointer("blue")},
			{Code: "CUSTOMS_DECLARATION_ARRANGED", Label: "报关已安排", ColorToken: statusColorPointer("violet")},
		}),
		defaultStatusTemplate("SI_SYSTEM", "海运进口内置状态流转", BusinessTypeSI, []StatusTemplateItem{
			{Code: "TRUCKING_ARRANGED", Label: "拖车已安排", ColorToken: statusColorPointer("amber")},
			{Code: "CUSTOMS_DECLARATION_ARRANGED", Label: "报关已安排", ColorToken: statusColorPointer("violet")},
			{Code: "BILL_EXCHANGED", Label: "已换单", ColorToken: statusColorPointer("teal")},
			{Code: "INSPECTION_ARRANGED", Label: "报检已安排", ColorToken: statusColorPointer("violet")},
		}),
		defaultStatusTemplate("AE_SYSTEM", "空运出口内置状态流转", BusinessTypeAE, []StatusTemplateItem{
			{Code: "BOOKED", Label: "已订舱", ColorToken: statusColorPointer("blue")},
			{Code: "SPACE_ALLOCATED", Label: "已配舱", ColorToken: statusColorPointer("cyan")},
			{Code: "TRUCKING_ARRANGED", Label: "拖车已安排", ColorToken: statusColorPointer("amber")},
			{Code: "DOCUMENT_CUTOFF", Label: "已截单", ColorToken: statusColorPointer("rose")},
			{Code: "CUSTOMS_DECLARATION_ARRANGED", Label: "报关已安排", ColorToken: statusColorPointer("violet")},
			{Code: "DOCUMENT_SIGNED", Label: "已签单", ColorToken: statusColorPointer("indigo")},
		}),
		defaultStatusTemplate("AI_SYSTEM", "空运进口内置状态流转", BusinessTypeAI, []StatusTemplateItem{
			{Code: "TRUCKING_ARRANGED", Label: "拖车已安排", ColorToken: statusColorPointer("amber")},
			{Code: "CUSTOMS_DECLARATION_ARRANGED", Label: "报关已安排", ColorToken: statusColorPointer("violet")},
			{Code: "BILL_EXCHANGED", Label: "已换单", ColorToken: statusColorPointer("teal")},
			{Code: "INSPECTION_ARRANGED", Label: "报检已安排", ColorToken: statusColorPointer("violet")},
		}),
		defaultStatusTemplate("LAND_SYSTEM", "陆运内置状态流转", BusinessTypeLand, []StatusTemplateItem{
			{Code: "TRUCKING_ARRANGED", Label: "拖车已安排", ColorToken: statusColorPointer("amber")},
		}),
		defaultStatusTemplate("RAIL_SYSTEM", "铁路内置状态流转", BusinessTypeRail, []StatusTemplateItem{
			{Code: "DOCUMENT_SIGNED", Label: "已签单", ColorToken: statusColorPointer("indigo")},
			{Code: "BOOKED", Label: "已订舱", ColorToken: statusColorPointer("blue")},
			{Code: "SPACE_ALLOCATED", Label: "已配舱", ColorToken: statusColorPointer("cyan")},
			{Code: "TRUCKING_ARRANGED", Label: "拖车已安排", ColorToken: statusColorPointer("amber")},
			{Code: "DOCUMENT_CUTOFF", Label: "已截单", ColorToken: statusColorPointer("rose")},
			{Code: "CUSTOMS_DECLARATION_ARRANGED", Label: "报关已安排", ColorToken: statusColorPointer("violet")},
		}),
	}
}

func defaultStatusTemplate(code, name string, businessType BusinessType, items []StatusTemplateItem) StatusTemplate {
	result := StatusTemplate{
		Code: code, Name: name, BusinessType: businessType, Version: 1,
		IsDefault: true, Enabled: true,
		Items: []*StatusTemplateItem{{Code: "DRAFT", Label: "新建", SortOrder: 0, Enabled: true, ColorToken: statusColorPointer("slate"), System: true}},
	}
	for index := range items {
		item := items[index]
		item.SortOrder = index + 1
		item.Enabled = true
		item.System = true
		result.Items = append(result.Items, &item)
	}
	return result
}

func statusColorPointer(value string) *string { return &value }

type OrderConfigRepo interface {
	ListNumberRules(context.Context, uuid.UUID) ([]*NumberRule, error)
	CreateNumberRule(context.Context, uuid.UUID, *NumberRule) (*NumberRule, error)
	UpdateNumberRule(context.Context, uuid.UUID, uuid.UUID, *NumberRule) (*NumberRule, error)
	AllocateNumber(context.Context, uuid.UUID, DocumentType, time.Time) (*NumberRule, int64, error)
	ListStatusTemplates(context.Context, uuid.UUID, BusinessType, *bool) ([]*StatusTemplate, error)
	CreateStatusTemplate(context.Context, uuid.UUID, *StatusTemplate) (*StatusTemplate, error)
	PublishStatusTemplate(context.Context, uuid.UUID, uuid.UUID, bool, time.Time) (*StatusTemplate, error)
	SetDefaultStatusTemplate(context.Context, uuid.UUID, uuid.UUID) (*StatusTemplate, error)
}

type OrderConfigUsecase struct {
	repo  OrderConfigRepo
	audit AuditRepo
	now   func() time.Time
}

func NewOrderConfigUsecase(repo OrderConfigRepo, audit AuditRepo) *OrderConfigUsecase {
	return &OrderConfigUsecase{repo: repo, audit: audit, now: time.Now}
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
	created, err := uc.repo.CreateNumberRule(ctx, organizationID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "number_rule.create", Result: "success", Details: map[string]string{"number_rule.id": created.ID.String(), "document_type": string(created.DocumentType)}}); err != nil {
		return nil, fmt.Errorf("write number rule create audit: %w", err)
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
	updated, err := uc.repo.UpdateNumberRule(ctx, organizationID, id, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "number_rule.update", Result: "success", Details: map[string]string{"number_rule.id": updated.ID.String(), "document_type": string(updated.DocumentType)}}); err != nil {
		return nil, fmt.Errorf("write number rule update audit: %w", err)
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
	if businessType != OrderBusinessSE && businessType != OrderBusinessSI && businessType != OrderBusinessAE && businessType != OrderBusinessAI {
		return "", ErrMasterDataInvalidArgument
	}
	return uc.nextNumber(ctx, organizationID, DocumentTypeOrder, string(businessType))
}

func (uc *OrderConfigUsecase) ListStatusTemplates(ctx context.Context, organizationID uuid.UUID, businessType BusinessType, published *bool) ([]*StatusTemplate, error) {
	if organizationID == uuid.Nil || businessType != "" && !businessType.Valid() {
		return nil, ErrStatusTemplateInvalid
	}
	return uc.repo.ListStatusTemplates(ctx, organizationID, businessType, published)
}

func (uc *OrderConfigUsecase) CreateStatusTemplate(ctx context.Context, organizationID, actorID uuid.UUID, input *StatusTemplate) (*StatusTemplate, error) {
	_, err := normalizeStatusTemplate(input)
	if err != nil {
		return nil, err
	}
	return nil, ErrStatusTemplateReadOnly
}

func (uc *OrderConfigUsecase) PublishStatusTemplate(ctx context.Context, organizationID, actorID, id uuid.UUID, isDefault bool) (*StatusTemplate, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrStatusTemplateInvalid
	}
	return nil, ErrStatusTemplateReadOnly
}

func (uc *OrderConfigUsecase) SetDefaultStatusTemplate(ctx context.Context, organizationID, actorID, id uuid.UUID) (*StatusTemplate, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrStatusTemplateInvalid
	}
	return nil, ErrStatusTemplateReadOnly
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

func normalizeStatusTemplate(input *StatusTemplate) (*StatusTemplate, error) {
	if input == nil {
		return nil, ErrStatusTemplateInvalid
	}
	output := *input
	output.Code = strings.ToUpper(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	if output.Code == "" || utf8.RuneCountInString(output.Code) > 64 || output.Name == "" || utf8.RuneCountInString(output.Name) > 100 || !output.BusinessType.Valid() || output.Version < 1 || len(output.Items) == 0 {
		return nil, ErrStatusTemplateInvalid
	}
	seen := make(map[string]struct{}, len(output.Items))
	items := make([]*StatusTemplateItem, 0, len(output.Items))
	for _, item := range output.Items {
		if item == nil {
			return nil, ErrStatusTemplateInvalid
		}
		copyItem := *item
		copyItem.Code = strings.ToUpper(strings.TrimSpace(copyItem.Code))
		copyItem.Label = strings.TrimSpace(copyItem.Label)
		copyItem.ColorToken = normalizedOptionalString(copyItem.ColorToken)
		if copyItem.Code == "" || utf8.RuneCountInString(copyItem.Code) > 64 || copyItem.Label == "" || utf8.RuneCountInString(copyItem.Label) > 100 || copyItem.SortOrder < 0 || copyItem.ColorToken != nil && utf8.RuneCountInString(*copyItem.ColorToken) > 64 {
			return nil, ErrStatusTemplateInvalid
		}
		if _, exists := seen[copyItem.Code]; exists {
			return nil, ErrStatusTemplateInvalid
		}
		seen[copyItem.Code] = struct{}{}
		items = append(items, &copyItem)
	}
	if _, hasDraft := seen["DRAFT"]; !hasDraft {
		return nil, ErrStatusTemplateInvalid
	}
	for _, item := range items {
		if item.Code == "DRAFT" && !item.Enabled {
			return nil, ErrStatusTemplateInvalid
		}
	}
	output.Items = items
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
