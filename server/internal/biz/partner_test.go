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

func (s *partnerRepoStub) ListAssignmentOptions(_ context.Context, _ uuid.UUID, options SelectorListOptions) (*PagedList[*PartnerAssignmentOption], error) {
	return &PagedList[*PartnerAssignmentOption]{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *partnerRepoStub) ListAuditLogs(context.Context, uuid.UUID, uuid.UUID, int, int) (*PartnerAuditLogList, error) {
	return &PartnerAuditLogList{}, nil
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
	if len(audit.events) != 1 || audit.events[0].Action != "partner.create" || audit.events[0].ResourceType != "partner" || audit.events[0].ResourceID != created.ID.String() || audit.events[0].Details["roles"] != "customer:true,supplier:true" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestPartnerListAuditLogsValidatesPagination(t *testing.T) {
	usecase := NewPartnerUsecase(&partnerRepoStub{}, &auditRepoStub{})
	if _, err := usecase.ListAuditLogs(context.Background(), uuid.New(), uuid.New(), 0, 20); err != ErrPartnerInvalidArgument {
		t.Fatalf("invalid page error = %v, want ErrPartnerInvalidArgument", err)
	}
	if _, err := usecase.ListAuditLogs(context.Background(), uuid.New(), uuid.New(), 1, MaxListPageSize); err != nil {
		t.Fatalf("maximum page size error = %v, want nil", err)
	}
	if _, err := usecase.ListAuditLogs(context.Background(), uuid.New(), uuid.New(), 1, MaxListPageSize+1); err != ErrPartnerInvalidArgument {
		t.Fatalf("invalid page size error = %v, want ErrPartnerInvalidArgument", err)
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

func TestPartnerNormalizesProfileAndAssignments(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo, &auditRepoStub{})
	userID := uuid.New()
	organizationID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, userID, &Partner{
		Code: "CUSTOMER", LegalName: "境内客户", UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		Roles: []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
		Profile: &PartnerProfile{
			NameEN: " ACME Logistics ", CountryCode: " cn ", ProvinceCode: "310000000000",
			CityCode: "310100000000", DistrictCode: "310115000000",
			CustomerTypes: []PartnerCustomerType{PartnerCustomerDirect},
			BusinessTypes: []PartnerBusinessType{PartnerBusinessSE, PartnerBusinessAI},
		},
		Assignments: []*PartnerAssignment{{Role: PartnerAssignmentSales, UserID: userID, OrganizationID: organizationID}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Profile == nil || created.Profile.NameEN != "ACME Logistics" || created.Profile.CountryCode != "CN" {
		t.Fatalf("normalized profile = %#v", created.Profile)
	}
	if len(created.Assignments) != 2 || created.Assignments[0].Role != PartnerAssignmentSales || created.Assignments[1].Role != PartnerAssignmentCreator {
		t.Fatalf("normalized assignments = %#v", created.Assignments)
	}
}

func TestPartnerRejectsInvalidProfileAndAssignments(t *testing.T) {
	usecase := NewPartnerUsecase(&partnerRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()
	base := Partner{
		Code: "CUSTOMER", LegalName: "境内客户", UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		Roles: []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
	}

	invalidProfiles := []*PartnerProfile{
		{CountryCode: "US", ProvinceCode: "310000000000"},
		{CountryCode: "CN", CityCode: "310100000000"},
		{CountryCode: "CN", ProvinceCode: "310000"},
		{CountryCode: "CN", CustomerTypes: []PartnerCustomerType{PartnerCustomerDirect, PartnerCustomerDirect}},
		{CountryCode: "CN", BusinessTypes: []PartnerBusinessType{"OCEAN"}},
	}
	for index, profile := range invalidProfiles {
		input := base
		input.Profile = profile
		if _, err := usecase.Create(context.Background(), organizationID, actorID, &input); err != ErrPartnerInvalidArgument {
			t.Fatalf("invalid profile %d error = %v, want ErrPartnerInvalidArgument", index, err)
		}
	}

	input := base
	input.Assignments = []*PartnerAssignment{
		{Role: PartnerAssignmentSales, UserID: uuid.New(), OrganizationID: organizationID},
		{Role: PartnerAssignmentSales, UserID: uuid.New(), OrganizationID: organizationID},
	}
	if _, err := usecase.Create(context.Background(), organizationID, actorID, &input); err != ErrPartnerInvalidArgument {
		t.Fatalf("duplicate assignment role error = %v, want ErrPartnerInvalidArgument", err)
	}

	input.Assignments = []*PartnerAssignment{{Role: PartnerAssignmentCreator, UserID: actorID, OrganizationID: organizationID}}
	if _, err := usecase.Create(context.Background(), organizationID, actorID, &input); err != ErrPartnerInvalidArgument {
		t.Fatalf("client creator assignment error = %v, want ErrPartnerInvalidArgument", err)
	}
}

func TestPartnerAllowsTwoInternalContacts(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()
	input := &Partner{
		Code: "CUSTOMER", LegalName: "境内客户", UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		Roles: []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
		Assignments: []*PartnerAssignment{
			{Role: PartnerAssignmentInternalContact, UserID: uuid.New(), OrganizationID: organizationID},
			{Role: PartnerAssignmentInternalContact, UserID: uuid.New(), OrganizationID: organizationID},
		},
	}
	created, err := usecase.Create(context.Background(), organizationID, actorID, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Assignments[0].SortOrder != 1 || created.Assignments[1].SortOrder != 2 {
		t.Fatalf("internal contact sort orders = %d, %d", created.Assignments[0].SortOrder, created.Assignments[1].SortOrder)
	}

	input.Assignments = append(input.Assignments, &PartnerAssignment{
		Role: PartnerAssignmentInternalContact, UserID: uuid.New(), OrganizationID: organizationID,
	})
	if _, err := usecase.Create(context.Background(), organizationID, actorID, input); err != ErrPartnerInvalidArgument {
		t.Fatalf("third internal contact error = %v, want ErrPartnerInvalidArgument", err)
	}
}

func TestPartnerNormalizesRoleSettlementRule(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()
	creditCurrency := " cny "
	creditLimit := int64(50000000)

	created, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code: "CUSTOMER", LegalName: "境内客户", UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		Roles: []*PartnerRole{{
			Type: PartnerRoleCustomer, Enabled: true,
			SettlementRule: &PartnerSettlementRule{
				StatementMode: PartnerStatementSingle, SettlementMethod: PartnerSettlementByTicket,
				SettlementCurrency: " cny ", CreditLimitMinor: &creditLimit, CreditCurrency: &creditCurrency, IsActive: true,
			},
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	rule := created.Roles[0].SettlementRule
	if rule == nil || rule.SettlementCurrency != "CNY" || rule.CreditCurrency == nil || *rule.CreditCurrency != "CNY" {
		t.Fatalf("normalized settlement rule = %#v", rule)
	}

	created.Roles[0].SettlementRule.SettlementMethod = PartnerSettlementMonthly
	created.Assignments = nil
	if _, err := usecase.Create(context.Background(), organizationID, actorID, created); err != ErrPartnerSettlementRuleInvalidArgument {
		t.Fatalf("monthly rule without settlement base error = %v, want ErrPartnerSettlementRuleInvalidArgument", err)
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
