package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type partnerRepoStub struct {
	created         *Partner
	updated         *Partner
	updateResult    *PartnerUpdateResult
	blacklistInput  PartnerBlacklistUpdate
	blacklistResult *PartnerBlacklistResult
}

func (s *partnerRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*Partner, error) {
	return nil, ErrPartnerNotFound
}

func (s *partnerRepoStub) List(_ context.Context, _ uuid.UUID, options PartnerListOptions) (*PartnerList, error) {
	return &PartnerList{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *partnerRepoStub) Create(_ context.Context, organizationID uuid.UUID, input *Partner) (*Partner, error) {
	s.created = input
	input.ID = uuid.New()
	input.OrganizationID = organizationID
	return input, nil
}

func (s *partnerRepoStub) Update(_ context.Context, organizationID, id uuid.UUID, input *Partner) (*PartnerUpdateResult, error) {
	s.updated = input
	if s.updateResult != nil {
		return s.updateResult, nil
	}
	input.ID = id
	input.OrganizationID = organizationID
	return &PartnerUpdateResult{Partner: input}, nil
}

func (s *partnerRepoStub) SetSupplierBlacklist(_ context.Context, organizationID, id uuid.UUID, input PartnerBlacklistUpdate) (*PartnerBlacklistResult, error) {
	s.blacklistInput = input
	if s.blacklistResult != nil {
		return s.blacklistResult, nil
	}
	return &PartnerBlacklistResult{Partner: &Partner{ID: id, OrganizationID: organizationID}}, nil
}

func (s *partnerRepoStub) Import(_ context.Context, _ uuid.UUID, _ PartnerImportMode, items []*Partner) (*PartnerImportResult, error) {
	return &PartnerImportResult{CreatedCount: len(items)}, nil
}

func TestPartnerCreateNormalizesAggregateAndAudits(t *testing.T) {
	repo := &partnerRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewPartnerUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code:                    " acme ",
		LegalName:               "  上海   安可 物流有限公司 ",
		UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		RegisteredAddress:       " 上海市 ",
		Roles: []*PartnerRole{
			{Type: PartnerRoleCustomer, Enabled: true},
			{Type: PartnerRoleSupplier, Enabled: true},
		},
		Contacts: []*PartnerContact{{Name: " 张三 ", Phone: " 13800000000 ", Email: "contact@example.com", IsPrimary: true}},
		Aliases:  []*PartnerAlias{{AliasName: " ACME  Logistics ", SortOrder: 1}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Code != "ACME" || created.LegalName != "上海   安可 物流有限公司" || created.NormalizedName != "上海 安可 物流有限公司" {
		t.Fatalf("normalized partner = %#v", created)
	}
	if created.RegisteredAddress != "上海市" || created.Contacts[0].Name != "张三" || created.Aliases[0].NormalizedAliasName != "ACME LOGISTICS" {
		t.Fatalf("normalized children = contacts %#v aliases %#v", created.Contacts, created.Aliases)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "partner.create" || audit.events[0].Details["roles"] != "customer:true,supplier:true" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestPartnerRejectsRoleAndPrimaryContactConflicts(t *testing.T) {
	usecase := NewPartnerUsecase(&partnerRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	if _, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code: "P001", LegalName: "无角色公司",
	}); err != ErrPartnerRoleRequired {
		t.Fatalf("empty roles error = %v, want ErrPartnerRoleRequired", err)
	}
	if _, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code: "P002", LegalName: "重复角色公司",
		Roles: []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}, {Type: PartnerRoleCustomer, Enabled: true}},
	}); err != ErrPartnerInvalidRole {
		t.Fatalf("duplicate roles error = %v, want ErrPartnerInvalidRole", err)
	}
	if _, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code: "P003", LegalName: "多主联系人公司",
		Roles: []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
		Contacts: []*PartnerContact{
			{Name: "张三", IsPrimary: true},
			{Name: "李四", IsPrimary: true},
		},
	}); err != ErrPartnerPrimaryContactConflict {
		t.Fatalf("primary contacts error = %v, want ErrPartnerPrimaryContactConflict", err)
	}
}

func TestPartnerRejectsInvalidUSCCAndDuplicateAlias(t *testing.T) {
	usecase := NewPartnerUsecase(&partnerRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	if _, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code: "P001", LegalName: "信用代码错误公司", UnifiedSocialCreditCode: "123",
		Roles: []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
	}); err != ErrPartnerInvalidArgument {
		t.Fatalf("invalid USCC error = %v, want ErrPartnerInvalidArgument", err)
	}
	if _, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code: "P002", LegalName: "重复别名公司",
		Roles:   []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
		Aliases: []*PartnerAlias{{AliasName: "Acme Logistics"}, {AliasName: " acme   logistics "}},
	}); err != ErrPartnerAliasExists {
		t.Fatalf("duplicate aliases error = %v, want ErrPartnerAliasExists", err)
	}
}

func TestPartnerTaxIdentifierDependsOnActiveRole(t *testing.T) {
	usecase := NewPartnerUsecase(&partnerRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	for _, roleType := range []PartnerRoleType{PartnerRoleCustomer, PartnerRoleSupplier} {
		if _, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
			Code: "DOMESTIC", LegalName: "境内往来单位",
			Roles: []*PartnerRole{{Type: roleType, Enabled: true}},
		}); err != ErrPartnerTaxIdentifierRequired {
			t.Fatalf("role %s error = %v, want ErrPartnerTaxIdentifierRequired", roleType, err)
		}
	}

	if _, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code: "FOREIGN", LegalName: "国外代理",
		Roles: []*PartnerRole{{Type: PartnerRoleForeignAgent, Enabled: true}},
	}); err != nil {
		t.Fatalf("foreign agent without tax identifier error = %v", err)
	}
}

func TestPartnerSetSupplierBlacklistRequiresReasonAndAudits(t *testing.T) {
	partnerID := uuid.New()
	organizationID := uuid.New()
	actorID := uuid.New()
	changedAt := time.Date(2026, time.August, 20, 8, 0, 0, 0, time.UTC)
	repo := &partnerRepoStub{blacklistResult: &PartnerBlacklistResult{
		Partner:               &Partner{ID: partnerID, OrganizationID: organizationID},
		PreviouslyBlacklisted: false,
	}}
	audit := &auditRepoStub{}
	usecase := NewPartnerUsecase(repo, audit)
	usecase.now = func() time.Time { return changedAt }

	if _, err := usecase.SetSupplierBlacklist(context.Background(), organizationID, actorID, partnerID, true, "   "); err != ErrPartnerBlacklistReasonRequired {
		t.Fatalf("empty reason error = %v, want ErrPartnerBlacklistReasonRequired", err)
	}
	updated, err := usecase.SetSupplierBlacklist(context.Background(), organizationID, actorID, partnerID, true, "  严重违约  ")
	if err != nil {
		t.Fatalf("SetSupplierBlacklist() error = %v", err)
	}
	if updated.ID != partnerID || !repo.blacklistInput.Blacklisted || repo.blacklistInput.Reason != "严重违约" || repo.blacklistInput.ChangedAt != changedAt || repo.blacklistInput.ChangedBy != actorID {
		t.Fatalf("blacklist input = %#v, updated = %#v", repo.blacklistInput, updated)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "partner.supplier_blacklist.set" || audit.events[0].Details["reason"] != "严重违约" || audit.events[0].Details["blacklisted"] != "true" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

var _ PartnerRepo = (*partnerRepoStub)(nil)
