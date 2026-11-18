# 执行计划：账单到期日固定

开始前：`git status` 确认工作区干净；先读 `prd.md`、`design.md`，再读 `implement.jsonl` 列出的规范。

## 阶段 1：后端（单独提交）

1. `server/internal/biz/finance_bill_fees.go`：删除 `UpdateFees` 刷新回调里改日期、改到期日、
   重取汇率的逻辑，以及 `FINANCE_BILL_DUE_DATE_BEFORE_CREATION`。
2. 按 design §2.2 方案 A，把 `refresh` 参数从 `FinanceBillRepo.UpdateFees` 的接口、
   `server/internal/data/finance_bill_fees.go` 的实现和全部测试替身中删掉。
   用 `grep -rn "UpdateFees(" server/internal` 找齐所有调用方。
3. 更新单测和集成测试（design §4 后端部分）。
4. 验证：
   ```bash
   go -C server build ./...
   go -C server vet ./internal/biz/... ./internal/data/...
   go -C server test -p 32 ./internal/biz/ -run 'FinanceBill|BillFee'
   go -C server test -p 32 ./internal/data/ -run 'FinanceBillFee|FinanceBillCorrection'
   git diff --check
   ```
   集成测试需要本地 PostgreSQL；如果无法运行，在提交说明里写明原因。
5. 提交：`fix(finance): 增减账单费用不再顺延到期日和重估汇率`

## 阶段 2：前端（单独提交）

6. 先查 `.trellis/spec/web/frontend/capability-navigation.md`，确认没有现成的同类能力。
7. 在 `web/src/features/finance/bill-creation/` 新增：到期展示纯函数 +
   「到期设置」表单片段组件，并从 `index.ts` 导出（design §3.1）。
8. 改 `BillGroupCard.tsx`、`BillCreationWorkbench.tsx`（表单合并与提交载荷）。
9. 改 `BillEditModal.tsx`、`pages/finance/bills/index.tsx` 的 `submitBill`（design §3.3）。
10. 改 `BillFeeMaintenance.tsx` 的提示文案（design §3.4）；改两处「创建日期」tooltip。
11. 更新、新增前端测试（design §4 前端部分）。
12. 验证：
    ```bash
    pnpm --dir web exec vitest run src/features/finance/bill-creation src/pages/finance/bills
    pnpm --dir web tsc
    pnpm --dir web exec biome check <改动文件>
    pnpm run check:architecture
    git diff --check
    ```
13. 浏览器实测。用户偏好用 REA 场景代替大范围 vitest：
    - 命令：`rea-local-browser capture-browser-scenario`；`browser.mode=launch`，
      `executable_path=/usr/bin/google-chrome-stable`，前端 `http://127.0.0.1:8001`，
      `after_each_step:["screenshot"]`。不要连接 19322 端口上用户已开的浏览器页签。
    - 登录分两步：点 `.ant-segmented-item:has-text('账号登录')` → 填账号 → 点「下一步」→
      填密码 → 点 `button[type=submit]`。密码通过 `secrets` 环境变量传入，不写进场景文件。
      测试账号来自 `pnpm run sync:dev` 种子（如 `liming`）。
    - 场景：建一张带账期的应收账单 → 增加一笔费用 → 到期日与汇率不变；
      在编辑弹窗把到期方式改为指定日期 → 保存 → 再打开、不做修改直接保存，到期日不变。
14. 提交：`feat(web): 建账与编辑账单可确认并直接指定到期日`

## 阶段 3：规范（可并入阶段 2 的提交，或单独提交 `docs:`）

15. `.trellis/spec/server/backend/finance-bill-currency.md`：
    - §3 契约中「未开票且没有费用锁定事实时……把 `bill_date` 更新为本次北京时间当天，重取该日账单汇率……」
      和「无账期而显式到期日早于本次增减日……」两条，改写为：增减费用不改 `bill_date`、`due_date`、
      `payment_terms_days` 和汇率字段，所有有效行按原账单汇率重算本币。
    - §4 错误矩阵中「增减费用缺新创建日汇率……」这一行删除。
    - §6 测试要求中「增减费用更新业务日期、账单头/行汇率与金额和账期到期日」这一条按新口径改写。
    - §7 Wrong vs Correct 的第二组改成新口径。
16. `.trellis/spec/domain/glossary.md` 的「账单」行：把「未开票增减费用刷新」改为
    「建账时确定，后续增减费用不变」。
17. `.trellis/spec/web/frontend/component-guidelines.md` 第 12 行附近如果提到账单日期刷新，同步改写。

## 收尾

18. `pnpm run check:fast`。
19. 在父任务 `11-18-payment-terms-due-date` 的跨子任务验收里勾选与本子任务有关的项目。

## 回滚点

- 阶段 1 和阶段 2 各一次提交，可以分别 revert。没有迁移，也没有契约变更。
