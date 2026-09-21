# 后端能力导航（Server Biz）

> 开发前先查：场景 → 业务入口 → 能力。入口一律指向 `server/internal/biz/` 的
> 领域文件；传输层 DTO 在 `internal/service/`、持久化在 `internal/data/`，
> 分层规则与禁令见 [directory-structure](./directory-structure.md)，import
> 边界由 `pnpm run check:layers:go` 自动强制。
> 业务术语与数据流见 [领域知识层](../../domain/architecture-map.md)。

## 场景 → 入口

| 场景 | 业务入口（biz 文件） | 主要能力 | 反模式 |
| --- | --- | --- | --- |
| 订单创建/查询/流转 | `order_usecase.go`、`order.go`、`order_transition.go` | 订单档案、状态流转、列表过滤、跨组织范围 | 在 service 写业务规则或绕过 biz 直查 data |
| 海运出口单证 | `sea_document.go`、`sea_master_bill.go`、`sea_cargo_allocation.go` | 操作票、共享 MBL、货物分摊（契约见专项 spec） | 单证版本判断散落调用方 |
| 海运改单/作废/Switch | `sea_document_change.go`、`order_shipping_document.go` | 改单历史、财务门禁、不可变版本 | 跳过 Preview 直接改单 |
| 订单业务锁 | `order_lock.go`、`order_auto_lock.go` | 手动/自动锁、解锁审批、固定锁序 | 并发写单证不加锁序 |
| 订单费用 | `order_fee.go`、`order_fee_supplement*.go` | 费用录入、补充费用、候选选项 | 候选合并丢失订单身份 |
| 账单 | `finance_bill.go`、`finance_bill_batch.go` | 建账、批量建账、固定费用币种边界 | 跨币种人工折算旁路 |
| 核销 | `finance_verification.go` | 费用核销、反核销、汇兑损益 | 绕过 idempotencyKey 重放 |
| 对冲 | `finance_netting.go` | 应收应付对冲、汇差 | 独立实现汇差口径 |
| 收付 | `finance_cashflow.go` | 收付款流水、散户出款账户刚性 | 散户出款走普通账户 |
| 发票 | `finance_invoice.go`、`partner_invoice_profile.go` | 发票登记、开票资料 | 修改他人组织发票 |
| 结算单位设置 | `settlement.go`、`finance_custom_setting.go` | 结算候选、自定义设置 | 信用判定复制第二套 |
| 提成方案与分配 | `finance_commission_rules.go` | 方案区间、员工分配、资格门禁 | 实际区间允许重叠 |
| 月度提成申请 | `finance_commission_application.go`、`finance_commission_lifecycle.go` | 申请、批准/驳回、Clawback 锁 | 绕过净额锁直接改提成 |
| 汇率 | `exchange_rate.go`、`exchange_rate_import.go`、`exchange_rate_reminder.go` | 周汇率双轨、导入、到期提醒 | 自造第二套汇率容灾链 |
| 主数据 | `masterdata.go`、`industry_reference.go`、`reference_data.go`、`fee_catalog.go` | 船公司/港口/机场/行政区划、费目目录 | 绕过 B 型统一读取谓词 |
| 往来单位档案 | `partner.go` | 档案、散客契约、角色黑名单 | 散客标识写多处 |
| 往来单位财务配置 | `partner_account.go`、`partner_contract.go`、`partner_credit.go`、`partner_settlement_rule.go`、`partner_shipping_preset.go`、`partner_attachment.go` | 账户、合同、信用、结算规则、运输预设、附件 | 信用口径与账单不一致 |
| 组织与权限 | `admin_organization.go`、`admin_role.go`、`admin_user.go`、`admin_user_membership.go`、`access/`（manifest） | 组织、角色库、用户、成员资格、权限清单 | 前端复制权限真相 |
| 登录与会话 | `auth.go` | 登录、组织切换、会话轮转 | 切换不走单事务轮转 |
| 审计 | `admin_audit.go`、`biz.go`（审计上下文） | 业务审计独立存储 | 审计写运行日志 |
| 业务标签 | `business_tag.go`、`enterprise_resource.go` | 标签分组、企业资源配置 | 每页自建标签查询 |
| 钉钉集成 | `dingtalk_registration.go`、`dingtalk_approval.go`、`dingtalk_invitation.go` | 注册双通道、审批、邀请 | 手机号/令牌入日志 |
| 后台任务/通知 | `background_task.go`、`notification.go`、`workbench.go` | 任务调度、通知、工作台聚合 | 工作台直查多仓储 |
| 订单辅助实体 | `order_cargo_item.go`、`order_container.go`、`order_milestone.go`、`order_personnel.go`、`order_attachment.go`、`order_abnormal_case.go`、`order_release_pod.go`、`orderconfig.go` | 货物、箱型、里程碑、人员、附件、异常、放货 | 辅助实体越权写他单 |
| 公司人员候选与资格 | data `company_personnel.go`，biz `PartnerRepo.ListAssignmentOptions` / `OrderRepo.ListPersonnelOptions` | 当前公司及部门/团队、同用户聚合、部门展示、岗位写入成员范围 | 穿透下属公司、候选与保存范围不一致 |
| 数据清理 | `object_deletion.go` | 受控删除 | 物理删除不走用例 |

## 维护约定

- 新增 biz 领域文件或用例时同步本表；入口路径必须真实存在。
- 导航只写「场景→入口」，字段与行为细节以 proto 与 Ent Schema 为真相源。
- 传输层与持久化入口按 `service/`、`data/` 目录同名约定定位，不在本表重复。
