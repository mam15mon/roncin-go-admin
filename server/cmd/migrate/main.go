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
	allowChecksumRepair := flag.Bool("allow-checksum-repair", false, "已执行迁移的校验和与当前文件不一致时重录迁移记录（仅开发环境使用，pnpm run migrate:dev 默认开启）")
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
	var orderOptionsSummary *data.DefaultOrderOptionsSyncSummary
	repairedCount := 0
	if err := migration.ApplyWithOptions(ctx, db, *dir, migration.Options{
		AllowChecksumRepair: *allowChecksumRepair,
		PostStep: func(conn *sql.Conn) error {
			// 权限清单与代码中的 access.Manifest 保持同步是发版的必要步骤，在迁移锁
			// 释放前完成，避免多实例发版时与其他迁移进程交错执行。
			var syncErr error
			permissionSummary, syncErr = data.SyncPermissionManifest(ctx, conn)
			if syncErr != nil {
				return syncErr
			}
			orderOptionsSummary, syncErr = data.SyncDefaultOrderOptions(ctx, conn)
			if syncErr != nil {
				return syncErr
			}
			// 角色库只归属工作台（总部/公司）：先把存量中锚定在部门/团队的角色改挂到
			// 其所属工作台，再自检断言不存在部门/团队锚定行，违反即终止本次发版迁移。
			return data.BackfillRoleWorkspaceAnchors(ctx, conn)
		},
		ChecksumRepaired: func(version, oldChecksum, newChecksum string) {
			repairedCount++
			fmt.Printf("已重录迁移 %s 的校验和（文件在应用后被修改过）：\n  旧 %s\n  新 %s\n", version, oldChecksum, newChecksum)
		},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "执行数据库迁移失败: %v\n", err)
		os.Exit(1)
	}
	if repairedCount > 0 {
		fmt.Println("提示：重录校验和只更新迁移记录，不会补执行文件修改对应的 DDL；若运行时报缺表缺列，请核对迁移差异或重建开发库。")
	}
	if err := data.BackfillSelectorSearchKeywords(ctx, db); err != nil {
		fmt.Fprintf(os.Stderr, "回填下拉候选项拼音检索键失败: %v\n", err)
		os.Exit(1)
	}
	membershipSummary, err := data.SyncBootstrapAdminCompanyMemberships(ctx, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "同步超管公司成员覆盖失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("权限清单已同步：新增 %d 项，更新 %d 项，移除 %d 项，补齐角色依赖 %d 项\n", permissionSummary.Created, permissionSummary.Updated, permissionSummary.Removed, permissionSummary.Attached)
	fmt.Printf("订单主数据种子已同步：补齐 %d 项\n", orderOptionsSummary.Created)
	fmt.Printf("超管公司覆盖已同步：覆盖公司 %d 家，新建角色 %d 个，新建成员关系 %d 条，重启用成员关系 %d 条，补挂角色分配 %d 条\n",
		membershipSummary.Companies, membershipSummary.CreatedRoles, membershipSummary.CreatedMemberships, membershipSummary.ReenabledMemberships, membershipSummary.CreatedRoleAssignments)
}
