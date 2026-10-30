# Research: 订单履约里程碑后端完整删除

- Query: 删除独立 OrderMilestone 功能所涉及的服务端源代码、生成物、Schema、权限、开发种子及相邻业务门禁。
- Scope: internal
- Date: 2026-10-30
- 状态: 规划研究；本次没有修改产品代码、应用迁移、操作数据库或执行 Git 变更。

## Findings

### 1. 当前能力与明确边界

- `server/api/order/v1/milestone.proto:12` 独立 `OrderMilestoneService` 仅提供 `ListMilestones` 与 `SetMilestone`。
- HTTP 为 `GET /api/v1/orders/{order_id}/milestones`、`PUT /api/v1/orders/{order_id}/milestones/{type}`（同文件 16、23 行）；权限 operation 为 `milestone.read/set`（15、21 行）。
- `server/internal/biz/order_milestone.go:12` 持有实体、仓储接口、用例；38/45 行读取与设置实现；60 行写审计 action `order.milestone.set`。
- `server/internal/service/order_milestone.go:12` DTO/UUID/时间转换与 usecase 调用；67 行 `parseOptionalRFC3339`、83 行 `timeStringPtr` 经全仓 service 搜索没有其他调用；biz 的 74 行 `derefString` 同样仅本功能使用，可随源文件整体删除。
- `server/internal/data/order_milestone.go:19/38` 仓储；列表按 order_id 查询，写入锁订单、校验统一编辑门禁与 expectedVersion，再按 order_id/type upsert。它不推进订单 `flow_status`，不承担主流程节点状态。
- 因此完整删除此三层不应改动订单状态机、订单流程事件、审核锁、POD、异常、费用等其他能力。

### 2. 必须删除或同步的手写源文件

| 文件/位置 | 处理 |
| --- | --- |
| `server/api/order/v1/milestone.proto` | 删除整份专属契约；不存在别的 proto import 它的命中。 |
| `server/internal/service/order_milestone.go` | 整文件删除，含仅专用 helpers。 |
| `server/internal/biz/order_milestone.go` | 整文件删除。 |
| `server/internal/data/order_milestone.go` | 整文件删除。 |
| `server/internal/data/ent/schema/order_milestone.go:11` | 删除 Ent Schema；只有 order_id 外键、type、模板节点投影、occurred_at、note、updated_by。 |
| `server/internal/data/ent/schema/order.go:125` | 删除 `edge.To("milestones", OrderMilestone.Type)`；组织和用户 Schema 不引用该实体。 |
| `server/internal/service/service.go:6` | 移除 `NewOrderMilestoneService` provider。 |
| `server/internal/biz/biz.go:6` | 移除 `NewOrderMilestoneUsecase` provider。 |
| `server/internal/data/data.go:35` | 移除 `NewOrderMilestoneRepo` provider。 |
| `server/internal/server/http.go:26/60` | 删除构造参数 `milestones *service.OrderMilestoneService` 与 HTTP 注册。 |
| `server/internal/server/grpc.go:25/57` | 删除同参数及 gRPC 注册。 |
| `server/internal/server/auth.go:202` | 删除 read-operation 分类中的 `OrderMilestoneRead`。 |
| `server/internal/access/manifest.go:155/156/309` | 删除 read/set 两个 operation 常量与 manifest definitions；414 行依赖说明删除“里程碑”举例。 |
| `server/cmd/sync-dev/main.go:49` | 删除 ent predicate import。 |
| `server/cmd/sync-dev/main.go:3024-3053` | 删除里程碑开发数据块（BOOKING_CONFIRMED 与 VESSEL_DEPARTED）及只被该块使用的 `ord01` 局部变量。 |
| `server/cmd/sync-dev/main.go:190/220/2949` | 同步输出/错误/章节注释，不再宣称注入里程碑。 |

### 3. 最重要的业务门禁引用：只移除退役事实，不删整个门禁

`server/internal/data/order_write_sea_master_bill.go:611` 的 `ensureSeaMasterBillHasNoDownstreamFacts` 被主单身份/航程更正路径调用（382、516 行）。当前条件：

1. `order.LockedAt != nil` 直接阻断（615-616）。
2. 非 DRAFT 的 `ShippingDocuments`（621）。
3. 任意 `ReleasePods`（622）。
4. 任意实际 `Containers`（623）。
5. **已填写 occurred_at 的 Milestones（624）**。
6. 有有效账单行的费用（625）。
7. 订单财务账单行（626）。
8. 订单提成行（627）。
9. 订单提成调整（628）。

本次只删第 5 项及 19 行 `ordermilestoneent` import，其余条件、统一可编辑门禁、共享成员门禁、锁序和错误 `ErrSeaMasterBillCorrectionBlocked` 保持。不要将 milestone 删除解释为放开全部已履约主单更正；也不要以 flowStatus 自动替代 milestone，现有注释明确订单流程状态本身不是下游事实（612-614）。

现有回归锚点：
- `server/internal/data/order_transaction_integration_test.go:477`：共享主单多成员阻断身份更正。
- 同文件 `:783-823`：已确认分单阻断主单身份和航程更正，失败后订单版本与航程无部分写入。
- `server/internal/data/sea_document_change_integration_test.go:521`：财务下游事实阻断改单执行。
- 同文件 `:572`：财务事实阻断单证结构切换。
- 同文件 `:647/:744`：箱货分配仍进入相关下游影响。
- 建议本次针对剩余门禁增加/补跑实际箱、POD、已建账费用等有价值场景，而不是为删除行写镜像断言。

### 4. 测试源必须同步

- `server/internal/access/manifest_test.go:211-212` 删除里程碑 read/set 依赖样例；保留货物/费用等资源依赖样例，增加退休 permission 不在 Manifest 的负向断言可帮助防回归。
- `server/internal/server/auth_test.go:404-405` 删除声明 operation 分类用例中的 milestone 两行。
- 同文件 `:651` 用 milestone.read 充当“无共享箱权限但有其他权限”的 principal，须换为仍存活且不授权共享箱的 operation（例如 `OrderCargoItemRead` 或 `OrderAttachmentRead`），不能删除整组拒绝用例。
- `server/internal/server/auth_middleware_shared_container_test.go:148` 同理替换用于模拟无权限的 milestone.read，保留不进入 handler 的断言。
- `server/internal/data/order_lifecycle_gate_integration_test.go:17/25/44-47/99/114-118` 删除 milestone predicate import、清理步骤、注释及代表性写入检查；草稿/POD/货物的终止和审核锁阻断/零部分写入检查必须完整保留。
- 没有找到专属 milestone usecase/service 单测源；所有其他手写 Go 引用已按关键字搜索覆盖。

### 5. 正式数据库迁移与权限数据

- `server/internal/data/ent/schema/order_milestone.go:28-30` 仅从本实体指向 orders 的外键；未发现其他 Schema 引用 OrderMilestone，也未发现应用手写 SQL 直接引用该表。
- 必须新增正式迁移删除 `order_milestones`（DROP TABLE，不需 CASCADE；它只拥有指向 orders 的 FK），不能仅删 Ent 代码却让正式迁移链最终仍残留表。
- `server/internal/data/migration_manifest_test.go:21` 会重放 SQL 中 CREATE/DROP 名称并与最新 Ent tables 比较；只删 Schema 不加 DROP 会失败。
- 不自动执行针对开发/生产库的删表；生成和验证迁移可用专用集成库隔离 Schema。真正应用会删除这套退役功能已有里程碑记录，需在方案中明确。
- **权限清理已有统一实现，无需新造兼容清理代码或手写权限 DELETE 迁移**：`server/internal/data/permission_manifest_sync.go:31-34` 说明在事务内同步清单，`:82` 删除清单外 Permission。`server/cmd/migrate/main.go:41-47` 的迁移 PostStep 自动调用它，普通角色和 administrator 关联由已有规则处理。
- 推荐迁移测试使用 `getIntegrationDataBeforeMigration(t, newVersion)` 构造前态并插入里程碑，再运行本次迁移、断言表不存在且订单/POD/附件/费用等未改；权限清理另跑 `TestSyncPermissionManifestPostgres` 或在同隔离测试调用 sync 后确认12条业务线 milestone 权限移除。

### 6. 历史迁移只保留，禁止为清空搜索命中而改写

这些是历史迁移事实，应保留并由新迁移收口：
- `server/migrations/20260820000000_baseline.sql:25-33` 历史模板建表，`:66-69` order_milestones 建表及索引。
- `server/migrations/20260824010000_expand_permission_manifest.sql` 历史模板与全局 milestone 权限。
- `server/migrations/20260824020000_split_order_permissions_by_business_type.sql` 与 `20260824020100_restore_order_permissions_by_business_type.sql` 的 milestone permission 历史生成。
- `server/migrations/20260829103000_remove_milestone_templates.sql` 已删除模板表，保持原样。
- `server/api/masterdata/v1/error_reason.proto:9` 的 retired milestone template error names 是 `reserved`，它不提供活功能；保留 protobuf 防复用声明，不能为了0关键字删掉。

### 7. 生成物与命令（只允许生成，不手工修补）

生成顺序建议先改全部源再执行：

```bash
# API、访问规则；删除已失去源文件的专属 milestone*.pb.go 产物，避免生成器不自动清旧文件。
make -C server api
# Ent（生成directive在internal/data/ent/generate.go:7），随后Wire。
go -C server generate ./internal/data/ent
go -C server generate ./cmd/server
# 或按仓库标准 go -C server generate，包含以上与server access rules。
pnpm run generate:web-client
pnpm run generate:permission-keys
```

- Protobuf生成配置 `server/buf.gen.yaml` 通过本地 pinned Go plugins 生成 Go/gRPC/HTTP 和 `server/openapi.yaml`。
- `server/internal/server/access_rules.go:8` 的 go:generate 从 proto 获取权限规则，`make api` 已调用 `go generate ./internal/server`。
- `server/cmd/server/wire_gen.go:3` 自带 Wire directive；删 ProviderSet、HTTP/GRPC参数后必须重新生成，而不是手改 Wire。
- `server/api/order/v1/milestone.pb.go`、`milestone_grpc.pb.go`、`milestone_http.pb.go` 是专属被删除源文件的生成物，应随契约一起删除，不留 Unimplemented 空壳。
- Ent生成会更新 client/tx/runtime/migrate/order相关边方法，并去掉 ordermilestone实体及predicate/query/create/update/delete文件。需核查生成器有无清除失去Schema的旧文件；若没有，只删除与已移除Schema完全对应的生成文件，不改其内容。
- 访问规则/主清单改变后前端 generated permissions、OpenAPI客户端/typings及索引必须同步。客户端是否自动清旧专属文件需检查生成脚本行为；主会话或前端研究负责。

### 8. 推荐验证顺序

1. `go -C server test -p 32 ./internal/access ./internal/server ./internal/service ./internal/biz`。
2. `go -C server test -p 32 ./internal/data -run 'TestMigrationManifestCoversEntSchema|TestOrderLifecycleGatePostgres|TestOrderCreateTransactionPostgres|TestSyncPermissionManifestPostgres|<本次迁移测试>'`。
3. 专用集成库环境必须显式 `RONCIN_INTEGRATION_DATABASE_SOURCE`，各Schema应用正式迁移，禁止Schema.Create/t.Parallel，并记录真实执行与skip区别。
4. `go -C server test -p 32 ./cmd/sync-dev` 确认开发seed编译与相邻功能未误删。
5. `git diff --check`，生成一致性复跑；最终 `pnpm run check:fast`（此次契约、权限、Ent和注册均变，完整门禁有充分依据）。

### 9. 需要同步的规范

- `.trellis/spec/server/backend/capability-navigation.md:40` 从辅助实体清单删除 order_milestone.go/里程碑。
- `.trellis/spec/server/backend/order-lock-and-document-version.md:130` 生命周期子资源示例不再列里程碑。
- 全仓domain/spec的其他命中由主会话统一清理；历史journal/task说明保留作为变更记录。

## External References

没有使用外部资料；依据当前仓库代码、固定生成配置和现有规范研究，不引入新依赖或外部设计。

## Related Specs

- 根 `AGENTS.md`：生成物、正式迁移、权限清单、无历史兼容、事务和删除数据授权边界。
- `.trellis/workflow.md`：planning阶段只做证据与文档，不启动实施。
- `.trellis/spec/server/backend/database-guidelines.md`：Ent为Schema源，正式SQL迁移、共享事务与schema全局对象禁并发。
- `.trellis/spec/server/backend/quality-guidelines.md`：专用集成库、正式迁移前态、被测时点。
- `.trellis/spec/server/backend/sea-export-document-contract.md`：主单身份/航程更正与下游事实保护。

## Caveats / Not Found

- 本次未连接或查询实际数据库，因此不能断言开发库 order_milestones 当前是否有记录。
- 没有在仓库找到现成Atlas migration diff命令；日常是Ent Schema源 + 正式SQL文件 + migration manifest/集成验证，不应擅自安装生成平台。
- 未运行生成器或测试，因为当前任务仍处planning。执行阶段需以真实结果验证旧产物清理、Wire编译及全部相邻门禁。
- `OrderMilestone` 不等于订单主流程状态/事件，不能一并删掉OrderFlowStatus或status transition history。
