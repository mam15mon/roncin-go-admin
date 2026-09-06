# 海运出口船公司改用航运公司主数据

## Goal

让海运出口订单直接选择组织内启用的 `shipping_lines` 船公司主数据，消除“船公司必须先建成
Partner 承运人角色”这一错误前置条件；订单链路只保存航运公司行业身份，真实需要付款、结算、
联系人或合同管理的订舱代理、船代、NVOCC 等主体继续作为 Partner 供应商维护。

## Background

- 海运出口选择船公司用于记录承运航线事实、船名航次、箱号及提单/箱号前缀，不直接向船公司
  付款结算，因此船公司不属于本系统中的商务往来单位角色。
- 当前前端船公司下拉调用 Partner 列表并限定 `role_type = carrier`；当前组织没有 carrier
  Partner，所以即使 `shipping_lines` 已导入 358 条且全部启用，下拉仍为空。
- 当前 Order、SeaTransportExecution 使用无外键的 `carrier_id`，SeaMasterBill 及其不可变版本
  则把同一 UUID 保存成带 Partner 外键的 `issuer_partner_id`。上一任务刚建立了三者相等的不变量，
  但主体类型仍然错误。
- 当前开发数据库有 0 张 SE 订单、0 张 MBL、0 张 MBL 版本、0 条 carrier Partner 角色；本次
  可以直接收敛到新模型，不需要旧语义兼容或数据猜测映射。
- HBL 的 `issuer_partner_id` 表示真实的客户或其他往来单位签发主体，与 MBL 船公司身份不同，必须保留。

## Requirements

### FR-1：海运出口船公司选择与展示

- 新建、详情编辑、拆票和整票改配中的“船公司”统一从当前组织已启用的 `shipping_lines` 查询，
  不再查询 Partner carrier 角色。
- 下拉支持服务端关键字检索 SCAC、中文名、英文名、中文全拼和拼音首字母，单次最多返回 50 条，
  不由前端循环加载全量 358 条数据。
- 候选项展示“中文名 / 英文名（SCAC）”；已有订单、共享 MBL、拆票和改配摘要展示可读名称，
  不把 UUID 当船公司名称显示。
- 新建或更换船公司只允许选择同组织且启用的 ShippingLine；历史引用即使后续被停用，仍可查看，
  且不得被普通无关字段保存静默替换。

### FR-2：订单、航程和共享 MBL 身份

- 使用语义明确的 `shipping_line_id` 取代 Order 与 SeaTransportExecution 的 `carrier_id`，取代
  SeaMasterBill 的 `issuer_partner_id`；禁止让同一 UUID 在不同表中分别假装 Partner 和 ShippingLine。
- 新写入的海运出口数据必须满足：
  `Order.shipping_line_id = SeaTransportExecution.shipping_line_id = SeaMasterBill.shipping_line_id`。
- MBL 唯一身份改为
  `(organization_id, shipping_line_id, normalized_master_no)`；相同组织、船公司和主单号仍进入现有
  共享 MBL 候选确认流程。
- SeaMasterBillInput 不再接收 MBL Partner 签发方；候选匹配、拆票和改配只传一个
  `shipping_line_id`，由服务端在现有事务和固定锁序内校验三方一致性。
- MBL 不可变版本只保存一个 `shipping_line_id` 船公司身份；删除重复的 issuer/carrier 双字段语义。

### FR-3：Partner carrier 角色退场

- 从 Partner Proto 枚举、Biz 类型、Ent Schema、转换逻辑和前端角色选项中删除 carrier 角色；Proto
  保留原编号和名称，禁止将枚举值 4 复用于其他角色。
- 数据库为 Partner role 增加只允许 customer、supplier、foreign_agent 的约束。
- 迁移前若发现 carrier 或其他非法 Partner role，必须原子停止并明确报错；不得静默删除、改成供应商
  或创建替代数据。
- 实际有应付结算关系的订舱代理、船代或 NVOCC 继续使用 Partner supplier，不由 ShippingLine 替代。

### FR-4：费用结算边界

- ShippingLine ID 不得作为费用 `settlement_party_id`、账单往来单位或其他 Partner 外键使用。
- 新增应付费用时，只能默认订舱代理 Partner；没有订舱代理时结算单位保持空白，由用户选择真实供应商，
  不再回退到船公司。
- 费用页的订单概要显示船公司可读名称，但不暗示其为付款对象。

### FR-5：迁移与兼容边界

- 新增一条不可变增量迁移，调整列名、索引、外键和 Partner role 约束；不得修改历史迁移。
- 因当前开发数据库没有 SE 业务数据，迁移不提供 Partner carrier 到 ShippingLine 的名称或代码猜测映射。
  若目标数据库存在 SE/MBL/MBL 版本数据，迁移必须在任何 DDL 前停止，由后续独立数据治理任务处理。
- API、OpenAPI、Go 绑定和 Web Client 统一改用 `shipping_line_id` / `shipping_line_name`；不保留旧 JSON
  字段或双写兼容分支。
- 当前未提交的船公司同步程序和 358 条已导入数据属于用户现有工作，本任务不得覆盖、回退或混入代码提交。

## Acceptance Criteria

- [x] AC-1：当前组织已有启用 ShippingLine 时，SE 新建页“船公司”下拉可返回数据；按 SCAC、
  中文名、英文名和拼音关键字均能检索，且只展示启用项。
- [x] AC-2：创建或更新 SE 订单后，Order、SeaTransportExecution、SeaMasterBill 保存同一个有效
  ShippingLine ID；跨组织、停用或不存在的 ID 被明确拒绝且零部分写入。
- [x] AC-3：相同 ShippingLine 与相同规范化 MBL 号仍触发共享候选确认；不同 ShippingLine 的相同
  MBL 号可以共存，候选版本、航程冲突和多成员修改限制无回归。
- [x] AC-4：拆票与整票改配的新目标和候选目标只维护一个 ShippingLine 身份，保存后三方一致；页面不再
  生成或提交 MBL `issuer_partner_id`。
- [x] AC-5：MBL 摘要、候选、不可变版本和费用订单概要显示 ShippingLine 名称；HBL 的客户/其他 Partner
  签发主体、唯一性、版本与改单行为保持不变。
- [x] AC-6：Partner 页面不再提供“承运人”角色，接口拒绝已保留的枚举值 4，数据库拒绝非法角色；
  customer、supplier、foreign_agent 的维护与结算规则正常。
- [x] AC-7：新增应付费用在有订舱代理时默认该 Partner，无订舱代理时为空；ShippingLine UUID 从不写入
  `settlement_party_id`。
- [x] AC-8：新迁移从空 Schema 和当前开发 Schema 均可成功执行并保留全部已有 ShippingLine（迁移测试
  固定覆盖 358 条，当前开发库实测 371 条）；构造旧 SE 数据或 carrier role 时迁移原子失败且不留下
  部分 DDL。
- [x] AC-9：Proto、Ent、OpenAPI、前端客户端与手写代码中不再存在 MBL/SE 的旧 carrier/issuer 字段；
  仅 HBL Partner issuer 和与业务无关的自然语言“carrier”可保留。
- [x] AC-10：针对性及完整 Web/Server 质量门禁通过，生成物仅由生成命令更新，用户原有未提交导入改动
  保持原样且不进入本任务提交。

## Out of Scope

- 不在本任务中建设航线、船舶、航次、挂港、船期或自动填充 ETD/ETA 的新主数据模型。
- 不根据 ShippingLine 自动生成箱号、MBL 号或航次；现有箱号与提单号校验/录入规则保持不变。
- 不把订舱代理、船代、NVOCC 等真实商务主体从 Partner 迁移到 ShippingLine。
- 不修改 HBL 的 SELF_ORGANIZATION、CUSTOMER_PARTNER、OTHER_PARTNER 签发规则。
- 不为已有生产 SE 数据设计自动映射、双读、双写、回退或兼容 API。
- 不调整海运出口页面整体布局。

## Key Decisions

- 数据字段统一明确命名为 `shipping_line_id`，不沿用语义含混的 `carrier_id`。
- MBL 船公司身份不再叫 issuer Partner；一个 `shipping_line_id` 同时承担共享主单身份和权威航程船公司。
- ShippingLine 是行业参考主数据，Partner 是商务往来主体，两者不建立一对一镜像或自动同步。
- 当前数据为空使直接迁移可行；发现旧业务数据时选择“停止迁移并单独治理”，不猜测映射。
