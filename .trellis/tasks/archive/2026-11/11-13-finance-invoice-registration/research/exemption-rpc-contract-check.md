# 免票 RPC 契约机械补修

日期：2026-11-14。

完整后端契约检查发现标记/撤销 RPC 共用 `SetBillInvoiceExemptionRequest` 与账单更新 `UpdateBillResponse`，违反仓库 Buf 对每个 RPC 独立请求/响应类型的规则。

已分别定义：

- `SetBillInvoiceExemptionRequest` / `SetBillInvoiceExemptionResponse`。
- `RevokeBillInvoiceExemptionRequest` / `RevokeBillInvoiceExemptionResponse`。

两个响应仍为统一信封（success/code/message/data/trace_id），data 为 FinanceBill；请求仍为 id/expected_version/reason。仅类型名调整，不改变 JSON 形状、路由、权限或业务语义。传输层各自包装响应，共享私有 helper 返回领域账单。相关服务测试使用独立撤销请求并检查成功信封与账单 ID。

生成按顺序执行 `make -C server api`、`pnpm run generate:web-client`，未手改生成物、未关闭 Buf 规则。

验证：`pnpm run lint:proto`、服务层 `TestBillInvoiceExemption*` 定向测试、`pnpm --dir web tsc` 与 `git diff --check` 全部通过。生成器已交还主会话，未提交 Git。
