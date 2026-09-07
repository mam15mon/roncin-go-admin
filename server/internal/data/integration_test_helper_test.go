package data

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

func getIntegrationData(t *testing.T) (*Data, func()) {
	t.Helper()
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminDB, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 集成测试数据库失败: %v", err)
	}
	if err := adminDB.PingContext(ctx); err != nil {
		_ = adminDB.Close()
		t.Fatalf("连接 PostgreSQL 集成测试数据库失败: %v", err)
	}

	schemaName := "roncin_test_iso_" + fmt.Sprintf("%x", uuid.New())
	quotedSchema := `"` + schemaName + `"`
	if _, err := adminDB.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		_ = adminDB.Close()
		t.Fatalf("创建隔离测试 Schema 失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := adminDB.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("删除隔离测试 Schema 失败: %v", err)
		}
		_ = adminDB.Close()
	})

	pgxConfig, err := pgx.ParseConfig(source)
	if err != nil {
		t.Fatalf("解析 PostgreSQL 集成测试连接串失败: %v", err)
	}
	pgxConfig.RuntimeParams["search_path"] = schemaName + ",public"
	isolatedSource := stdlib.RegisterConnConfig(pgxConfig)
	t.Cleanup(func() {
		stdlib.UnregisterConnConfig(isolatedSource)
	})
	migrationDB, err := sql.Open("pgx", isolatedSource)
	if err != nil {
		t.Fatalf("打开隔离迁移连接失败: %v", err)
	}
	migrationDB.SetMaxOpenConns(1)
	if err := migration.Apply(ctx, migrationDB, filepath.Join("..", "..", "migrations")); err != nil {
		_ = migrationDB.Close()
		t.Fatalf("在隔离 Schema 执行迁移失败: %v", err)
	}
	if err := migrationDB.Close(); err != nil {
		t.Fatalf("关闭隔离迁移连接失败: %v", err)
	}

	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:             "postgres",
		Source:             isolatedSource,
		AutoMigrate:        false,
		MaxOpenConnections: 8,
		MaxIdleConnections: 8,
	}}, nil)
	if err != nil {
		t.Fatalf("无法连接集成测试数据库: %v", err)
	}
	return data, cleanup
}
