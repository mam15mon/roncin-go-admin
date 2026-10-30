# 订单协作团队独立维护退役

## 1. 适用范围

2026-10-30 删除重复的“订单协作团队”独立维护入口及手工指派/删除接口。订单创建和详情中的人员分工、底层 `order_personnels` 关系及相关通知/提成依赖保留，不删除表或已有人员数据。

## 2. 接口签名

删除以下 RPC 与写端点，不保留兼容或空实现：

```text
OrderPersonnelService.AssignPersonnel
OrderPersonnelService.RemovePersonnel
POST   /api/v1/orders/{order_id}/personnel
DELETE /api/v1/orders/{order_id}/personnel/{id}
```

保留 `OrderPersonnelService.ListPersonnel` / `GET /api/v1/orders/{order_id}/personnel`，供订单详情表单只读回填；保留 `OrderPersonnel`、`OrderPersonnelRole`、`OrderPersonnelAssignmentInput` 及创建时的人员候选查询。

## 3. 数据与权限契约

- 前端删除抽屉、列表菜单与回调、专用角色选项、独立增删动作的审计展示映射；不删除历史审计数据。
- Service/Biz/Data 中删除独立Assign/Remove方法，人员仓储保留List。仍在核心订单创建中使用的角色、参数错误、组织成员校验和通知逻辑不得随抽屉删除。
- Manifest删除六业务类型的 `personnel.assign/remove` 共12权限，保留 `personnel.read`。权限目录/旧角色关联由正常迁移命令内置的 `SyncPermissionManifest` 清理，不另造权限同步或Schema迁移。
- `OrderPersonnelRole` 仍由 `order.proto` 的创建请求消费，不能删除整份 `order_personnel.proto` 或全部生成客户端；只删除专属RPC/消息，再按标准入口生成。
- 本次不改变既有表单的只读/可选规则、订单写入和三岗校验，也不改变审核锁、数据范围和提成归属。

## 4. 校验与错误行为

| 场景 | 预期 |
| --- | --- |
| 列表更多菜单 | 无协作团队维护入口，费用、自拼汇总等仍可访问 |
| 旧指派/删除端点 | 不进入业务handler，无法绕过订单表单增删分工 |
| 订单详情读取人员 | 原 `personnel.read` 规则仍生效，人员正常回填 |
| 创建订单 | 创建人及角色分工正常落库，组织成员校验/通知保留 |
| 旧角色有assign/remove授权 | 正常清单同步后移除，其他权限及人员记录保持 |

## 5. 场景

- 正常：从订单创建/详情入口查看人员，服务端仍按既有核心契约维护分工。
- 基准：权限同步前后订单与人员整行快照、只读List结果不变，原有效read/create授权保留。
- 错误：只隐藏菜单而保留写API，或因删除抽屉顺带删人员表、读权限、角色枚举及创建通知。

## 6. 必需验证

- 前端菜单移除与创建/详情人员表单、只读回填相邻测试；角色可授予计数及夹具更新。
- 真实隔离PostgreSQL：核心创建形成分工，构造旧12权限与关联后同步，断言精确清理且人员/订单/读结果保持；重复同步幂等。
- 公司与部门成员可用、跨公司人员拒绝并回滚的核心创建用例。
- API/Web客户端/权限键标准生成与幂等；完整前后端门禁和独立review。

## 7. 错误与正确

```text
错误：删除OrderPersonnel实体与所有RPC，导致详情人员无法回填、通知和提成失去关联
正确：仅移除独立编辑入口与写接口，保留读取及核心人员关系
```
