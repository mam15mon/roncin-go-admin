# 最终验收门禁

- `pnpm run check:fast`：退出码 0，全部通过，172.8 秒；日志 `/tmp/commission-company-check-fast.log`。
- WEB：390 测试文件通过，1 文件既有跳过；2111 用例通过、11 既有跳过。权限/Proto生成一致性、架构边界、重复检测、Biome、TypeScript 均通过。
- SERVER：标准单测、分层、重复检测、Proto lint、vet、漏洞扫描均通过，130.1 秒。标准门禁不注入 PostgreSQL 参数，不能据此把集成测试跳过算作实测通过。
- 本次真实 PostgreSQL 投影回归由实施与独立审核分别完整执行：均 6 子场景 PASS、无 SKIP，详见 implementation.md、review.md。
- 现场正式开发接口只读验收通过，详见 live-acceptance.md；没有重新操作恢复或已取消调整。
- 独立 review 无阻塞项；`git diff --check` 通过。
- 只读投影修复不改 Proto、Schema、依赖或构建配置，无需重生成和整站生产构建；为现场验证单独构建过当前 Go 服务并验证正式开发服务已自动重载。
