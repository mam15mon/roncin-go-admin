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

type creditLimitControlRepoStub struct {
	FinanceCustomSettingRepo
	policy *CreditLimitControlPolicy
}

func (r *creditLimitControlRepoStub) GetCreditLimitControlPolicy(_ context.Context, organizationID uuid.UUID) (*CreditLimitControlPolicy, error) {
	if r.policy == nil {
		return nil, nil
	}
	return r.policy, nil
}

func (r *creditLimitControlRepoStub) SaveCreditLimitControlPolicy(_ context.Context, _, actorID uuid.UUID, policy *CreditLimitControlPolicy, expectedVersion uint64, _ *AuditEvent) (*CreditLimitControlPolicy, error) {
	r.policy = &CreditLimitControlPolicy{OrganizationID: policy.OrganizationID, AllowSelectionWhenCreditExceeded: policy.AllowSelectionWhenCreditExceeded, Version: expectedVersion + 1, UpdatedBy: &actorID}
	return r.policy, nil
}

func TestGetCreditLimitControlPolicyDefaultsToReminderMode(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	usecase := NewFinanceCustomSettingUsecase(&creditLimitControlRepoStub{})

	policy, err := usecase.GetCreditLimitControlPolicy(context.Background(), organizationID)
	if err != nil {
		t.Fatalf("读取默认策略失败: %v", err)
	}
	if !policy.AllowSelectionWhenCreditExceeded {
		t.Fatalf("未保存过策略时应默认开启「超额后允许选择」（仅提醒模式）")
	}
	active, err := usecase.IsCreditLimitInterventionActive(context.Background(), organizationID)
	if err != nil {
		t.Fatalf("判定干预模式失败: %v", err)
	}
	if active {
		t.Fatalf("默认策略不应处于直接干预模式")
	}
}

func TestUpdateCreditLimitControlPolicyRoundTripAndIntervention(t *testing.T) {
	ctx := context.Background()
	organizationID := uuid.Must(uuid.NewV7())
	actorID := uuid.Must(uuid.NewV7())
	repo := &creditLimitControlRepoStub{}
	usecase := NewFinanceCustomSettingUsecase(repo)

	policy, err := usecase.UpdateCreditLimitControlPolicy(ctx, organizationID, actorID, false, 3)
	if err != nil {
		t.Fatalf("关闭超额允许选择失败: %v", err)
	}
	if policy.AllowSelectionWhenCreditExceeded || policy.Version != 4 {
		t.Fatalf("策略保存结果不符: %+v", policy)
	}
	active, err := usecase.IsCreditLimitInterventionActive(ctx, organizationID)
	if err != nil {
		t.Fatalf("判定干预模式失败: %v", err)
	}
	if !active {
		t.Fatalf("关闭「超额后允许选择」后应处于直接干预模式")
	}

	if _, err := usecase.UpdateCreditLimitControlPolicy(ctx, organizationID, uuid.Nil, true, 4); err != ErrFinanceCustomSettingInvalidArgument {
		t.Fatalf("操作人为空应返回参数错误, got %v", err)
	}
	if _, err := usecase.UpdateCreditLimitControlPolicy(ctx, uuid.Nil, actorID, true, 4); err != ErrFinanceCustomSettingInvalidArgument {
		t.Fatalf("组织为空应返回参数错误, got %v", err)
	}
}
