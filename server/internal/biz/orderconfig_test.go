package biz

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderConfigRepoStub struct {
	numberRules       []*NumberRule
	createdNumberRule *NumberRule
	updatedNumberRule *NumberRule
	allocatedRule     *NumberRule
	allocatedSequence int64
	allocateErr       error
	lastAllocOrgID    uuid.UUID
	lastAllocDocType  DocumentType
	lastAllocTime     time.Time
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

var _ OrderConfigRepo = (*orderConfigRepoStub)(nil)

func TestDefaultNumberRules(t *testing.T) {
	want := []NumberRule{
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
	docType := DocumentTypeQuotation

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

func TestOrderConfigNextOrderNumberUsesSupportedBusinessType(t *testing.T) {
	for _, businessType := range []OrderBusinessType{OrderBusinessSE} {
		t.Run(string(businessType), func(t *testing.T) {
			repo := &orderConfigRepoStub{
				allocatedRule:     &NumberRule{DateFormat: DateFormatNone, SequenceLength: 5},
				allocatedSequence: 7,
			}
			usecase := NewOrderConfigUsecase(repo, &auditRepoStub{})

			got, err := usecase.NextOrderNumber(context.Background(), uuid.New(), businessType)
			if err != nil {
				t.Fatalf("NextOrderNumber() error = %v, want nil", err)
			}
			if want := string(businessType) + "00007"; got != want {
				t.Fatalf("NextOrderNumber() = %q, want %q", got, want)
			}
			if repo.lastAllocDocType != DocumentTypeOrder {
				t.Fatalf("repo.lastAllocDocType = %q, want %q", repo.lastAllocDocType, DocumentTypeOrder)
			}
		})
	}
}

func TestOrderConfigNextOrderNumberRejectsUnimplementedBusinessType(t *testing.T) {
	usecase := NewOrderConfigUsecase(&orderConfigRepoStub{}, &auditRepoStub{})

	for _, businessType := range []OrderBusinessType{OrderBusinessSI, OrderBusinessAE, OrderBusinessAI, OrderBusinessLand, OrderBusinessRail} {
		if _, err := usecase.NextOrderNumber(context.Background(), uuid.New(), businessType); err != ErrMasterDataInvalidArgument {
			t.Fatalf("NextOrderNumber(%q) error = %v, want %v", businessType, err, ErrMasterDataInvalidArgument)
		}
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
			name: "removed coload house bill type on create",
			mutate: func(r *NumberRule) {
				r.DocumentType = DocumentType("coload_house_bill")
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
