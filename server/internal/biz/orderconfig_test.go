package biz

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderConfigRepoStub struct {
	numberRules          []*NumberRule
	createdNumberRule    *NumberRule
	updatedNumberRule    *NumberRule
	statusTemplates      []*StatusTemplate
	createdTemplate      *StatusTemplate
	publishedTemplate    *StatusTemplate
	defaultTemplate      *StatusTemplate
	allocatedRule        *NumberRule
	allocatedSequence    int64
	allocateErr          error
	lastAllocOrgID       uuid.UUID
	lastAllocDocType     DocumentType
	lastAllocTime        time.Time
	lastPublishOrgID     uuid.UUID
	lastPublishID        uuid.UUID
	lastPublishIsDefault bool
	lastPublishTime      time.Time
	lastSetDefaultOrgID  uuid.UUID
	lastSetDefaultID     uuid.UUID
}

func (s *orderConfigRepoStub) ListNumberRules(_ context.Context, _ uuid.UUID) ([]*NumberRule, error) {
	return s.numberRules, nil
}

func (s *orderConfigRepoStub) CreateNumberRule(_ context.Context, organizationID uuid.UUID, input *NumberRule) (*NumberRule, error) {
	s.createdNumberRule = input
	input.OrganizationID = organizationID
	if input.ID == uuid.Nil {
		input.ID = uuid.New()
	}
	return input, nil
}

func (s *orderConfigRepoStub) UpdateNumberRule(_ context.Context, organizationID, id uuid.UUID, input *NumberRule) (*NumberRule, error) {
	s.updatedNumberRule = input
	input.OrganizationID = organizationID
	input.ID = id
	return input, nil
}

func (s *orderConfigRepoStub) AllocateNumber(_ context.Context, organizationID uuid.UUID, documentType DocumentType, now time.Time) (*NumberRule, int64, error) {
	s.lastAllocOrgID = organizationID
	s.lastAllocDocType = documentType
	s.lastAllocTime = now
	if s.allocateErr != nil {
		return nil, 0, s.allocateErr
	}
	return s.allocatedRule, s.allocatedSequence, nil
}

func (s *orderConfigRepoStub) ListStatusTemplates(_ context.Context, _ uuid.UUID, _ BusinessType, _ *bool) ([]*StatusTemplate, error) {
	return s.statusTemplates, nil
}

func (s *orderConfigRepoStub) CreateStatusTemplate(_ context.Context, organizationID uuid.UUID, input *StatusTemplate) (*StatusTemplate, error) {
	s.createdTemplate = input
	input.OrganizationID = organizationID
	if input.ID == uuid.Nil {
		input.ID = uuid.New()
	}
	return input, nil
}

func (s *orderConfigRepoStub) PublishStatusTemplate(_ context.Context, organizationID, id uuid.UUID, isDefault bool, now time.Time) (*StatusTemplate, error) {
	s.lastPublishOrgID = organizationID
	s.lastPublishID = id
	s.lastPublishIsDefault = isDefault
	s.lastPublishTime = now
	if s.publishedTemplate != nil {
		return s.publishedTemplate, nil
	}
	return &StatusTemplate{
		ID:             id,
		OrganizationID: organizationID,
		Code:           "OCEAN_EXPORT",
		Version:        1,
		IsDefault:      isDefault,
		PublishedAt:    &now,
	}, nil
}

func (s *orderConfigRepoStub) SetDefaultStatusTemplate(_ context.Context, organizationID, id uuid.UUID) (*StatusTemplate, error) {
	s.lastSetDefaultOrgID = organizationID
	s.lastSetDefaultID = id
	if s.defaultTemplate != nil {
		return s.defaultTemplate, nil
	}
	return &StatusTemplate{
		ID:             id,
		OrganizationID: organizationID,
		Code:           "OCEAN_EXPORT",
		Version:        1,
		IsDefault:      true,
	}, nil
}

var _ OrderConfigRepo = (*orderConfigRepoStub)(nil)

func TestDefaultNumberRules(t *testing.T) {
	want := []NumberRule{
		{DocumentType: DocumentTypeOrder, Prefix: "OR", DateFormat: DateFormatYYYYMMDD, SequenceLength: 4, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeBooking, Prefix: "BK", DateFormat: DateFormatYYYYMMDD, SequenceLength: 4, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeHBL, Prefix: "HBL", DateFormat: DateFormatYYYYMMDD, SequenceLength: 4, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeMBL, Prefix: "MBL", DateFormat: DateFormatYYYYMMDD, SequenceLength: 4, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeBill, Prefix: "BI", DateFormat: DateFormatYYYYMMDD, SequenceLength: 4, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeStatement, Prefix: "ST", DateFormat: DateFormatYYYYMMDD, SequenceLength: 4, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypePayment, Prefix: "PY", DateFormat: DateFormatYYYYMMDD, SequenceLength: 4, ResetPolicy: ResetPolicyDaily, Enabled: true},
		{DocumentType: DocumentTypeInvoice, Prefix: "IV", DateFormat: DateFormatYYYYMMDD, SequenceLength: 4, ResetPolicy: ResetPolicyDaily, Enabled: true},
	}

	if got := DefaultNumberRules(); !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultNumberRules() = %#v, want %#v", got, want)
	}
}

func TestOrderConfigCreateNumberRuleNormalizesAndAudits(t *testing.T) {
	repo := &orderConfigRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewOrderConfigUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	input := &NumberRule{
		DocumentType:   DocumentTypeOrder,
		Prefix:         "  ord-se  ",
		DateFormat:     DateFormatYYYYMMDD,
		SequenceLength: 4,
		ResetPolicy:    ResetPolicyDaily,
		Enabled:        true,
	}

	created, err := usecase.CreateNumberRule(context.Background(), organizationID, actorID, input)
	if err != nil {
		t.Fatalf("CreateNumberRule() error = %v, want nil", err)
	}

	if created.Prefix != "ORD-SE" {
		t.Fatalf("created.Prefix = %q, want ORD-SE", created.Prefix)
	}
	if repo.createdNumberRule == nil || repo.createdNumberRule.Prefix != "ORD-SE" {
		t.Fatalf("repo.createdNumberRule prefix = %v, want ORD-SE", repo.createdNumberRule)
	}
	if len(audit.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(audit.events))
	}

	event := audit.events[0]
	if event.Action != "number_rule.create" {
		t.Fatalf("audit event action = %q, want number_rule.create", event.Action)
	}
	if event.OrganizationID == nil || *event.OrganizationID != organizationID {
		t.Fatalf("audit event OrganizationID = %v, want %v", event.OrganizationID, organizationID)
	}
	if event.UserID == nil || *event.UserID != actorID {
		t.Fatalf("audit event UserID = %v, want %v", event.UserID, actorID)
	}
	if event.Result != "success" {
		t.Fatalf("audit event result = %q, want success", event.Result)
	}
	wantDetails := map[string]string{
		"number_rule.id": created.ID.String(),
		"document_type":  string(DocumentTypeOrder),
	}
	if !reflect.DeepEqual(event.Details, wantDetails) {
		t.Fatalf("audit event details = %#v, want %#v", event.Details, wantDetails)
	}
}

func TestOrderConfigNextNumberFormatAndRepoArguments(t *testing.T) {
	fixedTime := time.Date(2026, 8, 20, 14, 30, 0, 0, time.FixedZone("CST", 8*3600))
	expectedUTC := fixedTime.UTC()
	organizationID := uuid.New()
	docType := DocumentTypeBooking

	testCases := []struct {
		name       string
		rule       *NumberRule
		sequence   int64
		wantNumber string
	}{
		{
			name: "yyyyMMdd with 4 digits padding",
			rule: &NumberRule{
				Prefix:         "BKG",
				DateFormat:     DateFormatYYYYMMDD,
				SequenceLength: 4,
			},
			sequence:   42,
			wantNumber: "BKG202608200042",
		},
		{
			name: "yyyyMM with 6 digits padding",
			rule: &NumberRule{
				Prefix:         "BKG-",
				DateFormat:     DateFormatYYYYMM,
				SequenceLength: 6,
			},
			sequence:   1,
			wantNumber: "BKG-202608000001",
		},
		{
			name: "yyyy with 3 digits padding",
			rule: &NumberRule{
				Prefix:         "EXP",
				DateFormat:     DateFormatYYYY,
				SequenceLength: 3,
			},
			sequence:   123,
			wantNumber: "EXP2026123",
		},
		{
			name: "none date format with 5 digits padding",
			rule: &NumberRule{
				Prefix:         "NO-DATE-",
				DateFormat:     DateFormatNone,
				SequenceLength: 5,
			},
			sequence:   7,
			wantNumber: "NO-DATE-00007",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &orderConfigRepoStub{
				allocatedRule:     tc.rule,
				allocatedSequence: tc.sequence,
			}
			usecase := NewOrderConfigUsecase(repo, &auditRepoStub{})
			usecase.now = func() time.Time {
				return fixedTime
			}

			gotNumber, err := usecase.NextNumber(context.Background(), organizationID, docType)
			if err != nil {
				t.Fatalf("NextNumber() error = %v, want nil", err)
			}
			if gotNumber != tc.wantNumber {
				t.Fatalf("NextNumber() = %q, want %q", gotNumber, tc.wantNumber)
			}
			if repo.lastAllocOrgID != organizationID {
				t.Fatalf("repo.lastAllocOrgID = %v, want %v", repo.lastAllocOrgID, organizationID)
			}
			if repo.lastAllocDocType != docType {
				t.Fatalf("repo.lastAllocDocType = %v, want %v", repo.lastAllocDocType, docType)
			}
			if !repo.lastAllocTime.Equal(expectedUTC) {
				t.Fatalf("repo.lastAllocTime = %v, want %v", repo.lastAllocTime, expectedUTC)
			}
		})
	}
}

func TestOrderConfigNextNumberSequenceExhausted(t *testing.T) {
	repo := &orderConfigRepoStub{
		allocatedRule: &NumberRule{
			Prefix:         "ORD",
			DateFormat:     DateFormatYYYYMMDD,
			SequenceLength: 3,
		},
		allocatedSequence: 1000,
	}
	usecase := NewOrderConfigUsecase(repo, &auditRepoStub{})

	_, err := usecase.NextNumber(context.Background(), uuid.New(), DocumentTypeOrder)
	if err != ErrNumberSequenceExhausted {
		t.Fatalf("NextNumber() error = %v, want %v", err, ErrNumberSequenceExhausted)
	}
}

func TestOrderConfigCreateStatusTemplateRejections(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()

	tests := []struct {
		name     string
		template *StatusTemplate
	}{
		{
			name: "missing DRAFT item",
			template: &StatusTemplate{
				Code:         "SE_DEFAULT",
				Name:         "海运出口默认模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*StatusTemplateItem{
					{Code: "BOOKED", Label: "已订舱", SortOrder: 1, Enabled: true},
					{Code: "COMPLETED", Label: "已完成", SortOrder: 2, Enabled: true},
				},
			},
		},
		{
			name: "DRAFT item is disabled",
			template: &StatusTemplate{
				Code:         "SE_DEFAULT",
				Name:         "海运出口默认模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*StatusTemplateItem{
					{Code: "DRAFT", Label: "草稿", SortOrder: 0, Enabled: false},
					{Code: "BOOKED", Label: "已订舱", SortOrder: 1, Enabled: true},
				},
			},
		},
		{
			name: "duplicate status code (case insensitive/trimmed)",
			template: &StatusTemplate{
				Code:         "SE_DEFAULT",
				Name:         "海运出口默认模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*StatusTemplateItem{
					{Code: "DRAFT", Label: "草稿", SortOrder: 0, Enabled: true},
					{Code: "BOOKED", Label: "已订舱", SortOrder: 1, Enabled: true},
					{Code: "  booked ", Label: "已订舱重复", SortOrder: 2, Enabled: true},
				},
			},
		},
		{
			name:     "nil template",
			template: nil,
		},
		{
			name: "empty code",
			template: &StatusTemplate{
				Code:         "   ",
				Name:         "模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*StatusTemplateItem{
					{Code: "DRAFT", Label: "草稿", SortOrder: 0, Enabled: true},
				},
			},
		},
		{
			name: "invalid business type",
			template: &StatusTemplate{
				Code:         "TEMPLATE",
				Name:         "模板",
				BusinessType: BusinessType("INVALID"),
				Version:      1,
				Items: []*StatusTemplateItem{
					{Code: "DRAFT", Label: "草稿", SortOrder: 0, Enabled: true},
				},
			},
		},
		{
			name: "version less than 1",
			template: &StatusTemplate{
				Code:         "TEMPLATE",
				Name:         "模板",
				BusinessType: BusinessTypeSE,
				Version:      0,
				Items: []*StatusTemplateItem{
					{Code: "DRAFT", Label: "草稿", SortOrder: 0, Enabled: true},
				},
			},
		},
		{
			name: "empty items",
			template: &StatusTemplate{
				Code:         "TEMPLATE",
				Name:         "模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items:        []*StatusTemplateItem{},
			},
		},
		{
			name: "nil item inside items list",
			template: &StatusTemplate{
				Code:         "TEMPLATE",
				Name:         "模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*StatusTemplateItem{
					{Code: "DRAFT", Label: "草稿", SortOrder: 0, Enabled: true},
					nil,
				},
			},
		},
		{
			name: "item with negative sort order",
			template: &StatusTemplate{
				Code:         "TEMPLATE",
				Name:         "模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*StatusTemplateItem{
					{Code: "DRAFT", Label: "草稿", SortOrder: -1, Enabled: true},
				},
			},
		},
		{
			name: "template code longer than 64 characters",
			template: &StatusTemplate{
				Code:         "12345678901234567890123456789012345678901234567890123456789012345",
				Name:         "模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*StatusTemplateItem{
					{Code: "DRAFT", Label: "草稿", SortOrder: 0, Enabled: true},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			usecase := NewOrderConfigUsecase(&orderConfigRepoStub{}, &auditRepoStub{})
			_, err := usecase.CreateStatusTemplate(context.Background(), organizationID, actorID, tc.template)
			if err != ErrStatusTemplateInvalid {
				t.Fatalf("CreateStatusTemplate() error = %v, want %v", err, ErrStatusTemplateInvalid)
			}
		})
	}
}

func TestOrderConfigSetDefaultStatusTemplate(t *testing.T) {
	repo := &orderConfigRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewOrderConfigUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	templateID := uuid.New()

	template, err := usecase.SetDefaultStatusTemplate(context.Background(), organizationID, actorID, templateID)
	if err != nil {
		t.Fatalf("SetDefaultStatusTemplate() error = %v, want nil", err)
	}

	if repo.lastSetDefaultOrgID != organizationID {
		t.Fatalf("repo.lastSetDefaultOrgID = %v, want %v", repo.lastSetDefaultOrgID, organizationID)
	}
	if repo.lastSetDefaultID != templateID {
		t.Fatalf("repo.lastSetDefaultID = %v, want %v", repo.lastSetDefaultID, templateID)
	}
	if !template.IsDefault {
		t.Fatalf("template.IsDefault = false, want true")
	}

	if len(audit.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(audit.events))
	}
	event := audit.events[0]
	if event.Action != "status_template.set_default" {
		t.Fatalf("audit event action = %q, want status_template.set_default", event.Action)
	}
	if event.OrganizationID == nil || *event.OrganizationID != organizationID {
		t.Fatalf("audit event OrganizationID = %v, want %v", event.OrganizationID, organizationID)
	}
	if event.UserID == nil || *event.UserID != actorID {
		t.Fatalf("audit event UserID = %v, want %v", event.UserID, actorID)
	}
	if event.Result != "success" {
		t.Fatalf("audit event result = %q, want success", event.Result)
	}
	wantDetails := map[string]string{
		"status_template.id":   template.ID.String(),
		"status_template.code": template.Code,
		"version":              "1",
	}
	if !reflect.DeepEqual(event.Details, wantDetails) {
		t.Fatalf("audit event details = %#v, want %#v", event.Details, wantDetails)
	}

	// Validate invalid arguments
	if _, err := usecase.SetDefaultStatusTemplate(context.Background(), uuid.Nil, actorID, templateID); err != ErrStatusTemplateInvalid {
		t.Fatalf("nil orgID error = %v, want %v", err, ErrStatusTemplateInvalid)
	}
	if _, err := usecase.SetDefaultStatusTemplate(context.Background(), organizationID, actorID, uuid.Nil); err != ErrStatusTemplateInvalid {
		t.Fatalf("nil templateID error = %v, want %v", err, ErrStatusTemplateInvalid)
	}
}

func TestOrderConfigNumberRuleValidationBoundaries(t *testing.T) {
	usecase := NewOrderConfigUsecase(&orderConfigRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	baseRule := func() *NumberRule {
		return &NumberRule{
			DocumentType:   DocumentTypeOrder,
			Prefix:         "ORD",
			DateFormat:     DateFormatYYYYMMDD,
			SequenceLength: 4,
			ResetPolicy:    ResetPolicyDaily,
			Enabled:        true,
		}
	}

	tests := []struct {
		name      string
		mutate    func(r *NumberRule)
		wantError error
	}{
		{
			name: "invalid date format",
			mutate: func(r *NumberRule) {
				r.DateFormat = DateFormat("invalid_date_format")
			},
			wantError: ErrMasterDataInvalidArgument,
		},
		{
			name: "invalid reset policy",
			mutate: func(r *NumberRule) {
				r.ResetPolicy = ResetPolicy("invalid_reset_policy")
			},
			wantError: ErrMasterDataInvalidArgument,
		},
		{
			name: "sequence length 0 (below minimum 1)",
			mutate: func(r *NumberRule) {
				r.SequenceLength = 0
			},
			wantError: ErrMasterDataInvalidArgument,
		},
		{
			name: "sequence length 13 (above maximum 12)",
			mutate: func(r *NumberRule) {
				r.SequenceLength = 13
			},
			wantError: ErrMasterDataInvalidArgument,
		},
		{
			name: "invalid document type on create",
			mutate: func(r *NumberRule) {
				r.DocumentType = DocumentType("unsupported_doc_type")
			},
			wantError: ErrMasterDataInvalidArgument,
		},
		{
			name: "prefix longer than 32 characters",
			mutate: func(r *NumberRule) {
				r.Prefix = "123456789012345678901234567890123"
			},
			wantError: ErrMasterDataInvalidArgument,
		},
		{
			name: "nil input",
			mutate: func(r *NumberRule) {
				// handled by passing nil
			},
			wantError: ErrMasterDataInvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rule *NumberRule
			if tc.name != "nil input" {
				rule = baseRule()
				tc.mutate(rule)
			}
			_, err := usecase.CreateNumberRule(context.Background(), organizationID, actorID, rule)
			if err != tc.wantError {
				t.Fatalf("CreateNumberRule() error = %v, want %v", err, tc.wantError)
			}
		})
	}
}
