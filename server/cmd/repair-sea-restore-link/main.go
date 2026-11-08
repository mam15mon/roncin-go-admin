// repair-sea-restore-link 仅显式修复旧恢复流程造成的 ACTIVE 无活动关系订单；默认预检，不改业务数据。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

func main() {
	org := flag.String("organization-id", "", "目标公司ID")
	order := flag.String("order-id", "", "目标订单ID")
	version := flag.Uint64("expected-version", 0, "当前订单版本")
	ended := flag.String("ended-link-id", "", "明确指定的本次退关ENDED关系ID")
	actor := flag.String("actor-id", "", "真实具备目标公司恢复权限的操作者ID")
	reason := flag.String("reason", "", "修复原因")
	apply := flag.Bool("apply", false, "通过全部预检后显式写入；缺省只读预检")
	flag.Parse()
	input, err := parseInput(*org, *order, *ended, *actor, *version, *reason)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	source := os.Getenv("DATABASE_SOURCE")
	if source == "" {
		fmt.Fprintln(os.Stderr, "缺少DATABASE_SOURCE")
		os.Exit(1)
	}
	storage, cleanup, err := data.NewData(&conf.Data{Database: &conf.Data_Database{Driver: "postgres", Source: source}}, slog.Default())
	if err != nil {
		fmt.Fprintln(os.Stderr, "数据库连接初始化失败，请检查环境配置")
		os.Exit(1)
	}
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	report, err := storage.RepairSeaOrderRestoreLink(ctx, input, *apply)
	if err != nil {
		fmt.Fprintln(os.Stderr, "补链预检/执行失败：", err)
		os.Exit(1)
	}
	if err = json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, "输出结果失败")
		os.Exit(1)
	}
}

func parseInput(org, order, ended, actor string, version uint64, reason string) (data.SeaOrderRestoreRepairInput, error) {
	values := []string{org, order, ended, actor}
	ids := make([]uuid.UUID, 4)
	for i, v := range values {
		id, err := uuid.Parse(v)
		if err != nil || id == uuid.Nil {
			return data.SeaOrderRestoreRepairInput{}, fmt.Errorf("公司、订单、退关关系及操作者必须是明确有效的UUID")
		}
		ids[i] = id
	}
	if version == 0 || reason == "" {
		return data.SeaOrderRestoreRepairInput{}, fmt.Errorf("必须指定当前expected-version和reason")
	}
	return data.SeaOrderRestoreRepairInput{OrganizationID: ids[0], OrderID: ids[1], EndedLinkID: ids[2], ActorID: ids[3], ExpectedVersion: version, Reason: reason}, nil
}
