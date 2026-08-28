package data

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
)

const permissionManifestLockKey int64 = 7_266_246_125_832_581_108

// PermissionManifestSyncSummary 汇总一次权限清单同步的变更计数。
type PermissionManifestSyncSummary struct {
	Created  int
	Updated  int
	Removed  int
	Attached int
}

type transactionStarter interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// SyncPermissionManifest 在单个事务内按 access.Manifest 同步权限目录、移除清单外
// 权限，并补齐所有角色缺失的传递依赖；administrator 角色始终持有完整清单。函数
// 不做 Schema 变更，也不创建或修改用户与组织，且只借用连接。
func SyncPermissionManifest(ctx context.Context, database transactionStarter) (*PermissionManifestSyncSummary, error) {
	sqlTx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin permission manifest sync: %w", err)
	}
	defer sqlTx.Rollback()
	if _, err := sqlTx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", permissionManifestLockKey); err != nil {
		return nil, fmt.Errorf("lock permission manifest sync: %w", err)
	}
	driver := entsql.NewDriver(dialect.Postgres, entsql.Conn{ExecQuerier: sqlTx})
	client := ent.NewClient(ent.Driver(driver))

	existing, err := client.Permission.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query permissions: %w", err)
	}
	existingByKey := make(map[string]*ent.Permission, len(existing))
	for _, item := range existing {
		existingByKey[item.Key] = item
	}

	summary := &PermissionManifestSyncSummary{}
	definitions := access.Manifest()
	manifestKeys := make([]string, 0, len(definitions))
	manifestIDs := make(map[string]uuid.UUID, len(definitions))
	for _, definition := range definitions {
		manifestKeys = append(manifestKeys, definition.Key)
		if current, found := existingByKey[definition.Key]; found {
			if current.Name != definition.Name || current.Group != definition.Group || current.Description != definition.Description {
				if _, updateErr := client.Permission.Update().Where(permission.IDEQ(current.ID)).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx); updateErr != nil {
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

	removed, err := client.Permission.Delete().Where(permission.KeyNotIn(manifestKeys...)).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("remove permissions outside manifest: %w", err)
	}
	summary.Removed = removed

	roles, err := client.Role.Query().WithPermissions().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	for _, currentRole := range roles {
		grantedKeys := make([]string, 0, len(currentRole.Edges.Permissions))
		grantedSet := make(map[string]struct{}, len(currentRole.Edges.Permissions))
		for _, item := range currentRole.Edges.Permissions {
			grantedKeys = append(grantedKeys, item.Key)
			grantedSet[item.Key] = struct{}{}
		}
		targetKeys := access.ResolveDependencies(grantedKeys)
		if currentRole.Code == "administrator" {
			targetKeys = manifestKeys
		}
		missingIDs := make([]uuid.UUID, 0, len(targetKeys))
		for _, key := range targetKeys {
			if _, ok := grantedSet[key]; ok {
				continue
			}
			id, ok := manifestIDs[key]
			if !ok {
				return nil, fmt.Errorf("role %s references permission outside manifest: %s", currentRole.Code, key)
			}
			missingIDs = append(missingIDs, id)
		}
		if len(missingIDs) == 0 {
			continue
		}
		if err := attachRolePermissions(ctx, sqlTx, currentRole.ID, missingIDs); err != nil {
			return nil, fmt.Errorf("attach permission dependencies to role %s: %w", currentRole.Code, err)
		}
		summary.Attached += len(missingIDs)
	}
	if err := sqlTx.Commit(); err != nil {
		return nil, fmt.Errorf("commit permission manifest sync: %w", err)
	}
	return summary, nil
}

func attachRolePermissions(ctx context.Context, tx *sql.Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	var statement strings.Builder
	statement.WriteString(`INSERT INTO "role_permissions" ("role_id", "permission_id") VALUES `)
	args := make([]any, 0, len(permissionIDs)+1)
	args = append(args, roleID)
	for index, permissionID := range permissionIDs {
		if index > 0 {
			statement.WriteString(", ")
		}
		fmt.Fprintf(&statement, "($1, $%d)", index+2)
		args = append(args, permissionID)
	}
	_, err := tx.ExecContext(ctx, statement.String(), args...)
	return err
}
