package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

// initializeCompanyAdministrator 提供公司初始角色，由系统管理显式分配成员；不自动授予任何账号。
func initializeCompanyAdministrator(ctx context.Context, client *ent.Client, organizationID uuid.UUID) error {
	exists, err := client.Role.Query().Where(role.OrganizationIDEQ(organizationID), role.CodeEQ("administrator")).Exist(ctx)
	if err != nil || exists {
		return err
	}
	permissions, err := client.Permission.Query().All(ctx)
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(permissions))
	for _, permission := range permissions {
		if access.PermissionAllowedInWorkspace(permission.Key, false) {
			ids = append(ids, permission.ID)
		}
	}
	_, err = client.Role.Create().SetOrganizationID(organizationID).SetCode("administrator").SetName("公司管理员").SetDataScope(role.DataScopeOrganization).AddPermissionIDs(ids...).Save(ctx)
	return err
}
