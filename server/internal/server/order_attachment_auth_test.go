package server

import (
	"testing"

	"github.com/google/uuid"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestOrderAttachmentCommandsRequireTheirOwnOrderPermission(t *testing.T) {
	org, otherOrg := uuid.New(), uuid.New()
	target := &biz.Order{ID: uuid.New(), OrganizationID: org, BusinessType: biz.OrderBusinessSE}
	uc := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: target}, nil, nil, nil, newReminderModeCreditControl(), nil)
	cases := []struct {
		request   any
		operation access.OrderOperation
	}{
		{&orderv1.PrepareAttachmentUploadRequest{OrderId: target.ID.String()}, access.OrderAttachmentRegister},
		{&orderv1.RegisterAttachmentRequest{OrderId: target.ID.String()}, access.OrderAttachmentRegister},
		{&orderv1.GetAttachmentAccessRequest{OrderId: target.ID.String(), Id: uuid.NewString()}, access.OrderAttachmentRead},
	}
	for _, test := range cases {
		principal := &biz.Principal{Organization: biz.Organization{Kind: biz.OrganizationKindCompany, ID: org}, OrganizationNodes: serverOrganizationNodes(org, otherOrg)}
		for _, scenario := range []struct {
			permission     access.OrderOperation
			organizationID uuid.UUID
			allowed        bool
		}{
			{access.OrderRead, org, false}, {test.operation, otherOrg, false}, {test.operation, org, true},
		} {
			principal.Organization.ID = scenario.organizationID
			principal.RoleGrants = []biz.RoleGrant{serverRoleGrant("attachment-role", biz.DataScopeAll, []string{access.OrderPermission(access.OrderBusinessSE, scenario.permission)})}
			resolved, direct := requestOrder(t.Context(), test.request, uc, orderOrganizationScopes(principal, test.operation, orderOperationWrites(test.operation)))
			if !direct || (resolved != nil) != scenario.allowed {
				t.Fatalf("%T allowed=%t resolved=%t", test.request, scenario.allowed, resolved != nil)
			}
		}
	}
}
