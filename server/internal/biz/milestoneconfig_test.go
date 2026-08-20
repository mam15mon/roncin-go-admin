package biz

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

type milestoneConfigRepoStub struct {
	templates            []*MilestoneTemplate
	createdTemplate      *MilestoneTemplate
	publishedTemplate    *MilestoneTemplate
	defaultTemplate      *MilestoneTemplate
	lastListOrgID        uuid.UUID
	lastListOptions      MilestoneTemplateListOptions
	lastCreateOrgID      uuid.UUID
	lastPublishOrgID     uuid.UUID
	lastPublishID        uuid.UUID
	lastPublishIsDefault bool
	lastPublishTime      time.Time
	lastSetDefaultOrgID  uuid.UUID
	lastSetDefaultID     uuid.UUID
}

func (s *milestoneConfigRepoStub) ListMilestoneTemplates(_ context.Context, organizationID uuid.UUID, options MilestoneTemplateListOptions) ([]*MilestoneTemplate, error) {
	s.lastListOrgID = organizationID
	s.lastListOptions = options
	return s.templates, nil
}

func (s *milestoneConfigRepoStub) CreateMilestoneTemplate(_ context.Context, organizationID uuid.UUID, input *MilestoneTemplate) (*MilestoneTemplate, error) {
	s.lastCreateOrgID = organizationID
	s.createdTemplate = input
	input.OrganizationID = organizationID
	if input.ID == uuid.Nil {
		input.ID = uuid.New()
	}
	return input, nil
}

func (s *milestoneConfigRepoStub) PublishMilestoneTemplate(_ context.Context, organizationID, id uuid.UUID, isDefault bool, now time.Time) (*MilestoneTemplate, error) {
	s.lastPublishOrgID = organizationID
	s.lastPublishID = id
	s.lastPublishIsDefault = isDefault
	s.lastPublishTime = now
	if s.publishedTemplate != nil {
		return s.publishedTemplate, nil
	}
	return &MilestoneTemplate{
		ID:             id,
		OrganizationID: organizationID,
		Code:           "MS_DEFAULT",
		Version:        1,
		IsDefault:      isDefault,
		PublishedAt:    &now,
	}, nil
}

func (s *milestoneConfigRepoStub) SetDefaultMilestoneTemplate(_ context.Context, organizationID, id uuid.UUID) (*MilestoneTemplate, error) {
	s.lastSetDefaultOrgID = organizationID
	s.lastSetDefaultID = id
	if s.defaultTemplate != nil {
		return s.defaultTemplate, nil
	}
	return &MilestoneTemplate{
		ID:             id,
		OrganizationID: organizationID,
		Code:           "MS_DEFAULT",
		Version:        1,
		IsDefault:      true,
	}, nil
}

var _ MilestoneConfigRepo = (*milestoneConfigRepoStub)(nil)

func TestMilestoneConfigCreateNormalizesAndAudits(t *testing.T) {
	repo := &milestoneConfigRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewMilestoneConfigUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	desc := "  里程碑描述  "
	category := "  customs_clearance  "

	input := &MilestoneTemplate{
		Code:         "  ms_ocean_export  ",
		Name:         "  海运出口里程碑  ",
		BusinessType: BusinessTypeSE,
		TradeTerm:    "  fob  ",
		Version:      1,
		Enabled:      true,
		Items: []*MilestoneTemplateItem{
			{
				Code:        "  booking_confirm  ",
				Label:       "  订舱确认  ",
				Description: &desc,
				Category:    &category,
				SortOrder:   1,
				Enabled:     true,
				DependsOn:   []string{"  cargo_ready  ", "cargo_ready", "  CARGO_READY  "},
			},
			{
				Code:      "  cargo_ready  ",
				Label:     "  备货完成  ",
				SortOrder: 0,
				Enabled:   true,
				DependsOn: []string{},
			},
		},
	}

	created, err := usecase.Create(context.Background(), organizationID, actorID, input)
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	if created.Code != "MS_OCEAN_EXPORT" {
		t.Fatalf("created.Code = %q, want MS_OCEAN_EXPORT", created.Code)
	}
	if created.TradeTerm != "FOB" {
		t.Fatalf("created.TradeTerm = %q, want FOB", created.TradeTerm)
	}
	if created.Name != "海运出口里程碑" {
		t.Fatalf("created.Name = %q, want 海运出口里程碑", created.Name)
	}

	if len(created.Items) != 2 {
		t.Fatalf("created.Items len = %d, want 2", len(created.Items))
	}

	item0 := created.Items[0]
	if item0.Code != "BOOKING_CONFIRM" {
		t.Fatalf("item0.Code = %q, want BOOKING_CONFIRM", item0.Code)
	}
	if item0.Label != "订舱确认" {
		t.Fatalf("item0.Label = %q, want 订舱确认", item0.Label)
	}
	if item0.Description == nil || *item0.Description != "里程碑描述" {
		t.Fatalf("item0.Description = %v, want 里程碑描述", item0.Description)
	}
	if item0.Category == nil || *item0.Category != "CUSTOMS_CLEARANCE" {
		t.Fatalf("item0.Category = %v, want CUSTOMS_CLEARANCE", item0.Category)
	}
	wantDependsOn := []string{"CARGO_READY"}
	if !reflect.DeepEqual(item0.DependsOn, wantDependsOn) {
		t.Fatalf("item0.DependsOn = %v, want %v", item0.DependsOn, wantDependsOn)
	}

	item1 := created.Items[1]
	if item1.Code != "CARGO_READY" {
		t.Fatalf("item1.Code = %q, want CARGO_READY", item1.Code)
	}
	if item1.Label != "备货完成" {
		t.Fatalf("item1.Label = %q, want 备货完成", item1.Label)
	}

	if repo.createdTemplate != created {
		t.Fatalf("repo.createdTemplate = %v, want %v", repo.createdTemplate, created)
	}
	if repo.lastCreateOrgID != organizationID {
		t.Fatalf("repo.lastCreateOrgID = %v, want %v", repo.lastCreateOrgID, organizationID)
	}

	if len(audit.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(audit.events))
	}
	event := audit.events[0]
	if event.Action != "milestone_template.create" {
		t.Fatalf("audit event action = %q, want milestone_template.create", event.Action)
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
		"milestone_template.id":   created.ID.String(),
		"milestone_template.code": "MS_OCEAN_EXPORT",
		"version":                 "1",
	}
	if !reflect.DeepEqual(event.Details, wantDetails) {
		t.Fatalf("audit event details = %#v, want %#v", event.Details, wantDetails)
	}
}

func TestMilestoneConfigCreateRejections(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()

	tests := []struct {
		name     string
		orgID    uuid.UUID
		template *MilestoneTemplate
	}{
		{
			name:  "nil organization ID",
			orgID: uuid.Nil,
			template: &MilestoneTemplate{
				Code:         "MS_TEST",
				Name:         "测试模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*MilestoneTemplateItem{
					{Code: "M1", Label: "节点1", Enabled: true},
				},
			},
		},
		{
			name:     "nil template",
			orgID:    organizationID,
			template: nil,
		},
		{
			name:  "dependency does not exist",
			orgID: organizationID,
			template: &MilestoneTemplate{
				Code:         "MS_TEST",
				Name:         "测试模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*MilestoneTemplateItem{
					{Code: "M1", Label: "节点1", Enabled: true, DependsOn: []string{"NON_EXISTENT"}},
				},
			},
		},
		{
			name:  "dependency on self",
			orgID: organizationID,
			template: &MilestoneTemplate{
				Code:         "MS_TEST",
				Name:         "测试模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*MilestoneTemplateItem{
					{Code: "M1", Label: "节点1", Enabled: true, DependsOn: []string{"m1"}},
				},
			},
		},
		{
			name:  "dependency on disabled item",
			orgID: organizationID,
			template: &MilestoneTemplate{
				Code:         "MS_TEST",
				Name:         "测试模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*MilestoneTemplateItem{
					{Code: "M1", Label: "节点1", Enabled: true, DependsOn: []string{"M2"}},
					{Code: "M2", Label: "节点2", Enabled: false},
				},
			},
		},
		{
			name:  "A->B->A cycle",
			orgID: organizationID,
			template: &MilestoneTemplate{
				Code:         "MS_TEST",
				Name:         "测试模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*MilestoneTemplateItem{
					{Code: "A", Label: "节点A", Enabled: true, DependsOn: []string{"B"}},
					{Code: "B", Label: "节点B", Enabled: true, DependsOn: []string{"A"}},
				},
			},
		},
		{
			name:  "duplicate item code (case-insensitive/trimmed)",
			orgID: organizationID,
			template: &MilestoneTemplate{
				Code:         "MS_TEST",
				Name:         "测试模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*MilestoneTemplateItem{
					{Code: "NODE_1", Label: "节点1", Enabled: true},
					{Code: "  node_1  ", Label: "节点1重复", Enabled: true},
				},
			},
		},
		{
			name:  "no enabled items",
			orgID: organizationID,
			template: &MilestoneTemplate{
				Code:         "MS_TEST",
				Name:         "测试模板",
				BusinessType: BusinessTypeSE,
				Version:      1,
				Items: []*MilestoneTemplateItem{
					{Code: "M1", Label: "节点1", Enabled: false},
					{Code: "M2", Label: "节点2", Enabled: false},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			usecase := NewMilestoneConfigUsecase(&milestoneConfigRepoStub{}, &auditRepoStub{})
			_, err := usecase.Create(context.Background(), tc.orgID, actorID, tc.template)
			if err != ErrMilestoneTemplateInvalid {
				t.Fatalf("Create() error = %v, want %v", err, ErrMilestoneTemplateInvalid)
			}
		})
	}
}

func TestMilestoneConfigListOptionsNormalization(t *testing.T) {
	organizationID := uuid.New()
	tradeTermInput := "  cif  "
	publishedTrue := true
	publishedFalse := false

	tests := []struct {
		name          string
		options       MilestoneTemplateListOptions
		wantTradeTerm *string
		wantPublished *bool
	}{
		{
			name: "normalizes trade term and preserves published true pointer",
			options: MilestoneTemplateListOptions{
				BusinessType: BusinessTypeSE,
				TradeTerm:    &tradeTermInput,
				Published:    &publishedTrue,
			},
			wantTradeTerm: stringPtr("CIF"),
			wantPublished: &publishedTrue,
		},
		{
			name: "preserves nil trade term and published false pointer",
			options: MilestoneTemplateListOptions{
				BusinessType: BusinessTypeAE,
				TradeTerm:    nil,
				Published:    &publishedFalse,
			},
			wantTradeTerm: nil,
			wantPublished: &publishedFalse,
		},
		{
			name: "preserves nil published pointer",
			options: MilestoneTemplateListOptions{
				BusinessType: BusinessTypeAI,
				TradeTerm:    stringPtr("fob"),
				Published:    nil,
			},
			wantTradeTerm: stringPtr("FOB"),
			wantPublished: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &milestoneConfigRepoStub{}
			usecase := NewMilestoneConfigUsecase(repo, &auditRepoStub{})

			_, err := usecase.List(context.Background(), organizationID, tc.options)
			if err != nil {
				t.Fatalf("List() error = %v, want nil", err)
			}

			if repo.lastListOrgID != organizationID {
				t.Fatalf("repo.lastListOrgID = %v, want %v", repo.lastListOrgID, organizationID)
			}

			if tc.wantTradeTerm == nil {
				if repo.lastListOptions.TradeTerm != nil {
					t.Fatalf("repo.lastListOptions.TradeTerm = %v, want nil", *repo.lastListOptions.TradeTerm)
				}
			} else {
				if repo.lastListOptions.TradeTerm == nil || *repo.lastListOptions.TradeTerm != *tc.wantTradeTerm {
					t.Fatalf("repo.lastListOptions.TradeTerm = %v, want %v", repo.lastListOptions.TradeTerm, *tc.wantTradeTerm)
				}
			}

			if repo.lastListOptions.Published != tc.wantPublished {
				t.Fatalf("repo.lastListOptions.Published pointer = %p (%v), want %p (%v)",
					repo.lastListOptions.Published, repo.lastListOptions.Published,
					tc.wantPublished, tc.wantPublished)
			}
		})
	}

	// 列表参数不合法时必须返回明确的领域错误。
	invalidTests := []struct {
		name    string
		orgID   uuid.UUID
		options MilestoneTemplateListOptions
	}{
		{
			name:    "nil organization ID",
			orgID:   uuid.Nil,
			options: MilestoneTemplateListOptions{},
		},
		{
			name:  "invalid business type",
			orgID: organizationID,
			options: MilestoneTemplateListOptions{
				BusinessType: BusinessType("INVALID"),
			},
		},
		{
			name:  "trade term longer than 16 chars",
			orgID: organizationID,
			options: MilestoneTemplateListOptions{
				TradeTerm: stringPtr("12345678901234567"),
			},
		},
	}

	for _, tc := range invalidTests {
		t.Run(tc.name, func(t *testing.T) {
			usecase := NewMilestoneConfigUsecase(&milestoneConfigRepoStub{}, &auditRepoStub{})
			_, err := usecase.List(context.Background(), tc.orgID, tc.options)
			if err != ErrMilestoneTemplateInvalid {
				t.Fatalf("List() error = %v, want %v", err, ErrMilestoneTemplateInvalid)
			}
		})
	}
}

func TestMilestoneConfigPublish(t *testing.T) {
	repo := &milestoneConfigRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewMilestoneConfigUsecase(repo, audit)

	fixedTime := time.Date(2026, 8, 20, 14, 30, 0, 0, time.FixedZone("CST", 8*3600))
	expectedUTC := fixedTime.UTC()
	usecase.now = func() time.Time {
		return fixedTime
	}

	organizationID := uuid.New()
	actorID := uuid.New()
	templateID := uuid.New()

	published, err := usecase.Publish(context.Background(), organizationID, actorID, templateID, true)
	if err != nil {
		t.Fatalf("Publish() error = %v, want nil", err)
	}

	if repo.lastPublishOrgID != organizationID {
		t.Fatalf("repo.lastPublishOrgID = %v, want %v", repo.lastPublishOrgID, organizationID)
	}
	if repo.lastPublishID != templateID {
		t.Fatalf("repo.lastPublishID = %v, want %v", repo.lastPublishID, templateID)
	}
	if !repo.lastPublishIsDefault {
		t.Fatalf("repo.lastPublishIsDefault = false, want true")
	}
	if !repo.lastPublishTime.Equal(expectedUTC) {
		t.Fatalf("repo.lastPublishTime = %v, want %v", repo.lastPublishTime, expectedUTC)
	}
	if repo.lastPublishTime.Location() != time.UTC {
		t.Fatalf("repo.lastPublishTime.Location() = %v, want UTC", repo.lastPublishTime.Location())
	}

	if len(audit.events) != 1 {
		t.Fatalf("audit events count = %d, want 1", len(audit.events))
	}
	event := audit.events[0]
	if event.Action != "milestone_template.publish" {
		t.Fatalf("audit event action = %q, want milestone_template.publish", event.Action)
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
		"milestone_template.id":   published.ID.String(),
		"milestone_template.code": published.Code,
		"version":                 "1",
	}
	if !reflect.DeepEqual(event.Details, wantDetails) {
		t.Fatalf("audit event details = %#v, want %#v", event.Details, wantDetails)
	}

	// 发布操作拒绝空组织或空模板标识。
	if _, err := usecase.Publish(context.Background(), uuid.Nil, actorID, templateID, false); err != ErrMilestoneTemplateInvalid {
		t.Fatalf("nil orgID error = %v, want %v", err, ErrMilestoneTemplateInvalid)
	}
	if _, err := usecase.Publish(context.Background(), organizationID, actorID, uuid.Nil, false); err != ErrMilestoneTemplateInvalid {
		t.Fatalf("nil templateID error = %v, want %v", err, ErrMilestoneTemplateInvalid)
	}
}

func TestMilestoneConfigSetDefault(t *testing.T) {
	repo := &milestoneConfigRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewMilestoneConfigUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	templateID := uuid.New()

	template, err := usecase.SetDefault(context.Background(), organizationID, actorID, templateID)
	if err != nil {
		t.Fatalf("SetDefault() error = %v, want nil", err)
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
	if event.Action != "milestone_template.set_default" {
		t.Fatalf("audit event action = %q, want milestone_template.set_default", event.Action)
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
		"milestone_template.id":   template.ID.String(),
		"milestone_template.code": template.Code,
		"version":                 "1",
	}
	if !reflect.DeepEqual(event.Details, wantDetails) {
		t.Fatalf("audit event details = %#v, want %#v", event.Details, wantDetails)
	}

	// 默认版本切换拒绝空组织或空模板标识。
	if _, err := usecase.SetDefault(context.Background(), uuid.Nil, actorID, templateID); err != ErrMilestoneTemplateInvalid {
		t.Fatalf("nil orgID error = %v, want %v", err, ErrMilestoneTemplateInvalid)
	}
	if _, err := usecase.SetDefault(context.Background(), organizationID, actorID, uuid.Nil); err != ErrMilestoneTemplateInvalid {
		t.Fatalf("nil templateID error = %v, want %v", err, ErrMilestoneTemplateInvalid)
	}
}
