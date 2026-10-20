# 空运出口共享主单与重量契约

## 1. 适用范围

AE + export 的建票、共享 MAWB、HAWB、操作流程与计费重。公共订单继续持有客户、
经营公司、人员、货物、费用、锁和财务状态；空运单证不得写入海运实体或引用海运快照外键。

入口：`biz/air_export.go`、`biz/air_weight.go`、`data/air_order_sync.go`；
Proto 真相源为 `api/order/v1/{order,air_export,air_weight,order_fee}.proto`。

## 2. 签名与存储

| 入口 | 契约 |
| --- | --- |
| `POST /api/v1/orders` | `businessType=AE`，`airExport{masterWaybillId?,expectedMasterVersion,master,houseNo,houseContent,houseStatus}` |
| `GET /api/v1/air-export/master-waybills` | `keyword,page,pageSize`；真实主单候选，不能按字符串隐式拼单 |
| `GET /api/v1/air-export/master-waybills/{id}` | 共同资料、活动成员分页、完整集合汇总、历史分页和阻断原因 |
| `GET .../{id}/link-candidates` | 已终止且原属本主单的恢复候选，服务端关键字过滤及分页 |
| `POST .../{id}/preview` / `PUT .../{id}` | `expectedVersion,input`；执行另带预览 `inputFingerprint` 和 `reason` |
| `POST .../{id}/members` | 显式恢复原主单已终止票；订单、主单版本及原 HAWB 均需匹配 |
| `GET /api/v1/air-export/orders/{order_id}/house-versions` | 本票分单不可变版本历史，校验本票读取范围及分页 |
| `POST /api/v1/air-weights/preview` | `scope=ORDER/MASTER,targetId,source=PLANNED/ACTUAL,volumetricDivisor,bubbleShareNumerator,bubbleShareDenominator` |
| `POST /api/v1/air-weights/confirm` | 同预览参数，另带 `inputFingerprint,expectedVersion,confirmedWeightKg,reason` |
| `GET /api/v1/air-weights/history` | 对象和分页；不可变确认历史包含货量来源和规则快照 |

Schema：`AirMasterWaybill`、`AirMasterWaybillOrderLink`、`AirHouseWaybill`、
`AirHouseWaybillVersion`、`AirMasterWaybillHistory`、`AirWeightConfirmation`。
`OrderLockRecord.air_document_snapshot` 单独保存空运锁单时的单证事实。
正式迁移：`20261021100000_air_export_waybills.sql`、`20261021110000_air_weight_confirmation.sql`。

## 3. 请求与响应不变量

### 身份、共享事实和生命周期

- 草稿与普通创建均要求有效 MAWB/HAWB。MAWB 去空格和连字符后保留 11 位数字：
  三位航司前缀、七位序号、序号 mod7 校验位；同公司号码唯一。
- 预计起降时间沿用公共订单必填与先后顺序校验，AE 从权威主单取得；
  航班号及后续单证内容可后补，在对应操作节点复验。
- HAWB 去首尾空白并统一大写；同主单内永久占号，跨主单可重复；一票一分。
  每票最多一个 ACTIVE Link，终止结束 Link，恢复创建新 Link，不复活历史行。
- 新建主单或明确选中候选是两条显式意图；同号冲突返回 409，不自动合并。
  选已有主单必须携带其 ID、版本和权威共同资料。
- 航司、机场、航班、起降时刻和截止时间由主单统一维护；本票私有保存不覆盖共享事实。
  本票客户、人员、货物、HAWB 内容和费用独立。普通保存不能改配到另一主单。
- 六个操作节点沿用公共状态枚举，AE 单独验证航班、货物、操作记录与签发资料。
  提货/拖车可跳过；超时须填处理原因，资料缺失仍硬阻断。签单、实际起飞/到达、
  财务结案彼此独立；运输时刻不自动推进成员状态。
- 同公司有效权限决定可读写范围。共享修改预览与提交均针对完整活动集合检查；
  锁定、终止中或结案成员阻断共享修改，本票私有修改只检查本票。
- 多票写入固定锁序：预读成员 → 订单主键排序 FOR UPDATE → 主单 → Link → HAWB/确认。
  锁内重验集合、公司、状态及版本，不能锁完主单再追加订单锁。
- 成员汇总由服务端完整集合计算，包装单位分组；主单申报量独立展示差额，
  不强行覆盖成员数量，也不能加总当前一页冒充总计。毛重/体积摘要及差值展示沿用
  货量表单的三位小数精度后去尾零，避免 double 求和尾数；展示处理不改重量确认计算。

### 重量与费用

- 内部输入明确为 kg 和 m³。PLANNED 取订单委托货量，ACTUAL 取本票货物明细，
  两种来源独立确认、独立版本；不混加，也不自动回退。
- 体积重 `V×1,000,000/D`；泡差 `max(体积重−G,0)`；分泡重量 `泡差×分子/分母`；
  参考计费重 `G+分泡重量`。默认 D=6000、比例 1；比例范围 [0,1]，分母必须为正。
- 中间计算使用有理数，最终参考值一次 HALF_UP 至 0.001 kg。无有效体积不给伪造建议；
  人工覆盖或缺参考时必须填依据。界面只开放 KG，不能从场景标签反推重量单位。
- 主单使用完整活动成员的相应来源独立计算；`max` 不可加，分单计费重之和不是主单结果。
- 确认锁内重读输入并复算，绑定来源指纹和当前确认版本。输入/成员/规则变化保留旧确认，
  读时标为过期；本票无关私有字段变化不应使本票重量确认失效。
- KG 默认仍是毛重；费用显式选择 `quantityBasis=CONFIRMED_CHARGEABLE_WEIGHT` 才取本票确认。
  不提供将整主单重量自动带给每票的入口。现有 RT 仍为 `max(CBM,KG/1000)`。
- 数量采用 `quantitySource`：DIRECT 绑定当前确认 ID/版本并要求数值相等；
  MANUAL_ADJUSTED 绑定参考确认并记录实际数量及调整依据；MANUAL 表达纯手工。
  服务端在订单锁内保存权威来源快照，客户端不能伪造采用值/单位。
- 旧费用不随重量更新；复制、补录和更正不得继承“当前 DIRECT”标记，必须重新选源或明确人工。
  审核冻结、建账、核销与提成继续使用公共财务门禁。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| MAWB/HAWB 缺失、校验位错误、机场/航司引用非法 | 400，建票事务不留下半票或部分关联 |
| 主单号码冲突、同主单 HAWB 重复 | 409，数据库唯一约束兜底 |
| 订单、主单、关系、预览指纹或重量确认版本过期 | 409，要求刷新重试 |
| 跨公司或无 AE 对应权限 | 按权限/不存在契约拒绝，不返回其他公司成员及汇总 |
| 任一活动成员锁定/终止中/结案 | 共享修改被阻断；无共享变更的本票编辑不误拦 |
| D 非正、比例越界/分母非正、数量精度或上限非法 | 400，不执行自由表达式、不使用负数哨兵 |
| 无确认或确认过期 | 费用预览 unavailable，说明原因，不改取毛重 |
| DIRECT 数量不等于当前确认、来源不属于本票 | 保存失败，不能伪造直接取数 |
| 缺参考/人工调整但未填依据 | 400，要求填写业务依据 |
| 审计/版本/关联任一步写入失败 | 整个事务回滚 |

## 5. 正常、基准与错误案例

- 正常：两票 A=100kg/1.2m³、B=300kg/0.6m³，D=6000/P=1；本票参考 200/300，
  主单参考 400，分单合计 500。两个数字分别保留。
- 正常：G=100kg、V=0.96m³、P=1/3 得到泡差 60、分泡 20、参考 120；
  1m³/D6000 的最终参考为 166.667。
- 基准：确认 130 后费用 DIRECT 带入 130，改成 135 必须转 MANUAL_ADJUSTED 并填依据；
  应付可独立手工填 120。后续货量变化不改这三笔已保存事实。
- 错误：仅凭相同号码拼单；用未锁票改另一锁票共享航班；缺确认时退回毛重；
  在每票重复带入主单 400kg；用 `0.333` 截断三分之一后计算。

## 6. 必需验证

- PostgreSQL 正式迁移冷启动、唯一性/活动关联、并发同号/版本、成员集合变化、锁单、
  终止恢复、事务回滚、公司隔离，集成用例不得 SKIP 或调用 Schema.Create。
- 重量算例、输入单位和最终舍入、PLANNED/ACTUAL 独立、确认历史/过期、费用保存并发来源复核。
- 浏览器：建票刷新、候选显式关联、共享修改预览、从主单继续建票、重量确认及费用带入、
  组织切换与迟到响应隔离。公共 SE 创建/单证/流程/KG/RT 定向回归。
- AE 应收/应付审核建账、发票、收付核销、提成的真实链路；通知只用受控预览，
  不向真实用户发送测试消息。

## 7. 错误与正确做法

错误：普通整单更新携带一份旧航班副本，写回 A 时顺便覆盖共享主单，绕过 B 的锁与权限；
或先锁主单，再按界面成员顺序追加订单锁。

正确：本票私有字段走公共订单版本门禁；共同资料通过主单 Preview → Update，
服务端先按主键锁完整活动订单集合，再锁主单并重验成员、状态、版本与指纹，
版本历史和审计与变更同事务提交。
