package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

// initializeCompanyAdministrator 仅建立公司角色；公司创建及命令入口另经事务补齐唯一初始管理员成员。
func initializeCompanyAdministrator(ctx context.Context, client *ent.Client, organizationID uuid.UUID) error {
	exists, err := client.Role.Query().Where(role.OrganizationIDEQ(organizationID), role.CodeEQ("administrator")).Exist(ctx)
	if err != nil || exists {
		return err
	}
	ids, err := administratorPermissionIDs(ctx, client, false)
	if err != nil {
		return err
	}
	_, err = client.Role.Create().SetOrganizationID(organizationID).SetCode("administrator").SetName("公司管理员").AddPermissionIDs(ids...).Save(ctx)
	return err
}

// administratorPermissionIDs 只使用 Manifest 内的权限，避免给初始管理员扩入来源不明的目录项。
func administratorPermissionIDs(ctx context.Context, client *ent.Client, system bool) ([]uuid.UUID, error) {
	keys := make([]string, 0)
	for _, definition := range access.Manifest() {
		if access.PermissionAllowedInWorkspace(definition.Key, system) {
			keys = append(keys, definition.Key)
		}
	}
	return client.Permission.Query().Where(permission.KeyIn(keys...)).IDs(ctx)
}
