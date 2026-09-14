package biz

import (
	"context"
	stderrors "errors"
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
	auditEvent      *AuditEvent
	listOptions     PartnerListOptions
	importedItems   []*Partner
	importedMode    PartnerImportMode
}

func (s *partnerRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*Partner, error) {
	return nil, ErrPartnerNotFound
}

func (s *partnerRepoStub) FindAuthorized(context.Context, uuid.UUID, []uuid.UUID) (*Partner, error) {
	return nil, ErrPartnerNotFound
}

func (s *partnerRepoStub) List(_ context.Context, _ []uuid.UUID, options PartnerListOptions) (*PartnerList, error) {
	s.listOptions = options
	return &PartnerList{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *partnerRepoStub) ListAssignmentOptions(_ context.Context, _ uuid.UUID, options SelectorListOptions) (*PagedList[*PartnerAssignmentOption], error) {
	return &PagedList[*PartnerAssignmentOption]{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *partnerRepoStub) ListAuditLogs(context.Context, uuid.UUID, uuid.UUID, int, int) (*PartnerAuditLogList, error) {
	return &PartnerAuditLogList{}, nil
}

func (s *partnerRepoStub) Create(_ context.Context, organizationID uuid.UUID, input *Partner, audit *AuditEvent) (*Partner, error) {
	s.created = input
	input.ID = uuid.New()
	input.OrganizationID = organizationID
	audit.ResourceID = input.ID.String()
	audit.Details["partner.id"] = input.ID.String()
	s.auditEvent = audit
	return input, nil
}

func (s *partnerRepoStub) Update(_ context.Context, organizationID, id uuid.UUID, input *Partner, audit *AuditEvent) (*PartnerUpdateResult, error) {
	s.updated = input
	s.auditEvent = audit
	if s.updateResult != nil {
		return s.updateResult, nil
	}
	input.ID = id
	input.OrganizationID = organizationID
	return &PartnerUpdateResult{Partner: input}, nil
}

func (s *partnerRepoStub) SetSupplierBlacklist(_ context.Context, organizationID, id uuid.UUID, input PartnerBlacklistUpdate, audit *AuditEvent) (*PartnerBlacklistResult, error) {
	s.blacklistInput = input
	s.auditEvent = audit
	if s.blacklistResult != nil {
		return s.blacklistResult, nil
	}
	return &PartnerBlacklistResult{Partner: &Partner{ID: id, OrganizationID: organizationID}}, nil
}

func (s *partnerRepoStub) Import(_ context.Context, _ uuid.UUID, mode PartnerImportMode, items []*Partner, audit *AuditEvent) (*PartnerImportResult, error) {
	s.importedMode = mode
	s.importedItems = items
	s.auditEvent = audit
	return &PartnerImportResult{CreatedCount: len(items)}, nil
}

func TestPartnerCreateNormalizesAggregateAndAudits(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo)
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
	if repo.auditEvent == nil || repo.auditEvent.Action != "partner.create" || repo.auditEvent.ResourceType != "partner" || repo.auditEvent.ResourceID != created.ID.String() || repo.auditEvent.Details["roles"] != "customer:true,supplier:true" {
		t.Fatalf("audit event = %#v", repo.auditEvent)
	}
}

func TestPartnerListAuditLogsValidatesPagination(t *testing.T) {
	usecase := NewPartnerUsecase(&partnerRepoStub{})
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

func TestPartnerCreateKeepsEmptyCode(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo)

	created, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), &Partner{
		LegalName:               "快捷新建往来单位",
		UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		Roles:                   []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Code != "" {
		t.Fatalf("code = %q, want empty（留空不自动生成）", created.Code)
	}
	if repo.auditEvent == nil || repo.auditEvent.Details["partner.code"] != "" {
		t.Fatalf("audit event = %#v", repo.auditEvent)
	}
}

type conflictPartnerRepoStub struct {
	partnerRepoStub
	createCalls int
	conflicts   int
	err         error
	codes       []string
}

func (s *conflictPartnerRepoStub) Create(ctx context.Context, organizationID uuid.UUID, input *Partner, audit *AuditEvent) (*Partner, error) {
	s.createCalls++
	s.codes = append(s.codes, input.Code)
	if s.err != nil {
		return nil, s.err
	}
	if s.createCalls <= s.conflicts {
		return nil, ErrPartnerCodeExists
	}
	return s.partnerRepoStub.Create(ctx, organizationID, input, audit)
}

func TestPartnerCreateDoesNotRetryExplicitCodeConflict(t *testing.T) {
	repo := &conflictPartnerRepoStub{conflicts: 3}
	usecase := NewPartnerUsecase(repo)

	_, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), &Partner{
		Code:                    "EXPLICIT",
		LegalName:               "显式编码冲突往来单位",
		UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		Roles:                   []*PartnerRole{{Type: PartnerRoleSupplier, Enabled: true}},
	})
	if !stderrors.Is(err, ErrPartnerCodeExists) {
		t.Fatalf("Create() error = %v, want ErrPartnerCodeExists", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", repo.createCalls)
	}
}

func TestPartnerCreateDoesNotRetryOtherErrors(t *testing.T) {
	repoErr := stderrors.New("create failed")
	repo := &conflictPartnerRepoStub{err: repoErr}
	usecase := NewPartnerUsecase(repo)

	_, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), &Partner{
		LegalName:               "仓储失败往来单位",
		UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		Roles:                   []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
	})
	if !stderrors.Is(err, repoErr) {
		t.Fatalf("Create() error = %v, want %v", err, repoErr)
	}
	if repo.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", repo.createCalls)
	}
}

func TestPartnerRejectsRoleAndPrimaryContactConflicts(t *testing.T) {
	usecase := NewPartnerUsecase(&partnerRepoStub{})
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
	usecase := NewPartnerUsecase(&partnerRepoStub{})
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

func TestPartnerCreateWithoutTaxIdentifier(t *testing.T) {
	usecase := NewPartnerUsecase(&partnerRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	// 个人客户与境外主体没有统一社会信用代码，不填税号时也必须能建档。
	for _, roleType := range []PartnerRoleType{PartnerRoleCustomer, PartnerRoleSupplier, PartnerRoleForeignAgent} {
		if _, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
			Code: "NOTAX", LegalName: "无税号往来单位",
			Roles: []*PartnerRole{{Type: roleType, Enabled: true}},
		}); err != nil {
			t.Fatalf("role %s create without tax identifier error = %v", roleType, err)
		}
	}
}

func TestPartnerNormalizesProfileAndAssignments(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo)
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
	usecase := NewPartnerUsecase(&partnerRepoStub{})
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
	usecase := NewPartnerUsecase(repo)
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
	usecase := NewPartnerUsecase(repo)
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
	usecase := NewPartnerUsecase(repo)
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
	if repo.auditEvent == nil || repo.auditEvent.Action != "partner.supplier_blacklist.set" || repo.auditEvent.Details["reason"] != "严重违约" || repo.auditEvent.Details["blacklisted"] != "true" {
		t.Fatalf("audit event = %#v", repo.auditEvent)
	}
}

func TestPartnerAssignmentFinanceAndDocumentRolesAreIndependent(t *testing.T) {
	if !PartnerAssignmentFinance.Valid() {
		t.Fatal("客户财务人员角色应当有效")
	}
	if !PartnerAssignmentDocument.Valid() {
		t.Fatal("客户单证人员角色应当有效")
	}
	if PartnerAssignmentFinance == PartnerAssignmentDocument {
		t.Fatal("客户财务人员与单证人员必须是独立角色")
	}
}

func TestFormatPartnerRolesAuditValueIsStable(t *testing.T) {
	roles := []*PartnerRole{
		{Type: PartnerRoleSupplier, Enabled: false},
		nil,
		{Type: PartnerRoleCustomer, Enabled: true},
	}

	if got, want := FormatPartnerRolesAuditValue(roles), "customer:true,supplier:false"; got != want {
		t.Fatalf("FormatPartnerRolesAuditValue() = %q, want %q", got, want)
	}
}

func TestPartnerCreateAndUpdatePreservesIsCasual(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo)
	organizationID := uuid.New()
	actorID := uuid.New()

	// 1. 创建散客
	created, err := usecase.Create(context.Background(), organizationID, actorID, &Partner{
		Code:                    "CASUAL01",
		LegalName:               "散客测试公司",
		UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		IsCasual:                true,
		Roles:                   []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
	})
	if err != nil {
		t.Fatalf("Create() casual partner error = %v", err)
	}
	if !created.IsCasual || repo.created == nil || !repo.created.IsCasual {
		t.Fatalf("预期创建散客伙伴 IsCasual = true，实际 created=%v, repo.created=%v", created.IsCasual, repo.created.IsCasual)
	}

	// 2. 更新为正式伙伴（转正）
	updated, err := usecase.Update(context.Background(), organizationID, actorID, created.ID, &Partner{
		ID:                      created.ID,
		Code:                    "CASUAL01",
		LegalName:               "散客测试公司（已转正）",
		UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
		IsCasual:                false,
		Roles:                   []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
	})
	if err != nil {
		t.Fatalf("Update() partner error = %v", err)
	}
	if updated.IsCasual || repo.updated == nil || repo.updated.IsCasual {
		t.Fatalf("预期更新伙伴转正 IsCasual = false，实际 updated=%v, repo.updated=%v", updated.IsCasual, repo.updated.IsCasual)
	}
}

func TestPartnerImportForcesIsCasualFalse(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo)
	organizationID := uuid.New()
	actorID := uuid.New()

	items := []*Partner{
		{
			Code:                    "IMP01",
			LegalName:               "导入客户一",
			UnifiedSocialCreditCode: "91310000MA1FL7A21Q",
			IsCasual:                true, // 试图导入为散客
			Roles:                   []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
		},
		{
			Code:                    "IMP02",
			LegalName:               "导入客户二",
			UnifiedSocialCreditCode: "91310000MA1FL7A22Q",
			IsCasual:                false,
			Roles:                   []*PartnerRole{{Type: PartnerRoleCustomer, Enabled: true}},
		},
	}

	result, err := usecase.Import(context.Background(), organizationID, actorID, PartnerImportInput{
		Source: "test.xlsx",
		Mode:   PartnerImportCreateOnly,
		Items:  items,
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.CreatedCount != 2 {
		t.Fatalf("预期导入 2 条，实际=%d", result.CreatedCount)
	}
	if len(repo.importedItems) != 2 {
		t.Fatalf("预期传给 repo 的 items 数量为 2，实际=%d", len(repo.importedItems))
	}
	for i, item := range repo.importedItems {
		if item.IsCasual {
			t.Fatalf("第 %d 条导入项目 IsCasual 应被强制设为 false，实际为 true", i)
		}
	}
}

func TestPartnerListFiltersByIsCasual(t *testing.T) {
	repo := &partnerRepoStub{}
	usecase := NewPartnerUsecase(repo)
	organizationIDs := []uuid.UUID{uuid.New()}

	// 1. 过滤散客
	isCasualTrue := true
	_, err := usecase.List(context.Background(), organizationIDs, PartnerListOptions{
		Page:     1,
		PageSize: 20,
		IsCasual: &isCasualTrue,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repo.listOptions.IsCasual == nil || *repo.listOptions.IsCasual != true {
		t.Fatalf("预期传递给 repo 的 IsCasual 为 true，实际=%v", repo.listOptions.IsCasual)
	}

	// 2. 过滤正式伙伴
	isCasualFalse := false
	_, err = usecase.List(context.Background(), organizationIDs, PartnerListOptions{
		Page:     1,
		PageSize: 20,
		IsCasual: &isCasualFalse,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repo.listOptions.IsCasual == nil || *repo.listOptions.IsCasual != false {
		t.Fatalf("预期传递给 repo 的 IsCasual 为 false，实际=%v", repo.listOptions.IsCasual)
	}

	// 3. 不带过滤
	_, err = usecase.List(context.Background(), organizationIDs, PartnerListOptions{
		Page:     1,
		PageSize: 20,
		IsCasual: nil,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repo.listOptions.IsCasual != nil {
		t.Fatalf("预期传递给 repo 的 IsCasual 为 nil，实际=%v", repo.listOptions.IsCasual)
	}
}

var _ PartnerRepo = (*partnerRepoStub)(nil)
