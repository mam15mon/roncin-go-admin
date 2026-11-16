# 实施计划

1. 新增 `templates/components/air/useAirMasterCandidate.ts`：防抖、精确核对查询、
   状态派生、差异计算、自动带入/关联 Effect、`adopt`/`switchMode`/`validate`。
2. 改 `AirMasterFields.tsx`：
   - 删除「主单关联方式」「选择共享主单」及 `AirMasterCandidateHint`；
   - 主运单号字段挂 `validate`；`ready` 时主单字段只读；
   - 新增就地核对卡片（未命中 / 已找到 / 差异 / 模式不符 / 不可关联 / 失败）；
   - URL 预选只写入主单资料与单证模式。
3. 回归 `AirDocumentSection.tsx`、`AirOrderWaybillFields.tsx` 对 `airMasterMode` 的读取。
4. 更新测试：`air-import-master-selection`、`air-preselected-master`、
   `air-create-layout`、`air-document-mode` 等；新增 Hook 场景测试覆盖 R3–R10
   （真实 wire 形态：`canAttach` 省略即 false、`memberCount` 省略即 0）。
5. 同步规范 `.trellis/spec/web/frontend/air-order-workspace.md`。

## 验证

```bash
pnpm --dir web exec vitest run src/pages/orders/templates src/pages/orders/order-kinds/air-export src/pages/orders/order-kinds/air-import
pnpm --dir web exec vitest run src/pages/orders
pnpm --dir web exec biome check <修改文件>
pnpm --dir web tsc
git diff --check
```

浏览器：AE 新建覆盖新建、加入、差异采用、不可关联；AI 新建抽查加入。

## 回滚点

Hook + 呈现一个提交，测试与规范随同；按提交 revert。
