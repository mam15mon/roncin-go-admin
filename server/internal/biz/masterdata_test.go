package biz

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

type masterDataRepoStub struct {
	listOptions  MasterDataListOptions
	created      *MasterDataItem
	updated      *MasterDataItem
	createAudit  *AuditEvent
	updateAudit  *AuditEvent
	importAudit  *AuditEvent
	importOrgID  uuid.UUID
	importMode   MasterDataImportMode
	importItems  []*MasterDataItem
	importResult *MasterDataImportResult
	importErr    error
}

func (s *masterDataRepoStub) List(_ context.Context, _ uuid.UUID, options MasterDataListOptions) (*MasterDataList, error) {
	s.listOptions = options
	return &MasterDataList{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *masterDataRepoStub) ListEnabled(context.Context, uuid.UUID) ([]*MasterDataItem, error) {
	return nil, nil
}

func (s *masterDataRepoStub) Create(_ context.Context, organizationID uuid.UUID, input *MasterDataItem, audit *AuditEvent) (*MasterDataItem, error) {
	s.created = input
	s.createAudit = audit
	input.OrganizationID = organizationID
	input.ID = uuid.New()
	audit.Details["master_data.id"] = input.ID.String()
	return input, nil
}

func (s *masterDataRepoStub) Update(_ context.Context, organizationID, id uuid.UUID, input *MasterDataItem, audit *AuditEvent) (*MasterDataItem, error) {
	s.updated = input
	s.updateAudit = audit
	input.OrganizationID = organizationID
	input.ID = id
	input.Code = "40HC"
	return input, nil
}

func (s *masterDataRepoStub) Import(_ context.Context, organizationID uuid.UUID, mode MasterDataImportMode, items []*MasterDataItem, audit *AuditEvent) (*MasterDataImportResult, error) {
	s.importOrgID = organizationID
	s.importMode = mode
	s.importItems = items
	s.importAudit = audit
	if s.importErr != nil {
		return nil, s.importErr
	}
	result := s.importResult
	if result == nil {
		result = &MasterDataImportResult{Items: items, Created: len(items)}
	}
	audit.Details["created"] = fmt.Sprintf("%d", result.Created)
	audit.Details["updated"] = fmt.Sprintf("%d", result.Updated)
	return result, nil
}

func TestDefaultOrderOptions(t *testing.T) {
	items := DefaultOrderOptions()
	if len(items) != 42 {
		t.Fatalf("DefaultOrderOptions() count = %d, want 42", len(items))
	}

	counts := map[MasterDataKind]int{}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.Kind != MasterDataKindContainerSpec && item.Kind != MasterDataKindServiceType && item.Kind != MasterDataKindCargoCategory {
			t.Fatalf("unexpected kind %q", item.Kind)
		}
		key := string(item.Kind) + "/" + item.Code
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate option %q", key)
		}
		seen[key] = struct{}{}
		counts[item.Kind]++
		if item.Code == "" || item.Name == "" || item.Source != "system" || item.SortOrder <= 0 || !item.Enabled {
			t.Fatalf("invalid default option: %#v", item)
		}
	}
	if counts[MasterDataKindContainerSpec] != 18 || counts[MasterDataKindServiceType] != 19 || counts[MasterDataKindCargoCategory] != 5 {
		t.Fatalf("option counts = %#v", counts)
	}
}

func TestDefaultCountryOptions(t *testing.T) {
	countries := DefaultCountryOptions()
	if len(countries) < 50 {
		t.Fatalf("DefaultCountryOptions() count = %d, expected >= 50", len(countries))
	}

	seen := make(map[string]struct{}, len(countries))
	for _, c := range countries {
		if c.Kind != MasterDataKindCountry {
			t.Fatalf("unexpected kind %q", c.Kind)
		}
		if _, exists := seen[c.Code]; exists {
			t.Fatalf("duplicate country code %q", c.Code)
		}
		seen[c.Code] = struct{}{}
		if c.Code == "" || c.Name == "" || c.NameEN == nil || *c.NameEN == "" || !c.Enabled {
			t.Fatalf("invalid country item: %#v", c)
		}
		if c.Attributes.Continent == nil || *c.Attributes.Continent == "" {
			t.Fatalf("missing continent for country %q", c.Code)
		}
		if c.Attributes.CurrencyCode == nil || *c.Attributes.CurrencyCode == "" {
			t.Fatalf("missing currencyCode for country %q", c.Code)
		}
	}
}

func TestMasterDataCreateNormalizesAndAudits(t *testing.T) {
	repo := &masterDataRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewMasterDataUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{Kind: MasterDataKindCurrency, Code: " cny ", Name: " 人民币 ", NameEN: stringPtr(" Renminbi "), Source: " ", SortOrder: 10})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Code != "CNY" || created.Name != "人民币" || created.NameEN == nil || *created.NameEN != "Renminbi" || created.Source != "manual" {
		t.Fatalf("normalized item = %#v", created)
	}
	if repo.createAudit == nil || repo.createAudit.Action != "master_data.create" {
		t.Fatalf("audit event = %#v", repo.createAudit)
	}
}

func TestMasterDataRejectsInvalidTEUFactor(t *testing.T) {
	usecase := NewMasterDataUsecase(&masterDataRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	if _, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{Kind: MasterDataKindCurrency, Code: "CNY", Name: "人民币", TEUFactor: stringPtr("1")}); err != ErrMasterDataInvalidArgument {
		t.Fatalf("currency TEU factor error = %v, want ErrMasterDataInvalidArgument", err)
	}
	if _, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{Kind: MasterDataKindContainerSpec, Code: "40HC", Name: "40尺高柜", TEUFactor: stringPtr("not-number")}); err != ErrMasterDataInvalidArgument {
		t.Fatalf("invalid TEU factor error = %v, want ErrMasterDataInvalidArgument", err)
	}
}

func TestMasterDataListRejectsInvalidPage(t *testing.T) {
	usecase := NewMasterDataUsecase(&masterDataRepoStub{}, &auditRepoStub{})
	if _, err := usecase.List(context.Background(), uuid.New(), MasterDataListOptions{Page: 0, PageSize: 20}); err != ErrMasterDataInvalidArgument {
		t.Fatalf("List() error = %v, want ErrMasterDataInvalidArgument", err)
	}
}

// 测试 Import 批量导入：将批次 kind/source 写入每项，编码大写，调用 repo 并记录 master_data.import 审计
func TestMasterDataImportNormalizesAndAudits(t *testing.T) {
	repo := &masterDataRepoStub{
		importResult: &MasterDataImportResult{
			Created: 1,
			Updated: 1,
		},
	}
	audit := &auditRepoStub{}
	usecase := NewMasterDataUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	input := MasterDataImportInput{
		Kind:   MasterDataKindCurrency,
		Source: "  custom_import  ",
		Mode:   MasterDataImportModeCreateOnly,
		Items: []*MasterDataItem{
			{
				Kind:   MasterDataKindCountry, // 应被批次 kind 覆盖
				Code:   " cny ",
				Name:   " 人民币 ",
				Source: " other ", // 应被批次 source 覆盖
			},
			{
				Code: " usd ",
				Name: " 美元 ",
			},
		},
	}

	result, err := usecase.Import(context.Background(), organizationID, actorID, input)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// 精确断言 repo 接收到的参数及字段规格化
	if repo.importOrgID != organizationID {
		t.Fatalf("repo.importOrgID = %v, want %v", repo.importOrgID, organizationID)
	}
	if repo.importMode != MasterDataImportModeCreateOnly {
		t.Fatalf("repo.importMode = %v, want %v", repo.importMode, MasterDataImportModeCreateOnly)
	}
	if len(repo.importItems) != 2 {
		t.Fatalf("repo.importItems count = %d, want 2", len(repo.importItems))
	}
	if repo.importItems[0].Code != "CNY" || repo.importItems[0].Kind != MasterDataKindCurrency || repo.importItems[0].Source != "custom_import" || repo.importItems[0].Name != "人民币" {
		t.Fatalf("repo.importItems[0] = %#v", repo.importItems[0])
	}
	if repo.importItems[1].Code != "USD" || repo.importItems[1].Kind != MasterDataKindCurrency || repo.importItems[1].Source != "custom_import" || repo.importItems[1].Name != "美元" {
		t.Fatalf("repo.importItems[1] = %#v", repo.importItems[1])
	}

	// 精确断言返回的导入结果
	if result.Created != 1 || result.Updated != 1 {
		t.Fatalf("result = %#v, want Created=1, Updated=1", result)
	}

	// 精确断言审计日志及详细字段
	if repo.importAudit == nil {
		t.Fatal("import audit event is nil")
	}
	event := repo.importAudit
	if event.Action != "master_data.import" {
		t.Fatalf("audit action = %q, want master_data.import", event.Action)
	}
	if event.OrganizationID == nil || *event.OrganizationID != organizationID {
		t.Fatalf("audit organizationID = %v, want %v", event.OrganizationID, organizationID)
	}
	if event.UserID == nil || *event.UserID != actorID {
		t.Fatalf("audit userID = %v, want %v", event.UserID, actorID)
	}
	if event.Details["master_data.kind"] != string(MasterDataKindCurrency) {
		t.Fatalf("audit details[master_data.kind] = %q, want %q", event.Details["master_data.kind"], MasterDataKindCurrency)
	}
	if event.Details["source"] != "custom_import" {
		t.Fatalf("audit details[source] = %q, want custom_import", event.Details["source"])
	}
	if event.Details["mode"] != string(MasterDataImportModeCreateOnly) {
		t.Fatalf("audit details[mode] = %q, want %q", event.Details["mode"], MasterDataImportModeCreateOnly)
	}
	if event.Details["created"] != "1" {
		t.Fatalf("audit details[created] = %q, want 1", event.Details["created"])
	}
	if event.Details["updated"] != "1" {
		t.Fatalf("audit details[updated] = %q, want 1", event.Details["updated"])
	}
}

// 测试 Import 各种非法参数返回 ErrMasterDataInvalidArgument：空批次、超过500条、重复编码（含大小写空格）、非法 kind、非法 mode、空 source
func TestMasterDataImportRejectsInvalidArguments(t *testing.T) {
	usecase := NewMasterDataUsecase(&masterDataRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	tests := []struct {
		name  string
		input MasterDataImportInput
	}{
		{
			name: "空批次 items",
			input: MasterDataImportInput{
				Kind:   MasterDataKindCurrency,
				Source: "excel",
				Mode:   MasterDataImportModeCreateOnly,
				Items:  []*MasterDataItem{},
			},
		},
		{
			name: "超过500条 items",
			input: MasterDataImportInput{
				Kind:   MasterDataKindCurrency,
				Source: "excel",
				Mode:   MasterDataImportModeCreateOnly,
				Items: func() []*MasterDataItem {
					items := make([]*MasterDataItem, 501)
					for i := 0; i < 501; i++ {
						items[i] = &MasterDataItem{Code: fmt.Sprintf("CODE%d", i), Name: "测试"}
					}
					return items
				}(),
			},
		},
		{
			name: "重复编码（完全相同）",
			input: MasterDataImportInput{
				Kind:   MasterDataKindCurrency,
				Source: "excel",
				Mode:   MasterDataImportModeCreateOnly,
				Items: []*MasterDataItem{
					{Code: "CNY", Name: "人民币1"},
					{Code: "CNY", Name: "人民币2"},
				},
			},
		},
		{
			name: "重复编码（大小写及前后空格）",
			input: MasterDataImportInput{
				Kind:   MasterDataKindCurrency,
				Source: "excel",
				Mode:   MasterDataImportModeCreateOnly,
				Items: []*MasterDataItem{
					{Code: " cny ", Name: "人民币1"},
					{Code: "CNY", Name: "人民币2"},
				},
			},
		},
		{
			name: "非法 kind",
			input: MasterDataImportInput{
				Kind:   MasterDataKind("invalid_kind"),
				Source: "excel",
				Mode:   MasterDataImportModeCreateOnly,
				Items: []*MasterDataItem{
					{Code: "CNY", Name: "人民币"},
				},
			},
		},
		{
			name: "非法 mode",
			input: MasterDataImportInput{
				Kind:   MasterDataKindCurrency,
				Source: "excel",
				Mode:   MasterDataImportMode("invalid_mode"),
				Items: []*MasterDataItem{
					{Code: "CNY", Name: "人民币"},
				},
			},
		},
		{
			name: "空 source（纯空格）",
			input: MasterDataImportInput{
				Kind:   MasterDataKindCurrency,
				Source: "   ",
				Mode:   MasterDataImportModeCreateOnly,
				Items: []*MasterDataItem{
					{Code: "CNY", Name: "人民币"},
				},
			},
		},
		{
			name: "包含 nil item",
			input: MasterDataImportInput{
				Kind:   MasterDataKindCurrency,
				Source: "excel",
				Mode:   MasterDataImportModeCreateOnly,
				Items: []*MasterDataItem{
					nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := usecase.Import(context.Background(), organizationID, actorID, tt.input)
			if err != ErrMasterDataInvalidArgument {
				t.Fatalf("Import() error = %v, want %v", err, ErrMasterDataInvalidArgument)
			}
		})
	}
}

// 验证字段专属规则：ParentCode 只允许 region，TEUFactor 只允许 container_spec
func TestMasterDataFieldConstraints(t *testing.T) {
	usecase := NewMasterDataUsecase(&masterDataRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	// 1. ParentCode 测试：只允许 region
	validParentCodeKinds := []MasterDataKind{MasterDataKindRegion}
	for _, kind := range validParentCodeKinds {
		_, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{
			Kind:       kind,
			Code:       "TEST",
			Name:       "测试",
			ParentCode: stringPtr("PARENT"),
		})
		if err != nil {
			t.Fatalf("Create() with ParentCode on valid kind %v error = %v", kind, err)
		}
	}

	invalidParentCodeKinds := []MasterDataKind{
		MasterDataKindCurrency,
		MasterDataKindCountry,
		MasterDataKindContainerSpec,
		MasterDataKindServiceType,
		MasterDataKindCargoCategory,
	}
	for _, kind := range invalidParentCodeKinds {
		_, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{
			Kind:       kind,
			Code:       "TEST",
			Name:       "测试",
			ParentCode: stringPtr("PARENT"),
		})
		if err != ErrMasterDataInvalidArgument {
			t.Fatalf("Create() with ParentCode on invalid kind %v error = %v, want %v", kind, err, ErrMasterDataInvalidArgument)
		}
	}

	// 2. TEUFactor 测试：只允许 container_spec
	_, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{
		Kind:      MasterDataKindContainerSpec,
		Code:      "20GP",
		Name:      "20尺普柜",
		TEUFactor: stringPtr("1.0"),
	})
	if err != nil {
		t.Fatalf("Create() with TEUFactor on container_spec error = %v", err)
	}

	invalidTEUFactorKinds := []MasterDataKind{
		MasterDataKindCurrency,
		MasterDataKindCountry,
		MasterDataKindRegion,
		MasterDataKindServiceType,
		MasterDataKindCargoCategory,
	}
	for _, kind := range invalidTEUFactorKinds {
		_, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{
			Kind:      kind,
			Code:      "TEST",
			Name:      "测试",
			TEUFactor: stringPtr("1.0"),
		})
		if err != ErrMasterDataInvalidArgument {
			t.Fatalf("Create() with TEUFactor on invalid kind %v error = %v, want %v", kind, err, ErrMasterDataInvalidArgument)
		}
	}
}

// 测试 upsert 模式原样传仓储
func TestMasterDataImportUpsertModePassThrough(t *testing.T) {
	repo := &masterDataRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewMasterDataUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	input := MasterDataImportInput{
		Kind:   MasterDataKindContainerSpec,
		Source: "edi",
		Mode:   MasterDataImportModeUpsert,
		Items: []*MasterDataItem{
			{
				Code:      "40hc",
				Name:      "40尺高柜",
				TEUFactor: stringPtr("2"),
			},
		},
	}

	_, err := usecase.Import(context.Background(), organizationID, actorID, input)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if repo.importMode != MasterDataImportModeUpsert {
		t.Fatalf("repo.importMode = %v, want %v", repo.importMode, MasterDataImportModeUpsert)
	}
	if repo.importOrgID != organizationID {
		t.Fatalf("repo.importOrgID = %v, want %v", repo.importOrgID, organizationID)
	}
	if len(repo.importItems) != 1 || repo.importItems[0].Code != "40HC" {
		t.Fatalf("repo.importItems = %#v", repo.importItems)
	}
}

func TestDefaultOrderOptionsIncludesContainerSpecs(t *testing.T) {
	options := DefaultOrderOptions()
	want := map[string]string{"20GP": "1", "40HQ": "2"}
	for _, option := range options {
		if option.Kind != MasterDataKindContainerSpec {
			continue
		}
		factor, exists := want[option.Code]
		if !exists {
			continue
		}
		if option.TEUFactor == nil || *option.TEUFactor != factor {
			t.Fatalf("箱型 %s TEU 系数 = %v，期望 %s", option.Code, option.TEUFactor, factor)
		}
		delete(want, option.Code)
	}
	if len(want) != 0 {
		t.Fatalf("默认订单选项缺少箱型：%v", want)
	}
}

var _ MasterDataRepo = (*masterDataRepoStub)(nil)
