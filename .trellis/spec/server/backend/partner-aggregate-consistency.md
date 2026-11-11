# 客商聚合子资源权限与默认抬头一致性

## 1. 适用范围

主档 Create/Update/Import 内嵌结算规则、主档聚合响应与导出，以及开票抬头新建/编辑切默认。
主档权限不授予独立结算规则权限；聚合写入与独立维护必须守住同一边界。
常用单证预设的启停与默认标记同样需要在创建和更新之间保持一致。

## 2. 签名与入口

`biz.PartnerSettlementRuleWriteAccess{Create, Update bool}` 是零值拒绝的纯 Go 动作能力，
经 `PartnerUsecase`、`PartnerRepo` 的 Create/Update/Import 显式传递至
`data.savePartnerRoleSettlementRules`。Service 按目标公司调用
`Principal.CanAccessOrganizationForPermission` 构造能力，不把权限存入 Partner。

`partnerToAPI`、`partnerExportItemToAPI` 与 `partnerRoleToAPI` 接收显式
`includeSettlementRule bool`；每个返回主档按其真实 OrganizationID 判断规则读权限。

`normalizePartnerShippingPreset` 是常用单证预设 Create/Update 共用业务校验入口，
`ErrPartnerShippingPresetInvalidArgument` 对应 HTTP 400。

## 3. 契约

- 内嵌规则 nil 表示本次不写规则，无需规则写权限。非 nil 时在现有事务及锁内按实际规则
  数量判定：零条需 create，一条需 update，多条保留聚合入口的明确拒绝。
- 主档、角色、规则及审计保持事务原子性，越权不得留下主档修改或部分导入结果。
- Get/List/Create/Update/UpdateRoles/SetPartnerRoleBlacklist/Export 七个响应入口统一裁剪；
  无规则读权限不返回嵌套规则，不能只隐藏前端分节。Import 只返回数量。
- 前端规则读/建/改权限分别使用现有 access 结果，不用通用 `canManagePartners` 放行。
  已有档案依据成功加载后的规则存在性选 create/update；查询失败、尚未完成不能视为空规则。
  普通主档保存省略无权修改的规则，并跳过该分节的写入校验；只读规则不阻断普通资料保存。
- 开票抬头清除旧默认状态也是内容变更。Create 和 Update 两条路径均在原父档案锁及事务内，
  对实际被清默认的行 `AddVersion(1)`；直接编辑项按原版本逻辑递增一次，其他非默认项不变。
- 常用单证预设的 `enabled=false,isDefault=true` 属矛盾请求，在共用 Biz 入口明确拒绝，
  不进入仓储事务、不清除旧默认，不静默把默认标记改成 false。禁用非默认项及启用默认项合法；
  不为修复自动清理旧数据。

## 4. 校验与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 只有主档权限，提交内嵌规则 | `ErrPermissionDenied`，整次写入回滚 |
| 有 create 无 update，角色已有一条规则 | 拒绝更新，不能将聚合写当作新建 |
| 有 update 无 create，角色尚无规则 | 拒绝新增 |
| 无内嵌规则，普通主档写入合法 | 正常保存，规则不变 |
| 无规则 read，读取聚合或导出 | 规则字段不输出；其他合法字段照常返回 |
| A 的默认状态被 B 替换后，提交 A 旧版本 | 版本冲突，不把默认切回 A |
| 创建/更新常用单证预设为禁用且默认 | 参数错误 400，原默认、目标资料、关联及审计不变 |
| 禁用非默认预设 / 启用默认预设 | 正常创建或更新；后者可替换原默认 |

## 5. 正常与错误场景

正常：只有主档编辑权的用户修改电话，载荷无结算规则且成功；有规则新增权的用户在已有
档案的空规则角色上新增规则。错误：把“已有档案页面”直接等同规则 update，或者凭主档
update 权写信用额度。默认抬头变化只影响后续新建发票的预选，不改已开票快照。

## 6. 必需验证

七个聚合响应有/无规则读权限及公司范围；Create/Update/Import 真实动作权限及整事务回滚；
零/一/多规则与停用规则；普通保存不带规则；规则查询失败保留输入并可重试。
真实 PostgreSQL 覆盖新建默认与编辑切默认、旧表单冲突、未受影响行版本和失败零部分写入。
常用单证预设另覆盖 Party/Text 两类的创建/更新矛盾组合不调用仓储，以及真实 Service→Biz→Data
在正式迁移 PostgreSQL 上拒绝时零写入；正常启用默认切换和禁用非默认创建/更新保持有效。

## 7. 错误与正确

错误：Service 锁外查到零条规则就授权 create，仓储锁内却更新随后出现的规则；或给聚合入口
默认完整权限。正确：Service 显式传目标公司动作能力，仓储锁内根据真实写入动作检查；
响应由 Service 按规则 read 裁剪。错误：只把旧默认置 false；正确：同语句推进其 Version。
单证预设错误：先清旧默认，写入时才用 `isDefault && enabled` 纠正；正确：进入仓储前拒绝矛盾组合。
