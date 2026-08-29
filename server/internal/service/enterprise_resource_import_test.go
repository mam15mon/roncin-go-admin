package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/enterprise_resource/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type enterpriseResourceImportRepoStub struct {
	importErr error
	conflicts []*biz.EnterpriseResourceImportConflict
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
func (stub *enterpriseResourceImportRepoStub) ListTagGroups(context.Context, uuid.UUID) ([]*biz.EnterpriseTagGroup, error) {
	return nil, nil
}
func (stub *enterpriseResourceImportRepoStub) CreateTagGroup(context.Context, uuid.UUID, *biz.EnterpriseTagGroup, *biz.AuditEvent) (*biz.EnterpriseTagGroup, error) {
	return nil, nil
}
func (stub *enterpriseResourceImportRepoStub) UpdateTagGroup(context.Context, uuid.UUID, uuid.UUID, *biz.EnterpriseTagGroup, *biz.AuditEvent) (*biz.EnterpriseTagGroup, error) {
	return nil, nil
}
func (stub *enterpriseResourceImportRepoStub) DeleteTagGroup(context.Context, uuid.UUID, uuid.UUID, *biz.AuditEvent) error {
	return nil
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
	ctx := biz.WithPrincipal(t.Context(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: organizationID}})
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
func (stub *enterpriseResourceImportRepoStub) Import(context.Context, uuid.UUID, uuid.UUID, []*biz.EnterpriseResource, bool, *biz.AuditEvent) ([]*biz.EnterpriseResource, int, int, []*biz.EnterpriseResourceImportConflict, error) {
	return nil, 0, 0, nil, stub.importErr
}

func TestCommitEnterpriseResourceImportPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("审计写入失败")
	usecase := biz.NewEnterpriseResourceUsecase(&enterpriseResourceImportRepoStub{importErr: wantErr}, nil)
	service := NewEnterpriseResourceService(usecase)
	ctx := biz.WithPrincipal(t.Context(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: uuid.New()}})
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
