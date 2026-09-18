// backfill-commission-snapshots 一次性存量提成行复算快照回填辅助入口。
//
// 以原提成行首次计算并写入 commission_amount 的 created_at 为唯一截止时点，
// 只使用原提成行不可变费用快照与截止时点前仍可追溯的账单行事实确定性重建
// 历史总应收/总应付分母，并按历史版本纯计算器复核已存提成结果。可还原行写
// READY + MIGRATED + 回填算法版本 + 证据集合哈希；无法确定唯一截止时点、历史
// 事实曾被原地修改或复算不一致的行标记 UNAVAILABLE + 稳定原因码，不覆盖原
// 提成金额、不填写猜测分母。执行结束输出不含敏感报文的迁移报告。
//
// 用法：DATABASE_SOURCE="postgres://..." go -C server run ./cmd/backfill-commission-snapshots
// 回填带 snapshot_status IS NULL 守卫，可安全重入；READY 行不会被再次改写。
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"

	"github.com/roncin/roncin-go-admin/server/internal/data"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
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
	report, err := data.BackfillCommissionLineSnapshots(ctx, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "存量提成行快照回填失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("存量提成行快照回填完成（算法版本 %s）\n", report.BackfillVersion)
	fmt.Printf("  处理行数：%d\n", report.TotalLines)
	fmt.Printf("  READY + MIGRATED：%d\n", report.ReadyCount)
	fmt.Printf("  UNAVAILABLE：%d\n", report.UnavailableCount)
	if len(report.UnavailableByReason) > 0 {
		reasons := make([]string, 0, len(report.UnavailableByReason))
		for reason := range report.UnavailableByReason {
			reasons = append(reasons, reason)
		}
		sort.Strings(reasons)
		for _, reason := range reasons {
			fmt.Printf("    %s：%d\n", reason, report.UnavailableByReason[reason])
		}
	}
}
