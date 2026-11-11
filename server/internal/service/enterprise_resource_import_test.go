package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/enterprise_resource/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type enterpriseResourceImportRepoStub struct {
	importCalls int
	overwrite   bool
	importErr   error
	conflicts   []*biz.EnterpriseResourceImportConflict
}

func (stub *enterpriseResourceImportRepoStub) SearchPartnerOptions(context.Context, uuid.UUID, string, int, int) ([]*biz.EnterpriseResourcePartnerOption, int64, error) {
	return nil, 0, nil
}
func (stub *enterpriseResourceImportRepoStub) SearchAssigneeOptions(context.Context, uuid.UUID, string, int, int) ([]*biz.EnterpriseResourceAssigneeOption, int64, error) {
	return nil, 0, nil
}
func (stub *enterpriseResourceImportRepoStub) ListRegionOptions(context.Context, int, *string, int, int) ([]*biz.EnterpriseResourceRegionOption, int64, error) {
	return nil, 0, nil
}
func (stub *enterpriseResourceImportRepoStub) ImageUsage(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}
func (stub *enterpriseResourceImportRepoStub) List(context.Context, uuid.UUID, biz.EnterpriseResourceListOptions) ([]*biz.EnterpriseResource, int64, error) {
	return nil, 0, nil
}
func (stub *enterpriseResourceImportRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*biz.EnterpriseResource, error) {
	return nil, nil
}
func (stub *enterpriseResourceImportRepoStub) Create(context.Context, uuid.UUID, uuid.UUID, *biz.EnterpriseResource, *biz.AuditEvent) (*biz.EnterpriseResource, error) {
	return nil, nil
}
func (stub *enterpriseResourceImportRepoStub) Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *biz.EnterpriseResource, *biz.AuditEvent) (*biz.EnterpriseResource, error) {
	return nil, nil
}
func (stub *enterpriseResourceImportRepoStub) Delete(context.Context, uuid.UUID, uuid.UUID, *biz.AuditEvent) error {
	return nil
}
func (stub *enterpriseResourceImportRepoStub) BatchPartners(context.Context, uuid.UUID, []uuid.UUID, []uuid.UUID, bool, *biz.AuditEvent) (int, error) {
	return 0, nil
}
func (stub *enterpriseResourceImportRepoStub) BatchAddressTypes(context.Context, uuid.UUID, []uuid.UUID, []biz.EnterpriseAddressType, bool, *biz.AuditEvent) (int, error) {
	return 0, nil
}
func (stub *enterpriseResourceImportRepoStub) BatchAssignees(context.Context, uuid.UUID, []uuid.UUID, []uuid.UUID, bool, *biz.AuditEvent) (int, error) {
	return 0, nil
}
func (stub *enterpriseResourceImportRepoStub) FindImportConflicts(context.Context, uuid.UUID, []*biz.EnterpriseResource) ([]*biz.EnterpriseResourceImportConflict, error) {
	return stub.conflicts, nil
}

func TestPreviewEnterpriseResourceImportRejectsAmbiguousOverwrite(t *testing.T) {
	organizationID := uuid.New()
	usecase := biz.NewEnterpriseResourceUsecase(&enterpriseResourceImportRepoStub{conflicts: []*biz.EnterpriseResourceImportConflict{
		{RowNumber: 1, ExistingResourceID: uuid.New(), ExistingShortName: "现有主体一", MatchedFields: []string{"company_name"}},
		{RowNumber: 1, ExistingResourceID: uuid.New(), ExistingShortName: "现有主体二", MatchedFields: []string{"business_code"}},
	}}, nil)
	service := NewEnterpriseResourceService(usecase)
	ctx := enterpriseImportContext(organizationID, access.EnterpriseResourceCreate, access.EnterpriseResourceUpdate)
	response, err := service.PreviewEnterpriseResourceImport(ctx, &v1.PreviewEnterpriseResourceImportRequest{
		ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_NOTIFY_PARTY,
		Rows: []*v1.EnterpriseResourceInput{{
			ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_NOTIFY_PARTY,
			ShortName:    "测试通知人", Enabled: true,
			Detail: &v1.EnterpriseResourceInput_Party{Party: &v1.EnterpriseResourceParty{CompanyName: "测试企业", CountryCode: "CN"}},
		}},
	})
	if err != nil {
		t.Fatalf("预览导入失败: %v", err)
	}
	if response.GetConflictCount() != 2 || response.GetOverwriteAllowed() {
		t.Fatalf("歧义冲突不应允许覆盖: %+v", response)
	}
	if len(response.GetRows()) != 1 || len(response.GetRows()[0].GetConflicts()) != 2 {
		t.Fatalf("预览未返回完整冲突明细: %+v", response.GetRows())
	}
}
func (stub *enterpriseResourceImportRepoStub) Import(_ context.Context, _, _ uuid.UUID, rows []*biz.EnterpriseResource, overwrite bool, _ *biz.AuditEvent) ([]*biz.EnterpriseResource, int, int, []*biz.EnterpriseResourceImportConflict, error) {
	stub.importCalls++
	stub.overwrite = overwrite
	if stub.importErr != nil {
		return nil, 0, 0, nil, stub.importErr
	}
	if overwrite {
		return rows, 0, len(rows), nil, nil
	}
	return rows, len(rows), 0, nil, nil
}

func TestCommitEnterpriseResourceImportPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("审计写入失败")
	usecase := biz.NewEnterpriseResourceUsecase(&enterpriseResourceImportRepoStub{importErr: wantErr}, nil)
	service := NewEnterpriseResourceService(usecase)
	ctx := enterpriseImportContext(uuid.New(), access.EnterpriseResourceCreate)
	_, err := service.CommitEnterpriseResourceImport(ctx, &v1.CommitEnterpriseResourceImportRequest{
		ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_SHIPPER,
		Rows: []*v1.EnterpriseResourceInput{{
			ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_SHIPPER,
			ShortName:    "测试发货人",
			Enabled:      true,
			Detail: &v1.EnterpriseResourceInput_Party{Party: &v1.EnterpriseResourceParty{
				CompanyName: "测试企业", CountryCode: "CN",
			}},
		}},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("仓储错误未原样向上传播: %v", err)
	}
}

var _ biz.EnterpriseResourceRepo = (*enterpriseResourceImportRepoStub)(nil)

func enterpriseImportContext(organizationID uuid.UUID, permissions ...string) context.Context {
	granted := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		granted[permission] = struct{}{}
	}
	return biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{ID: organizationID, Kind: biz.OrganizationKindCompany},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: organizationID, Kind: biz.OrganizationKindCompany}},
		RoleGrants:        []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: granted}},
	})
}

func enterpriseImportRows() []*v1.EnterpriseResourceInput {
	return []*v1.EnterpriseResourceInput{{
		ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_SHIPPER,
		ShortName:    "测试发货人", Enabled: true,
		Detail: &v1.EnterpriseResourceInput_Party{Party: &v1.EnterpriseResourceParty{CompanyName: "测试企业", CountryCode: "CN"}},
	}}
}

func TestEnterpriseResourceImportPermissionBoundary(t *testing.T) {
	for _, tc := range []struct {
		name        string
		permissions []string
		overwrite   bool
		allowed     bool
	}{
		{"仅新建可以导入新记录", []string{access.EnterpriseResourceCreate}, false, true},
		{"仅新建不能覆盖", []string{access.EnterpriseResourceCreate}, true, false},
		{"新建加编辑可以覆盖", []string{access.EnterpriseResourceCreate, access.EnterpriseResourceUpdate}, true, true},
		{"仅编辑不能导入", []string{access.EnterpriseResourceUpdate}, false, false},
		{"无权限不能导入", nil, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &enterpriseResourceImportRepoStub{}
			service := NewEnterpriseResourceService(biz.NewEnterpriseResourceUsecase(repo, nil))
			response, err := service.CommitEnterpriseResourceImport(enterpriseImportContext(uuid.New(), tc.permissions...), &v1.CommitEnterpriseResourceImportRequest{
				ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_SHIPPER,
				Rows:         enterpriseImportRows(), OverwriteConflicts: tc.overwrite,
			})
			if !tc.allowed {
				if !errors.Is(err, biz.ErrPermissionDenied) || repo.importCalls != 0 {
					t.Fatalf("越权请求不得进入仓储：err=%v calls=%d", err, repo.importCalls)
				}
				return
			}
			if err != nil || !response.GetSuccess() || repo.importCalls != 1 || repo.overwrite != tc.overwrite {
				t.Fatalf("合法导入失败：response=%v err=%v calls=%d", response, err, repo.importCalls)
			}
		})
	}
}

func TestPreviewEnterpriseResourceImportIncludesOverwritePermission(t *testing.T) {
	for _, canUpdate := range []bool{false, true} {
		permissions := []string{access.EnterpriseResourceCreate}
		if canUpdate {
			permissions = append(permissions, access.EnterpriseResourceUpdate)
		}
		repo := &enterpriseResourceImportRepoStub{conflicts: []*biz.EnterpriseResourceImportConflict{{RowNumber: 1, ExistingResourceID: uuid.New(), ExistingShortName: "已有企业", MatchedFields: []string{"company_name"}}}}
		service := NewEnterpriseResourceService(biz.NewEnterpriseResourceUsecase(repo, nil))
		response, err := service.PreviewEnterpriseResourceImport(enterpriseImportContext(uuid.New(), permissions...), &v1.PreviewEnterpriseResourceImportRequest{
			ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_SHIPPER, Rows: enterpriseImportRows(),
		})
		if err != nil || response.GetOverwriteAllowed() != canUpdate || response.GetConflictCount() != 1 {
			t.Fatalf("覆盖能力应同时满足冲突可消解与编辑权限：response=%v err=%v", response, err)
		}
	}
}

func TestEnterpriseResourceImportDeniedOverwriteCannotPartiallyWriteMixedRows(t *testing.T) {
	repo := &enterpriseResourceImportRepoStub{}
	service := NewEnterpriseResourceService(biz.NewEnterpriseResourceUsecase(repo, nil))
	rows := enterpriseImportRows()
	rows = append(rows, &v1.EnterpriseResourceInput{
		ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_SHIPPER,
		ShortName:    "另一个新资源", Enabled: true,
		Detail: &v1.EnterpriseResourceInput_Party{Party: &v1.EnterpriseResourceParty{CompanyName: "另一个新企业", CountryCode: "CN"}},
	})
	_, err := service.CommitEnterpriseResourceImport(enterpriseImportContext(uuid.New(), access.EnterpriseResourceCreate), &v1.CommitEnterpriseResourceImportRequest{
		ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_SHIPPER, Rows: rows, OverwriteConflicts: true,
	})
	if !errors.Is(err, biz.ErrPermissionDenied) || repo.importCalls != 0 {
		t.Fatalf("覆盖无权限时整个混合批次不能进入持久化: err=%v calls=%d", err, repo.importCalls)
	}
}

func TestEnterpriseResourceImportRejectsSystemWorkspace(t *testing.T) {
	repo := &enterpriseResourceImportRepoStub{}
	service := NewEnterpriseResourceService(biz.NewEnterpriseResourceUsecase(repo, nil))
	organizationID := uuid.New()
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{ID: organizationID, Kind: biz.OrganizationKindSystem},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: organizationID, Kind: biz.OrganizationKindSystem}},
		RoleGrants:        []biz.RoleGrant{{DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{access.EnterpriseResourceCreate: {}, access.EnterpriseResourceUpdate: {}}}},
	})
	_, err := service.CommitEnterpriseResourceImport(ctx, &v1.CommitEnterpriseResourceImportRequest{
		ResourceType: v1.EnterpriseResourceType_ENTERPRISE_RESOURCE_TYPE_SHIPPER, Rows: enterpriseImportRows(), OverwriteConflicts: true,
	})
	if !errors.Is(err, biz.ErrPermissionDenied) || repo.importCalls != 0 {
		t.Fatalf("系统工作区不能用经营权限导入: err=%v calls=%d", err, repo.importCalls)
	}
}
