# 订单列表字段与排序契约

## 1. 适用范围
修改 `ListOrders`、订单列表数据映射、海运可选列或列偏好时遵循。数据库模型不因增加展示列自动扩张；已有业务数据经轻量批量投影提供。

## 2. 签名
- `GET /api/v1/orders`：`sortBy` 为 `orderNo / createdAt / etd / eta`；`sortOrder` 为 `ascend / descend`。
- `Order` 列表摘要：`containerNos: string[]`、`attachmentCount?: number`、`customerServiceName?: string`。
- 页面列偏好：按订单品类、用户、组织隔离，使用既有 `useColumnSettings`。

## 3. 数据与展示约束
- Ent SQL Builder 的表别名和列名分别引用：`Ident(alias).WriteByte('.').Ident(field)`；`Ident("alias.field")` 可能生成一个含点的列名，单纯Go编译不会发现，新增排序必须在真实PostgreSQL执行。
- 排序在数据库分页前执行，未指定时创建时间倒序，使用稳定主键次级顺序。没有服务端支持的列不提供排序按钮。
- 箱号来自本票独占箱与本票共享箱分配，不得把同主单其他票的箱号一起显示；合并去重。
- 附件只统计有效引用，不返回完整附件内容；未授权不提供数量，授权无附件为明确零。前端不能把未授权/未设置转换为零。
- 客服姓名来自订单人员分配，不能用客户档案默认责任人替代已保存分工。
- 所有摘要仅对已授权本页订单批量查询；子资源读取资格继续裁剪。禁止逐行请求详情或复制全量明细到响应。
- 本票最终目的港与共同运输段卸货港是两个字段；无独立交付地址来源时不显示名为“目的地”的空列或用港口冒充。
- 三类备注分别映射 `allocationNotes → spaceNotes`、`bookingNotes → bookingNotes`、`operationNotes → operationNotes`，不可混用总备注。
- 海运专属字段放页面领域扩展，公共模板只消费稳定行模型。时间用 `formatDate`，金额用 `formatAmount`，数量用 `trimDecimal`。
- 订单列表提成列与专用聚合契约退役，不恢复摘要查询；财务与工作台计提业务保留。

## 4. 校验与错误矩阵
| 输入/状态 | 结果 |
| --- | --- |
| 未指定排序 | 创建时间倒序，稳定次序 |
| 合法字段与升降序 | 全筛选结果排序后分页 |
| 非法排序字段或方向 | 参数错误，不静默降级 |
| 授权无附件 | attachmentCount=0（optional保持存在） |
| 无附件读取权限 | attachmentCount不设置 |
| 无可选备注/时间/货值 | 展示“-”，不伪造事实 |

## 5. 正常与错误示例
- 正常：一票目的港洛杉矶、共同卸货港长滩，各自显示；本票无共享箱分配时不出现其他票箱号。
- 基础：未保存任何列偏好时展示精简操作视图；低频列可搜索开启。
- 错误：点击ETD排序只排序当前页；附件未授权显示“0”；ATD取点击已开航的操作时间。

## 6. 必要验证
- 仓储：箱号去重与本票归属、有效附件引用、客服分配、不同组织隔离、分页前升降序、相同值稳定顺序。
- Service：排序参数错误、子资源权限裁剪、optional数量零与未设置。
- 前端：真实wire缺省/显式零、时间金额格式化、三类备注、海空运列与偏好隔离、默认列、提成列缺席。
- 生成：Proto、Go绑定、OpenAPI与前端客户端一起生成，无手改生成物。

## 7. 错误与正确做法
```ts
// 错误：把未知/未授权数量伪造为零。
const count = order.attachmentCount ?? 0;
// 正确：保留存在性，由列渲染区分“-”和明确0。
const count = order.attachmentCount;
```

## 航空本票确认重量摘要

### 1. 适用范围
AE/AI 列表与详情的 `AirOrderInfo`，只读展示两种货量来源的最新本票确认；不改变费用采用来源，无 Schema 变更。

### 2. 签名
`AirOrderInfo.weight_summaries: repeated AirOrderWeightSummary`，ProtoJSON 为 `weightSummaries`。
`AirOrderWeightSummary` 含 `source`、`confirmedWeightKg`、`confirmationStale`、`confirmedAt`。
实现：`internal/data/air_order_weight_summary.go`，经 `airOrderToAPI` 输出。

### 3. 合同
- `source` 只能为 PLANNED/ACTUAL，各自按 version 取最新 ORDER 确认；MASTER 不混入，两个来源不替代。
- 无确认不返回该来源；`"0"` 是明确确认值。`confirmationStale` 为非 optional bool，省略表示 false。
- 过期状态复用 `BuildAirWeightPreview` 及同一货量指纹构造，禁止另写列表算法。
- 仅对已授权订单按公司批量查询；新增摘要固定至多四次查询/公司，前端不逐行补取。
- 本次没有改造既有共享主单装饰查询，不将摘要批量化等同于整套订单聚合无逐主单查询。

### 4. 校验与错误
| 状态 | 结果 |
| --- | --- |
| 来源无确认 | 不返回摘要，界面显示未确认 |
| 仅 ACTUAL 货量变化 | ACTUAL 需重确认，PLANNED 不随之过期 |
| 确认值为字符串 0 | 展示 0，不能当缺失 |
| 未知 source / 缺少或非法重量 | 前端映射显式报错，不伪造未确认 |

### 5. 示例
正常：PLANNED=120，ACTUAL=130且过期，两行分别展示；基础：空数组展示未确认。
错误：选择两来源中数值较大者作为“当前采用值”，或把主单重量分配给本票。

### 6. 验证
`air_weight_integration_test.go` 的真实 PostgreSQL 场景覆盖双来源最新版本、独立过期、未确认票；
`air_order_weight_summary_test.go` 覆盖 wire 省略 bool 与明确零；
前端 `list-query.air-weight.test.ts`、`air-list-columns.test.tsx` 验证映射和展示。

### 7. 正误对照
错误：`confirmedWeightKg || '0'` 把未知确认改成零。
正确：无摘要展示未确认，有摘要校验字符串后通过 `trimDecimal` 展示；CSV 保留来源分列。
