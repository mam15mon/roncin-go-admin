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
		logger.Info("permission manifest synced", "created", summary.Created, "updated", summary.Updated, "removed", summary.Removed, "attached", summary.Attached)
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
	organization, err := tx.Organization.Create().SetCode(config.organizationCode).SetName(config.organizationName).SetKind("headquarters").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("create organization: %w", err)
	}
	if err := data.CreateDefaultNumberRules(ctx, tx, organization.ID); err != nil {
		tx.Rollback()
		return err
	}
	if err := data.CreateDefaultOrderOptions(ctx, tx, organization.ID); err != nil {
		tx.Rollback()
		return err
	}
	if err := data.CreateDefaultCountries(ctx, tx, organization.ID); err != nil {
		tx.Rollback()
		return err
	}
	existingPermissions, err := tx.Permission.Query().All(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("query existing permissions: %w", err)
	}
	existingPermissionsByKey := make(map[string]*ent.Permission, len(existingPermissions))
	for _, item := range existingPermissions {
		existingPermissionsByKey[item.Key] = item
	}
	permissions := make([]*ent.Permission, 0, len(access.Manifest()))
	for _, definition := range access.Manifest() {
		permission := existingPermissionsByKey[definition.Key]
		if permission == nil {
			permission, err = tx.Permission.Create().SetKey(definition.Key).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx)
		} else {
			permission, err = permission.Update().SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx)
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("synchronize permission %s: %w", definition.Key, err)
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

// syncPermissions 按 access.Manifest 幂等同步权限目录。完整流程已由 cmd/migrate
// 在迁移后统一执行，此入口仅保留给开发阶段手工触发。
func syncPermissions(ctx context.Context, databaseSource string) (*data.PermissionManifestSyncSummary, error) {
	sqlDB, err := sql.Open("pgx", databaseSource)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return data.SyncPermissionManifest(ctx, sqlDB)
}
