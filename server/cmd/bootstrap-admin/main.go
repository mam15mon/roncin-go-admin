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
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
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
	args := os.Args[1:]
	syncOnly := slices.Contains(args, "--sync-permissions")
	seedBranchesOnly := slices.Contains(args, "--seed-branches")
	config, err := loadConfig(syncOnly || seedBranchesOnly)
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
	if seedBranchesOnly {
		created, seedErr := seedBranchCompanies(context.Background(), config.databaseSource)
		if seedErr != nil {
			logger.Error("seed default branch companies failed", "error", seedErr)
			os.Exit(1)
		}
		logger.Info("default branch companies seeded", "created", created)
		return
	}
	if err := bootstrap(context.Background(), config); err != nil {
		logger.Error("bootstrap admin failed", "error", err)
		os.Exit(1)
	}
	logger.Info("bootstrap admin completed", "username", config.username, "organization.code", config.organizationCode)
}

func loadConfig(toolsOnly bool) (*bootstrapConfig, error) {
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
	if config.organizationCode == "" {
		config.organizationCode = "SYSTEM"
	}
	if config.organizationName == "" {
		config.organizationName = "系统管理"
	}
	if toolsOnly {
		return config, nil
	}
	if config.username == "" || config.displayName == "" || config.plainPassword == "" || config.organizationCode == "" || config.organizationName == "" {
		return nil, fmt.Errorf("all BOOTSTRAP_* variables are required for full bootstrap")
	}
	return config, nil
}

func bootstrap(ctx context.Context, config *bootstrapConfig) error {
	return bootstrapWithMigrations(ctx, config, "migrations")
}

func bootstrapWithMigrations(ctx context.Context, config *bootstrapConfig, migrationDirectory string) error {
	sqlDB, err := sql.Open("pgx", config.databaseSource)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	defer client.Close()
	if err := migration.Apply(ctx, sqlDB, migrationDirectory); err != nil {
		return fmt.Errorf("apply formal migrations: %w", err)
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
	return data.WithClientTx(ctx, client, func(tx *ent.Tx) error {
		organization, err := tx.Organization.Create().SetCode(config.organizationCode).SetName(config.organizationName).SetKind("system").SetBaseCurrency("CNY").Save(ctx)
		if err != nil {
			return fmt.Errorf("create organization: %w", err)
		}
		// A 型主数据种子按 kind+code 幂等跳过；迁移种子（如费用类别）已落的行不重复写入。
		if err := data.CreateDefaultOrderOptions(ctx, tx); err != nil {
			return err
		}
		if err := data.CreateDefaultCountries(ctx, tx); err != nil {
			return err
		}
		existingPermissions, err := tx.Permission.Query().All(ctx)
		if err != nil {
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
				return fmt.Errorf("synchronize permission %s: %w", definition.Key, err)
			}
			if access.PermissionAllowedInWorkspace(permission.Key, true) {
				permissions = append(permissions, permission)
			}
		}
		// 默认分公司在权限目录同步后落库，使分公司内置的 administrator 角色直接携带全量权限。
		if _, err := data.CreateDefaultBranchCompanies(ctx, tx, organization.ID); err != nil {
			return fmt.Errorf("create default branch companies: %w", err)
		}
		adminRole, err := tx.Role.Create().SetOrganizationID(organization.ID).SetCode("administrator").SetName("系统管理员").SetDataScope(role.DataScopeAll).AddPermissions(permissions...).Save(ctx)
		if err != nil {
			return fmt.Errorf("create administrator role: %w", err)
		}
		account, err := tx.User.Create().SetUsername(config.username).SetDisplayName(config.displayName).SetPasswordHash(passwordHash).SetIsBootstrapAdmin(true).Save(ctx)
		if err != nil {
			return fmt.Errorf("create administrator: %w", err)
		}
		membership, err := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(organization.ID).SetPrimary(true).Save(ctx)
		if err != nil {
			return fmt.Errorf("create administrator membership: %w", err)
		}
		if _, err := tx.RoleAssignment.Create().SetMembershipID(membership.ID).SetRoleID(adminRole.ID).Save(ctx); err != nil {
			return fmt.Errorf("assign administrator role: %w", err)
		}
		return nil
	})
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

// seedBranchCompanies 在已完成 bootstrap 的库上幂等补建默认分公司种子。
// 完整 bootstrap 只允许跑在全新库上，存量开发库通过本入口补齐新增加的分公司。
func seedBranchCompanies(ctx context.Context, databaseSource string) (int, error) {
	sqlDB, err := sql.Open("pgx", databaseSource)
	if err != nil {
		return 0, fmt.Errorf("open database: %w", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.PingContext(ctx); err != nil {
		return 0, fmt.Errorf("connect database: %w", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	defer client.Close()
	systemWorkspace, err := client.Organization.Query().Where(organizationent.KindEQ(organizationent.KindSystem)).Only(ctx)
	if err != nil {
		return 0, fmt.Errorf("locate systemWorkspace: %w", err)
	}
	created := 0
	err = data.WithClientTx(ctx, client, func(tx *ent.Tx) error {
		var seedErr error
		created, seedErr = data.CreateDefaultBranchCompanies(ctx, tx, systemWorkspace.ID)
		return seedErr
	})
	return created, err
}
