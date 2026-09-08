# 执行计划：移除订单级 loadingTerms

## 顺序清单

1. **契约层**
   - [ ] `server/api/order/v1/order.proto`：删三处 `loading_terms`，加 reserved
   - [ ] `make -C server api` 重新生成绑定
   - [ ] 检查生成差异仅限 loading_terms 相关

2. **服务端手工代码**
   - [ ] `internal/biz/order_types.go` 删字段
   - [ ] `internal/biz/order_usecase.go` 删 trim 与长度校验片段
   - [ ] `internal/service/order_convert.go` 删读转换
   - [ ] `internal/service/order_write.go` 删 create/update 转换（含 nil 分支）
   - [ ] `internal/data/order_convert.go` 删持久化转换
   - [ ] `internal/data/order_write.go` 删两处 SetLoadingTerms
   - [ ] 测试中 LoadingTerms 引用同步删除

3. **Ent 与迁移**
   - [ ] `internal/data/ent/schema/order.go` 删字段定义
   - [ ] `go -C server generate` 重新生成 Ent
   - [ ] 新增 `server/migrations/20260908HHMMSS_drop_order_loading_terms.sql`
   - [ ] `pnpm run migrate:server` 本地执行并验证列已删除
     （`psql -c "\d orders"` 或 information_schema 查询）

4. **前端**
   - [ ] `SeaBasicInfoSection.tsx` 删字段块与 import、行注释
   - [ ] `common.ts` 删 `loadingTermsOptions`
   - [ ] `order-create-payload.ts` 删类型与映射
   - [ ] `orderDetailHelpers.ts` 删映射
   - [ ] `pnpm run generate:web-client` 重新生成 OpenAPI 输入与客户端
   - [ ] 前端测试中 loadingTerms 引用同步删除

5. **验证**
   - [ ] `grep -rn "loadingTerms\|loading_terms"` 全仓仅剩 proto reserved 与迁移 SQL
   - [ ] `go -C server build ./... && go -C server vet ./... && go -C server test ./...`
   - [ ] `pnpm --dir web tsc`；相关定向 vitest；涉及文件 Biome
   - [ ] 浏览器抽验新建页（业务信息无运输条款、提单信息运输条款仍在）

6. **收尾**
   - [ ] 单一提交：源文件 + 全部生成物 + 迁移（Conventional Commits）
   - [ ] 勾选 PRD 验收项、task.py finish + archive

## 验证命令

```bash
make -C server api
go -C server generate
pnpm run generate:web-client
pnpm run migrate:server
go -C server test ./...
go -C server vet ./...
pnpm --dir web tsc
pnpm --dir web exec vitest run <相关测试文件>
pnpm --dir web exec biome check <涉及文件>
```

## 回滚点

- 每个阶段独立可回退（git 工作区），迁移执行前为最终回滚点（执行后仅能 revert 代码，
  列数据不可恢复——已确认接受）。
