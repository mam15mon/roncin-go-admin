# 海管家交付前端静态规则与接口候选

## 证据与方法

本轮只读静态分析，没有执行下载的 JavaScript，没有操作浏览器、登录或请求业务 API。来源限定为成功 DOM 捕获实际引用的 `filecdn.hgj.com/packages/saas-web/1794456941597/js/` 资产，以及主会话捕获的已加载脚本。

- 当前脚本捕获：`loaded-sources-expanded.evidence.json`，REA `ev_b61f804124e6fdf136f4d117531759d06d27de3f7695e309f79c52a1b502cee7`。有 107 项脚本，93 项可提取 CDN 源码；跨页缓存脚本不证明当前路由执行了所有模块。
- AE 单源 REA 惰性语法分析：`ev_2e971c1d855647e861e51da367b27cd42acb101b537eea2ba2e3ca31ec2160a6`，读取 71,136 字节、1 个 JavaScript，19,023 个 AST 节点、0 解析失败。
- AI/SE/SI、重量共享模块和 app 五源 REA 分析：`ev_6c1f88ae4535a4a4d13908c374bdbd2aa36dd87fd756327d7317ad57f9c7ac2d`，读取 558,827 字节、5 个 JavaScript，131,278 个 AST 节点、0 解析失败。
- 两次 REA 语义图均为 **partial/truncated**，节点上限分别为每文件 10,000、总量 10,000/50,000；Webpack 动态导出和未选择来源的关系仍可能未知。不以静态图证明后端状态机、DTO 完整性或运行结果。
- 下文规则是对原始交付代码的人工逐段复核；原始源码、完整 REA JSON 与逐段 SHA-256 记录只保存在本机 `/home/vince/.local/share/rea/hgj/static-analysis/`，不提交客户业务数据。
- 范围编号对应 `rule-source-ranges.json`，偏移为**零基 UTF-8 字节半开区间 `[起点,终点)`**，不是 JavaScript UTF-16 字符偏移。可以用文件 SHA-256 和范围 SHA-256 复核。

## 可确认的前端规则

以下“确认”只表示在指定脚本中直接存在；服务端是否独立执行同样规则未验证。

### 必填、默认与保存载荷

| 规则 | 代码观察 | 范围编号 |
| --- | --- | --- |
| 委托单位 | 四类 `saveInfo` 先检查 `delegateCustomerName`；为空时提示并返回 | AE-save、AI-save、SE-save、SI-save |
| 其他必填 | 公共验证器读取 `design.order_buss_info / order_stowage / order_bill_lading / order_internal_info` 的 `required`；不是所有字段固定必填 | APP-required |
| 合并件重尺必填 | `systemField=1 && required` 时按 `!unitNumber || !numberUnit || !volume || !grossWeight` 判断，零会被判缺失；与普通空值函数把 0/false 当存在不同 | APP-required |
| 自动客户/内部编号 | 新建且编号配置为自动生成时，将对应两个设计项的 required 改为 false | APP-required |
| 人员 | 保存前另外调用 `PersonRef.verify()`，不能只看前述通用 required 校验宣称人员规则完整 | 四类 save |
| 传输形状 | 四类都把整个 `base` 对象传给共用 `orderSave`，其下有 `stowage`、`billLading`、`totalNumberGrossVolume`、人员等；自定义字段另构建 `customFieldOrderInfo.fieldInfos[{fieldGroup,info}]` | 四类 base/save、APP-custom-field |
| 标准化 | 保存前对 stowage 的字符串统一 `toUpperCase()`；对 billLading 顶层字符串除 `billRemark` 外大写。嵌套对象不在该循环中递归大写 | 四类 save |
| 初始服务项 | SE/AE 初始化为 `['订舱']`，SI/AI 初始化为 `['换单']` | 四类 base |
| 日期锚点 | 出口初始 stowage 声明 `estimatedTimeDeparture`，进口声明 `estimatedTimeArrival`；这不是证明对侧日期不支持，复制代码还处理 ETD/ETA/ATD/ATA | 四类 base、AE-time-copy |
| 通用默认 | 初始贸易条款 CIF、付款 FREIGHT PREPAID、运输条款 CY - CY、普货 DRY；但 `user.userCustomer` 条件可覆盖 FOB、FREIGHT COLLECT、SEAWAY BILL，因此不是所有用户固定默认 | 四类 base、PERSONAL-defaults |
| 文本变更校验 | 收发通知、地址、唛头、品名、HS 等配置 `checkFullWidthChar` 的 change 校验；其包装器对 chGoodsName 另传开关。未完整复核底层字符范围，不能宣称“禁止所有中文” | AE-fields-change-rules、APP-width-char |
| 存在订单前的入口 | 费用、箱货、单证制作等多数入口由 `base.orderNo && !isCopy` 及权限控制；新增页 DOM 出现隐藏入口不意味着首存前可用 | AE-guards、AE-document-entry |

四类服务端创建/修改是否分支、必填是否与前端一致、是否限制号码重复、事务和唯一约束均未通过请求实测验证。本系统现有强约束不会因为参考站点的动态必填而自动放宽。

### 空运航班字段

- `FlightNos` 初始一个 `{num:''}`，输入 trim、maxlength=10，只有当前项数小于 6 时显示添加按钮；删除按钮只在大于 1 项时出现。
- 点击添加前要求末项非空，输入会移除英文逗号；输出过滤空项，再以 `,` 拼接。已有 value 按逗号拆分，静态代码未看到载入时强制截到六项。
- 页面绑定 `base.stowage.voyage`，并非多个独立航段对象；本组件没有“每段独立机场、航司、ETD/ETA”。最多六个**可新增航班号**不能等同完整多航段运输。
- 参考范围：AE-flight-render、AE-flight-methods、AE-flight-model。AI bundle 同样包含 FlightNos，但本报告范围表只保留 AE 的同组件实现。

### 订单体积重、泡差与计费参考

- 字段映射：空运 `volumetricWeightRatio`（除数）、`volumetricWeight`（人工采用体积重）、`largerCargo`（泡差）、`bubbleProportion`（分泡比例）、`bubbleWeight`（分泡重）、`billingWeight`（采用计费量）、`billingWeightUnit`（KGS/CBM）。跨境/自拼共享组件采用另一组字段名，不能按名字混用。
- 除数候选：2000、10000/3（内部 -1）、5000、6000、7000；比例候选：1（“无分泡”）、0.1/0.2/0.3/0.4/0.5、1/3、2/3。
- 体积重参考 `decimalPlaces(actualVolume × 1,000,000 / divisor, 3)`；使用的是**实际**货量 `actualJmt`。点击参考值才写入采用体积重，不是自动把计算参考覆盖为正式值。
- 泡差 `max(adoptedVolumetricWeight − actualGrossWeight, 0)`；分泡重为泡差乘比例，分别经 3 位小数包装器。泡差为零时重置比例为字符串 `'1'`。
- 计费参考：实际毛重不小于采用体积重时取实际毛重，否则 `actualGrossWeight + (adoptedVolumetricWeight − actualGrossWeight) × ratio`。CBM 模式再乘除数/1,000,000；最后三位参考值。
- 采用体积重和采用计费量的输入 `min=0`、`max=999999999.999`、`step=0.001`；点击参考的带入也按相同上限截顶。参考为 0 时点击带入函数不写值。
- 源码使用 `eval` 求比例文本，**没有执行该代码**；本系统已有有理数/指纹/确认版本机制，应保留，不能照搬 eval 或前端浮点结果作为服务端依据。
- `decimalPlaces` 是两层 round 包装（先 10 位再目标位）；没有完整证明其底层舍入/浮点行为，不能直接标成与本系统 HALF_UP 相同。
- 参考范围：WEIGHT-fields、WEIGHT-formula、WEIGHT-controls、APP-round-wrapper。

### 订单状态、审核锁与列表导航

- 列表 `orderStatus` 筛选使用 0 未退关、2 已退关、4 已完结。批量完结前拒绝任何已锁行，并要求原状态为 0；取消完结要求 4。退关要求 0，取消退关要求 2。这里只观察前端条件，不证明后端原子性、并发防护和所有阻断。
- 批量锁/解锁及完结/退关先调用 `isPassAudit`；在 app 中该方法转到 `appFlowsAudit`，属于一个 POST 请求，其命名/布尔返回暗示资格校验，但不能从前端证明服务端没有日志等副作用。
- 列表普通行事件 `row-click`→OrderListTable.goEdit→列表 goEdit，目标 `/order/edit_<type>/<row.orderNo>`。不是 row-dblclick。`row.orderNo` 用于路由/请求，不能替换成用户看到的业务编号；显示号常称 `outerOrderNo`，当前列名到 rowField 的准确映射由服务端下发 `customBody.head.rowField` 决定，仅静态代码不足以证明当前列配置。
- **默认单元格 span 会 stopPropagation，只更新 contentClick 选中行**；点击文字不一定导航，需行/td 的可冒泡区域。不能把 click completed 当详情成功，必须检查实际 URL/页面。
- 参考范围：LIST-finish、LIST-shutout、LIST-lock、LIST-navigation、WTABLE-cell-click、WTABLE-row-click、ORDER-TABLE-row-click。

## 单证制作入口与航空运单结构

- 四类详情 `goDocumentMake` 没有直接调用 `saveInfo/orderSave`：先资格检查、拒绝委托单位未保存变化，再导航。空运到 DocumentMake，海运到 DocumentMakeSea；不代表目的页、服务端请求无副作用。
- 空运单证列表 created 读取路由；mounted 读取本地过滤条件并 `getDocList`，后者请求 manifest/billMultiList。列表本轮源码未见初始化自动保存。其 editDoc：billType 0/1→AirLadingManage，2→AirManifest，3→AirMark，4→AirMailMark。
- AirLadingManage 的 billType 0 显示 MAWB、1 显示 HAWB。其 mounted→initFunc 读取名称候选、订单、已有运单详情、主分单候选、常用资料、服务单位和费率；唯一观测到的 otherChargesList watcher 是表格 reloadData，未见该初始化链自动保存。该判断限于已复核入口，不是对所有子组件副作用的绝对保证。
- 航空运单 `base` 结构含：航空公司代码及流水号、主分单号、SCI、参考/账务信息、发收货账号及地址、代理/IATA、始发/目的、第一/二/三承运及地点、航班/日期、处理与通知信息、币种、声明运输/海关价值、保险、PPD/COLL、费用明细、签发日期/地点/签名等。初始声明值 NVD/NCV/XXX、PPD；服务端如何校验未知。
- **同名陷阱**：订单 `billLading.billingWeight` 是采用计费量；航空运单 `chargesInfo.billingWeight` 被 `getWeightCharge()` 设为费率明细 total 的和，再与估价费/税/代理费/承运费相加，是金额字段。不能用字段名全局映射默认值或复用同一类型。
- 航空运单保存先调用 saveAirDetail，再更新名称，再逐条保存待存费率明细；创建时用返回的 ID 换路由。前端串联多个写调用不能证明服务端整体原子性，不应直接照搬到本系统。
- 制作页改名、费用明细编辑、删除、保存和导出按钮是独立动作；只读调研不可因为“只是打开表单”而触发这些动作。
- 参考范围：AE-document-entry、AI-document-entry、APP-audit、APP-audit-endpoint、DOC-list-init/read/route、AWB-defaults/init/detail-read/charge-total/save。

## 接口候选（静态，不代表本轮请求已经执行）

| 客户端动作 | 路径或形状 | 意义边界 |
| --- | --- | --- |
| orderDetail | `saas-service/order/detail` | 读取，调用方传 orderNo/isCropOrder |
| orderSave | `saas-service/order/save` | 四类共用；本轮未发起，不能确认服务器必填和完整 DTO |
| orderSetStatus | `saas-service/order/setStatus` | 自定义操作节点，不应与 0/2/4 生命周期混为一套 |
| orderShutOut | `saas-service/order/shutOut?orderNo=…&type=…&source=…` | 候选生命周期写入口，未调用 |
| batchFinish / batchShutOut | `saas-service/order/batchShutOut?type=…`，退关附 source=1 | 完结 type 4、取消 0；退关 type 2、取消 0；批量体为所选 ID 列表 |
| batchLock | `saas-service/order/batchLock?lock=…` | body 为所选 ID 列表，未调用 |
| designOrder | `saas-service/defined/field/orderDetail` | 页面字段设计候选 |
| customFieldOrderInfo | `saas-service/custom/field/orderInfo` | 读取附加字段 |
| order/status/child-list、order/order-status、order/order-status-child | 类型或订单附在路径 | 状态配置，不证明全站固定六步或固定流程 |
| appFlowsAudit | `saas-support/applications/flows/audit` | POST 资格校验候选，未知服务端副作用 |
| getDocumentList | `saas-service/manifest/billMultiList` | 单证制作列表，参数 orderId/isCropOrder |
| getAirDetail / getAirBillLadingName / getRateDescription | `saas-service/air/getDetail`、`…/getBillLadingName`、`…/getRateDescription` | 运单详情/名称/费率读取 |
| getMAWBByBill | `saas-service/air/mawb` | 主运单候选/详情读取，不能从名称推断模型关系 |
| saveAirDetail | `saas-service/lcl/air/order/billLading/saveDetail` | 普通空运编辑器也使用，不能单凭 lcl 路径断定仅自拼业务 |
| editAirBillName / editRateDescription | `saas-service/air/editBillName`、`…/editRateDescription` | 分开的后续写调用，本轮未执行 |

GET/POST 的判断来自已交付通用请求辅助函数的客户端调用约定；若调用辅助函数的行为发生变化需再跟踪，路径名称本身不是方法证明。上表没有发起任何探测请求。

## 原始来源指纹

以下 URL 均来自实际 DOM 引用或扩域脚本捕获，不由路径猜测构造。静态字节复核不依赖第三方相同标题的资料。

| 来源文件 | SHA-256 |
| --- | --- |
| [AirExport~AirImport~DomesticTrade~LCL~Land~Logistics~OrderBooking~Railway~SeaExport~SeaImport~21833f~1faa7780.098b5e5d.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/AirExport~AirImport~DomesticTrade~LCL~Land~Logistics~OrderBooking~Railway~SeaExport~SeaImport~21833f~1faa7780.098b5e5d.js) | `2e460ff76a50e219e1413527c41c6120cd345fe80fba33448c6530ac0e999ad8` |
| [AirLadingManage~31ecd969.09820d28.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/AirLadingManage~31ecd969.09820d28.js) | `adb120888070b9e79854044e355d383bddbb51ceef57119f2fc73854854ae132` |
| [CrossDetail~EditAirExport~EditAirImport~EditBooking~EditLand~EditLcl~EditRailway~31ecd969.48159171.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/CrossDetail~EditAirExport~EditAirImport~EditBooking~EditLand~EditLcl~EditRailway~31ecd969.48159171.js) | `823ecaa8405a5b23b4de8b849ac131da248858899a8a4e4043c2d2fe2b9fe6df` |
| [DocumentMake~21833f8f.d1979acc.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/DocumentMake~21833f8f.d1979acc.js) | `91e545904469cfb345669170aaf42977948db299fb01e257aa96473f2000f66c` |
| [EditAirExport~21833f8f.ceb04dc7.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/EditAirExport~21833f8f.ceb04dc7.js) | `d02ea9de5ee7444fd8d83e930fdeca30b616549f8159bb32a2c6df7b36b6f831` |
| [EditAirImport~21833f8f.bf3376f8.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/EditAirImport~21833f8f.bf3376f8.js) | `2eff52e3f9c497ff0ccb129b8c4d3130a8d77934189d838cd0b3b502e8f97cb7` |
| [EditSeaExport~31ecd969.af321e09.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/EditSeaExport~31ecd969.af321e09.js) | `41ea5cef166f6cd5b1b58fb52158d6b295ddcca3e4eecd2da36a2a43dbc6d502` |
| [EditSeaImport~31ecd969.d4bb6e32.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/EditSeaImport~31ecd969.d4bb6e32.js) | `3bfae46c1104e3f7488d92ddc00052982436ac06233c3155f4cfd916fb9a8d13` |
| [app~c714bc7b.aad8b8e0.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/app~c714bc7b.aad8b8e0.js) | `27934724a2acc3d827ba5d19e74ffe236d679325707071ca0e0a3c7843ce9e64` |
| [chunk-374f66c4.a111524b.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/chunk-374f66c4.a111524b.js) | `2d9361878ff1ca633bb7e91149b046528f5da93effdbad09257f1e8d61b47c19` |
| [chunk-748cfcfe.fd7e1e76.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/chunk-748cfcfe.fd7e1e76.js) | `405f0336b5c0b353c1dc14f3a4db938dd7686a5668cd93df798293c3d44875fd` |

## 字节范围索引

| 范围编号 | 文件 | UTF-8 字节区间 |
| --- | --- | --- |
| AE-base | `EditAirExport~21833f8f.ceb04dc7.js` | `[30238,31623)` |
| AE-save | `EditAirExport~21833f8f.ceb04dc7.js` | `[45380,47120)` |
| AI-base | `EditAirImport~21833f8f.bf3376f8.js` | `[38289,39672)` |
| AI-save | `EditAirImport~21833f8f.bf3376f8.js` | `[53377,55117)` |
| SE-base | `EditSeaExport~31ecd969.af321e09.js` | `[54752,56215)` |
| SE-save | `EditSeaExport~31ecd969.af321e09.js` | `[74155,76026)` |
| SI-base | `EditSeaImport~31ecd969.d4bb6e32.js` | `[37517,38905)` |
| SI-save | `EditSeaImport~31ecd969.d4bb6e32.js` | `[54153,56000)` |
| AE-flight-render | `EditAirExport~21833f8f.ceb04dc7.js` | `[3108,4099)` |
| AE-flight-methods | `EditAirExport~21833f8f.ceb04dc7.js` | `[4102,4981)` |
| AE-flight-model | `EditAirExport~21833f8f.ceb04dc7.js` | `[18167,18567)` |
| AE-fields-change-rules | `EditAirExport~21833f8f.ceb04dc7.js` | `[32026,32714)` |
| AE-guards | `EditAirExport~21833f8f.ceb04dc7.js` | `[5497,6713)` |
| AE-split | `EditAirExport~21833f8f.ceb04dc7.js` | `[8768,9300)` |
| AE-load-defaults | `EditAirExport~21833f8f.ceb04dc7.js` | `[37242,37602)` |
| PERSONAL-defaults | `EditSeaExport~31ecd969.af321e09.js` | `[9404,9970)` |
| AE-time-copy | `EditAirExport~21833f8f.ceb04dc7.js` | `[40528,41528)` |
| WEIGHT-fields | `CrossDetail~EditAirExport~EditAirImport~EditBooking~EditLand~EditLcl~EditRailway~31ecd969.48159171.js` | `[1041,2090)` |
| WEIGHT-formula | `CrossDetail~EditAirExport~EditAirImport~EditBooking~EditLand~EditLcl~EditRailway~31ecd969.48159171.js` | `[2689,4467)` |
| WEIGHT-controls | `CrossDetail~EditAirExport~EditAirImport~EditBooking~EditLand~EditLcl~EditRailway~31ecd969.48159171.js` | `[10786,17390)` |
| APP-required | `app~c714bc7b.aad8b8e0.js` | `[243514,245204)` |
| APP-endpoints | `app~c714bc7b.aad8b8e0.js` | `[69428,71388)` |
| APP-design | `app~c714bc7b.aad8b8e0.js` | `[71467,71767)` |
| APP-states | `app~c714bc7b.aad8b8e0.js` | `[81291,81741)` |
| APP-width-char | `app~c714bc7b.aad8b8e0.js` | `[220548,220730)` |
| APP-round-wrapper | `app~c714bc7b.aad8b8e0.js` | `[224403,224746)` |
| APP-numeric-validation | `app~c714bc7b.aad8b8e0.js` | `[225985,226357)` |
| APP-custom-field | `app~c714bc7b.aad8b8e0.js` | `[77992,78592)` |
| LIST-navigation | `AirExport~AirImport~DomesticTrade~LCL~Land~Logistics~OrderBooking~Railway~SeaExport~SeaImport~21833f~1faa7780.098b5e5d.js` | `[77699,78211)` |
| LIST-finish | `AirExport~AirImport~DomesticTrade~LCL~Land~Logistics~OrderBooking~Railway~SeaExport~SeaImport~21833f~1faa7780.098b5e5d.js` | `[73867,75129)` |
| LIST-shutout | `AirExport~AirImport~DomesticTrade~LCL~Land~Logistics~OrderBooking~Railway~SeaExport~SeaImport~21833f~1faa7780.098b5e5d.js` | `[75944,77171)` |
| LIST-lock | `AirExport~AirImport~DomesticTrade~LCL~Land~Logistics~OrderBooking~Railway~SeaExport~SeaImport~21833f~1faa7780.098b5e5d.js` | `[73004,73867)` |
| WTABLE-cell-click | `chunk-748cfcfe.fd7e1e76.js` | `[1623,2103)` |
| WTABLE-row-click | `chunk-748cfcfe.fd7e1e76.js` | `[4135,4380)` |
| ORDER-TABLE-row-click | `chunk-374f66c4.a111524b.js` | `[51701,51821)` |
| AE-document-entry | `EditAirExport~21833f8f.ceb04dc7.js` | `[44135,44825)` |
| AI-document-entry | `EditAirImport~21833f8f.bf3376f8.js` | `[52132,52822)` |
| APP-audit | `app~c714bc7b.aad8b8e0.js` | `[229799,230223)` |
| APP-audit-endpoint | `app~c714bc7b.aad8b8e0.js` | `[135857,135997)` |
| DOC-list-init | `DocumentMake~21833f8f.d1979acc.js` | `[8885,9752)` |
| DOC-list-read | `DocumentMake~21833f8f.d1979acc.js` | `[10517,10953)` |
| DOC-list-route | `DocumentMake~21833f8f.d1979acc.js` | `[10953,12420)` |
| AWB-defaults | `AirLadingManage~31ecd969.09820d28.js` | `[69561,71308)` |
| AWB-init | `AirLadingManage~31ecd969.09820d28.js` | `[79020,79700)` |
| AWB-detail-read | `AirLadingManage~31ecd969.09820d28.js` | `[87597,88353)` |
| AWB-save | `AirLadingManage~31ecd969.09820d28.js` | `[85819,87597)` |
| AWB-charge-total | `AirLadingManage~31ecd969.09820d28.js` | `[84956,85819)` |
| APP-air-endpoints | `app~c714bc7b.aad8b8e0.js` | `[75403,76762)` |
| APP-doc-list-endpoint | `app~c714bc7b.aad8b8e0.js` | `[87033,87153)` |
| APP-mawb | `app~c714bc7b.aad8b8e0.js` | `[88190,88290)` |

## 仍未知与后续建议

- 参考站点后台数据库、事务原子性、版本号/悲观锁、真实重复校验、跨公司隔离与最终服务端错误均未验证。
- 自定义字段 required 只证明客户端遵循当前设计；下发配置内容及其他角色/公司下配置会变，必须以实际采样记录为准。
- 前端显示的航空运单尺寸/费率/打印表单不能证明输出 PDF 版式与航空公司实际接收格式合规；需要用户提供认可样张或继续读取现有模板后确认。
- 不把六航班号字段当多航段，不把订单计费重量当运单运费，不把“可见单证制作按钮”当已具备签发权限。
- 本系统可借用字段表达和参考计算交互，但应继续保留独立空运领域模型、明确来源的重量确认、服务端金额精度、共享事务及版本历史。

## 只读导航纠正

主会话发现把人类可见编号直接放进详情路由可以显示空详情标题，但未真正载入订单，并触发“委托单位已修改，请先保存”提示。该尝试不构成已有单据读取成功；必须改用已观察到的内部 orderNo（如复选框关联 ID），再以非空详情及读取响应验证。该观察由主会话负责补充运行证据，本静态报告只证明路由参数来源。

## 新增空白航空运单的读取边界复核

主会话取得真实 AI 单证制作列表运行证据 `ev_b083f056e7f7461bf4969d2d9f872a0f5eea771044164224195f2ea6d60cb493`：五类单证入口可见，当前列表为空。本段是对后续“单个新增”的静态预检，不冒充已打开空白编辑器的运行证据。

- 单证列表新增菜单：外层要求 `corpStatus !== 1 && !synergyModuleDisabled('单证制作')`，作为 popover reference 的新增按钮还要求 `!lock`。这只证明客户端入口条件，不代替服务端授权。
- 提单(MAWB)菜单直接调用 `clickAdd(0)`，提单(HAWB)直接调用 `clickAdd(1)`，中间没有提交、确认或保存函数。
- `clickAdd(t)` 只执行路由跳转：目标 `AirLadingManage`，携带原 DocumentMake 路由的 `type`、`supplierId`、内部 `orderNo`、展示 `outerOrderNo`、`lock`、`corpStatus`，新增设置 `billType=t`、`id=0`。
- 实际航空运单路由为 `/order/air_lading/:type/:supplierId/:orderNo/:outerOrderNo/:lock/:billType/:id/:corpStatus/:orderType?`。其 lock/corpStatus 的位置顺序不同于单证列表路由，不应抄列表路由后按相同顺序拼接。
- 编辑器 created 将 id 转为 `airBillLadingId` 数字；`id=0` 时 initFunc 不读取既有 getPageDetail，而只本地初始化 otherChargeDefault，并读取名称/订单/主分单候选/常用资料/单位等。名称候选读取后在本地取下一个“提单N”，不提交占号或创建。
- 本段复核的单个新增链与初始化链没有 `orderSave`、`saveAirDetail`、`editRateDescription` 调用。实际建运单仍由 `handleClickSave` 触发；本轮没有点击保存。该判断限于客户端调用链，无法证明所有读取端点的后台实现绝对无副作用。
- 批量新增是独立 `handleClickBatchAdd → AddHawbDialog.open`，不包含在上述无创建调用结论中，不以“单个新增无写”推及批量。
- 后续取证建议通过当前页面新增菜单进入，保留实际路由和网络记录；不自行猜内部 ID、客户 ID、外部号或 URL 参数。

新增范围仍采用零基 UTF-8 字节半开区间：

| 范围编号 | 文件 | UTF-8 字节区间 |
| --- | --- | --- |
| DOC-add-menu | `DocumentMake~21833f8f.d1979acc.js` | `[3060,4876)` |
| DOC-add-route | `DocumentMake~21833f8f.d1979acc.js` | `[13128,13518)` |
| APP-awb-route | `app~c714bc7b.aad8b8e0.js` | `[184529,184729)` |
| AWB-created | `AirLadingManage~31ecd969.09820d28.js` | `[73197,74533)` |
| AWB-new-name | `AirLadingManage~31ecd969.09820d28.js` | `[79700,80407)` |
