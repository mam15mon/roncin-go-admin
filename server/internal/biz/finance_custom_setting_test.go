package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type financeCustomSettingRepoStub struct {
	FinanceCustomSettingRepo
	saved           *BilledFeeEditPolicy
	expectedVersion uint64
}

func (r *financeCustomSettingRepoStub) SaveBilledFeeEditPolicy(_ context.Context, _, _ uuid.UUID, policy *BilledFeeEditPolicy, expectedVersion uint64, _ *AuditEvent) (*BilledFeeEditPolicy, error) {
	r.saved = policy
	r.expectedVersion = expectedVersion
	return policy, nil
}

func TestUpdateBilledFeeEditPolicyRejectsUnknownAndDuplicateFields(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	actorID := uuid.Must(uuid.NewV7())
	usecase := NewFinanceCustomSettingUsecase(&financeCustomSettingRepoStub{})

	for _, fields := range [][]BilledFeeEditableField{
		{BilledFeeEditableField("UNKNOWN")},
		{BilledFeeFieldQuantity, BilledFeeFieldQuantity},
	} {
		_, err := usecase.UpdateBilledFeeEditPolicy(context.Background(), organizationID, actorID, &BilledFeeEditPolicy{Enabled: true, EditableFields: fields}, 0)
		if err != ErrFinanceCustomSettingInvalidArgument {
			t.Fatalf("非法可修改字段应被拒绝，fields=%v err=%v", fields, err)
		}
	}
}

func TestUpdateBilledFeeEditPolicyPreservesSelectedFieldsAndVersion(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	actorID := uuid.Must(uuid.NewV7())
	repo := &financeCustomSettingRepoStub{}
	usecase := NewFinanceCustomSettingUsecase(repo)
	fields := []BilledFeeEditableField{BilledFeeFieldFeeName, BilledFeeFieldQuantity, BilledFeeFieldTaxRate}

	_, err := usecase.UpdateBilledFeeEditPolicy(context.Background(), organizationID, actorID, &BilledFeeEditPolicy{Enabled: true, EditableFields: fields}, 7)
	if err != nil {
		t.Fatalf("保存费用修改策略失败: %v", err)
	}
	if repo.saved == nil || !repo.saved.Enabled || repo.expectedVersion != 7 {
		t.Fatalf("策略开关或版本未传给仓储: policy=%+v version=%d", repo.saved, repo.expectedVersion)
	}
	for _, field := range fields {
		if !repo.saved.Allows(field) {
			t.Fatalf("字段 %s 应被保留为可修改", field)
		}
	}
}
