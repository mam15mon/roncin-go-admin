package data

import (
	"context"
	"sort"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/statustemplate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/statustemplateitem"

	"github.com/google/uuid"
)

type orderConfigRepo struct{ data *Data }

func NewOrderConfigRepo(data *Data) biz.OrderConfigRepo { return &orderConfigRepo{data: data} }

func (r *orderConfigRepo) ListNumberRules(ctx context.Context, organizationID uuid.UUID) ([]*biz.NumberRule, error) {
	items, err := r.data.db.NumberRule.Query().Where(numberrule.OrganizationIDEQ(organizationID)).Order(numberrule.ByDocumentType()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.NumberRule, 0, len(items))
	for _, item := range items {
		result = append(result, numberRuleToBiz(item))
	}
	return result, nil
}

func (r *orderConfigRepo) CreateNumberRule(ctx context.Context, organizationID uuid.UUID, input *biz.NumberRule) (*biz.NumberRule, error) {
	created, err := r.data.db.NumberRule.Create().SetOrganizationID(organizationID).SetDocumentType(numberrule.DocumentType(input.DocumentType)).SetPrefix(input.Prefix).SetDateFormat(numberrule.DateFormat(input.DateFormat)).SetSequenceLength(input.SequenceLength).SetResetPolicy(numberrule.ResetPolicy(input.ResetPolicy)).SetEnabled(true).Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, biz.ErrNumberRuleExists
		}
		return nil, err
	}
	return numberRuleToBiz(created), nil
}

func (r *orderConfigRepo) UpdateNumberRule(ctx context.Context, organizationID, id uuid.UUID, input *biz.NumberRule) (*biz.NumberRule, error) {
	existing, err := r.data.db.NumberRule.Query().Where(numberrule.IDEQ(id), numberrule.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrNumberRuleNotFound
		}
		return nil, err
	}
	updated, err := existing.Update().SetPrefix(input.Prefix).SetDateFormat(numberrule.DateFormat(input.DateFormat)).SetSequenceLength(input.SequenceLength).SetResetPolicy(numberrule.ResetPolicy(input.ResetPolicy)).SetEnabled(input.Enabled).Save(ctx)
	if err != nil {
		return nil, err
	}
	return numberRuleToBiz(updated), nil
}

func (r *orderConfigRepo) AllocateNumber(ctx context.Context, organizationID uuid.UUID, documentType biz.DocumentType, at time.Time) (*biz.NumberRule, int64, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	lockedRule, err := tx.NumberRule.Query().Where(numberrule.OrganizationIDEQ(organizationID), numberrule.DocumentTypeEQ(numberrule.DocumentType(documentType)), numberrule.EnabledEQ(true)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, 0, biz.ErrNumberRuleNotFound
		}
		return nil, 0, err
	}
	maximum := int64(1)
	for range lockedRule.SequenceLength {
		maximum *= 10
	}
	maximum--
	periodKey := biz.NumberPeriodKey(at, biz.ResetPolicy(lockedRule.ResetPolicy))
	sequence, err := tx.NumberSequence.Query().Where(numbersequence.RuleIDEQ(lockedRule.ID), numbersequence.PeriodKeyEQ(periodKey)).Only(ctx)
	if ent.IsNotFound(err) {
		sequence, err = tx.NumberSequence.Create().SetRuleID(lockedRule.ID).SetPeriodKey(periodKey).SetCurrentValue(1).Save(ctx)
	} else if err == nil {
		if sequence.CurrentValue >= maximum {
			_ = tx.Rollback()
			return nil, 0, biz.ErrNumberSequenceExhausted
		}
		sequence, err = sequence.Update().AddCurrentValue(1).Save(ctx)
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, 0, err
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}
	return numberRuleToBiz(lockedRule), sequence.CurrentValue, nil
}

func (r *orderConfigRepo) ListStatusTemplates(ctx context.Context, organizationID uuid.UUID, businessType biz.BusinessType, published *bool) ([]*biz.StatusTemplate, error) {
	query := r.data.db.StatusTemplate.Query().Where(statustemplate.OrganizationIDEQ(organizationID)).WithItems(func(query *ent.StatusTemplateItemQuery) {
		query.Order(statustemplateitem.BySortOrder(), statustemplateitem.ByCode())
	})
	if businessType != "" {
		query.Where(statustemplate.BusinessTypeEQ(statustemplate.BusinessType(businessType)))
	}
	if published != nil {
		if *published {
			query.Where(statustemplate.PublishedAtNotNil())
		} else {
			query.Where(statustemplate.PublishedAtIsNil())
		}
	}
	items, err := query.Order(statustemplate.ByBusinessType(), statustemplate.ByCode(), statustemplate.ByVersion()).All(ctx)
	if err != nil {
		return nil, err
	}
	return statusTemplatesToBiz(items), nil
}

func (r *orderConfigRepo) CreateStatusTemplate(ctx context.Context, organizationID uuid.UUID, input *biz.StatusTemplate) (*biz.StatusTemplate, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	template, err := tx.StatusTemplate.Create().SetOrganizationID(organizationID).SetCode(input.Code).SetName(input.Name).SetBusinessType(statustemplate.BusinessType(input.BusinessType)).SetVersion(input.Version).SetIsDefault(false).SetEnabled(true).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrStatusTemplateExists
		}
		return nil, err
	}
	for _, item := range input.Items {
		if _, err := tx.StatusTemplateItem.Create().SetTemplateID(template.ID).SetCode(item.Code).SetLabel(item.Label).SetSortOrder(item.SortOrder).SetEnabled(item.Enabled).SetNillableColorToken(item.ColorToken).SetSystem(item.System).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findStatusTemplate(ctx, organizationID, template.ID)
}

func (r *orderConfigRepo) PublishStatusTemplate(ctx context.Context, organizationID, id uuid.UUID, isDefault bool, publishedAt time.Time) (*biz.StatusTemplate, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	template, err := tx.StatusTemplate.Query().Where(statustemplate.IDEQ(id), statustemplate.OrganizationIDEQ(organizationID), statustemplate.EnabledEQ(true)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrStatusTemplateNotFound
		}
		return nil, err
	}
	if template.PublishedAt != nil {
		_ = tx.Rollback()
		return nil, biz.ErrStatusTemplateInvalid
	}
	if isDefault {
		if _, err := tx.StatusTemplate.Update().Where(statustemplate.OrganizationIDEQ(organizationID), statustemplate.BusinessTypeEQ(template.BusinessType), statustemplate.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if _, err := template.Update().SetPublishedAt(publishedAt).SetIsDefault(isDefault).Save(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrStatusTemplateDefaultConflict
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findStatusTemplate(ctx, organizationID, id)
}

func (r *orderConfigRepo) SetDefaultStatusTemplate(ctx context.Context, organizationID, id uuid.UUID) (*biz.StatusTemplate, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	template, err := tx.StatusTemplate.Query().Where(statustemplate.IDEQ(id), statustemplate.OrganizationIDEQ(organizationID), statustemplate.EnabledEQ(true), statustemplate.PublishedAtNotNil()).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrStatusTemplateNotFound
		}
		return nil, err
	}
	if !template.IsDefault {
		if _, err := tx.StatusTemplate.Update().Where(statustemplate.OrganizationIDEQ(organizationID), statustemplate.BusinessTypeEQ(template.BusinessType), statustemplate.IsDefaultEQ(true), statustemplate.IDNEQ(id)).SetIsDefault(false).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := template.Update().SetIsDefault(true).Save(ctx); err != nil {
			_ = tx.Rollback()
			if ent.IsConstraintError(err) {
				return nil, biz.ErrStatusTemplateDefaultConflict
			}
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findStatusTemplate(ctx, organizationID, id)
}

func (r *orderConfigRepo) findStatusTemplate(ctx context.Context, organizationID, id uuid.UUID) (*biz.StatusTemplate, error) {
	item, err := r.data.db.StatusTemplate.Query().Where(statustemplate.IDEQ(id), statustemplate.OrganizationIDEQ(organizationID)).WithItems(func(query *ent.StatusTemplateItemQuery) {
		query.Order(statustemplateitem.BySortOrder(), statustemplateitem.ByCode())
	}).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrStatusTemplateNotFound
		}
		return nil, err
	}
	return statusTemplateToBiz(item), nil
}

func numberRuleToBiz(item *ent.NumberRule) *biz.NumberRule {
	return &biz.NumberRule{ID: item.ID, OrganizationID: item.OrganizationID, DocumentType: biz.DocumentType(item.DocumentType), Prefix: item.Prefix, DateFormat: biz.DateFormat(item.DateFormat), SequenceLength: item.SequenceLength, ResetPolicy: biz.ResetPolicy(item.ResetPolicy), Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func statusTemplatesToBiz(items []*ent.StatusTemplate) []*biz.StatusTemplate {
	result := make([]*biz.StatusTemplate, 0, len(items))
	for _, item := range items {
		result = append(result, statusTemplateToBiz(item))
	}
	return result
}

func statusTemplateToBiz(item *ent.StatusTemplate) *biz.StatusTemplate {
	result := &biz.StatusTemplate{ID: item.ID, OrganizationID: item.OrganizationID, Code: item.Code, Name: item.Name, BusinessType: biz.BusinessType(item.BusinessType), Version: item.Version, IsDefault: item.IsDefault, PublishedAt: item.PublishedAt, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	for _, child := range item.Edges.Items {
		result.Items = append(result.Items, &biz.StatusTemplateItem{ID: child.ID, Code: child.Code, Label: child.Label, SortOrder: child.SortOrder, Enabled: child.Enabled, ColorToken: child.ColorToken, System: child.System})
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		if result.Items[i].SortOrder == result.Items[j].SortOrder {
			return result.Items[i].Code < result.Items[j].Code
		}
		return result.Items[i].SortOrder < result.Items[j].SortOrder
	})
	return result
}

var _ biz.OrderConfigRepo = (*orderConfigRepo)(nil)
