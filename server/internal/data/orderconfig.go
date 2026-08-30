package data

import (
	"context"
	"fmt"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"

	"github.com/google/uuid"
)

type orderConfigRepo struct{ data *Data }

func NewOrderConfigRepo(data *Data) biz.OrderConfigRepo { return &orderConfigRepo{data: data} }

func CreateDefaultNumberRules(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID) error {
	for _, rule := range biz.DefaultNumberRules() {
		if _, err := tx.NumberRule.Create().
			SetOrganizationID(organizationID).
			SetDocumentType(numberrule.DocumentType(rule.DocumentType)).
			SetPrefix(rule.Prefix).
			SetDateFormat(numberrule.DateFormat(rule.DateFormat)).
			SetSequenceLength(rule.SequenceLength).
			SetResetPolicy(numberrule.ResetPolicy(rule.ResetPolicy)).
			SetEnabled(rule.Enabled).
			Save(ctx); err != nil {
			return fmt.Errorf("创建默认单号规则 %s: %w", rule.DocumentType, err)
		}
	}
	return nil
}

func (r *orderConfigRepo) ListNumberRules(ctx context.Context, organizationID uuid.UUID) ([]*biz.NumberRule, error) {
	items, err := r.data.db.NumberRule.Query().Where(
		numberrule.OrganizationIDEQ(organizationID),
		numberrule.DocumentTypeNEQ(numberrule.DocumentTypeColoadHouseBill),
	).Order(numberrule.ByDocumentType()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.NumberRule, 0, len(items))
	for _, item := range items {
		result = append(result, numberRuleToBiz(item))
	}
	return result, nil
}

func (r *orderConfigRepo) CreateNumberRule(ctx context.Context, organizationID uuid.UUID, input *biz.NumberRule, audit *biz.AuditEvent) (*biz.NumberRule, error) {
	var created *ent.NumberRule
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var err error
		created, err = tx.NumberRule.Create().SetOrganizationID(organizationID).SetDocumentType(numberrule.DocumentType(input.DocumentType)).SetPrefix(input.Prefix).SetDateFormat(numberrule.DateFormat(input.DateFormat)).SetSequenceLength(input.SequenceLength).SetResetPolicy(numberrule.ResetPolicy(input.ResetPolicy)).SetEnabled(true).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrNumberRuleExists)
		}
		audit.Details["number_rule.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return numberRuleToBiz(created), nil
}

func (r *orderConfigRepo) UpdateNumberRule(ctx context.Context, organizationID, id uuid.UUID, input *biz.NumberRule, audit *biz.AuditEvent) (*biz.NumberRule, error) {
	var updated *ent.NumberRule
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.NumberRule.Query().Where(numberrule.IDEQ(id), numberrule.OrganizationIDEQ(organizationID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrNumberRuleNotFound, nil)
		}
		updated, err = existing.Update().SetPrefix(input.Prefix).SetDateFormat(numberrule.DateFormat(input.DateFormat)).SetSequenceLength(input.SequenceLength).SetResetPolicy(numberrule.ResetPolicy(input.ResetPolicy)).SetEnabled(input.Enabled).Save(ctx)
		if err != nil {
			return err
		}
		audit.Details["document_type"] = string(updated.DocumentType)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return numberRuleToBiz(updated), nil
}

func (r *orderConfigRepo) AllocateNumber(ctx context.Context, organizationID uuid.UUID, documentType biz.DocumentType, at time.Time) (*biz.NumberRule, int64, error) {
	var rule *biz.NumberRule
	var sequence int64
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var err error
		rule, sequence, err = allocateNumberInTx(ctx, tx, organizationID, documentType, at)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	return rule, sequence, nil
}

func allocateNumberInTx(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, documentType biz.DocumentType, at time.Time) (*biz.NumberRule, int64, error) {
	lockedRule, err := tx.NumberRule.Query().Where(numberrule.OrganizationIDEQ(organizationID), numberrule.DocumentTypeEQ(numberrule.DocumentType(documentType)), numberrule.EnabledEQ(true)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, 0, mapEntError(err, biz.ErrNumberRuleNotFound, nil)
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
			return nil, 0, biz.ErrNumberSequenceExhausted
		}
		sequence, err = sequence.Update().AddCurrentValue(1).Save(ctx)
	}
	if err != nil {
		return nil, 0, err
	}
	return numberRuleToBiz(lockedRule), sequence.CurrentValue, nil
}

func numberRuleToBiz(item *ent.NumberRule) *biz.NumberRule {
	return &biz.NumberRule{ID: item.ID, OrganizationID: item.OrganizationID, DocumentType: biz.DocumentType(item.DocumentType), Prefix: item.Prefix, DateFormat: biz.DateFormat(item.DateFormat), SequenceLength: item.SequenceLength, ResetPolicy: biz.ResetPolicy(item.ResetPolicy), Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

var _ biz.OrderConfigRepo = (*orderConfigRepo)(nil)
