package biz

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

func TestPrincipalNotificationPermissionsRequireSystemAdministrator(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*Principal)
		allowed bool
	}{
		{"系统管理员", func(*Principal) {}, true},
		{"系统普通角色", func(p *Principal) { p.RoleGrants[0].RoleCode = "operator" }, false},
		{"公司同码管理员", func(p *Principal) {
			p.Organization.Kind = OrganizationKindCompany
			p.OrganizationNodes[0].Kind = OrganizationKindCompany
		}, false},
		{"仅中文同名", func(p *Principal) {
			p.RoleGrants[0].RoleCode = "operator"
			p.RoleGrants[0].RoleName = "系统管理员"
		}, false},
		{"仅初始化标记", func(p *Principal) { p.RoleGrants[0].RoleCode = "operator"; p.IsBootstrapAdmin = true }, false},
		{"无效管理员授权", func(p *Principal) {
			p.RoleGrants[0].RoleCode = "operator"
			p.RoleGrants = append(p.RoleGrants, RoleGrant{RoleCode: "administrator", DataScope: DataScopeSelf})
		}, false},
		{"未知管理员范围", func(p *Principal) {
			p.RoleGrants[0].RoleCode = "operator"
			p.RoleGrants = append(p.RoleGrants, RoleGrant{RoleCode: "administrator", DataScope: DataScope("unknown")})
		}, false},
		{"系统节点停用", func(p *Principal) { p.OrganizationNodes[0].Disabled = true }, false},
		{"临时定位系统组织", func(p *Principal) { p.WorkspaceOrganizationID = uuid.New() }, false},
		{"缺少系统组织节点", func(p *Principal) { p.OrganizationNodes = nil }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, p, _ := newNotificationPreviewFixture()
			p.RoleGrants[0].Permissions[access.UserRead] = struct{}{}
			tc.change(p)
			for _, key := range []string{access.NotificationPreview, access.NotificationTestSend} {
				if p.HasPermission(key) != tc.allowed || p.HasPermissionInScope(key, DataScopeOrganization) != tc.allowed {
					t.Fatalf("权限%s允许状态错误", key)
				}
				if slices.Contains(p.PermissionKeys(), key) != tc.allowed {
					t.Fatalf("权限投影%s与鉴权不一致", key)
				}
				_, err := p.ResolvePermissionOrganizationScope(key)
				if (err == nil) != tc.allowed {
					t.Fatalf("范围解析%s:%v", key, err)
				}
			}
			if !p.HasPermission(access.UserRead) || !slices.Contains(p.PermissionKeys(), access.UserRead) {
				t.Fatal("相邻用户读取权限受到通知限制影响")
			}
		})
	}
}

func TestNotificationTestRejectsNonAdministratorBeforeSideEffects(t *testing.T) {
	for _, company := range []bool{false, true} {
		uc, repo, sender, p, _ := newNotificationPreviewFixture()
		if company {
			p.Organization.Kind = OrganizationKindCompany
			p.OrganizationNodes[0].Kind = OrganizationKindCompany
		} else {
			p.RoleGrants[0].RoleCode = "ordinary"
		}
		if _, err := uc.Catalog(t.Context(), p); err != ErrPermissionDenied {
			t.Fatalf("目录:%v", err)
		}
		if _, err := uc.Preview(t.Context(), p, "USER_AUTHORIZED", nil); err != ErrPermissionDenied {
			t.Fatalf("预览:%v", err)
		}
		if err := uc.Send(t.Context(), p, "USER_AUTHORIZED", nil); err != ErrPermissionDenied {
			t.Fatalf("发送:%v", err)
		}
		if len(repo.requested) != 0 || len(sender.messages) != 0 {
			t.Fatal("拒绝前仍读取绑定或发送消息")
		}
	}
}
