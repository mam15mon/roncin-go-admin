package data

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

// PermissionManifestSyncSummary 汇总一次权限清单同步的变更计数。
type PermissionManifestSyncSummary struct {
	Created  int
	Updated  int
	Attached int
}

// SyncPermissionManifest 按 access.Manifest 幂等同步权限目录，并为 administrator
// 角色补挂缺失权限。由 cmd/migrate 在数据库迁移后统一调用，本函数不做 Schema
// 变更，也不创建或修改用户与组织；已从 Manifest 删除的权限码不会被移除，需要
// 彻底清理时重置数据库后重新执行完整 bootstrap。函数只借用连接，不关闭传入的
// 数据库句柄。
func SyncPermissionManifest(ctx context.Context, db *sql.DB) (*PermissionManifestSyncSummary, error) {
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))

	existing, err := client.Permission.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query permissions: %w", err)
	}
	existingByKey := make(map[string]*ent.Permission, len(existing))
	for _, item := range existing {
		existingByKey[item.Key] = item
	}

	summary := &PermissionManifestSyncSummary{}
	manifestIDs := make(map[string]uuid.UUID, len(access.Manifest()))
	for _, definition := range access.Manifest() {
		if current, found := existingByKey[definition.Key]; found {
			if current.Name != definition.Name || current.Group != definition.Group || current.Description != definition.Description {
				if _, updateErr := client.Permission.UpdateOneID(current.ID).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx); updateErr != nil {
					return nil, fmt.Errorf("update permission %s: %w", definition.Key, updateErr)
				}
				summary.Updated++
			}
			manifestIDs[definition.Key] = current.ID
			continue
		}
		created, createErr := client.Permission.Create().SetKey(definition.Key).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx)
		if createErr != nil {
			return nil, fmt.Errorf("create permission %s: %w", definition.Key, createErr)
		}
		manifestIDs[definition.Key] = created.ID
		summary.Created++
	}

	adminRoles, err := client.Role.Query().Where(role.CodeEQ("administrator")).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query administrator roles: %w", err)
	}
	for _, adminRole := range adminRoles {
		granted, grantErr := adminRole.QueryPermissions().All(ctx)
		if grantErr != nil {
			return nil, fmt.Errorf("query permissions of role %s: %w", adminRole.Code, grantErr)
		}
		grantedKeys := make(map[string]struct{}, len(granted))
		for _, item := range granted {
			grantedKeys[item.Key] = struct{}{}
		}
		var missingIDs []uuid.UUID
		for key, id := range manifestIDs {
			if _, ok := grantedKeys[key]; !ok {
				missingIDs = append(missingIDs, id)
			}
		}
		if len(missingIDs) == 0 {
			continue
		}
		if _, attachErr := client.Role.UpdateOneID(adminRole.ID).AddPermissionIDs(missingIDs...).Save(ctx); attachErr != nil {
			return nil, fmt.Errorf("attach permissions to role %s: %w", adminRole.Code, attachErr)
		}
		summary.Attached += len(missingIDs)
	}
	return summary, nil
}
