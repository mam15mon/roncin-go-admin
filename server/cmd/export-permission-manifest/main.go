// export-permission-manifest 将 internal/access 的权限码清单导出为 JSON 数组，
// 供仓库根目录的 scripts/generate-permission-keys.mjs 生成前端权限键名常量。
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/roncin/roncin-go-admin/server/internal/access"
)

func main() {
	keys := make([]string, 0, len(access.Manifest()))
	for _, definition := range access.Manifest() {
		keys = append(keys, definition.Key)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(keys); err != nil {
		fmt.Fprintln(os.Stderr, "导出权限清单失败:", err)
		os.Exit(1)
	}
}
