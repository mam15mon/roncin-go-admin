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
	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
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
	config, err := loadConfig()
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
		summary, seedErr := seedBranchCompanies(context.Background(), config.databaseSource)
		if seedErr != nil {
			logger.Error("seed default branch companies failed", "error", seedErr)
			os.Exit(1)
		}
		logProvisioningSummary(logger, summary, "默认公司种子及初始管理员授权已处理")
		return
	}
	summary, err := bootstrapWithSummary(context.Background(), config, "migrations")
	if err != nil {
		logger.Error("bootstrap admin failed", "error", err)
		os.Exit(1)
	}
	logProvisioningSummary(logger, summary, "初始管理员初始化或补齐已处理")
}

func loadConfig() (*bootstrapConfig, error) {
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
	// 首次创建所需账号资料在事务内确认账号表为空后校验；重复执行不要求密码。
	return config, nil
}

func bootstrapWithMigrations(ctx context.Context, config *bootstrapConfig, migrationDirectory string) error {
	_, err := bootstrapWithSummary(ctx, config, migrationDirectory)
	return err
}

func bootstrapWithSummary(ctx context.Context, config *bootstrapConfig, migrationDirectory string) (*data.AdministratorProvisioningSummary, error) {
	sqlDB, err := sql.Open("pgx", config.databaseSource)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	defer client.Close()
	if err := migration.Apply(ctx, sqlDB, migrationDirectory); err != nil {
		return nil, fmt.Errorf("apply formal migrations: %w", err)
	}
	var summary *data.AdministratorProvisioningSummary
	err = data.WithClientTx(ctx, client, func(tx *ent.Tx) error {
		var provisionErr error
		summary, provisionErr = data.BootstrapAdministrator(ctx, tx, data.BootstrapAdministratorInput{
			Username: config.username, DisplayName: config.displayName, PlainPassword: config.plainPassword, HashPassword: password.Hash,
			OrganizationCode: config.organizationCode, OrganizationName: config.organizationName,
		})
		return provisionErr
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func logProvisioningSummary(logger *slog.Logger, summary *data.AdministratorProvisioningSummary, message string) {
	result := "已完成全部可用工作台授权"
	if len(summary.Skipped) > 0 {
		result = "已完成可处理项，仍有跳过项，请人工恢复后重跑"
	}
	logger.Info(message, "result", result, "username", summary.Username, "account.created", summary.AccountCreated, "companies.created", summary.CompaniesCreated, "memberships.created", summary.MembershipsCreated, "bindings.created", summary.BindingsCreated, "workspaces.satisfied", summary.Satisfied, "companies.skipped", len(summary.Skipped))
	for _, item := range summary.Skipped {
		logger.Warn("公司管理员授权已跳过", "organization.id", item.OrganizationID.String(), "organization.code", item.OrganizationCode, "reason", item.Reason)
	}
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
// 此入口不创建账号；补建默认公司并补齐唯一初始管理员的实际工作台授权。
func seedBranchCompanies(ctx context.Context, databaseSource string) (*data.AdministratorProvisioningSummary, error) {
	sqlDB, err := sql.Open("pgx", databaseSource)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	defer client.Close()
	systemWorkspace, err := client.Organization.Query().Where(organizationent.KindEQ(organizationent.KindSystem)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("locate systemWorkspace: %w", err)
	}
	var summary *data.AdministratorProvisioningSummary
	err = data.WithClientTx(ctx, client, func(tx *ent.Tx) error {
		var seedErr error
		summary, seedErr = data.CreateDefaultBranchCompanies(ctx, tx, systemWorkspace.ID)
		return seedErr
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}
