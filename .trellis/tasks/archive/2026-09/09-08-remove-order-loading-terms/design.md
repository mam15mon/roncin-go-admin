# 技术设计：移除订单级 loadingTerms

## 变更边界

跨层契约删除，单一提交包含源文件与全部生成物。不引入行为变更：删除字段后订单创建、
草稿更新的其余逻辑逐字不动。

## 各层改动

### 1. 契约层（server/api/order/v1/order.proto）

- `Order` 消息：删除 `optional string loading_terms = 50;`，消息内追加
  `reserved 50; reserved "loading_terms";`
- `CreateOrderRequest`：删除 field 44，同消息内 `reserved 44; reserved "loading_terms";`
- `UpdateOrderRequest`：删除 field 45，同消息内 `reserved 45; reserved "loading_terms";`
- 重新生成：`make -C server api`（pb/http 绑定）→ `pnpm run generate:web-client`
  （OpenAPI 与前端客户端、proto 常量）。

### 2. biz 层

- `order_types.go`：删 `LoadingTerms string` 字段。
- `order_usecase.go` normalizeOrder：删 `output.LoadingTerms = strings.TrimSpace(...)` 一行；
  长度校验长条件中删去 `utf8.RuneCountInString(output.LoadingTerms) > 100 ||` 片段。

### 3. service 层

- `order_convert.go`：读转换删 `LoadingTerms: stringPtrIfNotEmpty(item.LoadingTerms)`。
- `order_write.go`：create 转换删 `LoadingTerms: request.GetLoadingTerms()`；update 转换删
  `if request.LoadingTerms != nil { output.LoadingTerms = request.GetLoadingTerms() }` 分支。

### 4. data 层

- `order_convert.go`：持久化转换删 `LoadingTerms: item.LoadingTerms`。
- `order_write.go`：`Create`（SetLoadingTerms）与 `UpdateDraft`（SetLoadingTerms）各删一处。
- Ent schema `ent/schema/order.go`：删 `field.String("loading_terms").Optional().MaxLen(100)`，
  执行 `go -C server generate` 重新生成 Ent（runtime/mutation/order 等）。

### 5. 数据库迁移

新增 `server/migrations/20260908HHMMSS_drop_order_loading_terms.sql`：

```sql
-- 移除订单级运输条款：该字段与提单正文 transportTerms 重复且无业务消费方，列数据按决策直接丢弃。
ALTER TABLE "orders" DROP COLUMN "loading_terms";
```

- 遵循目录规范：不使用 IF EXISTS（若列不存在则报错，符合本仓库迁移哲学）；
  本地执行 `pnpm run migrate:server` 验证。
- 迁移在 Ent 代码生成之后编写，保证 `ent/migrate/schema.go` 与迁移链冷启动结构一致。

### 6. 前端

- `SeaBasicInfoSection.tsx`：删除「运输条款」`ProFormSearchableSelect` 字段块（第 7 行
  区域），同步更新行注释；文件顶部 import 中移除 `loadingTermsOptions`。
- `common.ts`：删除 `loadingTermsOptions` 常量（已确认仅 SeaBasicInfoSection 引用）。
- `order-create-payload.ts`：类型 `CreateOrderFormValues.loadingTerms` 与
  `buildCreateOrderPayload` 的 `loadingTerms` 映射删除。
- `orderDetailHelpers.ts`：详情表单值读取与 `buildUpdatePayload` 的映射删除。
- 生成物 `web/src/services/roncin/`、`web/types/` 由 generate:web-client 再生，不手改。

## 风险与取舍

- **数据丢失**：DROP COLUMN 不可逆，用户已明确确认（当前仅本地开发库）。
- **契约兼容**：proto reserved 防止字段号复用；前后端同仓同发，无灰度窗口问题。
- **测试**：后端 order 相关测试若引用 LoadingTerms 需同步删除断言；前端
  order-create-payload.test / orderDetailHelpers.test 同理。
- 不做旧值迁移到提单正文：两字段语义并非严格一一对应（订单侧是枚举下拉，提单侧是
  自由文本），静默搬运违反「不做静默纠错」约束。

## 回滚

单一提交，`git revert` 即可回滚代码；数据库列回滚需从迁移链外手工恢复（已确认无
需保留数据，故不提供 down 迁移）。
