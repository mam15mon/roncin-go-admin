# 四类订单字段逐项采纳建议

状态：调研建议，未经最终设计审批，不授权产品实现。日期：2026-11-16（北京时间）。本文件只负责字段采纳、归属及验收建议，不替代 PRD、设计或实际运行证据。

用户已确认的边界：本期实现 SE/SI/AE/AI；MAWB、HAWB 均结构化且支持制作、预览和 PDF 打印；航空支持直单和主分单；进口覆盖到货、换单、清关、提货、派送，按本票承接服务决定适用性；陆运、铁路、内贸水运、自拼、物流园报关只调研。用户已确认真正多航段、复杂空运拆拼改配及批量导入后续做；直单一主单一票且不混HOUSE，四类完整筛选CSV导出本期做。

读取依据：[表单字段矩阵](form-field-matrix.md)、[实施影响](implementation-impact.md)及下列当前源码。未访问外站、未读取凭据、未运行产品测试。海管家 DOM 字段是页面观察，不证明后台必填、唯一性、计算规则或默认值。下表是本报告唯一的逐项采纳表；“保留”表示保留现有语义并在四类验收适用性，“补全”表示建议本期补实现，“不采纳”表示不采用竞品表达方式，“后续”表示建议延期，具体技术/字段建议随父最终复核稿批准后实施；本表不独立授权执行。所有新增字段名与数值限制均为设计提案。

## 唯一逐项采纳表

| ID | 字段或行为 / 适用 | 建议 | 最薄字段归属与前后端契约 | 当前源码证据 | 对应验收 |
| --- | --- | --- | --- | --- | --- |
| F01 | 订单业务类型与进出口方向 / 四类 | 补全 | 保留 `Order.businessType/tradeDirection`；只允许 SE/AE+出口、SI/AI+进口；前端从注册项产生，服务端逐次校验，保存不能改变既有身份。 | `server/internal/biz/order_usecase.go:321`；`web/src/pages/orders/order-kinds/registry.ts:9` | 四类可建票，四种错配拒绝，跨类型 ID 操作与组织切换均不越权。 |
| F02 | 主单方向身份、共享候选 / 四类 | 补全 | 海空主单各自增加不可变方向/业务身份；候选、关联、预览、提交、恢复均比对，不凭同号自动拼单；唯一键增加方向。具体唯一范围按最终设计审批。 | `server/internal/data/ent/schema/sea_master_bill.go:51`；`air_master_waybill.go:36` | 同号进出口可隔离存储；跨方向候选不出现、直接提交对方主单 ID 失败；零活动成员仍保留方向。 |
| F03 | 自动订单号、编号时间 / 四类 | 保留 | 沿用单一订单编号规则及事务序列，业务代码取真实类型，不为 SI/AI 另做编号平台；编号时间保留既有创建语义。 | `server/internal/data/order_write.go:29`；`server/internal/biz/orderconfig.go:200` | SE/SI/AE/AI 前缀正确，并发不重号，创建失败不留半票；详情不可修改订单号。 |
| F04 | 委托单位、代码、人员默认带入 / 四类 | 保留 | `Order.customerId` 引用真实同公司 Partner；代码展示；销售/客服保留客户档案权威带入、恒只读且新建必填，服务端创建/换客户按档案覆盖；操作等可编辑岗位仅默认初始化，不强制覆盖用户手改。 | `web/src/pages/orders/templates/air-template.tsx:59`；`SeaBasicInfoSection.tsx:740`；`server/internal/data/order_sync.go:28` | 真实候选检索、已停用存量引用按现规则；销售/客服随新客户权威配置变化，操作等手改人员保持；客户缺两岗或跨公司引用拒绝，提成快照不重拍。 |
| F05 | 集运/跨境 / AE、AI，SE/SI适用时 | 补全 | 使用已有 `shipmentMode` 枚举；AE 补控件、默认/清空/创建/更新/回显映射。它是经营模式字段，不是拼单关系或服务多选，不自动创建跨境业务模块。 | `order.proto:580`；`ent/schema/order.go:66`；`air-export/form-adapter.ts:46` 的字段集合未包含该项 | 两值往返；未选保留 optional 语义；改经营模式不移动主分单成员或改费用。 |
| F06 | 客户业务号、内部业务号长度及查重 / 四类 | 保留 | 最终稿提案保留客户/内部编号各100字、无新全局查重唯一约束，不机械扩大为HG250字；前后端边界保持一致。 | `order_usecase.go:365`；`ent/schema/order.go:38`；`SeaBasicInfoSection.tsx:114` | 按批准上限做中英文边界与超限拒绝；不截断；未批准查重范围前不建设新全局唯一键。 |
| F07 | 贸易条款 / AE、AI及两海运 | 补全 | 沿用 `Order.tradeTerm` 已有枚举；AE 增可编辑选择和回显，不新表；不依据条款自动创建应收应付。 | `order.proto:572`；`ent/schema/order.go:62`；`air-export/form-adapter.ts:81` 已映射 | 已有条款值创建/编辑/清空往返，非法枚举失败，改条款不静默改费用或服务范围。 |
| F08 | 业务付款方式 / 四类 | 补全 | 已有 `Order.paymentTerm=PREPAID/COLLECT` 提供可编辑入口；AWB 的运费/其他费用 PP/CC另归单证，不能用一个隐藏默认字段覆盖全部印字。 | `air-export/form-adapter.ts:80`、`:110`；`SeaCargoSection.tsx:34`；`order.proto:573` | 到付刷新不变预付；缺值/非法值按明确合同处理；修改单证 PP/CC不自动改业务付款方式。 |
| F09 | 订舱代理 / AE、AI及两海运 | 补全 | 复用 `Order.bookingAgentId` 和供应商候选；AE 仅补 UI，已有映射保留。进口未承接订舱时不强制必填；不是换单代理别名。 | `order.proto:577`；`air-export/form-adapter.ts:82`；`SeaBasicInfoSection.tsx:740` | 同公司合法供应商可选、创建更新清空回显；外部变动保护；不自动产生供应商费用。 |
| F10 | 国外代理业务主体 / 四类 | 补全 | 复用 `Order.foreignAgentId`，AE 补 UI；按国外代理角色校验；单证国外代理文字为独立快照。 | `order.proto:600`；`air-export/form-adapter.ts:83`；`SeaBasicInfoSection.tsx:750` | 主体 ID 与单证文字分别保存；改主体不覆盖已保存印字；伪造类型/公司引用拒绝。 |
| F11 | 船代 / SE、SI | 保留 | 沿用公共船代字典 ID；空运不显示、不把其值当航司代理。 | `SeaBasicInfoSection.tsx:770`；`order.proto:605`；`data/order_sync.go:85` | 两海运候选可检索；空运请求夹带不适用引用显式处理，不污染航空实体。 |
| F12 | 换单代理 / SI；AI是否需要相同字段 | 补全 | 建议新增本票 `ImportHandling.exchangeAgentId`，引用同公司供应商 Partner，不借 bookingAgentId；SI 支持，AI保持字段可选且须有实际用途才显示。该引用不代表已完成换单。 | HG-SI见字段矩阵；当前 `order.proto:568` 无独立字段；供应商候选范本 `SeaBasicInfoSection.tsx:740` | SI 与订舱代理可分别选不同主体、独立回显清空；非本公司/不适用角色失败；未承接换单不强制。 |
| F13 | 换单时间 / SI、AI | 补全 | 本期以进口EXCHANGE完成事实completedAt表达；换单代理独立引用；不新增预计换单时间，不把docCutoff改名复用。 | `order.proto:589`–`:592` 为各出口截止；`air_export.proto:49` 为document_cutoff_at；HG-SI/AI见字段矩阵 | 到货/换单时间互不覆盖；带时区往返正确；未承接环节显示不适用，未填时间不能伪造已完成。 |
| F14 | 合约号 / AE、AI及两海运 | 补全 | 沿用本票 `contractNo`、100字，AE补控件；无需收缩为HG50字，不把它当报价关联。 | `air-export/form-adapter.ts:84`；`order.proto:601`；`SeaBasicInfoSection.tsx:781` | 51–100字可保存；100上限双端一致；空值清除；独立于主单号/报价。 |
| F15 | 货值及币种 / AE、AI及两海运 | 补全 | 已有 `cargoValue/cargoCurrency` 成对可选；AE补 UI、表单类型与双向映射；原始decimal字符串提交，最多18整数+4小数，币种真实主数据。 | `order.proto:602`；`order_usecase.go:418`、`:690`；`ent/schema/order.go:47`；`air-export/form-adapter.ts:46` 未包含 | 缺一拒绝、显式0可保存、非法币种/负数/超精度失败；显示formatAmount不反写原值；不自动生成费用。 |
| F16 | 保费及币种 / AE、AI及两海运 | 补全 | 已有 `insurancePremium/insuranceCurrency` 成对可选，独立于货值；AE补 UI/映射。保费是订单经营记录，不等于已投保，也不是AWB保险声明金额。 | `order.proto:606`；`order_usecase.go:421`；`SeaBasicInfoSection.tsx:806` | 同货值边界，独立币种往返；填写保费不调用保险接口、不自动建应付。 |
| F17 | 接单时间、货好时间 / AE、AI及两海运 | 补全 | 使用本票 `receivedAt/cargoReadyAt`，AE补 DatePicker↔RFC3339映射；可空，不回退到createdAt或航班起降时间。 | `order.proto:611`、`:614`；`order_usecase.go:441`；`SeaBasicInfoSection.tsx:823`、`:828` | 两时间不同时区往返为同一时刻；清空、未设置和非零时间均验证；列表/详情本地可读格式。 |
| F18 | 工厂 / 四类适用时 | 补全 | 沿用 `Order.factoryName` 200字可选文本；AE补 UI/映射；本期不新建工厂档案或联系人平台。 | `order.proto:610`；`ent/schema/order.go:56`；`SeaBasicInfoSection.tsx:836` | 200字中英文可保存、201拒绝、清空；无工厂的进口不被必填拦截。 |
| F19 | 订舱/配舱/操作/普通备注 / 四类 | 补全 | 分别沿用本票 bookingNotes/allocationNotes/operationNotes/notes，各1000字；AE已有映射，补当前缺失的订舱/配舱备注编辑入口。进口显示适用业务称谓，不擅自混用到货/换单事实。 | `air-export/form-adapter.ts:99`；`order.proto:615`；`order_usecase.go:365`；`AirMasterFields.tsx:461` | 四字段可分别修改/清空，刷新不串值；多行保留；1000/1001边界和并发输入保护。 |
| F20 | 内部八类人员 / 四类 | 保留 | 沿用CREATOR/OPERATOR/SALES/CUSTOMER_SERVICE/ASSOCIATE/DOCUMENT/COMMERCIAL/ASSOCIATE2及同公司候选；创建人不可被普通表单伪造修改。 | `SeaPersonnelSection.tsx:111`；`air-export/form-adapter.ts:132` | 四类建票角色正确，创建人权威；切公司清候选；晚到请求不写入另一组织。 |
| F21 | 进口服务承接范围与进度 / SI、AI | 补全 | 固定五环节 ARRIVAL/EXCHANGE/CUSTOMS/PICKUP/DELIVERY，每票选是否承接；每环节有状态/实际完成时间/责任人/备注及事件。无服务字典、标签或租户流程设计器；未承接不是已完成。 | 用户明确批准；`order.proto:293` 旧service_type_ids已reserved；`ent/schema/order.go:67` 现flow为出口 | 任意服务子集可保存、完成适用环节；未承接环节不阻断；缺实际资料不能“假完成”；服务变更、完结、锁、终止与版本检查一致。 |
| F22 | 海运托运类型、箱计划、实际箱 / SE、SI | 保留 | 沿用FCL/LCL/BREAK_BULK、箱计划和真实箱/共享分配；SI按业务适用后接入；空运不增加海运箱型。 | `ent/schema/order.go:64`；`order_usecase.go:451`；`SeaActualContainers.tsx` | FCL/LCL/散杂适用分支与件重尺守恒；SI当前票箱不能借SE跨方向成员，空运不显示海运箱型。 |
| F23 | 委托货量、实际货物明细 / 四类 | 保留 | 委托 `Order.totalPackages/totalPackageUnit/totalGrossWeightKg/totalVolumeCbm` 与实际CargoItem分离；整数件数、kg/m³，不因同页展示自动覆盖。 | `order.proto:594`、`:624`；`ent/schema/order.go:104`；`AirExportDetailFeatures.tsx:22` | 零/空/非零区分，负值/非法精度拒绝；实际明细变更不改委托量；主单统计覆盖完整成员，不求和当前页。 |
| F24 | 品类与危险品 UN/CLASS/截申报 / 四类 | 保留 | 复用cargoCategoryIds及本票危险信息；UN4数字、CLASS最多16；截申报与截单/截关独立。取消危险品的清空行为需前后端一致，不能只隐藏残值。 | `SeaBasicInfoSection.tsx:189`、`:216`、`:237`、`:247`；`order_usecase.go:421` | 勾选/取消连续操作、清空、非法UN拒绝；历史加载不得因初始化误清；四类请求一致，不放宽为任意文本。 |
| F25 | 空运危险品包装等级 / AE、AI | 保留 | 沿用PG_I/PG_II/PG_III，未设置保留“未填/不适用”；不因HG未显示而删除；不新增UN→PG联动或危险品规则库。 | `ent/schema/order.go:53`；`SeaBasicInfoSection.tsx:255`；`air-template.tsx:85` | 支持相应类别及可空；清危险品清PG；非法枚举拒绝，PG不是重量单位。 |
| F26 | 体积重、分泡、计费重 / AE、AI | 保留 | 保留ORDER/MASTER、PLANNED/ACTUAL各自确认及指纹；默认D6000、P1、0.001kg HALF_UP，支持明确分子分母；不运行用户自由表达式。 | `biz/air_weight.go:86`；`ent/schema/air_weight_confirmation.go:20`；`data/air_weight.go:61` | 保留1/3等精确算例；DIRECT无HAWB可确认；主单与本票不互相确认；变化标过期但不改历史费用。 |
| F27 | HG体积重/分泡快捷候选及KG/CBM单位菜单 | 不采纳 | 交付前端静态已能观察候选比例映射（见frontend-bundle-rules.md），但不自动照搬其菜单和eval求值；本期保留明确除数/有理数及KG，不将CBM当重量。 | 字段矩阵的HG候选只是隐藏DOM；`.trellis/spec/server/backend/air-export-contract.md` 重量与费用条款 | 不出现错误单位切换；1/3不截为0.333；未知候选不改变后端规则。 |
| F28 | 航司/起运目的机场/单航班/时刻 / AE、AI | 补全 | 复用AirMaster权威运输，拓展AI；航司/机场ID，flightNo最多40，ETD/ETA/实际起降RFC3339。不得从“港”文案改用Port FK。 | `air_export.proto:39`；`AirMasterFields.tsx:30`、`:137`；`biz/air_export.go:222` | AE/AI共享修改预览及锁内校验；机场/时刻列表筛选与详情一致；修改计划不自动写实际或进度。 |
| F29 | 航班新增、多航段/中转运输模型 / AE、AI | 后续 | 用户明确真正多航段后续做；本期保留一组权威运输航班/机场/时刻，运单纸面路线位仅作印字，不将逗号号码列表伪装独立航段。 | `air_export.proto:44` 仅单flight；HG新增入口未执行，不能认定其真实模型 | 选定范围后验收；未批准前不以自由文本“支持多航段”冒充实现。 |
| F30 | 海运船司、船名航次、码头、港口 / SE、SI | 补全 | 复用SeaMaster+TransportExecution，补SI适用入口；共同卸货港与本票最终目的港独立；码头现有字段适配不另建港口平台。 | `SeaTransportSection.tsx:681`、`:714`、`:742`；`.trellis/spec/domain/glossary.md`共同段/本票目的港 | 同主单成员读取同运输事实；本票后程目的港不被共享更新覆盖；跨方向共享拒绝。 |
| F31 | 截单/截关/VGM/截申报 / 适用四类 | 保留 | 各时间有独立语义，按运输/本票归属现模型；进口换单不占用截止字段。超时与缺资料分开，不因某项不存在而制造时间。 | `order.proto:589`；`air_export.proto:49`；`data/air_order_sync.go:204` | 时区正确；超时原因处理与资料硬门禁分开；可空字段不默认今天/ETD；进口未承接环节不强填。 |
| F32 | 航空主分单模式与号码 / AE、AI | 补全 | DIRECT/HOUSE显式；DIRECT一MAWB一票不混HOUSE且真实无HAWB；HOUSE每票真实分单；MAWB11位mod7、HAWB批次永久占号。最终稿提案创建选模式，保存后本期不提供模式互转。 | `biz/air_export.go:190`、`:301`；`data/air_order_sync.go:73`；`air_master_waybill_order_link.go:19` | 两模式建票/编辑/签发/锁/终止恢复；无HAWB是明确存在性；非法混合载荷失败；唯一性并发数据库兜底。 |
| F33 | 结构化收发货人、通知人/第二通知人、单证代理文字 / MAWB/HAWB | 补全 | 航空独立内容模型，主单/分单各自持有party文本及可选来源ID快照；建议party每项5000字。业务主体ID与印字分开，无自动同步覆写。 | `air_export.proto:74`现自由文本；海运范本`sea_document.proto:185`；`biz/sea_document.go:18` | 主/分分别保存、第二通知人不覆盖第一；来源变更不自动改已保存文字；5000/5001边界、长地址分页、历史独立读回。 |
| F34 | 唛头、单证英文货描、免责/附加说明 / 四类单证 | 补全 | SE/SI沿用SeaBillContent，MAWB/HAWB新独立marks/description/handling或clauses；最终稿提案航空marks/description各10000字，handling/clauses/remarks各5000字，保持委托goodsDescription1000独立。海运现上限不机械缩为HG值。 | `sea_document.proto:189`、`:199`；`biz/sea_document.go:18`；`order_usecase.go:365` | 四种长文本不串值、不截断、换行保留；保存/历史/PDF字符完整；业务货描不能覆盖正式单证货描。 |
| F35 | HG唛头10000对当前5000 / 海运及拟建航空 | 补全 | 航空独立marks/description字段，最终稿提案各10000字；主分各存内容并保留完整输出。现有海运长度合同不因对标缩短或机械改动。 | `biz/sea_document.go:18` 当前共享party长度常量含Marks；`SeaBillContentFormFields.tsx:859` | 批准后按新上限及超限测试；不能只放开输入再PDF裁切，不静默截断已有数据。 |
| F36 | 单证件重尺、计费量、费率金额 / MAWB/HAWB | 补全 | 正式单证独立lines/合计快照；件数整数、重量和计费量明确单位、费率金额decimal字符串。可显式从委托/实际/确认带入，保存后不随货物或费用自动覆盖。 | 现`air_export.proto:39`只有主单申报量；`sea_document.proto:191`为海运内容参考；`order_fee.proto:931`数量依据 | 本票和整主单数不重复带入；打印采用保存快照；0/空/非零和小数精度验证；打印金额不自动创建财务费用。 |
| F37 | AWB声明价值、保险声明、PP/CC、Handling、签发代理/地点/日期/签章文字 / MAWB/HAWB | 补全 | 是用户已批准的航空结构化制作必要设计候选；归航空文档内容/printDetails，主分独立。declaredValue、insuranceAmount不借订单cargoValue/insurancePremium；签发日期纯日期，发行主体引用与印字快照明确。准确字段清单/尺寸/枚举及版式待最终设计审批。  最终稿提案一套A4航空版式加续页，MAWB/HAWB身份差异明确；进口保存真实外部主体，不自动签发/加电子签章；具体分区以air-waybill-observations.md及父design为准。 | `air_export.proto:39`、`:69`无这些结构；海运打印范本`sea_document.proto:203`；`hbl/model.ts:105`签发主体指纹 | 已保存数据能制作主/分PDF；缺可选项留空；真实主体、不同主分内容和历史可验证；不伪造外部航司签名/电子签章，不冒称已实际放货。 |
| F38 | PDF预览/下载/打印 / MAWB/HAWB，海运保留 | 补全 | 航空独立版式；复用技术机制：lazy renderer、同域字体、同一Blob、版本/主体指纹、变动提醒和输入保护。不得复制海运“正本/电放”语义到AWB，页面模块不互相深引。 | `hbl/HblEditor.tsx:67`；`hbl/pdf/HblPdfPreview.tsx:34`、`:46`、`:54`、`:71`；`hbl/use-hbl-data.ts:46` | 主分实图与提字完整，超长内容续页；保存失败不出新稿、版本变化不下载旧稿；锁定可按读权限输出已保存内容。 |
| F39 | 引用报价及自动带费 | 后续 | 建议本期不新增报价→订单自动带入协议；HG禁用报价号/引用入口不足以说明已验证业务。现有手工合约/备注不能充当报价关联。 | `order.proto:568`当前无报价引用；字段矩阵HG入口观察 | 作为明确后续而非虚假按钮；不静默带入未经确认价格或审批结果。 |
| F40 | HG可配置服务类型/自由标签/租户流程 | 不采纳 | 本期只固定进口承接环节，不恢复service_type_ids、业务标签或流程平台；出口已有订舱/运输按既有流程复用。 | `order.proto:293`；根AGENTS业务标签退役要求 | API/Schema不恢复旧字典和标签关联；固定服务只改变本票承接与适用性，不成为权限真相。 |
| F41 | 表格导出 / 四类列表 | 补全 | 用户明确四类完整筛选CSV导出；专用无page导出请求与列表同条件同排序同公司授权，服务端同读快照返回完整行DTO，前端复用公共CSV，不循环翻页、不静默截页。 | `OrderListToolbar.tsx:134`可选onExportTable；`OrderListTemplate.tsx:714`未传该回调；`order.proto:16`现服务无订单导出RPC | 核对完整行数/排序/筛选/权限，覆盖超过200行；金额两位无千分位，时间本地可读，数量比例保精度，CSV转义/BOM复用现工具。 |
| F42 | 订单批量导入 / 四类 | 后续 | 用户明确订单批量导入后续做；本期不提供假导入按钮。未来独立设计模板/预检/逐行错误/重复号与主单关联/提交原子边界。 | 当前`order.proto:16`无订单导入RPC；历史AE任务`prd.md:96`曾延期批量导入（可被新决定覆盖） | 若批准，预检零业务写入、非法行明确、提交原子边界说明、重复导入不误拼主单；不能因有CSV导出便宣称可导入。 |
| F43 | 复杂拆拼/改配/多主单/公共成本分摊 / 四类 | 后续 | 用户明确复杂空运拆拼、跨主单改配后续做；已有SE高级能力保留并回归。SI先交付本稿正常票和共享单证/服务闭环，不新建独立进口高级拆拼分摊协议。 | `sea_order_change.proto:28`、`:46`；`air_export.go:334`普通保存禁止换主单；旧AE任务`prd.md:96` | 范围批准前不加伪入口；已有SE回归；批准后目标票与费用分摊守恒、失败全回滚。 |
| F44 | 仓库安检、运单库存、自动配载、航司/海关对接 | 后续 | 建议不因HG存在关联入口而纳入本期；当前固定清关环节记录不等于电子报关对接。 | 旧AE任务`prd.md:63`；当前`air_export.proto:8`仅共享运单服务 | 报告标后续，页面不虚称完成外部申报/舱位/库存动作；可登记资料不等于通道成功。 |
| F45 | 陆运/铁路/内贸水运/自拼/物流园报关 | 后续 | 本期仅调研，形成各自字段/流程/接口证据和差距，不实施菜单、实体或权限。后续实施必须单独批准。 | 用户明确范围；`order_usecase.go:321`当前非SE/AE拒绝 | 五类研究可逐项核对覆盖与未知；产品代码无暗中新增这五类写入口。 |

## 最薄实施合同及归属

### 1. 已有订单基础字段不增加数据库副本

F05、F07–F10、F14–F20尽量只补前端表达和完整映射。`Order`现有列是权威，不能在AirMaster或单证JSONB再建同名“业务副本”。AE适配器`web/src/pages/orders/order-kinds/air-export/form-adapter.ts:74`可扩充当前字段集合；金额保持字符串，DatePicker输入/ISO输出与详情dayjs回显分别处理。不要在全局拦截器按名称补默认值，也不扩大到全仓映射重构。

普通更新当前是optional字段协议，`server/internal/service/order_write.go:273`等只在请求字段出现时改值；“清空”应使用各类型明确合同（可清文本显式空串、枚举明确UNSPECIFIED/清除意图），不能把所有空值变undefined后声称支持清空。新增UI之前要逐项核对已有清空规则，必要时同步Proto和全部调用方；非空字段验证仍须服务端执行。

货值/保费的验证基线见`server/internal/biz/order_usecase.go:418`与`:690`；货币引用见`server/internal/data/order_sync.go:49`。业务货值、投保保费、AWB运输/海关声明价值、单证保险金额、单证运费与财务应收应付是不同事实，不因同为金额共享一列或自动联动。

### 2. 进口承接与事实采用固定的每票结构

建议`Order.importHandling`消息承载`exchangeAgentId`及固定steps；每个step的kind限五种，带`undertaken`、状态、`completedAt`、责任人和本票备注（具体字段名待设计）。`exchangeAgentId`是同公司Partner供应商引用；`completedAt`是换单事实，预计换单若批准另设`plannedAt`，不将一字段同时解释为计划/实际。到货实际时间与主运输实际到达若共用权威事实需在设计中明说，不能由每张票重复覆写共享到达时间。

仅在SI/AI请求和响应出现该消息；普通订单保存受订单expectedVersion/锁门禁；完成环节命令同样绑定预期版本，写入与事件共事务。批量服务设置优先一个明确保存协议，不散成五套页面状态。未承接步骤以不适用展示，不能推进为已完成；取消已完成环节/服务范围调整的原因和可操作条件仍需最终设计确认。

实现方式可用固定类型JSONB加事件或薄的每票步骤表；这涉及Schema与事务设计，主会话须在最终设计中选定，不以本建议自动生成。不得恢复已reserved的`service_type_ids/status_template_id`字段名或编号。

### 3. 空运运输、单证与本票内容分开

- AirMaster保存同方向共同运输、MAWB结构化内容和主单申报事实；AirLink保存DIRECT/HOUSE及当前/历史关联；HAWB仅HOUSE存在，保存本票结构化内容。主单与分单印字可由相同航空字段定义描述，但数据实例、版本与保存目标独立。
- 新增航空结构化消息应一次性替换自由文本合同，保留存量文本为明确备注的一次性迁移，不双读、不自动回退。DTO用optional house消息表达无HAWB；不能以字段默认空值推断直单。
- 新单证字段保存覆盖普通订单入口、制作页、版本历史、内容哈希和审核快照，防止制作页保存成功后整单保存又清空。共同单证修改按主单完整成员权限、版本及锁复验；本票私有字段不误改其他票。
- 签发主体、签发时间/地点、代理印字、条款、长货描和PDF格式属于单证。MAWB/HAWB与进口外部单证的“登记为已签发”与本系统“办理签发”不能混淆；本期输出是根据已保存资料制作文件，不默认外部电子签章或通道成功。

### 4. 验收顺序

先实现真实字段/契约，再补风险匹配验证。基础字段重点测四类新建→详情→编辑→清空→刷新；ProtoJSON省略零值/optional未设/真实必填缺失；外部变化保输入。新增Schema走正式迁移，保留开发数据；Proto及客户端通过生成器同步。

航空按DIRECT/HOUSE和AE/AI组合验证正文、签发、锁、恢复、完整成员统计和重量来源，追加PDF实图与提字/分页验证。进口按五环节适用子集验证；财务沿用当前公司、出口ETD/进口ETA，不因UI补字段改变金额或经济日期。

导入/导出、长度扩大、复杂拆拼改配以及多航段的待选项，必须在最终PRD收敛后再进入实现清单。表中“补全”只是对已确认目标的具体建议，不代表用户逐字段已批准。

## 本稿最终复核状态

范围问题已按用户后续回复同步：多航段/复杂空运拆拼改配与导入后续，完整CSV本期；直单一主一票不混HOUSE。F06保留现有长度，F13只记录实际换单时间，F35航空长文上限，F37A4与真实主体，F32模式创建后固定，均作为父design.md的具体提案交最终复核，不逐项重复询问已批准范围。表中每项的字段归属和验收仍须实现后验证，不能因为本表已写完就声称功能完成。
