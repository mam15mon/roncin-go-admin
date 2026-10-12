# 整票费用审核后端交付记录

2026-10-12；实施者 `fee_review_core`。本报告仅覆盖所分配的后端审核闭环及统一生成；补录/提成内部由其他实施者完成，前端由前端实施者完成，主线程负责种子、权限写锁、规格及最终门禁。未提交 Git，未重置或迁移用户开发业务库。

## 已实现

- 独立 `system.finance.fee.review` 权限；实时校验有效用户、成员、角色以及同一角色授权的组织范围。自审允许，亏损批准必须填写意见。授权读取固定 User → Membership → Role 共享锁，与授权变更父行排他锁一致。
- Order 上完整审核状态、费用内容版本、审核版本以及当前/批准事件指针；新增不可变审核事件，保存审核人/授权依据/意见/费用快照/指纹/幂等键。状态为 UNSUBMITTED、PENDING、APPROVED、REJECTED、REVOKED。
- 查看、预览、提交、撤回、批准、退回、撤销和批量预览/批准 RPC。每 RPC 独立 Req/Resp 满足 Buf lint，HTTP JSON 保持平铺。批量上限 200、重复拒绝、全部结构先校验，逐票独立事务并返回成功/失败。
- PENDING/APPROVED 冻结普通新增/编辑/删除/批量录费。正常费用修改在 Order 锁下推进内容版本；业务锁和提成派生财务锁仍维持各自限制。撤销批准必须没有任何有效账单（含草稿）及待审补录。
- 批准绑定完整经济内容快照 `ORDER_FEE_REVIEW_V1`。费用行版本、账单关联和结算单位展示名变化不使批准失效；费用 ID、补录来源、金额、币种、汇率及实际费用范围变化会改变指纹。
- 台账、订单状态筛选及当前页批量能力投影；费用台账与建账候选包含状态、待审补录、可建账以及具体阻塞原因。无成本查看资格不投影金额和利润。
- 建账读取、直接创建、批量创建统一校验有效批准及无待审补录。账单创建/更新/确认/取消/批量确认以完整排序 Order 集为先，再锁 Bill/Fee；费用归属漂移返回冲突。
- 海运拆票在费用实际转移时阻止已待审/批准订单；纯业务内容拆分不误伤费用批准。
- 通知与审核事件同事务落库，复用现有逐人钉钉机器人后台投递。提交发合格候选，批准/退回发提交人；幂等重放不重复投递。海运出口链接为 `/orders/sea-export/{id}/fees`；前端未注册的其余业务类型保留订单编码而不生成伪链接。
- 为补录实施者提供实时授权、有效批准校验、候选查询、版本递增、补录后新批准快照/撤销及结果通知 helpers；为种子提供 `LoadOrderFeeReviewSnapshot`，不复制费用映射。

## 主要源文件

权限：`server/internal/access/manifest.go`。

契约：`server/api/order/v1/order.proto`、`order_fee.proto`、`error_reason.proto`；财务台账 `server/api/finance/v1/settlement.proto`。

领域/仓储/服务：`server/internal/biz/order_fee_review.go`、`server/internal/data/order_fee_review.go`、`order_fee_review_notifications.go`、`server/internal/service/order_fee_review.go`，及订单/费用/账单/台账/拆票现有调用方。

Schema：`server/internal/data/ent/schema/order.go`、`order_fee_review_event.go`、`notification_delivery.go`。统一迁移同时包含补录/提成实施者 Schema 变化。

正式迁移：`server/migrations/20261012100000_order_fee_review.sql`。隔离 Ent/Atlas diff 按任务范围筛选，现有同名 CHECK 从 Ent 注解重新生成；已审阅没有无关类型/FK/索引删除。迁移前检查旧待审补录、草稿系统补录冲减及父提成未付但调整已付异常。临时生成命令源已移除。

## 统一生成（末轮已完成）

```sh
go -C server generate ./internal/data/ent
make -C server api
go -C server generate ./cmd/server
pnpm run generate:permission-keys
pnpm run generate:web-client
```

生成物包括 Ent、Protobuf/gRPC/HTTP、权限规则、OpenAPI、Wire、前端客户端及权限/枚举常量，均通过生成器更新。末轮 API 和前端客户端包含各单票独立 Req/Resp、补录最新注释以及提成调整 `can_execute/execution_blocked_reason`。

## 验证结果

- `buf lint`：通过。
- `go -C server test -p 32 ./internal/service ./internal/biz ./internal/data/ent/migrate`：通过。
- `go -C server vet ./internal/biz ./internal/data ./internal/service`：通过。
- `go -C server build ./...`：生成之后通过；主线程最终全门禁进一步覆盖末轮 RPC 名称同步。
- `git diff --check`：通过。
- 私密任务隔离数据库仅通过 `/tmp/roncin-fee-approval-db.env` 注入，完整正式迁移应用通过。未打印连接串、凭据，也未触碰实际开发库。
- 最后定向真实 PostgreSQL：`go -C server test -p 32 ./internal/data ./internal/biz ./internal/data/ent/migrate -run 'TestOrderFeeReview|TestFeeReview|TestGeneratedMigrateTables_CheckConstraints' -count=1`（由私密 env 包装）通过；data 22.366 秒。覆盖自审、冻结、幂等、并发批准、亏损意见、混合批量、实时授权/跨组织、批准与建账撤销竞态、列表筛选总数、通知唯一性及路径。未外发真实钉钉消息。
- 最后补充 `go -C server test ./internal/biz -run TestFeeReview -count=1`：通过。明确验证 PreviewBatch/ApproveBatch 200 票逐票处理、201 票进入仓储前拒绝、末项参数错误在首票写入前拒绝；以及经济指纹/币种汇率边界。

## 后续由主线程统一验收

最终 `check:fast`、全部隔离数据库集成测试、跨层 checker 和 Git 提交由主线程执行。历史提成/补录测试夹具正在由补录实施者同步新契约，其错误不由核心代码加兼容分支掩盖。审核核心与生成没有已知未实现业务项。
