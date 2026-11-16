# 研究： 四类订单筛选能力与 HGJ 差异

- 研究问题： 参考 HGJ 实际查询设置，判断本系统四类订单缺少且值得补充的筛选项，区分展示缺口与契约缺口。
- 研究范围：仓库内部及 已有本地 HGJ 捕获；未操作共享浏览器，未访问或修改业务记录。
- 日期： 2026-11-16

## 研究发现

### 证据与限制

- 本系统：精读 `web/src/components/ui/order-list-template/OrderListSearchFilter.tsx:34`、`:329`、`:425`、`:485`；`web/src/pages/orders/list-query.ts:26`；`server/api/order/v1/order.proto:522`、`:565`；`server/internal/service/order_query.go:142`；`server/internal/biz/order_types.go:385`；`server/internal/data/order_query.go:95`；相关 Ent Schema 与航空权威查询。
- HGJ 历史四类列表捕获在 `/home/vince/.local/share/rea/hgj/order-research/`。四份 summary 的 evidence_id：SE `ev_9c30968c89b2c1543bbaf9ed6120eb21851f2479ae9bcbd8ad743c4a86b063bf`；SI `ev_d3e52191a3677c1c789c3d247296e5168ca74d4de6d7c324ca603b7227c376b4`；AE `ev_f25a61c1a6c8113ade0c1acab01affe6d9cebb031b880b66af4520b858fb7397`；AI `ev_42d52cedc3d762c6eebf5ce151f82d5efe7941d0d64038e2b0853dd83ad97fe2`。本研究于本日读取历史捕获，仅使用列表筛选栏、到查询设置按钮以前的控件文字；未将之后的表头名称当成筛选。
- 本轮 SE 初始页面证据 `/home/vince/.local/share/rea/hgj/filters-01a1e359/sea-export-initial.evidence.json`，id `ev_f04efcf1d4f06253d9b0d34500cdc2773d67be194859352ede07f9bfd33e4061`。它证明当前存在查询设置入口，不能证明所有可配置条件及实际过滤结果。
- 静态运行产物 `static-analysis/chunk-104ad466.18f26004.js`，组件 `SeniorSearch`：`getShowField`（附近调用 `listQueryFields({tableName})`）从服务端取 `fieldList`，`SearchSetting` 接收 fieldList 和 setFieldSort；因此仅靠 JS 里出现某个字段控件不能断言某品类实际允许该筛选，仍需当前查询设置弹窗或对应字段 API 的证据。
- 本轮已取得四类实际查询设置成功证据，字段级目录 `/home/vince/.local/share/rea/hgj/filters-01a1e359/field-catalog.json`：SE 110、SI 102、AE 94、AI 91项。对应证据ID见下表。前两个 settings/settings2 是失败尝试，不作为成功证据。仅观察同账号当时可选字段，selected_at_observation不是系统默认；未保存，也未逐字段提交验证过滤实效。

| 品类 | 当前查询设置成功证据 | 字段数 |
| --- | --- | --- |
| SE | `ev_3046759687bb228d16408d9cea0e8ac3592b32d4a3945f20ca7cf7c7684918a0` | 110 |
| SI | `ev_0cf66e5829483d8e24f729195bec2c31e71f2178cfc70ae7a6a35a70588e8f07` | 102 |
| AE | `ev_fbec13fadee4458750dfea88e6fa8a698186e818d367706c8e4cfbe9c51e1bf8` | 94 |
| AI | `ev_4f04156aa09d5c8e1f5fb7aee7a5b8e82f5f95c3c057f4162cab72de58d8d972` | 91 |

- 捕获文件包含真实业务记录，不复制到 Git；报告只记录字段级脱敏结论。未提交目标 bundle、原始响应或截图。

### 当前四类订单实际已支持

当前四品类共用一套筛选 UI，航空模式只改变地点标签并隐藏船公司；没有航空公司替代项。导出复用 `buildOrderListRequestFilters`，后端共用 `orderListQuery`，新增筛选必须同步列表与完整导出。

| 字段／组合 | 适用品类 | 控件与匹配 | 前端→后端依据 | 规划建议 |
| --- | --- | --- | --- | --- |
| 订单号／主单号／加拼主单号／客户业务号／Booking No | SE/SI/AE/AI（各类业务无值时为空） | 类型选择+文本，包含匹配；主单走真实关联 | UI:329；proto:532；service:178；data:100 | 保留常用复合单号，类型及值作为一个可设置单元，移除一起清除 |
| 委托单位 | 四类 | 远程客商单选，ID精确 | UI:359；proto:528；data:166 | 常用 |
| 起运港／目的港，起运机场／目的机场 | 四类 | 远程单选ID精确 | UI:374、:396；data:170；air_order_query.go:46 | 常用；航空按最新真实主单地点含终止关联，不回退旧投影 |
| ETD、ETA | 四类 | 日期区间 | UI:418、:445；service:199；data:123；air_order_query.go:39 | 出口ETD、进口ETA优先常用，另一个可选 |
| 创建时间、状态时间、锁定时间 | 四类 | 日期区间；状态时间=存在区间内生命周期事件，并非当前状态更新时间 | UI:439；data:115、:125、:136 | 创建常用，其余可选；展示名称要说明语义 |
| 船公司 | SE/SI | 远程单选ID精确 | UI:464；proto:546；data:175 | 常用或可选，空运不能用shippingLineId冒充airlineId |
| 发货人简称、收货人简称 | 四类 | 文本包含及拼音检索键 | UI:482、:487；data:178、:181 | 可选 |
| 进程（未退关、已退关、已完结） | 四类 | 枚举映射termination/closure | UI:492；list-constants.ts:3 | 保留常用；不等同具体业务进度flowStatus |
| 操作、业务、客服、创建人员 | 四类 | 当前品类授权员工候选，按人员ID精确 | UI:300、:510；service:223；data:184 | 操作/业务常用或可选，客服/创建可选 |
| 是否锁定、分享状态 | 四类 | 全部/是/否三态 | UI:528、:541；proto:559；data:188、:195 | 可选；false明确提交，不能省略成全部 |

HGJ 历史可见栏与本系统重合项包括创建时间、复合单号、进程、ETA、起运港；AE显示航空公司和目的港；AI显示航空公司和起运港。以下缺口已用本轮字段目录核对：航空公司、海运船名/航次、订舱/国外代理、贸易/付款/托运类型、合约/内部编号、分单、箱号均在相应实际查询设置中出现。航空航班号**未在四类查询设置目录发现**，它是本系统已有空运模型的业务补充建议，不得写成HGJ筛选已见。

### 后端已支持但前端尚未提供

| 候选 | 价值、控件与匹配 | 适用 | 推荐 | 源码与最薄影响 |
| --- | --- | --- | --- | --- |
| 是否有当前异常 | 快速找待处理异常；全部/有/无 | 四类 | 常用可选项 | proto has_active_exception:530；service:155；data:151 从ACTIVE异常关系过滤。前端加字段、参数映射即可，列表/导出一起传 |
| 具体操作／进口服务进度 | 不用打开订单就能找某个待办环节；枚举精确 | 四类，选项按品类 | 可选 | proto flow_status:526；service:145；data:142；schema/order.go:72。有实现基础，不能混成“进程”替换，进口与出口选项不同 |
| 综合关键词 | 便捷搜索订单号、旧船名航次、货描、拼音索引、航空主分单号 | 四类 | 可选，但不要替代明确字段 | proto keyword:525；service:144；data:98。前端未在buildOrderListRequestFilters透传；加可选输入即可。现有keyword不保证船舶航次最新主单来源，也不匹配airline/flight_no，不应宣称涵盖这些精确候选 |

### 数据已存在，但前后端筛选契约均缺少

| 候选字段 | 业务价值 | 控件／匹配 | 适用 | 推荐与HGJ依据 | 源码与最薄补充影响 |
| --- | --- | --- | --- | --- | --- |
| 航空公司 | 按承运人批量安排空运操作；现有船司字段对航空完全隐藏 | 远程航司ID单选，服务端精确 | AE/AI | 常用；HGJ两空运栏实际已见 | schema/air_master_waybill.go:23 有airline_id；UI:464 仅非air显示船司。新增airline_id至List/Export proto→service→biz→orderListQuery；候选复用现有航司主数据。按最新真实航空主单关联及同公司过滤，与地点/日期口径一致 |
| 船名、航次 | 快速定位同船同航次所有订单 | 文本不区分大小写包含，独立字段或明确组合 | SE/SI | 可选，HGJ海运查询设置已见 | schema/sea_transport_execution.go:24、:25；order.go:101仅旧组合投影。新增专用参数，沿真实活动或业务权威航程查询；不能以keyword旧投影等价替代 |
| 航班号 | 依据同一航班批量跟进和异常处理 | 文本包含，大小写规范化 | AE/AI | 可选，HGJ查询设置未见，按本系统业务价值推荐 | schema/air_master_waybill.go:24；新增List/Export字段+最新主单关联过滤；现有keyword未查flight_no |
| 订舱代理 | 同代理对单和舱位跟进 | 当前公司代理客商远程单选ID | 四类（按表单实际业务适用） | 可选 | schema/order.go:48；已有booking_agent_id。新增筛选契约+ID条件，候选沿现有客商权限与角色，不独立遍历全量 |
| 国外代理 | 代理对账和海外操作筛选 | 当前公司国外代理单选ID | 四类（按已有订单字段） | 可选 | schema/order.go:49；新增foreign_agent_id契约+ID条件 |
| 船代 | 进口放货等船代跟进 | 公共船代字典单选ID | SE/SI | 延期可选，先确认实际使用频度 | schema/order.go:50；已有shipping_agent_id，不能与船公司或订舱代理混同 |
| 付款方式 | 区分预付／到付的操作关注 | PREPAID/COLLECT枚举精确 | 四类 | 可选 | schema/order.go:68；新增payment_term optional enum过滤，不能用0值表示全部 |
| 贸易条款 | 按FOB/CIF等责任边界找业务 | 枚举精确 | 四类 | 可选 | schema/order.go:67；新增trade_term optional enum过滤，选项复用已有common，不另建字典 |
| 运输类型（整箱／拼箱／散杂货） | 箱货操作分组 | shipment_type枚举精确 | SE/SI | 可选 | schema/order.go:69；新增shipment_type筛选，同步List/Export |
| 箱属（COC/SOC） | 自备箱专项操作 | container_ownership枚举 | SE/SI | 延期可选 | schema/order.go:70；小成本契约扩展，但未确认HGJ筛选或实际必要性 |
| 合约号 | 查同一客户／船司合约业务 | 文本包含 | 四类 | 可选 | schema/order.go:51；新增contract_no过滤，避免借复合单号类型混淆 |
| 分单号（HBL/HAWB） | 用户常拿分单号找票，现有复合单号没有此类型 | 新增house号码类型，包含；海运规范化号、航空保留业务符号规则 | 四类有HOUSE时 | 可选，较高价值 | schema/sea_house_bill.go:22、:23；air_house_waybill.go:19；biz/order_types.go:228目前只有5类号码。新增number enum类型和关联谓词；航空keyword已查house但用户无法明确选分单。不得把customer_reference当分单 |
| 箱号 | 装箱／放货查询常用识别 | 文本包含，规范化大小写 | SE/SI | 可选，HGJ海运查询设置已见，需明确本票箱归属 | schema/order_container.go:22；新增container_no列表/导出参数，关联订单箱及共享箱须明确同一口径；不是新增箱实体 |
| 内部业务编号 | 内部跨系统对单 | 文本包含，可纳入number类型 | 四类 | 延期可选，避免与客户业务号误合并 | schema/order.go:44；新增number枚举及内部号谓词 |
| 货物描述／品名 | 商品专项查询 | 文本包含，拼音视现有索引范围 | 四类 | 延期专用字段，当前综合keyword已部分覆盖 | schema/order.go:108；data:98已查GoodsDescription，无需先新建重复能力 |
| 订单日期 | 与创建系统时刻区分业务日期 | 纯日期区间 | 四类 | 延期，先明确业务日期是orderDate还是出口ETD/进口ETA | schema/order.go:114；现有ETD/ETA支持，不能新增歧义“业务日期” |

以上“推荐”是规划建议，用户尚未批准新增字段清单。建议收敛为下述9组，另有低成本人员补齐备选G10，默认栏保留精简，新增项主要作为可选条件；不照搬HGJ百项。


### 本期建议收敛为 9 组（人员补齐为第10组备选）

| 组 | 本期能力 | 默认呈现建议 | 成本与价值 |
| --- | --- | --- | --- |
| G1 承运人 | AE/AI补航空公司；SE/SI保留船公司 | 常用 | 最大现存品类缺口，只新增查询契约，无新模型 |
| G2 航程定位 | 海运船名+航次、航空航班号 | 可选 | 已有权威实体，关联过滤必须与详情一致；航班号为本系统补充 |
| G3 代理 | 订舱代理、国外代理 | 可选 | 已有订单UUID字段，低成本精确条件 |
| G4 贸易与付款 | 贸易条款、付款方式 | 可选 | 复用现有枚举，无新实体 |
| G5 海运托运类型 | 整箱/拼箱/散杂货 | 海运可选 | 复用shipmentType，空运不展示 |
| G6 号码补足 | 分单号、内部业务编号，继续保留客户业务号和Booking | 放入已有复合号码选择 | 不另挤占多个格子；关联查询+新enum选项 |
| G7 合约 | 合约号 | 可选 | 文本包含，既有字段 |
| G8 异常与具体进度 | 是否有当前异常、具体业务进度，保留原进程 | 可选 | 后端已支持，主要前端接线；不恢复费用独立审核 |
| G9 箱号 | 本票独占箱及真实共享箱分配的箱号 | 海运可选 | 已有完整列表投影，查询要复用同一本票归属，不能泄露其他成员箱 |

| G10 人员补齐（备选） | 单证、商务、关联人员1/2 | 可选，默认隐藏 | 四角色已在OrderPersonnel存在，缺列表查询接线；复用人员谓词即可，无新角色模型 |

最低交付若需缩小可先G1–G8、G9另列后续；是否采纳由最终PRD审阅确定。UN/危险品、截点日期、业务日期等虽有部分已有字段，因频率和语义待定暂不占本期9组。接单时间不能未经确认直接等于orderDate；业务时间也不能把ETD、ETA和创建时间混成一个隐含优先级。

### HGJ其余字段的分组取舍

| HGJ字段组 | 本系统事实／最薄可能补充 | 取舍及适用 |
| --- | --- | --- |
| 关联人员/关联人员2、单证人员、商务人员 | `schema/order_personnel.go:23` 已有ASSOCIATE/ASSOCIATE2/DOCUMENT/COMMERCIAL，但List/Export与UI仅四岗位 | 有模型、前后端筛选均缺；可作为G10补齐人员组，按同一orderPersonnelFilter扩展，复用授权候选；默认隐藏，无需新建角色或重命名客服/操作 |
| 接单时间、业务时间 | orderDate已有，语义与HGJ未验证；业务经营日期已有出口ETD/进口ETA | 延期；仅已有字段不代表口径可直接等价 |
| 货好时间、UN NO.、PG、货值币种 | order.go:53、:56、:60、:62有对应，可补时间区间/文本/枚举精确 | 可作为后续可选；不是默认常用，不新增模型 |
| 截单时间、截关时间、截申报时间 | order.go:63、:104–106与air_master_waybill.go:26有部分对应，但截单细分SI/doc需明确 | 延期；按海/空出口真实截点定义后补区间，进口不机械照搬 |
| ATD/ATA/ETB、驳船ETD/船名、开仓/开港/截港、码头 | 当前订单/主单航程Schema未找到全部同义结构化来源；某些已有业务进程操作时间不能充当实际航行时刻 | 不为筛选新增运输事件体系；待真实业务需求再做 |
| 中转港及代码、卸货港 | order.go:99–100有地点ID，sea_transport_execution.go:22有共同卸货港，航空缺对应中转实体 | 后续海运可选；中转港代码是同一个候选的搜索键，不拆两个同义格子；本票目的与共同卸货严格区分 |
| 目的地 | 当前最终目的港有字段，独立交付地址没有统一来源；规范禁止冒充空列 | 不新增歧义字段；保留现有目的港/机场 |
| SO号、订舱编号 | booking_no已有但HGJ两个名称是否同义未验证 | 继续已有Booking No，不因两个HGJ名称复制两份字段 |
| 换单代理 | SI可能与shipping_agent_id或进口服务provider有不同语义 | 延期，不能未经确认直接映射国外代理/船代 |
| 报关号、日期、贸易国、项号、净重、出境关别、报关行、车牌、车队/仓库 | 某些进口步骤有provider/引用号，但没有已确认统一报关/车队模块与同义查询模型 | 不为筛选建新领域；有结构化来源的具体字段可单独提需求 |
| 海污、闪点、货主箱标记、来源、业务代码一/二、舱位预报编号、匹配/通道状态 | 未发现当前Schema内同义业务闭环；字符串notes不算可检索结构化来源 | 延期，不新造状态/字段 |
| MBL/HBL份数、提单形式 | 现有单证正文JSON可能包含文书信息，不等于有统一订单查询语义；HOUSE/DIRECT模式另有明确业务事实 | 延期；不以HBL存在性简单等于HGJ提单形式 |
| 应收/应付录入状态、费用名称 | 现有OrderFee有真实方向、费用科目及有效性，需EXISTS关联与fee权限 | 有数据，前后端筛选均缺，延期财务专项；如采纳应定义“未录/已录”的有效费用与零金额口径 |
| 应收/应付对账、开票、核销状态/时间 | 有账单/发票/核销事实，但订单可能多费用、多账单、部分分配，不存在现成单一订单状态 | 延期，需聚合语义和权限，不借未建账/已锁定替代 |
| 应收/应付锁定及时间、费用审核状态 | 当前统一人工订单审核锁，不分收/付独立审核生命周期 | 不采纳HGJ旧式双锁；已有isLocked/lockedAt与统一审核事实 |
| 所属公司、协同公司、协同状态、关联主单/拆单、自拼订单号 | 公司为当前工作区边界；共享主单/拆票部分有模型，但跨公司协同和自拼业务不等价 | 不扩组织范围，不把isShared当协同；历史关联查询延期 |
| 委托单位代码 | 现有委托单位选择器应支持代码/中文/英文/别名/拼音检索并按ID过滤 | 保留一个委托单位格子，无需复制独立代码条件 |

### 模型无对应、已退役及不应照搬

| 参照能力 | 分类 | 不采纳理由 |
| --- | --- | --- |
| 业务标签／标签组 | 已退役 | proto:554 reserved tag_ids；根AGENTS与business-tag-retirement规范明确禁止恢复。HGJ列表“添加标签”操作不能当本系统新需求 |
| 独立订单费用审核状态／补录待审筛选 | 已退役旧契约 | List/Export reserved fee_review_status、has_pending_fee_supplement；费用与资料统一人工审核锁，当前是否锁定已覆盖，不能恢复旧生命周期 |
| 铁路／陆运／内贸／自拼专属条件 | 本期无对应业务闭环 | 四类订单范围限定SE/SI/AE/AI，schema枚举有LAND/RAIL不代表可用页面或批准业务 |
| HGJ归档／协同状态 | 本次未确认等价模型 | 不把is_shared当协同、不把closure当归档。暂无明确本系统对应语义，不新建业务状态只为模仿筛选 |
| 往来公司／所属公司独立切换 | 受当前工作区约束 | 业务请求绑定当前公司和权限，不能个人筛选越过授权组织 |
| 数量／重量／利润任意公式查询 | 有部分数据，不等于存在筛选能力 | 未取得实际HGJ筛选证据或当前批准规则，不建立通用查询平台；提成/利润权限须独立核对，不因列表可读就暴露财务金额 |

### 最薄实施路径与验证建议

1. 把现有筛选字段注册为可配置元数据，复合号码和日期范围是原子字段组；定义四品类可用字段，空运默认ETA/ETD与机场，海运有船司。
2. 后端已有字段优先只补UI映射；缺契约的字段先改List与Export proto及number enum，再生成Go/OpenAPI和前端客户端，service只转换校验，biz声明筛选选项，data唯一orderListQuery实现。
3. 航空公司／航班与海运船名航次读取权威实体来源，不借用旧投影或静默回退；空运终止订单读取口径与现有地点/日期一致。多关联含历史记录必须选正确业务来源，不用所有历史曾匹配导致假阳性。
4. 新动态候选搜索复用当前公司授权与后端keyword，覆盖代码、中英名、别名及拼音；不循环翻页拉全量。
5. 验证新字段命中／不命中、false三态、日期端点与北京时间、四品类隔离、两公司／单类权限、已退关关联、重排隐藏清值、列表/统计/导出同口径和超过200行完整导出。
6. 不添加业务状态或新实体以支撑尚未批准的参照字段；已有模型补查询与新增领域能力分开评估。

## Related Specs

- `.trellis/spec/domain/glossary.md` 与 `architecture-map.md`：海运主单航程、航空主单、统一审核、组织边界。
- `.trellis/spec/server/backend/order-list-fields.md`、`order-cross-module-export.md`：列表投影、排序与完整导出。
- `.trellis/spec/server/backend/air-export-contract.md`、`air-import-contract.md`：航空权威来源及类型边界。
- `.trellis/spec/server/backend/business-tag-retirement.md`：标签退役。
- `.trellis/spec/web/frontend/finance-current-workspace.md`：财务当前公司，作为跨模块边界参考而非把财务选择器直接复用到订单。

## 外部引用

只读取本机已捕获HGJ证据与运行产物，未访问互联网文档。该第三方页面仅用于字段与交互观察，不构成后端算法或实际过滤正确性的证明。

## 限制与未发现项

- 已有四类实际查询设置字段集合，但它们仅证明UI允许选择，不能证明权限、匹配算法或查询实效。历史summary只用于补充，最终字段对照依据本轮field-catalog。
- 没有在目标系统执行查询写操作；未执行本系统测试、迁移或真实浏览器验收。
- 大部分新增候选已有数据字段但API筛选契约缺少；不能为了减少改动把它们偷偷映射到keyword或展示本页过滤冒充服务端筛选。
