# 移除订单级运输条款字段并收敛到提单正文

## Goal

海运出口订单「业务信息」中的运输条款（`orders.loading_terms` / `loadingTerms`）与提单正文
`transportTerms` 语义重复，且订单级字段没有任何业务流程消费（后端仅 DTO 转换、trim、
长度校验、落库）。完整删除订单级字段，MBL/HBL 提单正文的 `transportTerms` 成为唯一
真相源。

## 背景与决策

- `loadingTerms` 下拉值为 CY-CY / CY-CFS / CFS-CY 等 7 个固定组合，属于装卸交接条款，
  与提单正文「运输条款」（例如 CY-CY / FCL-FCL）语义重叠。
- 提单正文具备完整单证领域模型：MBL/HBL 各自持有、内容持久化、版本历史、单证校验；
  订单级字段只是普通订单列，两处并存会产生重复录入、数据不一致、版本边界错位。
- 海运订单创建时强制提供主单信息，提单正文区块在新建页即存在，不存在「提单尚未建
  立」的字段空窗。
- 用户已确认采用**完整删除**方案：契约、后端各层、数据库列、前端全部移除，
  `orders.loading_terms` 列数据直接丢弃（当前仅本地开发库有数据）。
- 用户已确认创建 Trellis 任务走完整流程。

## Requirements

- proto：`Order.loading_terms`（field 50）、`CreateOrderRequest.loading_terms`（field 44）、
  `UpdateOrderRequest.loading_terms`（field 45）删除，字段号 `reserved` 防复用。
- 服务端：`biz/order_types.go` 字段、`biz/order_usecase.go` normalizeOrder 的 trim 与长度
  校验、`service/order_convert.go`、`service/order_write.go`（含 update 的 nil 分支）、
  `data/order_convert.go`、`data/order_write.go`（Create 与 UpdateDraft 的 SetLoadingTerms）
  全部移除。
- 数据库：Ent schema `order.go` 的 `loading_terms` 字段移除，重新生成 Ent 代码；新增版本
  化迁移 SQL `DROP COLUMN`（遵循 migrations/README：不用 IF EXISTS，中文头注释）。
- 前端：`SeaBasicInfoSection.tsx` 运输条款字段移除；`common.ts` 的 `loadingTermsOptions`
  移除（先确认无其他引用）；`order-create-payload.ts`、`orderDetailHelpers.ts` 的字段映
  射与类型移除。
- 生成物：`make -C server api`、`go -C server generate`、`pnpm run generate:web-client`
  重新生成并与源文件同组提交。
- 前后端相关测试同步更新，不留对已删字段的引用。

## 非目标

- 不修改提单正文 `transportTerms` 的任何行为（HOUSE/DIRECT 模式逻辑不变）。
- 不做订单字段与提单正文的自动同步。
- 不为被删列做数据备份、导出或兼容读取（已确认丢弃）。
- 不调整运输条款相关的其他字段布局。

## Acceptance Criteria

- [x] 仓库内 `loadingTerms` / `loading_terms` 无残留引用（proto reserved 声明、迁移 SQL 的 DROP COLUMN、pb 描述符对 reserved 名的编码字节除外）。
- [x] `go -C server build ./...`、`go -C server vet ./...`、`go -C server test ./...` 通过（全量）。
- [x] `pnpm run migrate:server` 在本地库执行成功，`information_schema` 查询确认 `orders.loading_terms` 列已删除。
- [x] 前端相关定向测试通过（4 个文件 22 例），`pnpm --dir web tsc` 无错误，涉及文件 Biome 通过。
- [x] 浏览器抽验新建页：业务信息不再有 CY/CFS/DOOR 运输条款下拉（placeholder 计数为 0）；提单信息 MBL 正文「运输条款」字段仍可见。
