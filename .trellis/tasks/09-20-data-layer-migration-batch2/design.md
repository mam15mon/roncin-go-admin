# 技术设计：数据层迁移第二批

## 设计基线

完全复用第一周期的设计：`.trellis/tasks/archive/2026-09/09-20-data-layer-direction/design.md`
的模式映射表、v5 红线、错误提示约定（silent / meta.errorMessage / 动态 Error
透出）、测试策略（renderWithClient + createTestQueryClient）。基建已就绪
（`src/utils/queryClient.ts` + Provider + `tests/queryClientTestUtils.tsx`），
本批无新基建。

## 各批次特有要点

### 批次 A：orders 数据链

- `use-order-lock-state`：被 `orders/detail.tsx` 真实渲染，其测试
  （detail-draft-lifecycle / detail-change-actions / detail-shared-container-workbench /
  orders-breadcrumbs，部分 `importOriginal` 半 mock）迁移后必须补
  QueryClientProvider；先 grep 消费方再动手。
- `use-order-create-options`：消费方 `orders/new.tsx`（new.test.tsx 需要
  Provider）。
- `use-order-fee-options`：消费方 `orders/fees.tsx`（fees.test.tsx 需要
  Provider）。
- `orders/fees.tsx` 内 `setTimeout(0)` 表单填充 hack 属于交互 hack 而非数据
  拉取，保持不动。

### 批次 B：finance/workbench 页

- `VerificationWorkbench`：三段链式 effect（组织/币种 → 结算方 → 候选）用
  `enabled: !!parentValue` 表达。
- `commissions/index.tsx`：挂载期组织候选 effects；导出/筛选规范化逻辑不动。
- `finance/fees/index.tsx`：组织候选 + 用户偏好两个 effect；原偏好加载无
  cancelled 旗标（卸载后 setState 隐患），迁移后由库天然修复，行为等价。
- `useWorkbenchOverview`：sequenceRef + reloadToken 手写复制品，整体替换；
  返回字段签名兼容（Welcome 等消费方零改动）。

### 批次 C：BillCreationWorkbench

- token/fingerprint 双令牌 + 350ms 手写防抖 → 防抖关键词进 queryKey +
  keepPreviousData；预览竞态由 queryKey 隔离保证。
- 测试是第一周期 act 治理的重点文件（sleepInAct / 手动 resolve 包 act 等），
  迁移后逐用例核对：等待语义仍成立、断言意图不变、用例数不减；scale 测试
  同步。允许删除仅服务于手写竞态的等待代码，但不得删除行为断言。

### 批次 D：partner-detail + split

- `partner-detail.tsx`：`loadPartnerData` 大聚合 effect（依赖均为 primitive，
  无重跑 bug）→ 按 5 路聚合语义选择单条聚合 query 或并行 query + 门控；
  总部业务边界提交（23459708）刚改过该文件（+53 行），以现状为准。
- `split.tsx`：splitContext 拉取 + 300ms 防抖预览 → query + 防抖关键词进
  queryKey；`orders/split.test.tsx` 有迟到语义断言，等价保留。

## 风险与回滚

- 每批次独立提交可单独 revert。
- 批次间文件互不相交，可并行；消费方测试的 Provider 补齐归属各自批次。
- 收口阶段跑全量 vitest 兜底扫 Provider 遗漏（第一周期的教训）。

## 兼容性

- 不改 OpenAPI 生成物与服务端；不触碰 `master-data-template`（用户自改保留）。
