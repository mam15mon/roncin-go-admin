# 执行计划：数据层迁移至 React Query v5

> 设计依据：同目录 design.md（含模式映射表、v5 红线、测试策略）。
> 每阶段独立提交；迁移红线：行为等价、文案逐处保持、stderr 零噪音。

## 阶段 0：基建（主代理亲自做）

- [x] 1. `pnpm --dir web add @tanstack/react-query@^5`（5.103.1）。
- [x] 2. 新增 `web/src/utils/queryClient.ts`（单例 + 默认配置 + 全局 cache
      onError 经 appFeedback 弹错、meta.errorMessage 文案）。
- [x] 3. `web/src/app.tsx` childrenRender 包 `<QueryClientProvider>`。
- [x] 4. 新增 `web/tests/queryClientTestUtils.tsx`
      （createTestQueryClient / renderWithClient）。
- [x] 5. 冒烟：探针验证 Provider 注入 + useQuery 真实可用 + 失败不重试
      （通过后已删探针）。
- [x] 6. 提交 `feat(web): 接入 React Query v5 基建`（b4f3147d）。

> **⏸ 暂停点（2026-09-20 03:35）**：检测到并行会话正在推进
> `09-20-headquarters-business-boundary`（总部业务边界，15 分钟前仍活跃），
> 其改动覆盖阶段 1 的 `MyApplicationHistoryDrawer` 与 commissions 域，并已
> 重生成 OpenAPI 服务层。用户决策：暂停阶段 1/2，待该任务落地后继续。
> 分支：`refactor/data-layer-react-query`（阶段 0 已提交，未含对方改动）。
> 恢复时注意：先 `git status` 核对并行改动是否已提交/清理，再重新核对
> 阶段 1 文件清单是否仍与设计一致（对方可能已改写其中部分组件）。

## 阶段 1：高痛点抽屉与弹窗（子代理并行）

- [ ] 7. 批次 1A：`MyReceivablesDrawer`、`MyCommissionDrawer`、
      `MyApplicationHistoryDrawer`、`MyApplicationCandidatesDrawer`
      （4 个 Drawer 同模式：open+query 驱动、sequenceRef 全部删除）。
- [ ] 8. 批次 1B：`CommissionCreateModal`（4 个候选 effect + 2 处手写
      setTimeout 防抖 → enabled/防抖关键词进 queryKey）。
- [ ] 9. 验证：定向 vitest 全绿零警告；`tsc` + biome（改动文件）。
- [ ] 10. 提交 `refactor(web): workbench 抽屉与佣金创建弹窗迁移 React Query`。

## 阶段 2：订单详情聚合

- [ ] 11. `orders/use-order-detail-data`：8 个 useState + requestIdRef/activeRef
      五重竞态收敛为声明式 query（含订单不存在/无权限分支文案等价）；
      对应测试改造。
- [ ] 12. 验证同上；提交 `refactor(web): 订单详情数据聚合迁移 React Query`。

## 阶段 3：规范沉淀与收尾

- [ ] 13. spec 写入：`state-management.md`（React Query 唯一模式、key 规范、
      meta 错误文案约定）+ `quality-guidelines.md`（禁新增手写竞态模板）。
- [ ] 14. 全量 `pnpm run check:fast`；`git diff --check`。
- [ ] 15. 剩余批次另立后续任务清单（审计表：use-order-create-options、
      use-order-fee-options、use-order-lock-state、VerificationWorkbench、
      BillCreationWorkbench、partner-detail、fees、split、cashflows、invoices）。
- [ ] 16. 归档或保留任务（后续批次多则保留任务继续推进，由用户定）。

## 验证命令速查

```bash
pnpm --dir web add @tanstack/react-query@^5
pnpm --dir web exec vitest run <测试文件相对 web/ 的路径>
pnpm --dir web tsc && pnpm --dir web exec biome check <改动文件>
pnpm run check:fast
```
