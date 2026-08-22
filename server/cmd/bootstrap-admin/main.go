package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type bootstrapConfig struct {
	databaseSource   string
	username         string
	displayName      string
	plainPassword    string
	organizationCode string
	organizationName string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	syncOnly := slices.Contains(os.Args[1:], "--sync-permissions")
	config, err := loadConfig(syncOnly)
	if err != nil {
		logger.Error("bootstrap configuration invalid", "error", err)
		os.Exit(1)
	}
	if syncOnly {
		summary, syncErr := syncPermissions(context.Background(), config.databaseSource)
		if syncErr != nil {
			logger.Error("sync permission manifest failed", "error", syncErr)
			os.Exit(1)
		}
		logger.Info("permission manifest synced", "created", summary.created, "updated", summary.updated, "attached_to_administrator", summary.attached)
		return
	}
	if err := bootstrap(context.Background(), config); err != nil {
		logger.Error("bootstrap admin failed", "error", err)
		os.Exit(1)
	}
	logger.Info("bootstrap admin completed", "username", config.username, "organization.code", config.organizationCode)
}

func loadConfig(syncOnly bool) (*bootstrapConfig, error) {
	config := &bootstrapConfig{
		databaseSource:   os.Getenv("DATABASE_SOURCE"),
		username:         strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_USERNAME"))),
		displayName:      strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_DISPLAY_NAME")),
		plainPassword:    os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		organizationCode: strings.TrimSpace(os.Getenv("BOOTSTRAP_ORGANIZATION_CODE")),
		organizationName: strings.TrimSpace(os.Getenv("BOOTSTRAP_ORGANIZATION_NAME")),
	}
	if config.databaseSource == "" {
		return nil, fmt.Errorf("DATABASE_SOURCE is required")
	}
	if syncOnly {
		return config, nil
	}
	if config.username == "" || config.displayName == "" || config.plainPassword == "" || config.organizationCode == "" || config.organizationName == "" {
		return nil, fmt.Errorf("all BOOTSTRAP_* variables are required for full bootstrap")
	}
	return config, nil
}

func bootstrap(ctx context.Context, config *bootstrapConfig) error {
	sqlDB, err := sql.Open("pgx", config.databaseSource)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	defer client.Close()
	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("synchronize fresh schema: %w", err)
	}
	exists, err := client.User.Query().Exist(ctx)
	if err != nil {
		return fmt.Errorf("check existing users: %w", err)
	}
	if exists {
		return fmt.Errorf("users already exist; bootstrap is only allowed on a fresh database")
	}
	passwordHash, err := password.Hash(config.plainPassword)
	if err != nil {
		return err
	}
	tx, err := client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	organization, err := tx.Organization.Create().SetCode(config.organizationCode).SetName(config.organizationName).Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("create organization: %w", err)
	}
	if err := data.CreateDefaultNumberRules(ctx, tx, organization.ID); err != nil {
		tx.Rollback()
		return err
	}
	permissions := make([]*ent.Permission, 0, len(access.Manifest()))
	for _, definition := range access.Manifest() {
		permission, createErr := tx.Permission.Create().SetKey(definition.Key).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx)
		if createErr != nil {
			tx.Rollback()
			return fmt.Errorf("create permission %s: %w", definition.Key, createErr)
		}
		permissions = append(permissions, permission)
	}
	adminRole, err := tx.Role.Create().SetOrganizationID(organization.ID).SetCode("administrator").SetName("系统管理员").SetDataScope(role.DataScopeAll).AddPermissions(permissions...).Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("create administrator role: %w", err)
	}
	account, err := tx.User.Create().SetUsername(config.username).SetDisplayName(config.displayName).SetPasswordHash(passwordHash).Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("create administrator: %w", err)
	}
	membership, err := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(organization.ID).SetPrimary(true).Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("create administrator membership: %w", err)
	}
	if _, err := tx.RoleAssignment.Create().SetMembershipID(membership.ID).SetRoleID(adminRole.ID).Save(ctx); err != nil {
		tx.Rollback()
		return fmt.Errorf("assign administrator role: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap transaction: %w", err)
	}
	return nil
}

type permissionSyncSummary struct {
	created  int
	updated  int
	attached int
}

// syncPermissions 按 access.Manifest 幂等同步权限目录，并为 administrator 角色补挂
// 缺失权限。仅面向开发阶段的增量同步：不创建或修改用户与组织，已从 Manifest
// 删除的权限码不会被移除，需要彻底清理时重置数据库后重新执行完整 bootstrap。
func syncPermissions(ctx context.Context, databaseSource string) (*permissionSyncSummary, error) {
	sqlDB, err := sql.Open("pgx", databaseSource)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	defer client.Close()
	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("synchronize schema: %w", err)
	}

	existing, err := client.Permission.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query permissions: %w", err)
	}
	existingByKey := make(map[string]*ent.Permission, len(existing))
	for _, item := range existing {
		existingByKey[item.Key] = item
	}

	summary := &permissionSyncSummary{}
	manifestIDs := make(map[string]uuid.UUID, len(access.Manifest()))
	for _, definition := range access.Manifest() {
		if current, found := existingByKey[definition.Key]; found {
			if current.Name != definition.Name || current.Group != definition.Group || current.Description != definition.Description {
				if _, updateErr := client.Permission.UpdateOneID(current.ID).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx); updateErr != nil {
					return nil, fmt.Errorf("update permission %s: %w", definition.Key, updateErr)
				}
				summary.updated++
			}
			manifestIDs[definition.Key] = current.ID
			continue
		}
		created, createErr := client.Permission.Create().SetKey(definition.Key).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx)
		if createErr != nil {
			return nil, fmt.Errorf("create permission %s: %w", definition.Key, createErr)
		}
		manifestIDs[definition.Key] = created.ID
		summary.created++
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
		summary.attached += len(missingIDs)
	}
	return summary, nil
}
