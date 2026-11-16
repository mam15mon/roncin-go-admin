# 真实详情与运行接口契约采样

北京时间 2026-11-16。以下只记录字段、类型和规则，不包含原始业务号、内部ID、人员/客商/联系方式、金额或令牌。原始文件仅本机保存。

## 证据索引

| 观察 | REA 证据 | 原始文件 | SHA-256 |
| --- | --- | --- | --- |
| 海运出口真实详情 | `ev_651969b46afa6b664ec97ad1a11aad1ddd86962412349974d67da3e3e50bbccf` | `sea-export-detail-record.evidence.json` | `11bfdcb17212da0f5947decaa3472042988b630b5000bb605e0538e274955cb4` |
| 海运进口真实详情 | `ev_8c97bd76293d7514d11c4e9761a0b1e20d983109cb2bf025b119df73032eca95` | `sea-import-detail-record.evidence.json` | `0631f1ab24c03993af649ce1179aa573bceb160b1b691bee6c8393cfd2663758` |
| 空运出口真实详情 | `ev_bd9ee4c9bd5b1832c565afbdcec98b562634aa5c103af0d6f31cfcdf19724d71` | `air-export-detail-record.evidence.json` | `dd79ccba420dec49e5680f0b376ee700ba2610e56832b9a2c4cdd3a524be786e` |
| 空运进口真实详情 | `ev_69ddc82f6a71a11088d308b9d4d3514fb30a4c11eca9f547b7f765b7e4f819e7` | `air-import-detail-record.evidence.json` | `bcd4f4f01546204565c3150667b917a3934bbc6b2324cc1ece02549761d3516e` |
| 空运进口列表完整响应样本 | `ev_8fc4fe467632277c210e6ec8192bc899bc171bf4d4baf98ed1ab4780fdb2186f` | `ai-query-recorded-dto.evidence.json` | `532def4988ad69ea63b94210f4140fd77a7c97d6625ff8dcd3a50d12cc45163e` |
| 空运进口详情及字段/流程配置 | `ev_ea16dd32bcd879db76c32b738ef7c98b0dff979cad6d44c0d8cf5cdada98c326` | `ai-detail-runtime-dto.evidence.json` | `dd137d90a7c6aa59dd1f86ac7787fbad3c5774725801f23b121816994ff340c7` |
| 空运进口费用只读页 | `ev_a5b223118f66f804b79cc250d8456f32cfcec4f5def1f43850ba8a67eb17acb6` | `ai-fees-readonly.evidence.json` | `25b8cccbba187b8785985348979de5574b0f386ac568f725456fbb2a2b148d2d` |

所有记录总体完整性均提示 `missing_sections=[events]`。本报告只引用实际保留的 DOM、截图和具体网络请求/响应，不声称采全流量。列表早期响应曾因 64 KiB 的 body-size-limit 及 1 MiB 的 redaction allocation-limit 未保留；最终根据观察到的 233754 字节原始响应和 2493378 字节分配需求，改用 256 KiB body / 4 MiB allocation 取得完整单条响应。上述限制不等于服务器响应损坏。

## 有效详情与导航纠正

- 列表显示的订单编号实际对应 `outerOrderNo`，内部 `orderNo` 对应隐藏的简云订单编号；本轮 AI 列定义响应直接证实，不能拿外部号代替内部ID。
- 已按列表观察到的内部ID打开四类单据，并以截图中的已填业务号、委托单位等非空值核实真实加载；相关私密值不写入本文。
- 早期 `*-detail-direct.evidence.json` 使用展示号而只显示空详情标题，全部排除为无效详情证据；`*-detail.evidence.json` 对号码文本单击也只选行而未导航，不能当详情成功。
- 前端默认号码 span 阻止冒泡；本轮后来通过序号单元格触发行点击成功，再捕获真实详情 DTO。
- 新手引导遮罩会阻挡单证按钮。早期按钮超时不代表没有功能；已经在同一场景跳过引导再验证成功。

## 空运进口列表 DTO（一个账户、一个查询样本）

`POST /saas-service/order/index`：
- 请求字段：`bussType:string`、`labelName:array`、`orderNos:array`、`orderStatus:array`、`createTimeStart:string`、`createTimeEnd:string`、`pageIndex:number`、`sortAttributes:string`、`sortSequence:string`。当前 `bussType` 是中文业务名称“空运进口”。
- 根响应：`success:boolean`、`code:number`、`count:number`、`data:array`、`keys:array`、`pageSize:number`、`customListMate:object`、`customBody:array`。本次返回 40 行、117 项列定义，这只是采样，不是全库总量或平台上限。
- `customBody[].head` 包含 `rowField/rowName/enRowName/isShow/rowOrder/mustSelect/ifSequence/customField` 等。当前“订单编号”映射 outerOrderNo，“简云订单编号”映射 orderNo；企业列配置和业务行是两个维度。
- 行对象同时包含业务、运输、人员、财务进度等很多字段。它是列表投影，不能直接当主单或单证持久化 Schema。
- `POST /saas-service/order/getAbnormalList` 请求 `{data:内部ID数组}`，响应为每票 `abnormal/processedAbnormal/orderNo`；本次异常可为 null，不应把 null 误当空字符串或接口失败。

## 空运进口详情与配置 DTO

`GET /saas-service/order/detail`：
- 根响应 `{success,code,msg,data,additional}`。`data` 有 `orderNo/outerOrderNo/bussType`、委托/代理、服务项、人员、`orderStatus/orderLocking/version`、货值保险、危险品、`stowage`、`billLading`、委托件重尺、客商联系投影等。
- `stowage` 直接观察到 `portStart/portArrive/portUnloading/portTransit/midPort/destination`、ETD/ETA/ATD/ATA、`changeOrderTime`、主分单号、航次/航班及其他海运兼容投影。字段出现在空运响应不等于空运实际采用每项。
- `billLading` 直接观察到收发/一二通知人文本及模型、品名/唛头/HS、付款及运输条款、件重尺、体积重/泡差/分泡/计费量、`billingWeightUnit`、`version` 等。
- 本样本 `orderStatus` 是整数，`orderLocking` 是布尔，`version` 是整数；部分运输时刻是 null。存在 version 字段不能证明后端实现了乐观锁，空时刻也不能证明时区或日期格式。
- 顶层联系人字段是个人信息投影，报告只记录存在，不保留值；本系统继续按现有客商/企业资源边界，不复制一套联系人字符串。

`GET /saas-service/defined/field/orderDetail`：
- 返回字段设计数组，本次有 39 项，元素含 `field/fieldName/fieldGroup/inputType/maxLength/required/systemField/customField/toCustomer` 等。
- 本次该配置数组没有 required=true 项；不能据此认为委托单位或人员可省略，因为交付前端另外有显式校验。也不能把本公司配置推广为其他公司的固定规则。

`GET /saas-service/order/getStatus`：
- 返回流程节点数组，元素含 `key/strKey/value/order/select/dateKey/isDefault/orderStatusType/childNodeList/statusId/processId/responsiblePerson/head` 等。
- 本 AI 样本有四项：拖车已安排、报关已安排、已换单、报检已安排。它们的 select 均 false、order 均 0；不能将这次返回次序当强制线性状态机或所有进口通用配置。
- 操作节点与 `orderStatus` 的退关/完结、`orderLocking` 的审核锁是不同事实。参考站点支持配置，不自动成为本期需要建设通用流程引擎的依据。

## 空运进口费用联动（只读）

- 从真实详情点击费用录入，观察到 POST `saas-support/applications/flows/audit` 和 GET `saas-service/fee/detail`，未点击新增/保存/删除/核销等写动作。
- 费用资格请求字段 `{permissionCode,recordIds,verifyModule,bussTypeCode}`，本次对应 order_fee 和进口类型；布尔资格响应不证明最终保存权限和并发保证。
- 费用详情 `data={accruedFee,receivableFee,flowReceivableFeeCantEdit,flowAccruedFeeCantEdit,flowAllFeeCantEdit}`；本样本应付容器有费用行，应收容器为 null。null 不代表整个费用请求失败，不应一律要求两侧均是对象。
- 容器内 `feeDetails[]` 可观察到数量、单价、原币总额、折本币、费用汇率、账单汇率、税率/税额/未税额、结算单位、费用/核销状态、发票号、计账/开票/到期等日期、权限投影、分摊/协同等字段。
- 费用汇率、账单汇率、结算汇率是不同字段；不能按一个 exchangeRate 把整条费用覆盖。保存算法、金额舍入和跨单据原子性未测试。
- 同时观察费用字典、费率方案/有效汇率和利息读取。这只是当前 AI 单票样本，不宣称所有财务模块和每类业务全流程已验证。

## 仍未知

- 未提交订单、运单或费用；服务端必填/查重、错误语义、数据库唯一索引、并发冲突、锁与事务原子性未知。
- 没有修改节点、退关/完结、锁定/解锁；生命周期条件目前只有前端静态证据及现状读取。
- 没有生成真实 PDF、执行在线签发或收费/税务/报关动作；制作表单与最终纸面/输出完整性仍需本系统实现后验收。
- 只有当前公司/角色采样，不证明海管家跨公司隔离或最小权限模型。本系统权限和当前公司边界按现有规范独立实现。
