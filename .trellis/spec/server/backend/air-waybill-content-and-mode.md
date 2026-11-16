# 航空直单与结构化运单正文

## 1. 适用范围与入口

航空订单创建、普通保存、主单共享修改、历史、审核和前端编辑均遵循本合同。
当前交付开放 AE；AI 复用同一领域结构，须在进口任务完成真实权限与服务闭环后开放，
不能借 AE 权限放行。主单进出口身份见[主单方向身份](./order-master-business-identity.md)，
运输和重量算法见[空运共享主单与重量](./air-export-contract.md)。

源入口为 `api/order/v1/air_export.proto`、`biz/air_waybill_content.go`、
`biz/air_waybill_validation.go`、`data/air_order_sync.go`、`data/air_export.go`。
正文与运输、订单经营字段、经营费用分别保存，不能引用 `SeaBillContent`。

## 2. 接口与存储签名

`POST /api/v1/orders` 的 `airExport`：

```ts
type AirOrderInput = {
  documentMode: 'DIRECT' | 'HOUSE'; // 创建必填；更新如传必须与原模式一致
  masterWaybillId?: string;
  expectedMasterVersion?: string;
  master?: AirMasterWaybillInput; // 新主单资料；已有主单按 ID/版本取得权威资料
  house?: {
    houseNo: string;
    status?: 'DRAFT' | 'CONFIRMED' | 'RELEASED';
    content?: AirWaybillContent;
    expectedVersion?: string;
  };
  expectedLinkVersion?: string;
  changeReason?: string;
};
```

响应使用 `airExport{documentMode,master,house?,linkId,linkVersion,active}`；
`house` 自带 `id/houseNo/status/content/version/contentHash`，旧扁平的
`houseId/houseNo/houseStatus/houseContent/houseVersion` 已退役。
主单正文位于 `master.content.waybillContent`，其哈希在 `master.contentHash`。
主单候选额外返回 `documentMode/canAttach/attachBlockedReason`；
`canAttach` 是非 optional bool，成功响应中省略即 false，不能按票数猜资格。

`PUT /api/v1/air-export/master-waybills/{id}` 仍经 Preview → Update，提交
打开时的 `expectedVersion`、预览指纹与修改原因。主单历史、分单历史分别保存
正文和内容哈希。正文是纯 Go 值与 JSONB；Ent 类型仅限 data 层。

`OrderLockStateData.airDocumentFingerprint` 与 `LockOrderRequest.airDocumentFingerprint`
绑定审核弹窗打开时的航空模式、主单、可选分单及 Link 身份与版本。
主单正文改变可以不改变 Order.version，因此不能只用订单版本和费用指纹审核。

正式迁移 `20261116140000_air_waybill_content_and_mode.sql` 为 Link 增加无默认的
DIRECT/HOUSE 约束，将旧分单正文与不可变版本全文迁入 `remarks` 并同步内容哈希。
迁移先检查关系、公司、方向与历史归属，缺少真实分单的旧关系显式拒绝，不能猜成直单。
不双读旧字符串、不删除旧正文、不重置数据库。

## 3. 请求、数值与并发不变量

### 模式与存在性

- DIRECT 无 HAWB 实体和版本；夹带 `house` 消息直接拒绝。固定 DIRECT 主单只能有
  一张当前操作票，当前成员归零后允许同模式新挂一票；不能与任意历史 HOUSE 混挂。
- HOUSE 每票一份真实 HAWB，可多票共享主单。同主单号码永久占用，跨主单可重复。
  普通保存不能改模式、主单或已保存分单号，不能用清空分单绕过历史。
- 普通保存省略 `house` 保持分单；省略 `house.content` 保持正文；`content:{}`
  显式清空正文。非 optional 的空字符串、false、数值零与消息未设置语义不同。
- 新主单和已有主单关联是显式意图；已有主单取权威资料，不能用前端运输副本覆盖它。
  恢复只允许原主单的原票和原模式；DIRECT 恢复不要求分单号，HOUSE 恢复需原号码。

### 正文印字

- `AirWaybillContent` 覆盖收发货账号/地址、通知人、代理/IATA/账号、印发人、路线印字、
  航班日期、账务/处理/SCI、唛头/品名、币种/声明、运价行、其他费用、合计及签发备注。
  多组路线是印字，不能推进实际运输或建立多航段执行。
- 收发/通知/代理/印发文本、账务/处理/条款/备注及签发声明等大段文本各最多 5000 字；
  唛头和品名最多 10000 字；运价等级 20 字、商品项目号 100 字；其他普通印字 200 字。
  正文不 trim，前后空格、换行和长文完整保存。
- 金额、费率、重量、数量为十进制字符串：非负，整数部分最多 16 位，小数最多 8 位；
  件数为整数。运价行与其他费用各最多 200 行。前端不转 Number、不加千分位后提交。
- `carriageValue/customsValue/insuranceValue` 分别使用 `{kind:'AMOUNT',amount:'0'}`
  或本字段唯一的 `NVD/NCV/XXX` 文本令牌；文本不夹带数值，不能把未知当零。
- `issueDate/flightDate` 为严格 `YYYY-MM-DD` 纯日期，不转换时区；运输时刻仍为 RFC3339。
- 运价行金额显式录入，服务端不按重量×费率自动报价；重量运费按行金额求和。
  `AS ARRANGED` 或有未定金额行时，所属付款栏运费及总计保持未知，不写零。
  运价行或 AS ARRANGED 必须明确运单币种及重量付款 PPD/COLL。
- 其他费每行明确金额、币种、PPD/COLL、AGENT/CARRIER；当前只接受与运单相同的币种。
  `prepaid/collect` 的估价费与税额可录入；运费、代理/承运人其他费及总计由服务端 decimal
  汇总。不反写经营费用、不自动采用订单货值、保费或重量确认。
- 表单可选声明/汇总控件注册出的空子消息由所属领域
  `serializeAirWaybillContent` 省略；保留正文 `{}`、明确空字符串、false 和字符串 `'0'`。
  不在全局拦截器按字段名补值，不过滤非法的非空内容。

### 锁、版本与编辑基线

- 写入经统一事务；固定 Order 主键排序集合 → MAWB → Link → HAWB/重量记录，
  锁内复验完整集合、真实公司/业务类型、模式、状态、权限和预期版本。
- 分单正文/状态真改时 House.version 与 Link.version 各增一并追加不可变分单历史；
  普通整票保存 Order.version 增一。无分单变化不推进 House/Link。
- 普通整票保存如明确提交 `expectedMasterVersion > 0`，必须在 MAWB 锁内匹配当前版本，
  不能因省略主单正文而忽略此基线。未提交主单预期版本的纯本票更新不写主单，
  不扩为所有私有保存强制携带主单版本。
- 主单正文真改推进 Master.version、主单历史和哈希；正文不联动 Order.version。
  权威 ETD/ETA 等运输资料继续依现有日期及汇率同步逻辑处理。
- MASTER 重量成员指纹使用不可变 Link ID 与模式表达成员关系；分单正文改动不应仅因
  Link.version 推进就使重量过期。成员、货量、实际来源与规则变化仍按既有算法过期。
- 航空审核在上述锁内比对打开时指纹，快照明确记录模式、MAWB、可选 HAWB、Link 版本。
  前端发现当前指纹与弹窗快照不同则禁止确认，不临提交另取新指纹借给旧审核。
- 订单及主单编辑前台每 15 秒与回焦检查变动，隐藏/非当前页面暂停。探测只提示并保留输入，
  不替换提交基线；刷新覆盖脏输入先确认，取消和失败保持原稿。
- 整票保存仅接受本次确切成功快照：Order 恰增一，分单真改时 House/Link 恰增一，
  主单身份版本等于提交基线。响应含更晚变动时保留旧稿并提示刷新，不借后续版本。

## 4. 校验与错误矩阵

| 场景 | 行为 |
| --- | --- |
| 创建未给合法模式、HOUSE 缺真实分单、DIRECT 夹带分单 | 400，事务不留半票 |
| 直单第二当前票、历史混模式、并发关联改变集合 | 409，保持既有成员 |
| 订单/主单/分单/Link/审核指纹过期 | 409「已被更新，请刷新后重试」，保留输入 |
| 分单重复号或永久占用号码 | 409，数据库唯一约束兜底 |
| 普通修改模式、主单或已保存分单号 | 拒绝，不偷偷改配或新建分单 |
| 非法声明令牌、负数/超精度、非法日期、异币种其他费 | 400，不自动纠正 |
| 已签发分单正文/状态变更无原因 | 拒绝；首次签发及仅改本票经营字段不误拦 |
| 任一共享成员锁定/无权/终止或集合不完整 | 共享修改拒绝，本票无关修改沿用既有门禁 |
| 审计或历史写入失败 | 整个事务回滚，当前正文/版本/哈希一致 |

## 5. 正常、基准与错误示例

- 正常：DIRECT 创建只有主单正文；终止结束 Link，原票恢复仍无分单并保留历史。
- 正常：HOUSE 两票共享 MAWB，各自正文与签发状态独立；主单改正文不改分单。
- 基准：`0.1+0.2` 行金额汇总为十进制 `0.3`；声明数值 `'0'` 不变 NVD；
  `issueDate:'2026-11-16'` 不变 UTC 时刻。
- 错误：打开“其他费用”空分节就生成有币种要求的假费用；将未设置消息当 `{}` 清空；
  旧审核弹窗用最新主单指纹完成审核；直单为满足旧读取逻辑虚构一个 HAWB。

## 6. 必需验证与断言点

- PostgreSQL 按正式迁移冷启动及旧状态升级：旧正文/历史逐字相等、哈希与 Go 一致，
  关系异常迁移失败且 DDL/迁移记录回滚；无运行期 Schema.Create。
- DIRECT/HOUSE 创建、并发第二票/混挂、终止/恢复、唯一号永久占用、跨公司/方向、
  完整成员锁序、失败回滚；真实 DIRECT 的 ORDER/MASTER 重量及费用来源分别验证。
- 主分独立版本/哈希/历史、省略保持与 `{}` 清空、锁冲突、过期 House/Link、
  主单正文变更后旧审核指纹拒绝及新快照正确；AS ARRANGED/声明零值/纯日期/长文。
- 前端真实 wire 省略默认值/显式零值/optional 未设，动态 Form.List 路径、小数字符串与
  换行不丢；模式选择/候选 false 省略/只读，以及在途输入/后续版本/A→B→A 隔离。
- 主单前台 15 秒/回焦/隐藏/离页、探测失败持续提醒、刷新取消/失败/成功和刷新在途编辑；
  保留真实 React Query、表单和离开守卫，不把 mock 测试称为真实浏览器或事务验收。
- 精确已执行命令与缺口见本组任务 `research/`；PDF 制作与输出在后续任务验收，
  本合同不把结构字段存在视作 PDF 可用。

## 7. 错误与正确做法

错误：普通订单更新把整份主单副本写回；主单正文更新后旧审核只检查 Order.version；
保存后重读任何版本都采纳为旧稿提交基线。

正确：本票、主单正文分别通过各自命令写入；航空审核提交打开时单证指纹并锁内复验；
保存后只采纳属于本次提交的确切版本，其他变动持续提醒并保留输入。
