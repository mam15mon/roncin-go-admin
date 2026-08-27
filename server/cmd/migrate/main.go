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
	if err := migration.Apply(ctx, db, *dir); err != nil {
		fmt.Fprintf(os.Stderr, "执行数据库迁移失败: %v\n", err)
		os.Exit(1)
	}
	if err := data.BackfillSelectorSearchKeywords(ctx, db); err != nil {
		fmt.Fprintf(os.Stderr, "回填下拉候选项拼音检索键失败: %v\n", err)
		os.Exit(1)
	}
}
