# 调研：空运主分运单制作字段与页面格式

- 问题：海管家单证制作支持哪些类型？MAWB/HAWB 新增编辑器有哪些字段、控件限制与差异，可据此确定本系统结构化航空运单和 PDF 的哪些需求？
- 范围：读取主会话既有三份 REA 运行证据和本任务静态报告，不操作浏览器、不请求外站、不读取凭据、不保存单据。
- 日期：2026-11-16（北京时间）。
- 用户已确认本期主、分空运单均结构化，支持制作、预览和 PDF 打印。具体版式、进口签发语义、输出份联仍需设计确认；随后用户另行批准 AE/AI 直单（无 HAWB）与主分单两模式；多航段、复杂拆拼仍未获批。

## 来源与证据边界

| 简写 | 原始文件 | REA 证据 | 原始 SHA-256 |
| --- | --- | --- | --- |
| DOC | `air-import-record-document-maker.evidence.json` | `ev_b083f056e7f7461bf4969d2d9f872a0f5eea771044164224195f2ea6d60cb493` | `060d832ae73ee5c756fc51093092a2ae65ea899259a5452d2f4b791a83410657` |
| MAWB | `mawb-empty.evidence.json` | `ev_fd34e177ac56d31dab2e37da0c42fa00ae2a1c9dd69bbbafeea28f7b718d5f43` | `6dbb9bbeff0271875689f14f1b358fa532a2611efec88771a233ed1f79e8931d` |
| HAWB | `hawb-empty.evidence.json` | `ev_048eca8e2bd38c86d8e2df1c596ad97cb5614a337ced34ed5317a31d6be7815f` | `1c74ea5e0e69edaa426ead05f04511c47fc609d201504ae660622fa2128bb4ec` |

本机目录：`/home/vince/.local/share/rea/hgj/order-research/`。DOC 是既有空运进口操作票的单证制作列表；MAWB/HAWB 是该入口打开的未保存新增单证（路由 id=0），不是已保存运单样本。“空表单”表示没有新增保存，**不等于完全没有关联信息**：界面包含由原操作票带来的关联候选，不能把这些值写进字段默认值或测试样本。

三份记录的导航/点击/等待步骤均 completed，最终 DOM/URL/截图 captured；但场景总体 `completeness.status=incomplete`、`missing_sections=['events']`，不能称整个观察完整。事件 dropped=0 也不能证明跨窗口、跨域或所有活动均覆盖。

本报告通过 HTMLParser 只提取分区标签、控件类型、占位提示、maxlength/disabled/readonly 以及内联隐藏祖先；未读取字段个人信息、企业地址、电话或真实主分单号并写入文档。此种 DOM 解析不证明计算样式可见性、窗口像素布局或 PDF 印刷输出。

## 单证制作列表

DOC 直接出现五种类型：

| 类型 | 本轮覆盖 |
| --- | --- |
| 提单（MAWB） | 已打开未保存新增编辑器 |
| 提单（HAWB） | 已打开未保存新增编辑器 |
| 舱单（Manifest） | 仅列表类型入口，编辑器/格式未验证 |
| 货物标签（IATA Cargo Label） | 仅列表类型入口，条码/尺寸/打印未验证 |
| 邮袋标签（IATA Pouch Label） | 仅列表类型入口，编辑器/格式未验证 |

列表还有新增、批量新增、类型筛选，列包括单证类型/名称、主单或分单号、分件数、件数单位、分毛重 KGS、分体积 CBM、创建人/时间、操作。

该真实操作票的制作列表为空；这只证明这个样本和当前账号没有返回可见制作行，**不证明全公司没有运单，也不证明任何类型不支持**。不能据“暂无数据”推断关系基数、号码是否允许重复、删除历史、审核或复制后的行为。

## 编辑器分区与印字标签

下表 MAWB/HAWB 均在实际 DOM 中出现，差异在下一节单列。英文拼写按页面原文保留，不能把网页输入分节等同标准 AWB 的正式版式。

| 分区 | 观察到的标签/字段 | 设计意义与限制 |
| --- | --- | --- |
| 顶部身份 | 提单名称、编辑、主/分单号关联 | 单证名称与运单号码不是同一字段，关联选择与印字号码也不能混同 |
| Shipper | Account Number、Name and Address | 独立托运人账号及完整文本，不只客户简称 |
| AWB Consignment Details | AWB Number、Airport of Departure；HAWB 另有 HAWB Number | 号码按航空公司数字代号与序号/检验号拆框；不证明已校验 3+8 位或 mod7 |
| Consignee | Account Number、Name and Address | 独立收货人资料 |
| Issuer | Issued by | 印发人文本，不证明当前公司有航司签发权限 |
| Issuing Carrier’s Agent | Name and City、IATA Code、Account No. | 代理文本、IATA 代码、账号分别表达 |
| Accounting Information | Details | 单证财务说明文本，不能当费用台账 |
| Rounting and Flight Bookings | Departure、Route To、By First Carrier、To/By 两组、Destination、Flight、Date | 有三组承运/地点印字位及航班日期；**不证明订单运输是三段权威航程模型** |
| Shipment Reference Information | Reference Number、Information (1)、Information (2) | 参考号与补充文字，不等同本系统客户业务编号 |
| Charges Declaration | Currency、CHCG、Value for Carriage、WT/VAL、Value for Customs、Other、Amount of Insurance | WT/VAL 和 Other 各出现 PPD/COLL；声明值可为文本，不应强制一律 decimal |
| Handing Information | SCI、Requirements | 独立 SCI 和注意事项 |
| Rate Description | Pieces、Gross Weight(KGS)、Rate Class、Item No.、Chargeable Weight、Rate/Charge、Total、Nature and Quality、操作 | 运价明细表和新增入口，本次无数据行，行内控件规则尚未验证 |
| Charges Summary | Payment Method、Weight Charge、AS ARRANGED、Valuation Charge、Tax、Total Other Charges Due Agent、Total Other Charges Due Carrier、Total | 金额摘要；AS ARRANGED 为单独勾选，不是零值金额的别名 |
| Other Charges | 输入框、自动计算；费用名称、计费单位、数量、币种、单价、总价、操作 | 存在文本与计算式表格两种呈现内容，隐藏/折叠挂载可能混入 DOM；未切换、未验证参与总计方式 |
| CC Charges in Destination Currency | Currency Conversion Rates、CC Charges in Dest. Currency、Charges at Dest、Total Collect | 目的站币种及汇兑金额；不等同订单费用/结算/开票汇率 |
| Shipper’s Certification | 证明文本、Signature | 独立声明与签字印字位 |
| Carrier’s Execution | 承运人填写文本、Date、Place、Signature | 日期/地点/签名，不自动代表已经签发 |
| 补充 | 运输条款、唛头、通知人/第二通知人、提单备注 | 通知人页签需逐个验证；本轮只捕获当前 textarea |

页面采用若干 `form-box`、`detail-info-wrap`、`flex`、`width50` 等 HTML 分组；例如托运人与 AWB 细节为独立区块，声明信息有成对字段，运价使用表格。以上仅为 DOM 结构证据，**未测量 A4 尺寸、边距、字号、线条、纸张方向、页眉页脚和标准 AWB 方格**。即使分节标签可用于输出需求，也不能据此宣称已复刻模板布局。

## MAWB 与 HAWB 的直接差异

| 维度 | MAWB 空表单 | HAWB 空表单 |
| --- | --- | --- |
| 标题/复制 | MAWB制作、复制MAWB | HAWB制作、复制HAWB |
| 顶部关联 | 主单号及主单号关联 | 分单号、分单号关联以及主单号关联 |
| 独有身份印字 | AWB Number 拆框 | 同样 AWB Number 拆框，并额外 HAWB Number input（maxlength=20） |
| 从订单带入 | 本份 DOM 没有“带入详情页收发通”文案 | 有“带入详情页收发通”入口；本轮未执行 |
| 其他主要分区 | 上表全部 | 上表全部 |

HAWB 编辑器有主单关联并不证明“一主单必须一分单”、共享的具体写入边界、或者没有主单时一定拒绝。两类制作页共有收费结构，也不代表真实结算共用一条账。

## HTML 字段格式与限制

以下数值都是捕获 DOM 中的 maxlength，不是服务端校验结论；number/date 类未显示文本长度限制不代表无数值或时区约束。

| 字段 | 类型/HTML 最大长度 |
| --- | --- |
| 提单名称 | text，50，当前 disabled；旁有独立编辑入口 |
| 托运人/收货人账号 | text，各 50 |
| 托运人/收货人完整文本 | textarea，各 1000，带企业资源搜索提示 |
| 航空公司数字代号 / 序号及检验号 | text，各 20；不能直接推定业务位数 |
| 航空公司印字 | text，50 |
| HAWB 分单号 | HAWB 独有 text，20 |
| 印发人 / 财务说明 | textarea，各 500 |
| 国外代理姓名地址 / 账号 / IATA 代码 | textarea 1000 / text 50 / text 5 |
| 始发机场 / 航线到 / 第一承运人 | text 50 / 100 / 20 |
| 后续 To/By | 两组，每个 text 20 |
| 目的地 / 航班 | text，各 50 |
| 航班日期 | date 组件内部 text，无本轮 maxlength；未验证提交格式 |
| 参考号码 / 信息1 / 信息2 | text 50 / textarea 50 / textarea 50 |
| 运费代号 CHCG / 运输声明价值 / 海关声明价值 / 保险金额 | text，各 10 |
| SCI | 与 SCI 标签相邻的 text，20；无占位文案，仍需运行期绑定核对 |
| 注意事项 / 托运人证明 / 承运人填写栏 | textarea，各 500 |
| 其他费用说明 | textarea，1000 |
| 托运人签字 / 承运人地点 / 签名 | text，各 20 |
| 运输条款 / 唛头 / 当前通知人 / 提单备注 | text 50 / textarea 10000 / textarea 1000 / textarea 1000 |

付款方式、币种、WT/VAL、Other 和主分单关联的 input 是 readonly 的 Select 内框，不能据 readonly 断言不可选择。按钮“保存/复制/编辑”和隐藏对话框存在不代表未保存状态下都可用。

重量/费用的单位与语义应分别建模：

- 运价明细的 Gross Weight 明示 KGS；Chargeable Weight、Rate/Charge、Total 是独立列，不能把 Total 当重量。本次运价表为空，未验证行控件精度、负数/零、进位、Rate Class 枚举和货物多行映射。
- 运费摘要六项（计重运费、声明价值附加费、税金、向代理/承运人费用、总计）均出现 number input；没有提交或改值，不能认定是否允许手工覆盖、哪些字段自动计算。
- 目的地收费四项（汇率、折外币费用、目的站费用、到付总计）也为 number input；换算方向与舍入未知。
- Other Charges 表格出现 KGS 与“票”等单位候选内容以及若干费用代码，但未操作，不把 DOM 中预填/隐藏行认定为四类业务通用默认费率。
- 货币符号和金额位数不能由 number input 推出；本系统金额、汇率、数量分别按既有 decimal/formatAmount/trimDecimal 约定处理。

## 静态代码证据与动态观察的对应

本任务 `frontend-bundle-rules.md` 的 AWB-defaults / AWB-init / AWB-detail-read / AWB-charge-total / AWB-save 分段提供已交付 JavaScript 的静态观察，**不是本轮三份 DOM 场景执行证明**：

- 静态初始模型含 NVD/NCV/XXX 和 PPD；本轮未读取框架运行期 value 属性，也未改变声明字段，不能据 DOM 候选或占位提示认定所有账户固定初值。
- 静态 `chargesInfo.billingWeight` 按费率明细 total 合计并进入费用总额，实际上是金额；它与订单 `billLading.billingWeight` 的采用计费量不同。运行期表格为空，本轮没有验证计算结果，但设计上必须避免按字段名强行共享重量类型。
- 静态 AirLadingManage 根据 billType 0/1 分别显示 MAWB/HAWB；运行期两编辑器的标题/HAWB 独有号码控件与之相符，不能进一步据此断言数据库关系。
- 静态保存链包括 saveAirDetail、修改名称及逐条保存待存费率；本轮没有点击保存，不能证明服务端原子性、幂等或并发控制。本系统应保留统一事务与版本约束，不照抄多写请求的潜在部分成功。

## 本轮网络元数据与限制

对已捕获 events.items 过滤 XHR/fetch，未读取请求/响应业务体：

| 场景 | 观察到的有关请求 |
| --- | --- |
| DOC | POST `/saas-support/applications/flows/audit`；GET `/saas-service/manifest/billMultiList` |
| MAWB / HAWB | GET `/saas-service/air/getBillLadingName`；GET `/saas-service/manifest/billMultiList` |

资格检查为 POST，路径名字不能证明它一定无副作用；本报告仅说明观察到该请求。三份场景没有显示点击保存/改名/费率保存；上面列出的网络子集没有相应写路径，但鉴于总体 missing events，不能以“未捕获”声称不存在任何副作用或接口。

静态 getAirDetail、getMAWBByBill、getRateDescription 等候选没有在此子集观察到，可能和新建 id=0/请求条件有关，仍是未知，不据静态存在虚构本轮已执行。

## 对本系统的实施建议（待最终设计审批）

1. 为主、分运单分别建立结构化印字内容及版本，区别于订单运输权威事实；重复的印字字段可以共享纯 Go 内容值对象，但不能让空运依赖海运单证类型。
2. 关联 ID/版本与可编辑印字快照分离；修改印字航线不应无提示修改共享运输，修改主运单影响成员仍按固定锁序和版本检测执行。
3. 明确托运人/收货人/代理、声明、运价明细、其他费用、签发及签名区；账号、IATA、声明值和印字号码均不是纯金额。
4. PDF 输出使用同一结构化快照预览/下载/打印，保留版本、模板和公司抬头指纹；长文本完整续页，不在输入合法但打印截断。
5. 金额/数量/汇率分别规范，禁止根据 `billingWeight` 这样的歧义名称直接映射。AWB 运费展示与实际订单费/账单必须显式关联，不自动导入或改写真实费用。
6. 本期已确认主分单制作打印，用户随后另行批准无 HAWB 直单；没有确认 Manifest、两类标签、完整多航段或复杂拆拼，不能通过单证类型列表将它们自动加入实施。

## 尚未验证

- 未保存任何 MAWB/HAWB，未验证号码校验、唯一性、必填、名称修改、复制、关联换单、服务器权限、版本冲突、并发与事务。
- 未新增费率行、切换收费计算模式或付款条款、切换通知人页签，未确认币种/预付到付默认和金额公式。
- 未点击导出、预览、下载或打印，没有生成/检查 PDF；不能宣称纸张尺寸、印刷布局、可扫描标签或条码已验收。
- 仅有空运进口样本入口，未取得相同编辑器在空运出口/其他身份下的运行对比；静态共享组件不能替代跨类型权限实测。
- 未查舱单/标签编辑器，未验证空制作列表以外的已有单证行；没有据此声称五类单证全覆盖。
- 未运行本系统单元/集成测试，本报告只用于需求与设计证据。

相关规范：`.trellis/spec/server/backend/air-export-contract.md`、`.trellis/spec/server/backend/sea-hbl-production.md`、`.trellis/spec/server/backend/sea-export-document-contract.md`、`.trellis/spec/web/frontend/state-management.md`、`.trellis/spec/web/frontend/type-safety.md`。
