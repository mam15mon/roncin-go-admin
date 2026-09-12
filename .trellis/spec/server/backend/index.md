# Server 后端开发规范（Go / Kratos / Ent）

> 本层规范的唯一真相源是根目录 `AGENTS.md`；本目录是面向 AI 任务执行的浓缩版。
> 冲突时以 `AGENTS.md` 为准。

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | 分层职责与依赖方向 | ✅ |
| [Database Guidelines](./database-guidelines.md) | Ent、事务封装、并发锁、分页 | ✅ |
| [Sea Export Document Contract](./sea-export-document-contract.md) | 海运操作票、共享 MBL、运输执行与 HBL 跨层契约 | ✅ |
| [Order Lock and Document Version](./order-lock-and-document-version.md) | 海运出口订单业务锁、不可变版本与共享 MBL 固定锁序 | ✅ |
| [Sea Document Change History](./sea-document-change-history.md) | 海运提单改单、作废、Switch、财务门禁与不可变历史 | ✅ |
| [Finance Bill Currency](./finance-bill-currency.md) | 普通账单固定费用币种边界、账单日汇率、核销/对冲本位币口径与预计开票快照 | ✅ |
| [Exchange Rate Single Rate](./exchange-rate-single-rate.md) | 单一折本币基准汇率：总部唯一维护、各单据按业务日期消费、直连缺失经基准币交叉套算 | ✅ |
| [Finance Commission Lock](./finance-commission-lock.md) | 提成净额财务锁：Clawback 冲减、净额归零释放、双落点同口径 | ✅ |
| [Organization Shared Masterdata](./organization-shared-masterdata.md) | 总部共享主数据读取模式：本组织+总部并查、同码本组织优先、SQL 下推去重 | ✅ |
| [Partner Casual Contract](./partner-casual-contract.md) | 散客往来单位跨层契约：伙伴级标识、账期零底线、出款账户刚性 | ✅ |
| [Auth Session Org Switch](./auth-session-org-switch.md) | 登录组织选择、应用内切换与会话轮转契约：成员资格唯一入场券、单事务轮转、并发语义 | ✅ |
| [DingTalk Registration Approval](./dingtalk-registration-approval.md) | 钉钉注册双通道：邀请自动激活（降级红线）、认领审批、按目标组织路由通知、手机号与令牌安全 | ✅ |
| [Error Handling](./error-handling.md) | 领域错误与驱动错误映射 | ✅ |
| [Quality Guidelines](./quality-guidelines.md) | 分层禁令、生成物、常用命令 | ✅ |
| [Logging Guidelines](./logging-guidelines.md) | 结构化日志与审计 | ✅ |

## Pre-Development Checklist

动手写代码前确认：

1. 接口变更是否已先修改 `.proto` 契约（禁止手改 `*.pb.go` 等生成物）。
2. 涉及的层是否清楚：DTO 转换在 `service`、业务规则在 `biz`、Ent 查询在 `data`。
3. 是否需要事务 / 并发防护；新实体是否需要 `version` 字段（判定不清先问用户）。
4. 列表接口分页是否复用 `internal/biz` 公共校验（`1 <= pageSize <= 200`）。
5. 新增权限码是否登记 `internal/access/manifest.go` 并重新生成前端权限键。
6. 订单锁、解锁或共享 MBL 写入是否遵循不可变版本与固定锁序规范。
7. 单改、作废或 Switch 是否先 Preview、锁内重算，并保留不可变历史与财务门禁。
8. 跨组织访问是否按具体权限筛选角色、保留权限与组织范围来源绑定，并在仓储查询中
   显式应用实体 ID 与允许组织集合。
9. 普通账单是否始终按费用币种拆分，契约中不存在关闭分币种、目标账单币种或跨币种人工折算旁路。
10. 涉及汇率、核销、对冲、提成锁或总部共享主数据的改动是否对照对应专项规范
    （exchange-rate-single-rate / finance-bill-currency / finance-commission-lock /
    organization-shared-masterdata）。

## Quality Check

提交前确认：

- `go -C server vet ./...` 与 `go -C server test ./...` 通过（按风险选取最小集）。
- 契约变更已重新生成绑定代码与 OpenAPI / 前端客户端，生成物无手改痕迹。
- 迁移文件已随代码提交，未绕过 Ent 手写业务 SQL。

**语言**：文档与面向开发者的注释使用中文。
