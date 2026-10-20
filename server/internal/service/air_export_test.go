package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestAirRestoreLinkRequiresLifecyclePermission(t *testing.T) {
	p := &biz.Principal{Organization: biz.Organization{ID: uuid.New(), Kind: biz.OrganizationKindCompany}, RoleGrants: []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.OrderPermission(access.OrderBusinessAE, access.OrderUpdate): {}}}}}
	service := NewAirExportService(nil)
	_, e := service.LinkAirMasterWaybill(biz.WithPrincipal(context.Background(), p), &v1.LinkAirMasterWaybillRequest{Id: uuid.NewString(), OrderId: uuid.NewString(), ExpectedOrderVersion: 1, ExpectedMasterVersion: 1, HouseNo: "H001", Reason: "恢复"})
	if !errors.Is(e, biz.ErrPermissionDenied) {
		t.Fatalf("只有update不得恢复生命周期 %v", e)
	}
}
