package biz

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	v1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrMilestoneTemplateNotFound        = errors.NotFound(v1.ErrorReason_MILESTONE_TEMPLATE_NOT_FOUND.String(), "里程碑模板不存在")
	ErrMilestoneTemplateExists          = errors.Conflict(v1.ErrorReason_MILESTONE_TEMPLATE_EXISTS.String(), "里程碑模板版本已存在")
	ErrMilestoneTemplateInvalid         = errors.BadRequest(v1.ErrorReason_MILESTONE_TEMPLATE_INVALID.String(), "里程碑模板不合法")
	ErrMilestoneTemplateDefaultConflict = errors.Conflict(v1.ErrorReason_MILESTONE_TEMPLATE_DEFAULT_CONFLICT.String(), "默认里程碑模板设置冲突")
)

type MilestoneTemplateItem struct {
	ID          uuid.UUID
	Code        string
	Label       string
	Description *string
	Category    *string
	SortOrder   int
	Enabled     bool
	DependsOn   []string
}

type MilestoneTemplate struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Name           string
	BusinessType   BusinessType
	TradeTerm      string
	Version        int
	IsDefault      bool
	PublishedAt    *time.Time
	Enabled        bool
	Items          []*MilestoneTemplateItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MilestoneTemplateListOptions struct {
	BusinessType BusinessType
	TradeTerm    *string
	Published    *bool
}

type MilestoneConfigRepo interface {
	ListMilestoneTemplates(context.Context, uuid.UUID, MilestoneTemplateListOptions) ([]*MilestoneTemplate, error)
	CreateMilestoneTemplate(context.Context, uuid.UUID, *MilestoneTemplate) (*MilestoneTemplate, error)
	PublishMilestoneTemplate(context.Context, uuid.UUID, uuid.UUID, bool, time.Time) (*MilestoneTemplate, error)
	SetDefaultMilestoneTemplate(context.Context, uuid.UUID, uuid.UUID) (*MilestoneTemplate, error)
}

type MilestoneConfigUsecase struct {
	repo  MilestoneConfigRepo
	audit AuditRepo
	now   func() time.Time
}

func NewMilestoneConfigUsecase(repo MilestoneConfigRepo, audit AuditRepo) *MilestoneConfigUsecase {
	return &MilestoneConfigUsecase{repo: repo, audit: audit, now: time.Now}
}

func (uc *MilestoneConfigUsecase) List(ctx context.Context, organizationID uuid.UUID, options MilestoneTemplateListOptions) ([]*MilestoneTemplate, error) {
	if organizationID == uuid.Nil || options.BusinessType != "" && !options.BusinessType.Valid() {
		return nil, ErrMilestoneTemplateInvalid
	}
	if options.TradeTerm != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*options.TradeTerm))
		if utf8.RuneCountInString(normalized) > 16 {
			return nil, ErrMilestoneTemplateInvalid
		}
		options.TradeTerm = &normalized
	}
	return uc.repo.ListMilestoneTemplates(ctx, organizationID, options)
}

func (uc *MilestoneConfigUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *MilestoneTemplate) (*MilestoneTemplate, error) {
	if organizationID == uuid.Nil {
		return nil, ErrMilestoneTemplateInvalid
	}
	normalized, err := normalizeMilestoneTemplate(input)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateMilestoneTemplate(ctx, organizationID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "milestone_template.create", Result: "success", Details: map[string]string{"milestone_template.id": created.ID.String(), "milestone_template.code": created.Code, "version": strconv.Itoa(created.Version)}}); err != nil {
		return nil, fmt.Errorf("write milestone template create audit: %w", err)
	}
	return created, nil
}

func (uc *MilestoneConfigUsecase) Publish(ctx context.Context, organizationID, actorID, id uuid.UUID, isDefault bool) (*MilestoneTemplate, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrMilestoneTemplateInvalid
	}
	published, err := uc.repo.PublishMilestoneTemplate(ctx, organizationID, id, isDefault, uc.now().UTC())
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "milestone_template.publish", Result: "success", Details: map[string]string{"milestone_template.id": published.ID.String(), "milestone_template.code": published.Code, "version": strconv.Itoa(published.Version)}}); err != nil {
		return nil, fmt.Errorf("write milestone template publish audit: %w", err)
	}
	return published, nil
}

func (uc *MilestoneConfigUsecase) SetDefault(ctx context.Context, organizationID, actorID, id uuid.UUID) (*MilestoneTemplate, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrMilestoneTemplateInvalid
	}
	updated, err := uc.repo.SetDefaultMilestoneTemplate(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "milestone_template.set_default", Result: "success", Details: map[string]string{"milestone_template.id": updated.ID.String(), "milestone_template.code": updated.Code, "version": strconv.Itoa(updated.Version)}}); err != nil {
		return nil, fmt.Errorf("write milestone template set default audit: %w", err)
	}
	return updated, nil
}

func normalizeMilestoneTemplate(input *MilestoneTemplate) (*MilestoneTemplate, error) {
	if input == nil {
		return nil, ErrMilestoneTemplateInvalid
	}
	output := *input
	output.Code = strings.ToUpper(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	output.TradeTerm = strings.ToUpper(strings.TrimSpace(output.TradeTerm))
	if output.Code == "" || utf8.RuneCountInString(output.Code) > 64 || output.Name == "" || utf8.RuneCountInString(output.Name) > 100 || !output.BusinessType.Valid() || utf8.RuneCountInString(output.TradeTerm) > 16 || output.Version < 1 || len(output.Items) == 0 {
		return nil, ErrMilestoneTemplateInvalid
	}

	items := make([]*MilestoneTemplateItem, 0, len(output.Items))
	byCode := make(map[string]*MilestoneTemplateItem, len(output.Items))
	for _, item := range output.Items {
		if item == nil {
			return nil, ErrMilestoneTemplateInvalid
		}
		copyItem := *item
		copyItem.Code = strings.ToUpper(strings.TrimSpace(copyItem.Code))
		copyItem.Label = strings.TrimSpace(copyItem.Label)
		copyItem.Description = normalizedOptionalString(copyItem.Description)
		copyItem.Category = normalizedUpperOptionalString(copyItem.Category)
		if copyItem.Code == "" || utf8.RuneCountInString(copyItem.Code) > 64 || copyItem.Label == "" || utf8.RuneCountInString(copyItem.Label) > 100 || copyItem.Description != nil && utf8.RuneCountInString(*copyItem.Description) > 500 || copyItem.Category != nil && utf8.RuneCountInString(*copyItem.Category) > 64 || copyItem.SortOrder < 0 {
			return nil, ErrMilestoneTemplateInvalid
		}
		if _, exists := byCode[copyItem.Code]; exists {
			return nil, ErrMilestoneTemplateInvalid
		}
		copyItem.DependsOn = normalizeMilestoneDependencies(copyItem.DependsOn)
		byCode[copyItem.Code] = &copyItem
		items = append(items, &copyItem)
	}

	hasEnabled := false
	for _, item := range items {
		if item.Enabled {
			hasEnabled = true
		}
		for _, dependencyCode := range item.DependsOn {
			dependency, exists := byCode[dependencyCode]
			if !exists || dependencyCode == item.Code || !dependency.Enabled {
				return nil, ErrMilestoneTemplateInvalid
			}
		}
	}
	if !hasEnabled || milestoneDependenciesHaveCycle(items) {
		return nil, ErrMilestoneTemplateInvalid
	}
	output.Items = items
	return &output, nil
}

func normalizeMilestoneDependencies(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.ToUpper(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func milestoneDependenciesHaveCycle(items []*MilestoneTemplateItem) bool {
	dependencies := make(map[string][]string, len(items))
	for _, item := range items {
		dependencies[item.Code] = item.DependsOn
	}
	visiting := make(map[string]bool, len(items))
	visited := make(map[string]bool, len(items))
	var visit func(string) bool
	visit = func(code string) bool {
		if visiting[code] {
			return true
		}
		if visited[code] {
			return false
		}
		visiting[code] = true
		for _, dependency := range dependencies[code] {
			if visit(dependency) {
				return true
			}
		}
		visiting[code] = false
		visited[code] = true
		return false
	}
	for code := range dependencies {
		if visit(code) {
			return true
		}
	}
	return false
}

var _ MilestoneConfigRepo = (MilestoneConfigRepo)(nil)
