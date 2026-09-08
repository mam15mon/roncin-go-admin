# 移除海运订单列表页失效的编辑弹窗入口

## Goal

海运出口订单列表的行内编辑按钮仍打开旧版 EditOrderModal，其提交 payload 缺少 seaMasterBill/seaDocument，会被后端 normalizeOrder 以『海运出口订单必须提供主单信息』拒绝；收敛该入口，使列表编辑与详情页主编辑路径一致。

## 背景与问题

- 海运出口订单列表（`web/src/pages/orders/list.tsx`）每一行都渲染「编辑」按钮（不分订单状态），点击后打开 `EditOrderModal`。
- `EditOrderModal` 提交的 UpdateOrder payload 不携带 `seaMasterBill` / `seaDocument`，而后端 `internal/biz/order_usecase.go` 的 `normalizeOrder` 对海运出口订单强制要求主单信息，因此该弹窗对海运订单保存必然失败。
- `ORDER_KIND_CONFIGS`（`web/src/pages/orders/common.ts`）当前只注册 `sea-export` 一种业务类型，空运没有可达路由，不存在依赖该弹窗的其他在线业务。
- 详情页已是完整的编辑入口（复用与新建页相同的分节模板），列表不应再提供第二条不完整的编辑路径。

## Requirements

- 海运出口订单列表行操作不再提供「编辑」入口；订单编辑统一通过详情页完成。
- 移除 `list.tsx` 中对 `EditOrderModal` 的引用与挂载；该组件此后无任何使用方，随之删除 `web/src/pages/orders/components/modals/EditOrderModal.tsx`。
- `OrderListTemplate`（`web/src/components/ui/order-list-template/`）是通用模板，保留其 `onEditOrder` 能力本身不做修改；通过不再传入该 prop 使按钮不渲染（模板已有 `!readonly && onEditOrder` 守卫）。
- 不改变列表其他操作（详情、费用核算、里程碑、单证、标签等）的行为。

## 非目标

- 不修改后端契约与校验逻辑（P3 的 UpdateOrderRequest 收紧另行立项）。
- 不处理表单扁平字段收敛、共享主单确认被清空提示等评审遗留项（P2）。
- 不调整 `OrderListTemplate` 的通用能力或为未来业务类型预置编辑入口。

## Acceptance Criteria

- [x] 海运出口订单列表行操作中不再出现「编辑」按钮，其余操作不受影响。
- [x] 仓库中不再存在 `EditOrderModal` 文件及其导入引用；`grep -r "EditOrderModal" web/src` 无结果（仅剩 git 忽略的 `.umi-production` 构建缓存，再构建即再生）。
- [x] `pnpm --dir web exec vitest run src/pages/orders/orders.test.ts src/pages/orders/list-query.test.ts src/pages/orders/list-constants.test.ts src/pages/orders/list-resources.test.ts src/pages/orders/list-documents-action.test.ts` 通过（24 个用例）；`list.tsx` 通过 Biome 检查（含应用其 import 排序安全修复，存量问题一并整理）。
- [x] 因删除组件导致的类型引用清理完成，`pnpm --dir web tsc` 无新增错误。
