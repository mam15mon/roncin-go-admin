package server

import (
	"testing"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/google/uuid"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestAuthorizationFeeQuantityPreview(t *testing.T) {
	organizationID := uuid.New()
	order := &biz.Order{ID: uuid.New(), OrganizationID: organizationID, BusinessType: biz.OrderBusinessSE}
	request := &orderv1.PreviewFeeQuantityRequest{OrderId: order.ID.String(), BillingUnitId: uuid.NewString(), Source: "ACTUAL"}
	for _, tc := range []struct {
		name       string
		permission string
		otherOrg   bool
		cookie     string
		wantError  error
	}{
		{name: "海出费用读取允许", permission: access.OrderPermission(access.OrderBusinessSE, access.OrderFeeRead), cookie: "sid=valid-token"},
		{name: "没有费用读取权限拒绝", permission: access.OrderPermission(access.OrderBusinessSE, access.OrderRead), cookie: "sid=valid-token", wantError: biz.ErrPermissionDenied},
		{name: "只有海进费用读取权限拒绝", permission: access.OrderPermission(access.OrderBusinessSI, access.OrderFeeRead), cookie: "sid=valid-token", wantError: biz.ErrPermissionDenied},
		{name: "其他公司即使全范围也拒绝", permission: access.OrderPermission(access.OrderBusinessSE, access.OrderFeeRead), otherOrg: true, cookie: "sid=valid-token", wantError: biz.ErrPermissionDenied},
		{name: "未登录拒绝", permission: access.OrderPermission(access.OrderBusinessSE, access.OrderFeeRead), wantError: biz.ErrSessionRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspaceID := organizationID
			if tc.otherOrg {
				workspaceID = uuid.New()
			}
			principal := &biz.Principal{
				Organization:      biz.Organization{ID: workspaceID, Kind: biz.OrganizationKindCompany},
				OrganizationNodes: serverOrganizationNodes(workspaceID, organizationID),
				RoleGrants:        []biz.RoleGrant{serverRoleGrant("operator", biz.DataScopeAll, []string{tc.permission})},
			}
			invoke, state := newAuthorizationTestMiddleware(t, principal, order)
			ctx := transport.NewServerContext(t.Context(), &middlewareTransport{operation: orderv1.OperationOrderFeeServicePreviewFeeQuantity, cookie: tc.cookie})
			_, err := invoke(ctx, request)
			if err != tc.wantError {
				t.Fatalf("鉴权错误=%v，期望%v", err, tc.wantError)
			}
			if state.called != (tc.wantError == nil) {
				t.Fatalf("处理器调用=%v，鉴权错误=%v", state.called, err)
			}
			if state.called && state.orgID != organizationID {
				t.Fatalf("处理器组织=%s，期望订单组织%s", state.orgID, organizationID)
			}
		})
	}
}
