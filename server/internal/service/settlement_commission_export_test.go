package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

func commissionExportPrincipal(org uuid.UUID) *biz.Principal {
	return &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "export", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceCommissionExport: {}}}}}
}

func TestExportCommissionsRejectsIdentityPermissionAndCompanyBeforeUsecase(t *testing.T) {
	org := uuid.New()
	for _, name := range []string{"无身份", "仅查询权限", "系统工作台", "未知工作台", "越权公司"} {
		t.Run(name, func(t *testing.T) {
			service, repo := newCommissionService(org)
			p := commissionExportPrincipal(org)
			request := &v1.ExportCommissionsRequest{}
			ctx := context.Background()
			switch name {
			case "无身份":
			case "仅查询权限":
				p.RoleGrants[0].Permissions = map[string]struct{}{access.FinanceCommissionRead: {}}
			case "系统工作台":
				p.Organization.Kind = biz.OrganizationKindSystem
				p.OrganizationNodes[0].Kind = biz.OrganizationKindSystem
			case "未知工作台":
				p.Organization.Kind = "UNKNOWN"
				p.OrganizationNodes[0].Kind = "UNKNOWN"
			case "越权公司":
				id := uuid.NewString()
				request.OrganizationId = &id
			}
			if name != "无身份" {
				ctx = biz.WithPrincipal(ctx, p)
			}
			response, err := service.ExportCommissions(ctx, request)
			if err == nil || response != nil || repo.exportCalls != 0 || repo.exportAuditLog != nil {
				t.Fatalf("response=%v err=%v queries=%d audit=%v", response, err, repo.exportCalls, repo.exportAuditLog)
			}
		})
	}
}

func TestExportCommissionsWireStatusPresenceAndUnknown(t *testing.T) {
	for _, tt := range []struct {
		name, wire string
		invalid    bool
	}{
		{name: "省略状态", wire: `{}`},
		{name: "合法状态", wire: `{"status":"FINANCE_COMMISSION_STATUS_PAID"}`},
		{name: "显式零枚举", wire: `{"status":0}`, invalid: true},
		{name: "未知数字枚举", wire: `{"status":999}`, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			org := uuid.New()
			service, repo := newCommissionService(org)
			request := &v1.ExportCommissionsRequest{}
			if err := protojson.Unmarshal([]byte(tt.wire), request); err != nil {
				t.Fatal(err)
			}
			response, err := service.ExportCommissions(biz.WithPrincipal(context.Background(), commissionExportPrincipal(org)), request)
			if tt.invalid {
				if !errors.Is(err, biz.ErrCommissionInvalid) || response != nil || repo.exportCalls != 0 || repo.exportAuditLog == nil || repo.exportAuditLog.Result != "failure" || repo.exportAuditLog.Details["failure_stage"] != "validation" {
					t.Fatalf("response=%v err=%v queries=%d audit=%#v", response, err, repo.exportCalls, repo.exportAuditLog)
				}
				if _, exists := repo.exportAuditLog.Details["status"]; exists {
					t.Fatal("非法状态原文进入审计")
				}
			} else if err != nil || response == nil || repo.exportCalls != 1 || repo.exportAuditLog == nil || repo.exportAuditLog.Result != "success" {
				t.Fatalf("response=%v err=%v queries=%d audit=%#v", response, err, repo.exportCalls, repo.exportAuditLog)
			}
		})
	}
}
