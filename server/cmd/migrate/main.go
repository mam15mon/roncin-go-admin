package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dir := flag.String("dir", "migrations", "迁移文件目录")
	flag.Parse()

	source := os.Getenv("DATABASE_SOURCE")
	if source == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_SOURCE 不能为空")
		os.Exit(1)
	}
	db, err := sql.Open("pgx", source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "连接数据库失败: %v\n", err)
		os.Exit(1)
	}
	var permissionSummary *data.PermissionManifestSyncSummary
	if err := migration.ApplyWithPostStep(ctx, db, *dir, func(conn *sql.Conn) error {
		// 权限清单与代码中的 access.Manifest 保持同步是发版的必要步骤，在迁移锁
		// 释放前完成，避免多实例发版时与其他迁移进程交错执行。
		var syncErr error
		permissionSummary, syncErr = data.SyncPermissionManifest(ctx, conn)
		return syncErr
	}); err != nil {
		fmt.Fprintf(os.Stderr, "执行数据库迁移失败: %v\n", err)
		os.Exit(1)
	}
	if err := data.BackfillSelectorSearchKeywords(ctx, db); err != nil {
		fmt.Fprintf(os.Stderr, "回填下拉候选项拼音检索键失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("权限清单已同步：新增 %d 项，更新 %d 项，移除 %d 项，补齐角色依赖 %d 项\n", permissionSummary.Created, permissionSummary.Updated, permissionSummary.Removed, permissionSummary.Attached)
}
