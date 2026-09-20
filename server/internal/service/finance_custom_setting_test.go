package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type financeCustomSettingServiceRepoStub struct {
	biz.FinanceCustomSettingRepo
	policy *biz.BilledFeeEditPolicy
}

func (s *financeCustomSettingServiceRepoStub) GetBilledFeeEditPolicy(_ context.Context, organizationID uuid.UUID) (*biz.BilledFeeEditPolicy, error) {
	return &biz.BilledFeeEditPolicy{OrganizationID: organizationID, EditableFields: []biz.BilledFeeEditableField{}}, nil
}

func TestCustomSettingUpdatesRejectMissingExpectedVersion(t *testing.T) {
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:       uuid.New(),
		Organization: biz.Organization{Kind: biz.OrganizationKindCompany, ID: uuid.New()},
	})

	t.Run("账单费用修改策略", func(t *testing.T) {
		service := &SettlementService{}
		_, err := service.UpdateBilledFeeEditPolicy(ctx, &v1.UpdateBilledFeeEditPolicyRequest{})
		if err != biz.ErrFinanceCustomSettingInvalidArgument {
			t.Fatalf("未传 expected_version 时错误为 %v，期望 %v", err, biz.ErrFinanceCustomSettingInvalidArgument)
		}
	})
}

func TestGetBilledFeeEditPolicyReturnsCurrentOrganizationUpdateCapability(t *testing.T) {
	organizationID := uuid.New()
	service := &SettlementService{customSettingUsecase: biz.NewFinanceCustomSettingUsecase(&financeCustomSettingServiceRepoStub{})}
	newContext := func(includeRead, includeUpdate bool) context.Context {
		permissions := map[string]struct{}{}
		if includeRead {
			permissions[access.FinanceBillRead] = struct{}{}
		}
		if includeUpdate {
			permissions[access.FinanceBillConfigure] = struct{}{}
		}
		return biz.WithPrincipal(context.Background(), &biz.Principal{
			UserID: uuid.New(), Organization: biz.Organization{Kind: biz.OrganizationKindCompany, ID: organizationID},
			OrganizationNodes: []biz.OrganizationScopeNode{{Kind: biz.OrganizationKindCompany, ID: organizationID}},
			RoleGrants:        []biz.RoleGrant{{RoleCode: "settings", DataScope: biz.DataScopeOrganization, Permissions: permissions}},
		})
	}

	response, err := service.GetBilledFeeEditPolicy(newContext(true, true), &v1.GetBilledFeeEditPolicyRequest{})
	if err != nil || !response.GetCanUpdate() {
		t.Fatalf("当前组织同时具备读取和更新权限应可编辑: response=%#v err=%v", response, err)
	}
	response, err = service.GetBilledFeeEditPolicy(newContext(true, false), &v1.GetBilledFeeEditPolicyRequest{})
	if err != nil || response.GetCanUpdate() {
		t.Fatalf("当前组织缺少更新权限时不应可编辑: response=%#v err=%v", response, err)
	}
	if _, err := service.GetBilledFeeEditPolicy(newContext(false, true), &v1.GetBilledFeeEditPolicyRequest{}); err != biz.ErrPermissionDenied {
		t.Fatalf("缺少账单读取权限错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
	if _, err := service.UpdateBilledFeeEditPolicy(newContext(true, false), &v1.UpdateBilledFeeEditPolicyRequest{ExpectedVersion: wrapperspb.UInt64(0)}); err != biz.ErrPermissionDenied {
		t.Fatalf("缺少账单更新权限错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
}

func TestPolicyToAPICarriesUpdatedByName(t *testing.T) {
	actorID := uuid.New()
	billed := billedFeeEditPolicyToAPI(&biz.BilledFeeEditPolicy{OrganizationID: uuid.New(), Enabled: true, Version: 1, UpdatedBy: &actorID, UpdatedByName: "张三"})
	if billed.GetUpdatedByName() != "张三" {
		t.Fatalf("账单费用修改策略操作人姓名 = %q，期望 %q", billed.GetUpdatedByName(), "张三")
	}
	billedNoName := billedFeeEditPolicyToAPI(&biz.BilledFeeEditPolicy{OrganizationID: uuid.New()})
	if billedNoName.UpdatedByName != nil {
		t.Fatalf("操作人不可考时不应下发姓名，实际 %#v", billedNoName.UpdatedByName)
	}
	credit := creditLimitControlPolicyToAPI(&biz.CreditLimitControlPolicy{OrganizationID: uuid.New(), AllowSelectionWhenCreditExceeded: true, Version: 2, UpdatedBy: &actorID, UpdatedByName: "李四"})
	if credit.GetUpdatedByName() != "李四" {
		t.Fatalf("信用额度管控策略操作人姓名 = %q，期望 %q", credit.GetUpdatedByName(), "李四")
	}
	creditNoName := creditLimitControlPolicyToAPI(&biz.CreditLimitControlPolicy{OrganizationID: uuid.New(), AllowSelectionWhenCreditExceeded: true})
	if creditNoName.UpdatedByName != nil {
		t.Fatalf("操作人不可考时不应下发姓名，实际 %#v", creditNoName.UpdatedByName)
	}
}

func (s *financeCustomSettingServiceRepoStub) SaveBilledFeeEditPolicy(_ context.Context, organizationID, _ uuid.UUID, policy *biz.BilledFeeEditPolicy, _ uint64, _ *biz.AuditEvent) (*biz.BilledFeeEditPolicy, error) {
	s.policy = policy
	policy.OrganizationID = organizationID
	return policy, nil
}

func TestHeadquartersConfigurePolicyWithoutBusinessWrite(t *testing.T) {
	id := uuid.New()
	repo := &financeCustomSettingServiceRepoStub{}
	service := &SettlementService{customSettingUsecase: biz.NewFinanceCustomSettingUsecase(repo)}
	principal := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: id, Kind: biz.OrganizationKindHeadquarters}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: id, Kind: biz.OrganizationKindHeadquarters}}, RoleGrants: []biz.RoleGrant{{DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{access.FinanceBillConfigure: {}, access.FinanceBillRead: {}, access.FinanceBillUpdate: {}}}}}
	ctx := biz.WithPrincipal(context.Background(), principal)
	response, err := service.GetBilledFeeEditPolicy(ctx, &v1.GetBilledFeeEditPolicyRequest{})
	if err != nil || !response.GetCanUpdate() {
		t.Fatalf("总部配置能力应保留: %v", err)
	}
	if _, err := service.UpdateBilledFeeEditPolicy(ctx, &v1.UpdateBilledFeeEditPolicyRequest{ExpectedVersion: wrapperspb.UInt64(0), Enabled: true}); err != nil || repo.policy == nil {
		t.Fatalf("总部治理配置应成功保存: %v", err)
	}
	if _, err := organizationIDsForPermission(principal, access.FinanceBillUpdate, true); err != biz.ErrPermissionDenied {
		t.Fatalf("配置能力不能变成业务更新权限: %v", err)
	}
}
