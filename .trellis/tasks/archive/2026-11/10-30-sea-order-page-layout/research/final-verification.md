# 海运订单排版最终复核

日期：2026-11-15。用户要求将现存任务全部完成并合并。本次按现有批准范围收尾，不再扩展版式或业务功能。

## 实现与合并状态

- `d7427230`：新建与详情共用六列底栅格，船名、港口、客户等长字段跨列，WEEK 并入 ETD，备注收拢。
- `1eb846a5`：小节标题与维护操作对齐、详情提单模式标签统一、备注折叠预览。
- `2c888e87`：货物与商业六列布局、提单正文窄列标题换行及关联放货记录分组。
- 三个提交均已通过 `git merge-base --is-ancestor <提交> main`，无需重复合并。
- 对照 `sea-template.tsx`、`FormRow.tsx`、`SeaTransportSection.tsx`、`SeaFinalDestinationField.tsx` 与详情航程维护入口核查，未发现范围内产品阻断缺陷。

## 本轮补齐

- 新增 `web/tests/e2e/sea-order-layout.e2e.ts`，复用现有稳定性 API 夹具，加载真实新建及详情组件；全部业务请求在浏览器截获，不写开发业务数据库。
- 分别在 390、1024、1440、1920px 运行新建与详情共八个浏览器场景，检查船名、航次、港口独立可见及整页无水平溢出，保存整页截图；均通过，耗时约 1.6 分钟。
- 人工查看 390px 与 1440px 新建截图，确认小屏单列、桌面六列与提单区标题无互相遮盖。其余尺寸通过同一浏览器断言与截图留证；不将合成接口验收描述为真实后端业务验收。
- PRD 中早期待定布局选择更新为已定单页分节方案；验收项按本轮证据勾选。

## 验证结果

- 定向 Vitest：7 文件、43 用例通过（28s），覆盖布局卡片归属、备注、独立目的港/共享航程、船名航次和查询链接、维护权限与 ETD/ETA。
- 命令：`pnpm --dir web exec vitest run src/pages/orders/templates/sea-create-layout.test.tsx src/pages/orders/templates/sea-create-notes.test.tsx src/pages/orders/templates/components/sea/SeaFinalDestinationField.test.tsx src/pages/orders/templates/components/sea/SeaFinalDestinationField.shared.test.tsx src/pages/orders/templates/components/sea/SeaVesselVoyageField.test.tsx src/pages/orders/templates/components/sea/SeaMasterBillFields.maintenance.test.tsx src/pages/orders/templates/components/sea/SeaScheduleDateFields.test.tsx`。
- 浏览器：`pnpm --dir web exec playwright test tests/e2e/sea-order-layout.e2e.ts`，8 用例通过。
- `pnpm --dir web tsc` 退出 0。
- 新增浏览器文件 Biome 检查通过，`git diff --check` 通过。
- 主会话最终 `pnpm run check:fast` 退出 0：前端 2572 用例通过、11 项既有跳过，后端测试、vet、架构与漏洞扫描通过，总耗时 237.2 秒。

## 规范同步判定

未引入新业务契约；现有组件规范与海运共享航程规范已覆盖单页分节、全宽、目的港独立性、权限及输入保护，无需新增重复规范。
