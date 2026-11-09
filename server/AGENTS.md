# Roncin 服务端开发约束

本文件适用于 `server/`。先遵循[根协作规范](../AGENTS.md)，再查[后端能力导航](../.trellis/spec/server/backend/capability-navigation.md)与[后端规范索引](../.trellis/spec/server/backend/index.md)。这里补充服务端入口，不另建一套契约或业务规则。

## 分层与事实来源

- `api/<领域>/<版本>/` 的 `.proto` 是 HTTP/gRPC 契约唯一真相源；请求 DTO 在 `internal/service` 与领域对象互转。
- `internal/service` 只做边界校验、转换及用例调用，不读 Ent、不直接访问数据库。
- `internal/biz` 持有纯 Go 领域对象、用例、仓储接口和领域错误，不依赖 `data` 或 `service`，不泄漏 Ent/Protobuf 消息。
- `internal/data` 实现仓储、处理持久化转换与驱动错误映射；Ent 类型留在本层。客户端经 `Data.client(ctx)` 获取，共享事务内不得直连连接池。
- `internal/server` 注册传输、中间件、健康检查、静态资源及 worker，不承担业务决策；`cmd/` 负责入口和 Wire 装配。
- 权限码与依赖以 `internal/access/manifest.go` 为准。公司经营读写限定当前公司工作区，系统工作台不提供经营权限旁路；角色不再配置数据范围。

## 写入与数据边界

- 事务复用 `Data.WithTx`；跨仓储事务复用 `biz.Transactor.WithinTransaction`。按主键排序加锁、锁内校验版本及权威事实；失败整体回滚，事务提交后重读完整响应。
- 多人编辑实体按根规范使用 `ForUpdate` 与 `expectedVersion`；主数据是否需要版本按实体用途判断，不给所有资源机械套用同一更新模板。
- 常规分页复用 biz 公共校验，`page >= 1`、`1 <= pageSize <= 200`；选择器由服务端关键字过滤并覆盖拼音。具体字段与更新语义以该资源 `.proto` 为准，不强制引入仓库未采用的 AIP/fieldmask 协议。
- 数据结构以 Ent Schema 声明、正式 SQL 迁移执行。集成验证使用有序正式迁移，禁止运行期自动建表与并发 `Schema.Create`。
- 订单审核锁、逐费账单占用、费用补审与提成经济重算分别遵循对应专项规范；不得恢复独立费用预审、提成财务锁或业务标签。

## 生成与验证

从仓库根目录运行：

```bash
make -C server api                   # Go 绑定、服务端 OpenAPI 与接口权限规则
make -C server config                # 配置 Protobuf
pnpm run generate:web-client         # 消费 OpenAPI，生成前端客户端与 Proto 常量
pnpm run generate:permission-keys    # 修改 Manifest 后更新前端权限键
```

Ent Schema、Wire 等生成入口见源码中的 `go:generate`；需要时运行 `go -C server generate ./...`。禁止手改 `*.pb.go`、`*_grpc.pb.go`、`*_http.pb.go`、`wire_gen.go`、Ent 生成物或 `openapi.yaml`，生成物与源文件同组提交。

功能实现后按风险补验证，禁止 TDD。开发中先运行受影响 Go 包测试；任务验收按根规范使用 `pnpm run check:server` 或 `pnpm run check:fast`，真实库测试 SKIP 不计为数据库验收通过。普通文档修改只执行适用的文档检查，不以构建或写库验收代替事实核对。

文档、开发者注释和提交说明使用中文。保护未提交工作与秘密；运行日志为结构化 JSON，业务审计独立存储。每组可验证修改使用中文 Conventional Commit 提交，源文件及必要生成物同组审阅。
