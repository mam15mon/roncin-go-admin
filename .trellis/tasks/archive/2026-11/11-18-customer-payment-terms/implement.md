# 执行计划：客户收款约定收口

开始前：
- `git status` 确认工作区干净。
- 先读 `prd.md`、`design.md`，再读 `implement.jsonl` 里列出的规范。
- 建议先完成兄弟任务 `11-18-bill-due-date-fixed`（不是强制依赖）。

## 阶段 1：Schema、迁移、契约与服务端

1. 改 Ent Schema（design §1.1），执行 `go -C server generate ./...`。
2. 写正式迁移（design §1.2），文件名用当前链尾之后的时间戳。第 0 步是额度币种守卫，
   口径要和 `research/preflight-inventory.sql` 的 E 段保持一致。
   ⚠️ 迁移文件一旦放进 `server/migrations/`，`pnpm dev`、`pnpm run migrate:dev` 都会自动把它应用到开发库。
   **在第 8 步取得用户确认之前，不要启动 `pnpm dev`，也不要执行 `migrate:dev`。**
   如果用户正开着开发服务，先提醒用户：重启服务就会触发迁移。
3. 改 Proto（design §2），执行 `make -C server api`，再执行 `pnpm run generate:web-client`；检查生成差异。
4. 改 Biz、Data、Service（design §3），包括 `finance_bill_credit.go` 的信用汇总、`savePartnerRoleSettlementRules`、开发种子。
5. 如果改了 `internal/access/manifest.go` 的权限名称或描述，执行 `pnpm run generate:permission-keys`。
6. **在隔离库验证迁移：** 写迁移集成测试（design §6「迁移集成测试」的全部用例，包括两种守卫失败和守卫不误伤），
   在独立的 PostgreSQL 上跑通。这些测试在 `internal/data` 包里，所以要等第 4 步完成、包能编译后再写。
   迁移执行器会把整个文件放在一个事务里执行（`internal/platform/migration/runner.go`），守卫失败时整份迁移都会回滚。
   同时补写、更新其余后端测试（design §6 后端部分）。
7. 验证（这一步还不碰开发库）：
   ```bash
   go -C server build ./...
   go -C server vet ./...
   go -C server test -p 32 ./internal/biz/ -run 'PartnerSettlement|PartnerCredit|FinanceBill'
   go -C server test -p 32 ./internal/data/ -run 'Partner|Migration|FinanceBillCredit'
   go -C server test -p 32 ./internal/service/ -run 'Partner'
   git diff --check
   ```
8. **只读盘点开发库，然后暂停：**
   ```bash
   set -a && . ./.env.local && set +a
   psql "$DATABASE_SOURCE" -X -v ON_ERROR_STOP=1 -P pager=off \
     -f .trellis/tasks/11-18-customer-payment-terms/research/preflight-inventory.sql
   ```
   注意：盘点脚本按旧表结构编写，必须在开发库执行迁移**之前**运行。
   把预计删除的记录（B、C 段）、合并前后的取值（D 段）和币种异常（E 段）整理给用户。
   E 段不为空时，迁移一定会失败，要先请用户在客户档案里修正额度。
   **用户确认后，才执行** `pnpm run migrate:dev`。
   规划时的盘点快照见 `research/preflight-result-2026-11-18.md`，只供参考，必须重新盘点。
9. 迁移完成后，连续执行两次 `pnpm run sync:dev`，确认可以重复执行。

## 阶段 2：前端

10. 新建 `web/src/pages/partners/components/PaymentTermsFields.tsx`（design §4.1）。
11. 改 `SettlementSection.tsx`、`partner-detail.tsx`（design §4.2）。
12. 改 `SettlementRulesPanel.tsx`、`partner-secondary.tsx`（design §4.3）。
13. 清理全部前端对废弃字段和枚举的引用：
    ```bash
    grep -rn "settlementMethod\|statementMode\|settlementBase\|settlementDay\|settlementCycleDays\|settlementCurrency\|creditDays\|PartnerSettlementMethod\|PartnerStatementMode\|PartnerSettlementBase" web/src --include=*.ts --include=*.tsx | grep -v services/roncin
    ```
    （`pages/finance/cashflows` 里的「结算方式」是资金流水的收付方式，和这里无关，不要改。）
14. 更新前端测试（design §6 前端部分）。
15. 验证：
    ```bash
    pnpm --dir web exec vitest run src/pages/partners
    pnpm --dir web tsc
    pnpm --dir web exec biome check <改动文件>
    pnpm run check:architecture
    git diff --check
    ```
16. 浏览器实测。用户偏好用 REA 场景代替大范围 vitest：
    - 命令：`rea-local-browser capture-browser-scenario`；`browser.mode=launch`，
      `executable_path=/usr/bin/google-chrome-stable`，前端 `http://127.0.0.1:8001`，
      `after_each_step:["screenshot"]`。不要连接 19322 端口上用户已开的浏览器页签。
    - 登录分两步：点 `.ant-segmented-item:has-text('账号登录')` → 填账号 → 点「下一步」→
      填密码 → 点 `button[type=submit]`。密码通过 `secrets` 环境变量传入，不写进场景文件。
      测试账号来自 `pnpm run sync:dev` 种子（如 `liming`）。
    - 场景：
      1. 打开一个客户档案 → 收款约定改为「建账后 45 天」→ 保存 → 到建账工作台选该客户的应收费用 → 默认账期为 45；
      2. 改为「未约定」→ 保存 → 建账默认无账期；
      3. 打开一个供应商档案 → 看不到收款约定分节；
      4. 列表页侧边抽屉的「收款约定」页签显示与主档页一致。

## 阶段 3：规范

17. 按 design §5 更新规范文档。如果兄弟任务已经完成，复核 `glossary.md` 中「账单」和「收款约定」两条词条的口径一致。

## 提交

- 契约、生成物、Schema、迁移和服务端实现放进同一次提交：
  `refactor(partner): 客户结算规则收口为单条收款约定`
- 前端与规范可以放在同一次提交，或紧随其后单独提交：
  `refactor(web): 客户档案只维护收款期限与信用额度`
- 两次提交都完成后，才算这个子任务可以交付（契约改动不能只上线一半）。

## 收尾

18. `pnpm run check:fast`（这次改了契约、生成物和迁移，属于高风险变更，必须跑完整门禁）。
19. 在父任务 `11-18-payment-terms-due-date` 的跨子任务验收里勾选与本子任务有关的项目。

## 回滚点

- 迁移执行前：可以直接 revert 代码提交。
- 迁移执行后：被删除的规则行只能靠数据库备份恢复。所以第 8 步必须先盘点、再取得用户确认。
