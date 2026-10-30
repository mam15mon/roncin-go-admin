# 独立订单履约里程碑退役

## 1. 适用范围

2026-10-30 彻底移除独立 `OrderMilestone` 功能。订单主流程操作进度、退关/完结、审核与单据/箱货/财务能力继续保留；不得把已退役里程碑恢复成隐藏入口或兼容接口。

## 2. 已删除签名

以下 RPC 与 HTTP 端点不再注册，OpenAPI 和生成客户端不再声明：

```text
OrderMilestoneService.ListMilestones
OrderMilestoneService.SetMilestone
GET /api/v1/orders/{order_id}/milestones
PUT /api/v1/orders/{order_id}/milestones/{type}
```

Ent 不再定义 `OrderMilestone` / `Order.milestones`；正式迁移
`20261030140000_remove_order_milestones.sql` 严格执行 `DROP TABLE "order_milestones"`，
不使用 `CASCADE` 或 `IF EXISTS`。原记录随表删除，订单和其他业务表不属于删除目标。

## 3. 跨层契约

- 删除 Proto 源、三层实现、ProviderSet、HTTP/gRPC/Wire 接线、前端抽屉、菜单/回调、开发种子及专属审计展示映射。不删除历史审计日志。
- 权限清单不再注册六业务类型的 `milestone.read/set` 共12项权限。`cmd/migrate` 现有 `SyncPermissionManifest` 事务删除清单外权限，`role_permissions.permission_id` 的级联外键清理关联；不另造权限清理框架或手写重复同步 SQL。
- 主单更正的下游事实校验仅移除里程碑谓词，保留审核锁、有效单证、POD、实际箱、账单和提成等事实门禁；订单操作进度不是里程碑替代门禁。
- Ent 与 Web 客户端生成器按最新源清理旧实体/服务。Buf 不自动删除旧源对应生成文件，须精确移除 `milestone.pb.go` / `milestone_grpc.pb.go` / `milestone_http.pb.go`，再标准生成，禁止手改聚合生成内容。
- 历史正式迁移和 Protobuf `reserved` 声明保留其历史/防字段复用含义，不能为全仓关键字清零而删除。

## 4. 校验与错误矩阵

| 场景 | 预期 |
| --- | --- |
| 订单更多菜单 | 没有履约里程碑，操作进度/费用/箱货/人员等仍可用 |
| 请求旧端点 | 不进入业务 handler，无兼容或空实现 |
| 正式前态已有里程碑 | 升级后专属表/索引移除，订单及相邻子资源不变 |
| 原角色获里程碑权限 | 正常迁移后不再持有该权限，其余授权保留 |
| 主单有POD或实际箱 | 身份/航程更正仍被阻断，无部分写入 |
| 正式前态缺少目标表 | 严格失败，不能静默跳过结构漂移 |

## 5. 正常、基准与错误场景

- 正常：完整删除纵向能力并生成当前契约；已有数据库通过增量正式迁移升级。
- 基准：空库重放全部正式迁移，末态与最新 Ent 表集合一致，不再有里程碑表。
- 错误：只隐藏菜单、保留无消费者服务/模型/权限，或者误删订单操作进度与主单保护。

## 6. 必须验证

- 隔离 PostgreSQL Schema：前态插入旧里程碑及目标授权后升级，断言表/权限移除，订单、相邻实体、授权与审计保留；重复正常迁移/同步无新增变更。
- 订单生命周期和主单更正相邻路径回归，POD/实箱继续阻断，版本与航程无部分更新。
- 前端更多菜单、角色可授予权限与审计未知动作回归；生成物重跑幂等。
- 最终完整前后端门禁；真实数据库验证单独记录PASS/SKIP，不将缺连接导致的跳过当验收。

## 7. 错误与正确示例

```text
错误：删菜单 → 留里程碑接口、Ent模型和权限
正确：删除纵向源与生成物 → 正式增量删表 → 复用权限同步 → 隔离迁移和相邻路径验收

错误：为去里程碑依赖，删掉整个主单下游事实校验
正确：仅移除HasMilestonesWith，保留POD、实箱、单证、审核锁及财务门禁
```
