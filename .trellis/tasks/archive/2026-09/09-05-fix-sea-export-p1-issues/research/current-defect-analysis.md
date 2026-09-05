# 海运出口四项 P1 当前缺陷分析

## 勘察范围

本文件记录 2026-09-05 当前分支的只读代码勘察结果，用于实施与独立检查。产品边界以本任务 `prd.md` 为准。

## P1-1：SE 列表仍进入旧主分单据抽屉

- `web/src/pages/orders/list.tsx:174-177` 对所有业务类型把 `onOpenDocuments` 绑定到 `ShippingDocumentDrawer.open`。
- `web/src/pages/orders/list.tsx:301-305` 无条件挂载该旧抽屉。
- `web/src/components/ui/order-list-template/OrderListTemplate.tsx:489-493` 固定显示“主分单据管理”并调用上述回调。
- 真实 SE 单证入口已经存在于 `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx`，区块 key 为 `sea-document`，详情页通过 `GetSeaOrderDocuments` 加载当前 MBL/HBL。

结论：不需要新建海运单证页面；SE 列表应跳转现有订单详情，非 SE 保留旧抽屉。

## P1-2：ReleasePod 只关联旧 OrderShippingDocument

- `server/api/order/v1/order_release_pod.proto:65-102` 的响应、新增和更新请求只有 `shipping_document_id`。
- `server/internal/biz/order_release_pod.go:60-72` 的领域对象只有 `ShippingDocumentID`。
- `server/internal/data/ent/schema/order_release_pod.go:17-41` 只有旧分单外键和索引。
- `server/internal/data/order_release_pod.go:23-64` 的查询直接使用 `r.data.db`；`validateShippingDocument` 只验证旧 `OrderShippingDocument`。
- `web/src/pages/orders/release-pod-panel.tsx:75-188` 只查询旧分单并用 `shippingDocumentId` 回显。
- 真实 MBL/HBL 聚合已由 `GET /api/v1/orders/{order_id}/sea-documents` 返回：`SeaOrderDocuments.master_bill` 与 `house_bills`。
- 当前 HBL 删除命令位于 `server/internal/data/sea_document.go:749-892`，已经按 Order → MBL → Link → HBL 加锁，并在同一事务更新 Link、删除 HBL和写操作日志，适合在 HBL 锁后加入 ReleasePod 状态检查。

结论：需要新增两个数据库真实外键，以 API 的显式 SeaDocumentType + ID 表达；HBL 关联删除必须加入现有 HBL 删除事务，不能由前端串行调用两个删除接口。

## 用户确认的关联删除规则

- 海运单证页面展示每张 MBL/HBL 的关联放货记录。
- 删除 HBL 时，若全部关联记录为待签收或已签收，页面列出记录并询问是否一起删除。
- 用户取消后零变更；确认后 HBL、关联记录和操作日志原子处理。
- 任一关联记录为已回单时，HBL 和所有关联记录均禁止删除。
- 产品只称“操作日志”，不新增独立审计功能。

## P1-3：只有删除请求遗漏 expectedVersion

- `server/api/order/v1/order_container.proto:92-97` 与 `order_cargo_item.proto:89-94` 已要求 `expected_version`。
- `web/src/pages/orders/components/drawers/ContainerDrawer.tsx:145-163` 的更新请求已发送记录版本，但 `:166-170` 的删除请求只发送 `orderId/id`。
- `web/src/pages/orders/components/drawers/CargoItemDrawer.tsx:132-148` 的更新请求已发送记录版本，但 `:151-155` 的删除请求只发送 `orderId/id`。

结论：只修两个 `removeItem` 调用及其版本缺失守卫，不改 Proto 或后端删除契约。

## P1-4：Execute 前的无锁 Preview 抢先返回 400

- `server/internal/biz/sea_order_change.go:631-756` 中，Execute 在事务前调用 `PreviewSplit`，Preview 失败会直接返回。
- `server/internal/data/sea_order_change.go:527-539` 的 Preview 通过 `GetChangeActions` 根据当前可变状态生成阻断；竞争胜方完成后，失败方可能看到 HBL 数量不足。
- `server/internal/data/sea_order_change.go:946-968` 的真正执行事务先 `FOR UPDATE` 锁 Order，并在业务状态门禁前比较 `ExpectedVersions.OrderVersion`，本可稳定返回 409。
- 后续事务代码继续锁定并比较 Link、Allocation、HBL、货物、箱、费用、候选 MBL/TE，并在锁后重算分配和守恒。
- `server/internal/data/sea_order_change_integration_test.go:1864-1929` 已明确要求两个不同幂等键的并发拆票只能一成功一 `SEA_ORDER_SPLIT_VERSION_CONFLICT`。

结论：独立 Preview API 保留；Execute 移除事务前依赖可变状态的 Preview，把现有锁内重验作为提交权威。实施时必须通过既有非法输入、守恒和业务门禁测试确认错误没有被泛化成 409。

## 工作区隔离

勘察时存在多项其他窗口前端改动，其中 `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` 与海运页面相邻但不属于本任务计划文件。实施前需再次核对状态；不得覆盖、重置或纳入提交。
