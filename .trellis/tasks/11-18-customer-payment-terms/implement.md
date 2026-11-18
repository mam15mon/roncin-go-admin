# 执行计划：客户收款约定收口

开始前：
- `git status` 确认工作区干净。
- 先读 `prd.md`、`design.md`，再读 `implement.jsonl` 里列出的规范。
- 建议先完成兄弟任务 `11-18-bill-due-date-fixed`（不是强制依赖）。

## 阶段 1：Schema、迁移、契约与服务端

1. 改 Ent Schema（design §1.1），执行 `go -C server generate ./...`。
2. 写正式迁移（design §1.2），文件名用当前链尾之后的时间戳。
3. **执行迁移前暂停：** 告诉用户本迁移会删除开发库里的非客户规则、停用规则和重复规则，
   取得确认后再执行 `pnpm run migrate:dev`。
4. 改 Proto（design §2），执行 `make -C server api`，再执行 `pnpm run generate:web-client`；检查生成差异。
5. 改 Biz、Data、Service（design §3），包括 `finance_bill_credit.go` 的信用汇总、`savePartnerRoleSettlementRules`、开发种子。
6. 如果改了 `internal/access/manifest.go` 的权限名称或描述，执行 `pnpm run generate:permission-keys`。
7. 补写和更新后端测试，包括迁移集成测试（design §6 后端部分）。
8. 验证：
   ```bash
   go -C server build ./...
   go -C server vet ./...
   go -C server test -p 32 ./internal/biz/ -run 'PartnerSettlement|PartnerCredit|FinanceBill'
   go -C server test -p 32 ./internal/data/ -run 'Partner|Migration|FinanceBillCredit'
   go -C server test -p 32 ./internal/service/ -run 'Partner'
   pnpm run sync:dev   # 连续执行两次，确认可重复
   git diff --check
   ```

## 阶段 2：前端

9. 新建 `web/src/pages/partners/components/PaymentTermsFields.tsx`（design §4.1）。
10. 改 `SettlementSection.tsx`、`partner-detail.tsx`（design §4.2）。
11. 改 `SettlementRulesPanel.tsx`、`partner-secondary.tsx`（design §4.3）。
12. 清理全部前端对废弃字段和枚举的引用：
    ```bash
    grep -rn "settlementMethod\|statementMode\|settlementBase\|settlementDay\|settlementCycleDays\|settlementCurrency\|creditDays\|PartnerSettlementMethod\|PartnerStatementMode\|PartnerSettlementBase" web/src --include=*.ts --include=*.tsx | grep -v services/roncin
    ```
    （`pages/finance/cashflows` 里的「结算方式」是资金流水的收付方式，和这里无关，不要改。）
13. 更新前端测试（design §6 前端部分）。
14. 验证：
    ```bash
    pnpm --dir web exec vitest run src/pages/partners
    pnpm --dir web tsc
    pnpm --dir web exec biome check <改动文件>
    pnpm run check:architecture
    git diff --check
    ```
15. 浏览器实测。用户偏好用 REA 场景代替大范围 vitest：
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

16. 按 design §5 更新规范文档。如果兄弟任务已经完成，复核 `glossary.md` 中「账单」和「收款约定」两条词条的口径一致。

## 提交

- 契约、生成物、Schema、迁移和服务端实现放进同一次提交：
  `refactor(partner): 客户结算规则收口为单条收款约定`
- 前端与规范可以放在同一次提交，或紧随其后单独提交：
  `refactor(web): 客户档案只维护收款期限与信用额度`
- 两次提交都完成后，才算这个子任务可以交付（契约改动不能只上线一半）。

## 收尾

17. `pnpm run check:fast`（这次改了契约、生成物和迁移，属于高风险变更，必须跑完整门禁）。
18. 在父任务 `11-18-payment-terms-due-date` 的跨子任务验收里勾选与本子任务有关的项目。

## 回滚点

- 迁移执行前：可以直接 revert 代码提交。
- 迁移执行后：被删除的规则行只能靠数据库备份恢复。所以第 3 步必须先取得用户确认。
