# 进口订单与按票承接服务

## 1. 适用范围

SI 使用现有海运主单、航程、可选 HBL、箱货、费用和人工审核锁；业务方向为 IMPORT，
主单与订单类型必须同为 SI。SE 继续原出口流程。进口服务模型同时供后续 AI 使用，
不能仅因结构存在就用 AE 权限开放 AI 创建。已批准范围不包含任意流程配置器或标签能力。

代码入口：`biz/order_import_handling.go`、`data/order_import_handling.go`、
`service/order_import_handling.go`、`api/order/v1/order_import_handling.proto`。
方向身份及完整共享集合规则见[主单方向身份](./order-master-business-identity.md)。

## 2. 接口与数据库签名

| 接口 | 合同 |
| --- | --- |
| `GET /api/v1/orders/{order_id}/import-handling` | 真实订单品类 read 权限，返回本票服务及 Order.version |
| `PUT` 同路径 | 真实品类 update；`expectedVersion,exchangeAgentId?,steps[5],reason` |
| `GET .../import-handling/events` | 本票 read；标准 page>=1，pageSize1..200，不可变前后事实历史 |

服务响应 `OrderImportHandling{orderId,orderVersion,exchangeAgentId?,exchangeAgentName,steps}`。
每项 `ImportHandlingStep{kind,undertaken,occurredAt?,responsibleUserId?,responsibleUserName,
referenceNo,notes,attachmentIds}`；kind 固定 ARRIVAL/EXCHANGE/CUSTOMS/PICKUP/DELIVERY。
时间为 RFC3339，完成时间响应保留合法小数秒精度（RFC3339Nano），避免原样保存
被误判成完成事实修改；响应姓名是只读投影，不作为身份或权限真相。事件包含
`id/orderId/actorId/actorName/occurredAt/reason/before/after`，界面不以 UUID 代替姓名。

Ent `OrderImportHandling(order_id UNIQUE,state JSONB)` 只保存当前本票服务事实；
`OrderImportHandlingEvent(order_id,actor_id,occurred_at,reason,before JSONB,after JSONB)`
全部追加不可变，按订单/发生时间索引。唯一并发版本为 Order.version，不新增服务镜像版本。
正式迁移 `20261116160000_sea_import_handling.sql` 同步服务表、进口 flow 约束、
旧进口空承接初态及 SI 审核海运快照外键 CHECK。Schema 初始化只执行有序正式 SQL。

## 3. 业务、状态与前端合同

### 五项完整快照与事实

- 建票初态为五项均未承接，无伪造完成时间；服务由建票后独立面板维护。
  PUT 提交五项完整且不重复的快照，普通整票保存不写此资源。
- 未承接为“不适用”，承接但未设置 occurredAt 为“未完成”，实际时间存在为“已完成”。
  undertaken 为非 optional bool，ProtoJSON 省略表示 false；完成时间为 optional，
  未设置与非法空字符串不同，不能按 TypeScript `?` 猜字段语义。
- 可承接任意子集并行办理；共享运输实际到达不自动完成本票到货或同主单其他票。
  服务完成不自动签单、放货、审核、建账、核销或发放提成。
- 参考号最多 200 字、备注 2000 字、原因 500 字；每项附件最多 50 个，禁止重复 ID。
  换单代理为当前公司启用且未拉黑的供应商；责任人属于当前公司有效人员；
  附件必须是当前票且附件资产属于当前公司，不接受另一票或公司的引用。
- 取消承接清除当前完成时间，保留当前备注、参考号、责任人与附件；旧完成事实留在事件。
  取消已承接项目前前端明确确认，取消确认不发写请求。
- 既有完成项目的时间、适用性、责任人、参考号、备注、附件任一事实改变必须有原因；
  已完成换单的换单代理改变同样要求原因。新增承接及未完成资料补录不误要求撤回原因。

### 进度、锁与完结

| 当前服务集合 | 服务端派生 flow |
| --- | --- |
| 无承接 | IMPORT_NOT_APPLICABLE |
| 有承接且无完成 | IMPORT_NOT_STARTED |
| 部分完成 | IMPORT_IN_PROGRESS |
| 所有承接项目完成 | IMPORT_COMPLETED |

- 服务保存事务内派生上述 flow，客户端不能直接逐步写入进口摘要；SI/AI 不走出口
  已订舱/已签单跳转。数据库同时限制出口不得写进口 flow、进口不得写出口 flow。
- 完结检查当前承接服务与既有必要资料/费用门禁，未承接项目不阻断。
  已 CLOSED 的服务变更必须先通过显式重开命令；不改变全站其他普通资料的完结后编辑规则。
- 保存先锁目标 Order，比较 expectedVersion，再检查真实公司/进口类型、审核锁、
  ACTIVE 与 OPEN 状态，复验引用，更新服务/Order.version+1/flow/事件/审计同事务。
  任一步失败全回滚；提交后普通上下文重读，不在共享事务中运行独立原生 SQL。
- SI 审核包含真实 MBL、航程及其不可变版本、可选 HBL，不使用空或旧通用单证快照。
  终止/恢复及放货按真实 SI 单证与本公司权限复验，不能借 SE 权限或跨方向主单。
- SI 维护共同航程的预览/执行使用真实 SI update 权限；SE 沿用 reassign。
  权限源、传输门禁、服务端目标及完整成员复验、前端按钮须一致；不因复用共同航程
  开放SI整票改配/拆票，也不能要求不存在的SI reassign权限。
  两RPC统一声明路由动作 `transport.update`，由 `access.OrderPermission` 映射上述
  已有权限，不新增重复权限码。主单/航程候选按请求中显式的businessType授权，不能硬编码SE。

### 前端基线与复用

- SI 复用集中 registry、路由、OrderFormTemplate、海运字段与实际箱货，不复制整套 SE 页面。
  详情、费用、HBL 成功读取边界须严格匹配路由的 businessType；缺失/0/错类型拒绝，
  不为旧测试夹具保留 if-present 放行。路由标题仍唯一来自 config/routes.ts。
- `pages/orders/import-handling/ImportHandlingPanel.tsx` 为模块内服务入口，
  `model.ts` 统一 wire 和表单转换；分别维护承接/完成/人员/附件及原因，不改主单正文。
  当前公司、真实品类 read/update 与父表单可写策略共同约束；无候选权限时保留既有选择。
- 已有票前台每 15 秒及回焦检查，隐藏/非当前页面暂停；探测新快照只提醒，不能回填编辑。
  检查失败保留既有变动提醒和输入。服务 ModalForm 固定打开快照，不借新的 Order.version。
- 显式重新读取覆盖未保存服务输入前确认；取消/失败不丢输入，刷新在途又有编辑不采纳旧响应。
  A→B→A、组织及编辑会话变化拒绝迟到数据；保存等待期间的新输入保持，不自动关闭旧稿。
- 独立服务成功回调刷新聚合但不 reset 整票表单。只在 saved.orderVersion=previous+1、
  Form 基线仍为 previous、重读订单恰为 saved.version、MBL/航程/Link/HBL 身份版本未变时，
  才推进整票 Form.version 与本人检查参考点；更晚版本继续保留旧草稿并提示刷新。

### SI 外部 HBL 登记归档

- 本公司自签继续 RONCIN_SELF 原四种输出；SI 外部真实 issuerPartner 由服务端派生
  `printTemplate=EXTERNAL_ARCHIVE`，保留签发来源/外部主体 ID 和名称。
- 现有 HBL 编辑维护同一正文/印字/版本，外部归档使用“外部提单登记资料（归档）”A4资料页，
  展示真实外部签发方、已保存版本及全部登记字段，不借当前公司英文名或虚构外部签章/条款。
  不提供本公司正本/电放法律版本选择，不把制作PDF称为办理签发或放货。
- 预览、下载、打印共用保存快照和同一 Blob；生命周期指纹包含真实单证/正文/印字、
  签发主体和显式模板修订值，续页同样标记有界单号及保存版本。长文逐行续页。必须实际 PDF 提字/看图验收，
  布局树存在 Text 不代表真实渲染可见；数值行高按渲染器实际点数核查。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| 缺少/重复/未知服务项目，未承接却有完成时间，非法时间/长度/附件集合 | 400，不默补未知环节 |
| 服务已完成事实改变而无原因 | 400，提示撤回/取消/纠正原因 |
| 过期 Order.version | 409「已被更新，请刷新后重试」，不推进服务/flow/事件 |
| 审核锁、终止中/已终止 | 拒绝服务写入，保留事实与输入 |
| 已完结服务变更 | 先显式重开，不能静默重置 closure |
| 不同公司/方向主单、外票附件或非法换单代理/人员 | 拒绝并完整回滚 |
| SI 缺真实审核单证快照或迁移旧快照无法验证 | 显式失败，不能填假单证或放宽 CHECK |
| 外部 HBL 缺真实签发方 | 明确阻止归档输出，不改成本公司自签 |

## 5. 正常、基准与错误示例

- 正常：只承接换单和提货，两项完成后可满足服务完结条件；清关/派送不适用。
- 正常：同一主单 A 票自提、B 票派送，完成 B 派送不改变 A；共享 ETA 仍统一投影。
- 基准：撤回已完成换单填写原因、保留前后事实；取消确认或保存失败保留服务输入。
- 错误：按共享到港时间自动完成五项；服务保存后把迟到 Order.version 借给整票旧稿；
  外部归档PDF套本公司自签抬头；为缺 businessType 的旧 mock 放行错误路由。

## 6. 必需验证

- 真实 PostgreSQL SI DIRECT/HOUSE 创建/修改、实际 MBL/航程/HBL/箱货、费用日期 ETA、
  审核/终止恢复/放货/完结；SE 相邻路径及方向/公司拒绝不回退。
- 五服务任意子集/并行、无承接、撤回与更正原因、闭环/重开、人员/供应商/附件归属、
  锁/终止/过期版本、同主单其他票隔离、审计故障完整回滚与事件追加。
- 正式迁移旧数据/快照完整保留、非法旧审核快照拒绝及 DDL/revision 回滚；
  生成幂等，无 Schema.Create，不在现开发库清数据。
- 前端 wire false省略/optional未设、严格身份、五步完整快照、取消确认、失败/迟到/在途输入、
  15秒与回焦/隐藏/离页、刷新取消/失败/成功、精确服务回调不覆盖整票草稿。
- 外部归档真实 PDF 全文字与多页实图、本地日期/金额、真实主体/无自签法律版本，
  原 RONCIN_SELF 四版相邻回归；生产build及同域资源路径另留实际证据。

## 7. 错误与正确做法

错误：`SI` 复用 SE 的出口进度按钮；以 `if(order.businessType)` 才检查路由身份；
服务完成直接换成后台更晚版本回填整票草稿。

正确：服务命令原子派生进口进度，路由与真实类型严格相等；服务保存只采纳本次明确
成功的 Order+1 与共享身份不变快照，保留其他输入，最终服务端仍逐次锁内复验。
