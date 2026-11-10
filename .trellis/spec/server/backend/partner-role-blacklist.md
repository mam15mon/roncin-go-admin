# 往来单位角色级黑名单契约

## 1. 适用范围与触发条件

- 客户、供应商、国外代理的黑名单状态属于 `PartnerRole`，不属于整个往来单位档案。
- 新建或修改订单中的客户、订舱代理、国外代理引用时必须执行本契约。
- 黑名单只禁止新增业务关联；历史订单履约、账单、收付款、核销及清欠不得复用该门禁。

## 2. 接口签名

- RPC：`SetPartnerRoleBlacklist(SetPartnerRoleBlacklistRequest)`。
- HTTP：`POST /api/v1/partners/{id}/role-blacklist`。
- 请求字段：`id`、`role_type`、`blacklisted`、`reason`。
- 权限：复用 `business.partner.blacklist`，不建立角色各自的第二套权限。

## 3. 数据与行为契约

- `role_type` 只接受 customer、supplier、foreign_agent，且目标角色必须已经存在；黑名单操作不得自动创建角色。
- 同一档案不同角色的状态互相独立；列表操作直接提交当前页面对应的 `role_type`，不弹窗选择角色或状态。
- `reason` 是可选说明，去除首尾空白后不超过 500 个 Unicode 字符；加入和解除均允许不填，Web 不提供说明输入框。
- 解除时清空角色当前的原因、时间和操作人字段，但审计事件必须保留目标角色、操作人与状态，有说明时保留说明。
- 已拉黑角色不得通过角色替换、停用后重加或导入更新被移除；重新启用不得清空黑名单字段。
- 通用选择器保留黑名单单位可见，服务端写事务是权威门禁。
- 列表过滤契约（2026-09 起）：`ListPartners` 与 `ExportPartners` 支持
  `optional bool blacklisted` 查询参数，按目标 `role_type` 的角色记录过滤。
  `blacklisted=true` 时不要求该角色 `enabled=true`——拉黑与停用正交，
  已停用的黑名单档案必须仍出现在黑名单视图；`blacklisted=false` 语义为
  「存在该类型、启用且未拉黑的角色」；缺省不过滤，行为与历史完全一致。
- 客户列表提供「全部客户 / 黑名单客户 / 散客」页签。散客页签强制使用
  `is_casual=true`，搜索与重置保留该范围；切换页签和返回列表使用已提交查询快照。
  `ExportPartners` 同样接受 `optional bool is_casual`，与列表使用相同筛选；
  缺省不筛选，`false` 仅正式伙伴，`true` 仅散客。供应商和国外代理不增加散客页签。

订单字段与角色映射：

| 订单字段 | 要求角色 |
| --- | --- |
| 客户 | customer |
| 订舱代理 | supplier |
| 国外代理 | foreign_agent |


船代 `shipping_agent_id` 已改为公共 `MasterDataItem(kind=shipping_agent)`，不再映射 Partner supplier 或套用角色黑名单；新选择与旧停用引用规则见[公共船代目录](./organization-shared-masterdata.md#船代公共字典与订单引用)。

## 4. 校验与错误矩阵

| 条件 | 错误 reason | 行为 |
| --- | --- | --- |
| 非法角色类型 | `PARTNER_INVALID_ROLE` | 拒绝黑名单操作 |
| 目标角色不存在 | `PARTNER_BLACKLIST_ROLE_REQUIRED` | 拒绝且不自动建角色 |
| 可选说明超过 500 字 | `PARTNER_BLACKLIST_REASON_TOO_LONG` | 拒绝状态修改 |
| 尝试移除任意已拉黑角色 | `PARTNER_BLACKLISTED_ROLE` | 拒绝角色替换或导入 |
| 新订单或草稿换入黑名单角色 | `ORDER_PARTNER_ROLE_BLACKLISTED` | 返回包含角色含义的中文错误，整笔事务回滚 |

## 5. Good / Base / Bad Cases

- Good：草稿把订舱代理从 A 换成被拉黑的 B，后端按 supplier 角色拒绝并回滚。
- Base：订单继续保留后来被拉黑的原关联，允许保存其他字段；可选关联清空也允许。
- Bad：前端从候选项隐藏全部黑名单档案，或更新订单时不区分原关联与新换入关联；这会误伤历史履约和结算。

## 6. 必需测试

- Biz：三种合法角色、非法角色、无说明允许加入与解除、超长说明拒绝、Trim 后入仓及审计 `role_type`。
- Biz：ListPartners 的 `blacklisted` true/false/nil 三态透传（
  `TestPartnerListFiltersByBlacklisted`，范本 `TestPartnerListFiltersByIsCasual`）。
- Data：客户、订舱代理、国外代理三个字段的新建门禁；草稿换入拒绝、原关联保留允许、可选关联清空允许；同档案不同角色独立判定；错误 reason 正确。
- Data：角色替换和导入不能移除任意已拉黑角色，重新启用保留状态。
- PostgreSQL 集成：订单对去重且排序后的档案行使用 `FOR SHARE`，拉黑使用同一档案行 `FOR UPDATE`，验证并发提交先后不会产生检查空窗。
- Web：列表菜单按当前页面的 `PartnerRole` 状态显示「加入黑名单 / 移出黑名单」，点击立即提交当前页面的 `roleType` 与目标状态，不弹窗、不要求说明；加入成功切到黑名单页签，解除成功刷新当前列表；失败保留当前视图，处理中禁止重复提交。
- Web / Service：散客页签的搜索、重置、导出、返回恢复和其他页签切换；导出透传 `is_casual` 三态及已有公司、角色、状态与黑名单条件。

## 7. 错误与正确实现

错误：只查询 `PartnerRole.blacklisted`，随后在另一个事务写订单；并发拉黑可在检查与写入之间穿透。

正确：订单写事务内先将所有关联档案 UUID 去重、排序，再对档案行 `ORDER BY id FOR SHARE`；随后读取角色状态并完成订单写入。黑名单修改在事务中对同一档案行 `FOR UPDATE`，从而由数据库锁确定先后顺序。
