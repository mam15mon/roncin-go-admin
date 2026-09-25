# 执行计划：SearchFilterTemplate 已选条件 chips 回显行

> 前置：`task.py start` 后才进入实现。全程不碰 `web/src/services/roncin/` 生成物，无契约/权限变更。

## 步骤清单

### 1. 模板实现（`web/src/components/ui/search-filter-template/`）

- [x] `SearchFilterTemplate.tsx`：
  - [x] 新增 `committed` state（`TValues`）与 `labelMapRef`（`useRef`）；
  - [x] `handleFinish` 固化 committed；`handleReset` 清空 committed 与 labelMap；
  - [x] `renderFieldInput` 对 select/searchable-select 合并 onChange 捕获 label；
  - [x] 新增 `resolveChipText` 纯函数（类型分派 + 空值过滤 + formatDate）；
  - [x] 新增 chips 行渲染（grid：字段 Row 上方；bar：表单行下方；custom：跳过）；
  - [x] `deleteChip(name)`：setCommitted → form.setFieldValue(name, undefined) → onSearch(剔除后对象)；
  - [x] 「清除全部」：form.resetFields() → setCommitted({}) → onReset()。
- [x] `types.ts`：如需导出 chip 相关类型则最小化新增；不改既有类型形状。

### 2. 组件测试（`SearchFilterTemplate.test.tsx`）

- [x] 按 design.md §6 新增 6 组用例（grid 回显/空值/删除/日期与清空、bar、远程 label）。

### 3. 消费方核查（只读确认，不改代码）

- [x] 抽查 `commissions/index.tsx`、`partners/index.tsx`、`admin/users.tsx` 的 onSearch 回调为整体替换语义；
- [x] 确认无页面依赖「onSearch 收到的对象必须等于表单原始值」的隐含行为。

### 4. 定向验证

```bash
pnpm --dir web exec vitest run src/components/ui/search-filter-template/SearchFilterTemplate.test.tsx
pnpm --dir web biome:lint -- src/components/ui/search-filter-template
pnpm --dir web tsc
pnpm --dir web test:changed
pnpm run check:architecture
```

- [x] 全部通过；`tsc` 因共享组件变更必跑。

### 5. 提交

- [x] `git diff --check`；
- [x] Conventional Commit：`feat(web): SearchFilterTemplate 增加已选条件 chips 回显与删除`；
- [x] 不包含无关文件（`web/src/features/finance/bill-creation/BillCandidateSelectionStep.tsx` 有他人未提交改动，勿纳入）。

## 回滚点

- 单一组件 + 测试文件改动，`git revert` 单提交即可整体回滚；无数据/契约影响。

## 评审闸口

- 实现前：本计划与 prd/design 经用户确认后 `task.py start`。
- 实现后：跑完 §4 定向验证再报完成；验收对照 prd.md Acceptance Criteria 逐条勾选。
