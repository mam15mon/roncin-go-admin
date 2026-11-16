# 订单与单证组审阅记录

## 审阅范围与工作区

仅在 `/home/vince/.codex/worktrees/server-state-migration/roncin-go-admin` 审阅订单组清单 `/tmp/order-documents-owned.json` 与相邻回归；未写源 main 工作区、未提交、未回退其他代理差异。原实现定向回归为 29 文件 104 例通过；任务最终完整门禁由主会话负责，本记录不将中间全量失败报告成通过。

## 已确认并修复

1. `SeaOrderChangeHistoryDrawer.tsx`：列表/详情仅切换 `enabled`，关闭抽屉不会移除 Query 观察者，在途 signal 不取消，重开可能复用旧请求。列表与详情键补 `open`；分页占位只在同订单且前后均打开时保留。新增 `SeaOrderChangeHistoryDrawer.query.test.tsx` 验证关闭取消、重开迟到列表/详情隔离及首次失败不冒充空记录或计数零。
2. `SeaDocumentHistoryActions.tsx`：无限历史相同关闭问题；版本/事件键补打开状态，重开重置明确指向打开键，保留失效前缀及第一页重置语义。新增回归断言两个请求 signal 均取消且重开从第一页读取。
3. `SeaDocumentHistoryActions.tsx`：预览仅绑定表单 revision，而父级 `SeaDocumentSection` 复用组件且未按主/分单 ID 和版本设置 React key。版本或关联单证变化时可能沿用旧预览并用新基线执行。预览结果补本地订单/单证/版本身份，在渲染和迟到错误判断中比对；载荷快照、幂等键及原业务确认保留。新增版本改变后执行禁用、重取预览采用新版本，以及换单证后迟到预览不可执行回归。
4. `UnlockRequestHistoryDrawer.tsx`：相同关闭在途问题，键补 `open`，新增关闭取消/重开迟到隔离回归；原轮询前台/隐藏/恢复/关闭断言保留。
5. `detail-change-actions.test.tsx`：第一轮全量门禁发现 4 例失败。根因是新增 signal/options 后旧精确参数断言不匹配，首例提前退出后未消费的 `mockImplementationOnce` 又因 `clearAllMocks` 不清队列污染后续。只同步请求选项断言和逐例 mockReset，保留逆序、失败不恢复、手工刷新及旧同步隔离所有业务断言。原 12 例已通过。
6. 两个海运历史组件已有就地 Alert 与请求 `skipErrorHandler`，Query meta 却仍为 errorMessage，造成全局 QueryCache 再通知。三处变更历史查询、两处单证历史查询改为 `meta: { silent: true }`，符合既有错误展示契约。

## 实际路径审查

- 同批订单、变更历史和无 documents prop 的单证首次失败明确展示错误，不返回合法空态。
- 单证查询首次成功才接受编辑快照；背景重查不覆盖正文或 link/master 版本，原回归继续通过。
- 修订/作废与模式切换预览保留输入载荷与编辑身份；航程预览保留输入变化即失效和迟到失败隔离。
- 列表分页只保留同订单活动抽屉占位；无限事件历史先按原始返回条数判断下一页，再过滤本单证，原加载更多/重置测试继续通过。
- 查询键仍使用现有 orders 前缀，未新增公共接口或独立结果缓存。

## 本轮精确修改文件

实现（3）：

- `web/src/pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.tsx`
- `web/src/pages/orders/components/detail/UnlockRequestHistoryDrawer.tsx`
- `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx`

测试（4）：

- `web/src/pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.query.test.tsx`（新增）
- `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.query.test.tsx`（新增）
- `web/src/pages/orders/components/detail/UnlockRequestHistoryDrawer.query.test.tsx`
- `web/src/pages/orders/detail-change-actions.test.tsx`

另更新本记录。现有原实现 29 文件清单不包含新增的两个 query 测试和相邻 `detail-change-actions.test.tsx`，主会话提交时须一并纳入。

## 最终验证

- 本轮上述 7 个源码/测试文件 Biome：通过。
- TypeScript：通过，日志 `/tmp/review-orders-tsc.log`。
- `git diff --check`：通过。
- 定向 Vitest：8 文件、37 例全部通过，总耗时 15.63 秒；没有单 suite 超过 20 秒，无需拆分。日志 `/tmp/review-orders-final-tests.log`。

实际定向文件：

- `src/pages/orders/components/detail/UnlockRequestHistoryDrawer.query.test.tsx`
- `src/pages/orders/components/detail/UnlockRequestHistoryDrawer.test.ts`
- `src/pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.query.test.tsx`
- `src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.query.test.tsx`
- `src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.test.tsx`
- `src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.test.tsx`
- `src/pages/orders/templates/components/sea/SeaDocumentSection.query.test.tsx`
- `src/pages/orders/detail-change-actions.test.tsx`

## 未修复事项与规范同步

本组没有遗留待决设计或接口问题。未扩展到费用、公共候选、财务模块；全量门禁和最终提交仍由主会话完成。

SWR focus 的 Node TimeoutNaNWarning 已由实施组证实源 main 基线同样存在（`/tmp/main-air-edit-baseline.log`），未静音或修改依赖；该基线噪音不属于本组迁移修复。

已建议主会话在其拥有的 `state-management.md` 补充：`enabled: false` 不移除 Query 观察者，需要关闭取消和重开隔离时，将 open 等实际活动身份纳入 key，并避免关闭态桥接 placeholder。主会话已接手规范同步，本代理不并写共享规范。
