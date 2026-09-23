# 实现计划：费用科目种子丰富

前置：prd.md 权威清单（122 条）为准；所有验证命令在仓库根目录执行。

## 步骤

1. [x] PRD 权威清单落定
2. [x] 生成检索键：临时 Go 程序（server/cmd/tmp-fee-keywords，跑完删除）调用
   `internal/platform/searchtext.Build(name_zh, name_en, alias)`，为 122 条模板 +
   5 类别 + 5 异常 + 7 计费单位输出 search_keywords，写入迁移 SQL。
3. [x] 编写 `server/migrations/20260923xxxxxx_fee_catalog_seed.sql`：
   a. 计费单位 ON CONFLICT DO NOTHING；
   b. 费用类别/异常情况 ON CONFLICT (kind, code) DO NOTHING；
   c. 模板 VALUES 批量 INSERT ... ON CONFLICT (fee_code) DO NOTHING，
      charge_category_id/billing_unit_id/abnormal_case_id 按 code 子查询解析，
      缺失引用 fail-fast（RAISE EXCEPTION）；
   d. UPDATE 模板 THC/STORAGE 类别 → PORT_OPS；
   e. DO $$ 公司追加块（仿 20260922100000 第 79-120 行）：同码跳过、
      应税劳务按 (公司,名称) 找不到则建、插 fee_settings。
4. [x] 集成测试 `internal/platform/migration/fee_catalog_seed_integration_test.go`：
   - 全链迁移 + 一家公司 → 断言模板 122、公司科目 122、劳务/单位/类别/异常计数；
   - 预置一条本地同码科目（如 OF）→ 断言追加跳过、本地行未被改写。
5. [x] 验证：
   - `RONCIN_POSTGRES_MIGRATION_TEST=1 RONCIN_INTEGRATION_DATABASE_SOURCE=... go -C server test ./internal/platform/migration/`（全新库冷启动 + 新测试）；
   - `pnpm run migrate:dev`（开发库）+ psql 抽查：计数、THC 类别、DAY 单位、KC 异常挂接、
     公司 122 条、search_keywords 抽查与 Build 输出一致；
   - 重放一次确认幂等。
6. [x] 收尾：删除临时生成器；spec 若需更新（种子/迁移注意事项）走 trellis-update-spec；
   按 Phase 3.4 提交（迁移 + 测试 + 任务产物同一提交）。

## 风险与回滚

- 迁移单事务，失败整体回滚；不删除任何既有行，只 INSERT/有限 UPDATE。
- 开发库重放安全（DO NOTHING）；生产严格校验和，只执行一次。
- 回滚点：revert 迁移文件即可，种子数据无下游引用（新科目未被订单引用）。
