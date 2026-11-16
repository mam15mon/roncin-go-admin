# 空运进口业务闭环

## 1. 适用范围

AI+IMPORT复用航空主分运单、真实货物、重量、公共费用和按票进口服务。
本规范约束AI实施与验收，不以枚举或页面注册代表业务已通过；具体运行结果见当前任务research。
AE继续出口流程，SE/SI继续海运实体；本期不增加真正多航段、复杂空运拆拼改配或独立进口费用算法。

前置合同：[航空正文与模式](./air-waybill-content-and-mode.md)、
[航空运输与重量](./air-export-contract.md)、[主单方向身份](./order-master-business-identity.md)、
[进口五服务](./order-import-handling.md)。

## 2. 签名与真相源

- 建票/普通更新沿用 `OrderService` 的 `airExport: AirOrderInput` 字段，
  创建显式 `businessType=AI, tradeDirection=IMPORT, documentMode=DIRECT|HOUSE`。
- 共享航空HTTP接口继续 `/api/v1/air-export/master-waybills` 及其ID子资源；
  不因路径含export另建一份进口模型。候选 `businessType=AE|AI` 必填，缺失/其他类型拒绝。
- 主单预览/执行、成员恢复、分单历史沿用现有航空RPC及版本/指纹字段。
  ID目标的权限使用目标自身不可变身份，不能由请求中另一类型代替。
- 重量沿用 `/api/v1/air-weights/preview|confirm|history`；
  `scope=ORDER|MASTER, targetId, source=PLANNED|ACTUAL`，既有规则和确认字段不变。
- 进口服务沿用 `/api/v1/orders/{order_id}/import-handling` 和events，
  `Order.version`为唯一并发基线；普通整票保存与服务保存不互相清空。
- 权威字段仍为 `AirMasterWaybill.business_type`、运输输入及独立正文，
  `AirMasterWaybillOrderLink.document_mode`、可选真实HAWB及不可变版本；不建镜像事实。

## 3. 请求、权限与读写合同

### 类型与当前公司

- AI只能IMPORT，AE只能EXPORT；普通更新不改既有业务类型、方向、主单和模式。
- 候选按显式类型匹配对应read；按ID的共享读取/写入、恢复、分单历史及重量分别解析
  当前公司内目标Master或Order真实AE/AI，复用Manifest对应read/update/transition。
  HTTP生成路由、Authorization、service与仓储需一致；AI权限不能借AE，反之亦然。
- ORDER重量按Order类型，MASTER重量按Master类型；不能先用AE过滤成员而静默漏掉AI。
  目标及完整成员公司/类型异常应显式失败，不缩小统计集合冒充成功。
- DIRECT真实无HAWB且当前单成员，不能混历史HOUSE；HOUSE各票独立HAWB。
  恢复创建原主单/原模式的新Link，历史不复活，号码永久占用规则不变。

### 权威运输、费用与重量

- 列表、地点/日期筛选、详情及经营报表从最近合法航空关联取得同一权威主单机场和时刻；
  终止后仍读最后关联。缺失关系不回退Order旧地点/日期伪造权威事实。
- AI经营日期取权威ETA并换算 `ExchangeRateBusinessLocation` 中国自然日；
  AE仍取ETD。共享改期在原事务中同步成员日期和可更新系统汇率费用；手工汇率及受保护
  财务事实的费用保持原值。已建账但无保护事实的费用沿既有规则更新，账单行汇率和折本币
  快照不变。缺适用汇率或成员锁等失败保持主单、订单、费用与历史全部原样。
- PLANNED和ACTUAL、ORDER和MASTER各自独立确认；主单完整成员汇总，
  本票费用只采本票确认。货量/成员变化使确认过期但保留旧值和历史，不追改旧费用。
- 手工费用及确认数量来源仍按现有十进制、精度、来源ID/版本和最终锁内复验，
  不把运单印字费用当经营费用，也不将主单重量给每张票重复带入。

### 服务、审核与页面基线

- AI操作摘要只由固定五服务原子派生，任意承接子集可并行，未承接为不适用。
  不套用出口逐步flow，运输实际到达不自动完成本票服务。完结/锁/终止独立。
- 审核固化航空模式、MAWB、可选HAWB及Link版本/哈希指纹，不写海运快照外键。
  主单更新未必推进Order.version，不能只比较订单版本忽略旧审核快照。
- 前端共用航空字段/adapter/主单页及进口服务Panel；集中路由使用
  `/orders/air-import` 与 `/orders/air-import/masters/:masterId`，queryKey携类型和当前公司。
- 每次成功读取严格验证ID、公司和业务类型；缺失/零/错类型不为旧mock放行。
  前台15秒及回焦检查，隐藏/非当前页暂停；探测仅持续提醒，保留输入和提交基线。
  主单探测缓存须同时区分hook实例身份和编辑会话，离开后重进同一公司/type/id
  不采纳旧实例迟到响应或沿用旧探测结果，A→B→A同样隔离。
- 共享主单编辑和恢复关联弹窗均固定打开时版本及编辑serial；提交后await重读期间若打开
  新会话，旧ModalForm完成不能关闭/重置新稿。恢复不能用后台最新query的主单版本替换
  打开基线，卸载或换组织/目标后的迟到响应不触发新会话成功副作用。
- 服务保存只在Order恰增一、表单仍为原版本、重读恰为本次保存版本、
  航空Master/Link/可选House身份和版本全部未变时推进主表版本；不reset主表、不借后续版本。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| AI/EXPORT、AE/IMPORT、缺合法模式或DIRECT夹HAWB | 参数错误，不留半票 |
| 候选未传AE/AI或只持另一方向read | 拒绝，不能默认AE或借AE权限 |
| 跨公司/方向Master、Order、重量目标或恢复关系 | 不可见或业务拒绝，无部分变更 |
| Master完整活动成员公司/类型异常或集合并发变化 | 显式冲突，不漏统计、不后置追加Order锁 |
| 版本或审核/重量指纹过期 | HTTP409，中文提示刷新，保留事实与页面输入 |
| 共享改期缺可更新费用的适用汇率或成员被锁 | 原子回滚主单/成员/费用/历史 |
| 进口服务已有完成事实变更但缺原因，或CLOSED未重开 | 明确拒绝，不自动清空旧完成事实 |
| 保存响应包含更晚Order/Master/Link/House版本 | 前端不借新基线，保留草稿并持续提示刷新 |

## 5. 正常、基准与错误场景

- 正常：只有AI业务权限的公司成员创建DIRECT，无HAWB仍可本票/主单分别确认重量；
  费用显式选择本票确认来源，另一张AE同号主单保持独立。
- 基准：ETA从UTC15:30改到16:30时，中国经营日期跨午夜；列表、详情、费用汇率日一致，
  ETD不作为进口日期回退。修改缺周汇率时所有相关事实回滚。
- 错误：仓储已允许AI，但HTTP注解仍固定AE；主表有未保存输入，服务保存后采纳后台
  更晚Master版本，导致旧草稿覆盖他人修改。

## 6. 必需验证

- 真实PostgreSQL的AI DIRECT/HOUSE创建/私有修改/共享主单/成员完整统计、
  终止恢复、真实审核与锁、五服务/完结、跨公司方向、并发版本及失败回滚；不使用Schema.Create，
  数据库SKIP不计通过。沿用有序正式SQL和随机隔离Schema，不以开发库手工状态作基准。
- AI ORDER/MASTER及PLANNED/ACTUAL独立确认、过期/历史/费用来源，AE相邻算法不变；
  权威机场筛选、最近终止关系、ETA中国午夜/周汇率、缺汇率回滚及受保护费用/账单快照不变。
- HTTP使用真实生成路由、Authorization、Service/Usecase验证仅AI读/写、AE借权拒绝、
  跨公司、ORDER/MASTER目标；仓储stub的权限测试与真实PG结果分别记录，不互相冒称。
- 前端真实wire省略默认值/optional未设、双模式/清空/候选类型、只读权限、服务确切基线及
  主表输入保护；既有主单轮询/刷新/组织代际及AE/SI相邻。
- 真实浏览器至少一模式从新建UI提交、另一模式受控API建票，验证仅AI账号、主单/重量/
  服务与外部变动输入保护。航空实际PDF和四类完整财务/CSV由后续依赖组验收，本组不代替。

## 7. 错误与正确

错误：`business.order.ae.read`固定写在所有航空RPC，或按AE查询成员后返回AI主单汇总。

正确：候选显式选择AE/AI；ID目标由当前公司内实体的不可变business_type确定授权，
仓储锁内复验完整成员及版本，前端同类型路由和缓存消费真实响应。
