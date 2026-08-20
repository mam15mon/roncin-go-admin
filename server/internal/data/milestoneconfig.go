package data

import (
	"context"
	"sort"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/milestonetemplate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/milestonetemplateitem"

	"github.com/google/uuid"
)

type milestoneConfigRepo struct{ data *Data }

func NewMilestoneConfigRepo(data *Data) biz.MilestoneConfigRepo {
	return &milestoneConfigRepo{data: data}
}

func (r *milestoneConfigRepo) ListMilestoneTemplates(ctx context.Context, organizationID uuid.UUID, options biz.MilestoneTemplateListOptions) ([]*biz.MilestoneTemplate, error) {
	query := r.data.db.MilestoneTemplate.Query().
		Where(milestonetemplate.OrganizationIDEQ(organizationID)).
		WithItems(func(query *ent.MilestoneTemplateItemQuery) {
			query.Order(milestonetemplateitem.BySortOrder(), milestonetemplateitem.ByCode())
		})
	if options.BusinessType != "" {
		query.Where(milestonetemplate.BusinessTypeEQ(milestonetemplate.BusinessType(options.BusinessType)))
	}
	if options.TradeTerm != nil {
		query.Where(milestonetemplate.TradeTermEQ(*options.TradeTerm))
	}
	if options.Published != nil {
		if *options.Published {
			query.Where(milestonetemplate.PublishedAtNotNil())
		} else {
			query.Where(milestonetemplate.PublishedAtIsNil())
		}
	}
	items, err := query.Order(milestonetemplate.ByBusinessType(), milestonetemplate.ByTradeTerm(), milestonetemplate.ByCode(), milestonetemplate.ByVersion()).All(ctx)
	if err != nil {
		return nil, err
	}
	return milestoneTemplatesToBiz(items), nil
}

func (r *milestoneConfigRepo) CreateMilestoneTemplate(ctx context.Context, organizationID uuid.UUID, input *biz.MilestoneTemplate) (*biz.MilestoneTemplate, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	template, err := tx.MilestoneTemplate.Create().
		SetOrganizationID(organizationID).
		SetCode(input.Code).
		SetName(input.Name).
		SetBusinessType(milestonetemplate.BusinessType(input.BusinessType)).
		SetTradeTerm(input.TradeTerm).
		SetVersion(input.Version).
		SetIsDefault(false).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrMilestoneTemplateExists
		}
		return nil, err
	}
	for _, item := range input.Items {
		builder := tx.MilestoneTemplateItem.Create().
			SetTemplateID(template.ID).
			SetCode(item.Code).
			SetLabel(item.Label).
			SetNillableDescription(item.Description).
			SetNillableCategory(item.Category).
			SetSortOrder(item.SortOrder).
			SetEnabled(item.Enabled).
			SetDependsOn(item.DependsOn)
		if _, err := builder.Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findMilestoneTemplate(ctx, organizationID, template.ID)
}

func (r *milestoneConfigRepo) PublishMilestoneTemplate(ctx context.Context, organizationID, id uuid.UUID, isDefault bool, publishedAt time.Time) (*biz.MilestoneTemplate, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	template, err := tx.MilestoneTemplate.Query().Where(milestonetemplate.IDEQ(id), milestonetemplate.OrganizationIDEQ(organizationID), milestonetemplate.EnabledEQ(true)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrMilestoneTemplateNotFound
		}
		return nil, err
	}
	if template.PublishedAt != nil {
		_ = tx.Rollback()
		return nil, biz.ErrMilestoneTemplateInvalid
	}
	if isDefault {
		if _, err := tx.MilestoneTemplate.Update().Where(milestonetemplate.OrganizationIDEQ(organizationID), milestonetemplate.BusinessTypeEQ(template.BusinessType), milestonetemplate.TradeTermEQ(template.TradeTerm), milestonetemplate.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if _, err := template.Update().SetPublishedAt(publishedAt).SetIsDefault(isDefault).Save(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrMilestoneTemplateDefaultConflict
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findMilestoneTemplate(ctx, organizationID, id)
}

func (r *milestoneConfigRepo) SetDefaultMilestoneTemplate(ctx context.Context, organizationID, id uuid.UUID) (*biz.MilestoneTemplate, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	template, err := tx.MilestoneTemplate.Query().Where(milestonetemplate.IDEQ(id), milestonetemplate.OrganizationIDEQ(organizationID), milestonetemplate.EnabledEQ(true), milestonetemplate.PublishedAtNotNil()).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrMilestoneTemplateNotFound
		}
		return nil, err
	}
	if !template.IsDefault {
		if _, err := tx.MilestoneTemplate.Update().Where(milestonetemplate.OrganizationIDEQ(organizationID), milestonetemplate.BusinessTypeEQ(template.BusinessType), milestonetemplate.TradeTermEQ(template.TradeTerm), milestonetemplate.IsDefaultEQ(true), milestonetemplate.IDNEQ(id)).SetIsDefault(false).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := template.Update().SetIsDefault(true).Save(ctx); err != nil {
			_ = tx.Rollback()
			if ent.IsConstraintError(err) {
				return nil, biz.ErrMilestoneTemplateDefaultConflict
			}
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findMilestoneTemplate(ctx, organizationID, id)
}

func (r *milestoneConfigRepo) findMilestoneTemplate(ctx context.Context, organizationID, id uuid.UUID) (*biz.MilestoneTemplate, error) {
	item, err := r.data.db.MilestoneTemplate.Query().
		Where(milestonetemplate.IDEQ(id), milestonetemplate.OrganizationIDEQ(organizationID)).
		WithItems(func(query *ent.MilestoneTemplateItemQuery) {
			query.Order(milestonetemplateitem.BySortOrder(), milestonetemplateitem.ByCode())
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrMilestoneTemplateNotFound
		}
		return nil, err
	}
	return milestoneTemplateToBiz(item), nil
}

func milestoneTemplatesToBiz(items []*ent.MilestoneTemplate) []*biz.MilestoneTemplate {
	result := make([]*biz.MilestoneTemplate, 0, len(items))
	for _, item := range items {
		result = append(result, milestoneTemplateToBiz(item))
	}
	return result
}

func milestoneTemplateToBiz(item *ent.MilestoneTemplate) *biz.MilestoneTemplate {
	result := &biz.MilestoneTemplate{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		Code:           item.Code,
		Name:           item.Name,
		BusinessType:   biz.BusinessType(item.BusinessType),
		TradeTerm:      item.TradeTerm,
		Version:        item.Version,
		IsDefault:      item.IsDefault,
		PublishedAt:    item.PublishedAt,
		Enabled:        item.Enabled,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	for _, child := range item.Edges.Items {
		result.Items = append(result.Items, &biz.MilestoneTemplateItem{
			ID:          child.ID,
			Code:        child.Code,
			Label:       child.Label,
			Description: child.Description,
			Category:    child.Category,
			SortOrder:   child.SortOrder,
			Enabled:     child.Enabled,
			DependsOn:   append([]string(nil), child.DependsOn...),
		})
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		if result.Items[i].SortOrder == result.Items[j].SortOrder {
			return result.Items[i].Code < result.Items[j].Code
		}
		return result.Items[i].SortOrder < result.Items[j].SortOrder
	})
	return result
}

var _ biz.MilestoneConfigRepo = (*milestoneConfigRepo)(nil)
