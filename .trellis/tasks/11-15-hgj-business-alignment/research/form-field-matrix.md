# 调研：四类海空运新增表单字段与现有系统差距

- 问题：四类订单新增表单有哪些分节、字段、控件格式、选项和默认展示？与本仓库已实现 SE/AE 相比，哪些是表达缺口，哪些已有底层契约但未在 AE 表单提供？
- 范围：混合；读取既有 REA 成功证据和当前源码，不操作外站、不读取凭据、不修改产品代码。
- 日期：2026-11-16（北京时间）。

## 证据与方法

| 简写 | 模块 | REA 证据 | 本机原始文件 |
| --- | --- | --- | --- |
| HG-SE | 海运出口新增 | `ev_a7a90f9ec3771fc263165537698491a22f620b0f1230517aff76e8d8b9630a46` | `sea-export-create.evidence.json` |
| HG-SI | 海运进口新增 | `ev_947417afe5f26ac78758eafa96b7a32c8fc4d888a7c0eca84a810da9586ab764` | `sea-import-create.evidence.json` |
| HG-AE | 空运出口新增 | `ev_b2c44ae4d0a8651260dbe99eff9b4f8ecea5ef834222b432f48460773a75bcbe` | `open-air-create.evidence.json` |
| HG-AI | 空运进口新增 | `ev_5f159b2b5218f2acc007c9ce6235860c3c643acfcfaca1bf8e2298dca05f2081` | `air-import-create.evidence.json` |

原始目录为 `/home/vince/.local/share/rea/hgj/order-research/`；各证据完整性摘要见同目录任务研究 `create-form-observations.md`。本报告只保留字段与规则，不复制个人、客商或单据数据。

本次用标准 HTMLParser 按 DOM 祖先关系提取控件及最近字段标签，并检查自身/祖先的内联 `display: none`、`hidden`，区分隐藏弹窗。没有使用浏览器计算样式、控件边界框或截图逐项核实，因此下文“控件出现”只证明成功证据中有对应 DOM；“没有内联隐藏标记”也不等于已证明视觉可见。选项下拉往往已经隐藏挂载，不能把整段 DOM 文本当作当前选中值。

## 文件与源码模式

| 文件 | 用途 |
| --- | --- |
| `web/src/pages/orders/templates/air-template.tsx:24` | AE 五分节入口；新建与详情共用 |
| `web/src/pages/orders/templates/components/air/AirMasterFields.tsx:30` | MAWB、航司、机场、单组航班与时刻 |
| `web/src/pages/orders/order-kinds/air-export/form-adapter.ts:47` | AE 公共字段序列化；部分字段可序列化但未在当前 AE 表单渲染 |
| `web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx:636` | SE 业务、客商、货值/保费/工厂等字段 |
| `web/src/pages/orders/templates/components/sea/SeaTransportSection.tsx:626` | 海运时刻、航程、港口及截止时间 |
| `web/src/pages/orders/templates/components/sea/SeaBillContentFormFields.tsx:555` | 海运结构化收发通知人等单证字段 |
| `server/api/order/v1/order.proto:568` | 通用创建 DTO；不少基础字段可复用，不应重复建表 |
| `server/api/order/v1/air_export.proto:39` | AE 权威主单内容，只含一组航司/机场/航班/时刻 |
| `server/api/order/v1/air_export.proto:69` | 目前 HAWB 只有号码、状态和自由文本 |
| `server/internal/biz/order_usecase.go:321` | 只开放 SE/AE+EXPORT，进口当前没有真实创建闭环 |
| `server/internal/biz/order_usecase.go:365` | 基础字段服务端长度与数值校验 |

“未在 AE 表单渲染”以 `getAirTemplateSections` 所返回的组件及其实现为边界；不等于后端字段不存在，也不等于任何隐藏入口都不可能编辑。

## 分节和四类差异

| 维度 | HG-SE | HG-SI | HG-AE | HG-AI | 本系统差距/复用 |
| --- | --- | --- | --- | --- | --- |
| 主分节 | 业务信息、配舱信息、提单信息、内部信息 | 同左 | 同左 | 同左 | 本系统 AE 为业务与客户、订舱与运输、本票货物、分单信息、内部人员五卡；分节名称不同不是功能缺失，见 `air-template.tsx:24` |
| 托运类型 | 整箱、拼箱、散杂候选 | 相同候选 | 新增 DOM 未见此组 | 新增 DOM 未见此组 | SE 已有 shipmentType；不能把海运箱型语义强加空运 |
| 海空承运信息 | 船公司、船代、船名、航次 | 同左，另有换单代理 | 航空公司、航班号及新增入口 | 同左 | AE 已有真实航司/机场；SI/AI 需要明确共享运输模型 |
| 港口/机场展示名 | 起运港、目的港、卸货港、中转港、中转港代码 | 同左 | 起运港、目的港、卸货港、中转港代码 | 同左 | HG 空运仍使用“港”字，不证明海港数据源；本系统 AE 以机场 ID 为权威 |
| 截止/换单 | ETD、ETA、截单、截关 | ETD、ETA、换单时间 | ETD、ETA、截单 | ETD、ETA、换单时间 | AE 现有截单/报关截止；进口换单时间需要独立业务语义，不将其当出口截单改名 |
| 重量 | 委托/实际件重尺 | 相同 | 委托/实际件重尺、体积重、泡重、分泡重、计费重量 | 相同 | AE 已有重量确认，缺的不是再做公式框，而是单证与采用值的完整表达 |
| 特殊项 | 海运英文品名区有免责条款选项 | 同左 | 对应位置未见免责条款控件 | 同左 | 不从 HG 海运表单直接推导 AWB 免责格式 |

## 公共业务字段差距矩阵

以下“四类”均指 HG 四类成功新增 DOM。SI/AI 本系统仍被服务端门禁拒绝；表中“已有”主要描述通用契约与 SE/AE，不能据此宣称进口已支持。

| 字段/行为 | HG 观察 | 本系统当前证据 | 判断与建议 |
| --- | --- | --- | --- |
| 订单编号与编号时间 | 显示保存后自动生成，时间仅初次保存前可改的提示 | `air-template.tsx:40`、`:47`；SE `SeaBasicInfoSection.tsx:636`、`:645`；AE adapter `:129` | 复用现有编号；HG 提示不是服务端不可修改的验证 |
| 报价编号/引用报价 | 提示未引用的禁用字段和引用入口（SE 此次字段快照未稳定出现） | 通用订单 DTO 未见报价关联字段，需另查报价领域 | 仅入口观察，不能承诺可自动带入费用或审批结果 |
| 委托单位/代码 | 客商选择；单位代码为展示项 | AE `air-template.tsx:63`、`:70`，SE `SeaBasicInfoSection.tsx:558`、`:661` | 已有公共能力，复用服务端候选和人员默认带入 |
| 集运/跨境 | 两个单选候选 | SE `SeaBasicInfoSection.tsx:671`；AE 五卡未渲染 shipmentMode，AE adapter 未序列化该字段 | AE 是表单/映射缺口，不能误报全仓缺字段 |
| 服务类型 | 多选；进口包含换单/仓储/报检，空运含包板 | `order.proto:294` 保留已移除 `service_type_ids` 名称，CreateOrderRequest 无当前服务项字段 | 不能恢复旧标签/服务字典来直接对齐；需要确认具体业务动作是否进本期 |
| 货物品类 | 普货、冷藏、超限、危险品、散杂件货等候选文本 | AE `air-template.tsx:79`、`:85`；公共 cargoCategoryIds | 已有候选能力，选项和值需来自本系统主数据，不复制 HG 内部 ID |
| 客户业务编号/企业内部编号 | 两项 maxlength=250，旁有重复校验入口 | AE `air-template.tsx:49`、`:53`；`order_usecase.go:365` 服务端上限 100；`SeaBasicInfoSection.tsx:114` 输入上限 100 | 明确长度差异，查重唯一范围未验证；不能只改 UI 为 250 |
| 贸易条款 | 四类均有选择控件 | 通用 `order.proto:572`；AE adapter `:81` 已往返，但 AE 五卡未提供可见选择项 | AE 字段展示缺口，应先确认 UI 和保存意图；不是后端从零新增 |
| 订舱代理/国外代理 | 四类均有选择控件 | SE `SeaBasicInfoSection.tsx:740`、`:750`；AE adapter `:82`、`:83` 已往返，但 AE 五卡未提供对应字段 | 可复用领域选择能力，不恢复逐页面请求竞态写法 |
| 换单代理 | HG-SI 业务分节出现选择项；HG-AI 未见同名项 | 当前通用创建 DTO 未见明确 exchange/release-agent 独立字段 | SI 需定义角色和引用关系，不能静默借订舱代理保存 |
| 合约号 | 四类 maxlength=50 | 通用 `order.proto:601`，服务端 `order_usecase.go:365` 上限 100；AE adapter `:84` 支持但五卡未提供控件 | 现有支持长于 HG，不应为表面对齐缩短并截断 |
| 货值/保费及币种 | 货值币种选择；保费控件存在，货值/保费文本 maxlength=-1 | SE `SeaBasicInfoSection.tsx:792`、`:806`；通用 `order.proto:602`、`:603`、`:606`、`:607`；AE adapter 当前未传上述金额 | AE 若纳入应同时补 UI 和映射；`-1` 不能作为后端无上限证据 |
| UN NO./CLASS NO./截申报 | 四类有文本与时间控件；UN/CLASS 文本 maxlength=-1 | AE 复用 `SeaDangerousGoodsFields`；`SeaBasicInfoSection.tsx:216`、`:237`、`:247`，UN 输入 4，CLASS 16 | 保留本系统明确校验；HG 未验证格式不能取代危险品规则 |
| 包装类别 | 本轮四类 HG 新建 DOM 未见同名字段 | AE `air-template.tsx:88` 明确 showPackingGroup，危险品组件 `:258` | 本系统已有能力，不因 HG 未显而删除；IATA 适用范围以现有规范为准 |
| 接单时间/货好时间/工厂 | 四类出现，工厂 maxlength=200 | SE `SeaBasicInfoSection.tsx:823`、`:828`、`:836`；通用 `order.proto:611`、`:614`、`:610`；AE adapter 未映射这些字段 | AE 表达/映射缺口，不能只依赖 DTO 已有而宣称支持 |
| 订舱/配舱/操作备注 | 三个 1000 字 textarea | 通用 `order.proto:615` 起；AE adapter `:100`–`:102` 支持，但 AE 运输卡只显式展示 operationNotes（`AirMasterFields.tsx:461`） | 订舱/配舱备注需补可编辑入口；保存映射已有 |
| 内部八类人员 | 创建/操作/业务/客服/关联/单证/商务/关联2；创建人员控件禁用 | `SeaPersonnelSection.tsx:111`–`:139`，AE `air-template.tsx:127` 共用 | 当前已有对应角色骨架；不能凭 HG 人员下拉 DOM 扩大组织数据范围 |

## 运输、货量与单证字段矩阵

| 字段 | HG DOM 约束/候选 | 本系统当前证据 | 差距/保留规则 |
| --- | --- | --- | --- |
| 海/空主分单号 | 四类 maxlength=50；空运仍用 MBL/HBL 占位文字 | AE MAWB `AirMasterFields.tsx:48` maxLength 20，并校验归一后 11 位 mod7；HAWB `air-template.tsx:136` maxLength 64；后端 `air_export.go:190`、`:205` | HG 空运占位文字不代表海运号规则；主分单必填/唯一性/格式继续按本系统已确认契约 |
| 航空公司 | 空运 maxlength=100 autocomplete | AE 航司查询 `AirMasterFields.tsx:74`，权威字段 `air_export.proto:41` 为 airline_id | 保留主数据 ID，不退回自由文本 |
| 航班号 | 两空运 maxlength=10，有新增入口 | AE `AirMasterFields.tsx:127` maxLength 40；`air_export.proto:44` 单一 flight_no | “新增”可能多航班但未执行，不能宣称已证明多航段关系；多航段仍待产品范围决定 |
| 起运/目的/卸货/中转 | HG 控件 maxlength=100，中转代码=10 | AE `air_export.proto:42`、`:43` 只有起运/目的机场；SE `SeaTransportSection.tsx:742`、`:756`、`:771` 多地点字段 | AE 卸货/中转表达缺口需先确认是否本期多航段或单次经停；不能复用海港 FK |
| 船名/航次/码头 | 海运 50/50/30 | SE `SeaTransportSection.tsx:681`、`:714`；通用 vessel_voyage 长度 100（`order_usecase.go:365`） | 码头独立语义待对照实体，前端拆分字段不等于独立后端类型 |
| ETD/ETA/截止 | 新增 DOM 只见文本输入及名称，未证明格式/时区/日期边界 | AE `AirMasterFields.tsx:148`、`:155`、`:186`，另有实际起降；`air_export.proto:45`–`:50` | 已有时刻权威主单，不将未验证 HG 日期控件作为本系统 UTC/本地契约 |
| 发货人/收货人 | 四类 textarea maxlength=1000，支持企业资源搜索提示 | SE `SeaBillContentFormFields.tsx:555`、`:625` 结构化内容；AE `house_content` 自由文本 | AE 结构化单证为明确差距；不能用公共简称字段替代完整收发货文本 |
| 通知人/第二通知人 | 四类有页签标签和当前通知人 textarea；本轮没有切换第二通知人页签 | SE `SeaBillContentFormFields.tsx:710`、`:718`；AE 无结构化对应项 | 第二通知人的独立限制/保存语义未验证，不能从当前 textarea 推定两者相同 |
| 单证国外代理 | 四类 textarea maxlength=1000 | SE `SeaBillContentFormFields.tsx:771`；业务层 foreignAgentId 与单证文本不是同一字段 | AE 若制作 AWB 要区分代理主体引用和印字快照 |
| 唛头 | 四类 textarea maxlength=10000 | SE `SeaBillContentFormFields.tsx:859`；后端 `sea_document.go:18` MaxSeaPartyTextLength=5000；AE 无独立 marks | 对齐长度需跨 Proto/校验/打印完整考虑；不扩大字段后再在 PDF 截断 |
| 英文品名 | 四类 textarea maxlength=5000 | SE `SeaBillContentFormFields.tsx:918`；单证服务端 MaxSeaDescriptionLength=10000；公共 goodsDescription 仅 1000（`order_usecase.go:365`） | 必须区分委托描述和单证印字；AE 仅有公共 1000 字描述加自由文本，未有结构化 AWB 印字 |
| 委托/实际件数、毛重、体积 | number 控件，包装单位选择；文本显示 KGS/CBM；DOM 未给统一 min/step | AE `air-template.tsx:95`、`:105`、`:113`；实际货物在详情扩展；SE `SeaBillContentFormFields.tsx:1049` 起 | 本系统已有两来源，保持实际与委托分离；HG 同页数值不证明实际从货物明细推导 |
| 体积重比率 | 空运候选 2000、10000/3、5000、6000、7000；候选下拉隐藏挂载 | 现有确认模型接受明确正除数，`air-export-contract.md:68` 起 | 可作为快捷候选，不能运行任意表达式；HG 舍入与表达式求值未验证 |
| 泡重/分泡比例/分泡重 | 空运候选无分泡、一九/二八/三七/四六/五五、1/3、2/3；参考显示数值 | 现有独立分子/分母及来源指纹 `air_weight.go` | “一九”的哪个数用于分泡未验证，不能按名称映射后端比例 |
| 计费重量和单位 | 空运 number+单位下拉，DOM 有 KGS/CBM 选项文本 | 现有确认只开放 KG，费用显式采用确认重量（`air-export-contract.md:77` 起） | 不因为 CBM 选项存在就把体积当重量；单位切换的业务含义仍未知 |
| 付款方式/运输条款/提单形式 | 四类有选择/文本；运输条款 maxlength=50；付款候选含英文预付/到付与中文别名 | SE `SeaBillContentFormFields.tsx:1260`、`:1290`、`:1314`；AE 通用 paymentTerm 为隐藏字段，`SeaCargoSection.tsx:34`，adapter `:80` 缺值取 1 | AE 需区分业务付款枚举与 AWB 印字，不能仅有隐藏默认值而声称用户可编辑 |

## 控件限制与默认展示的可信度

1. 四份快照中，可从 `el-radio__input is-checked` / `el-checkbox__input is-checked` 类名观察到：四类均显示集运、普货；两海运显示整箱；两出口显示订舱服务项；两进口显示换单服务项。这些是**该已登录用户、该时点的 CSS 状态**，不是干净账户的默认值合同，也没有验证保存结果。
2. 上述选中控件的原始 input 未携带 HTML `checked` 属性；不能只查 `checked` 属性就断言全部未选，也不能把 CSS `is-checked` 当数据库字段值。
3. 海运箱型相关隐藏弹窗中另有“仅一次”/“保持同步”单选；其祖先含 `display: none`。这不代表当前订单采用了任何同步规则。表单下方 AI 助手、OCR、弹窗的输入框不属于业务字段，不进入缺口列表。
4. 创建人员有 disabled/readonly 组合；普通下拉内部 input 的 readonly 只是禁止直接输入，不能推定整个下拉不可选择。
5. `maxlength=-1` 不构成有效的业务最大长度；numeric input 未出现 maxlength 也不代表可无限精度、负值或任意大数。服务端规则必须另行验证。
6. `value` 属性不一定反映框架运行期 DOM property，因此未把 input 空属性认定为空默认值。货值币种、重量单位、体积比率、分泡比例、日期、运输/提单条款的初始选中值均未独立验证。
7. 本系统明确 AE 默认值可从代码证明：`air-export/form-adapter.ts:108` 起为 paymentTerm=1、NEW 主单、CTNS 包装单位、可用 GENERAL 品类；HAWB 初始 DRAFT 见 `air-template.tsx:148`。这些是本系统代码事实，不是海管家默认值。

## 已确认单证范围与优先差距建议

用户已确认：本期主、分空运单均结构化，并支持制作、预览和 PDF 打印。具体版式、字段归属及进口单证维护/签发语义仍待研究和设计确认；无 HAWB 直单随后已由用户另行批准；多航段及复杂改配/拆拼仍待范围决定。以下为实施建议，不构成最终设计批准。

- F1：先完善 AE 当前已有基础 DTO 字段的编辑与完整往返：贸易条款、代理、合约、货值/保费、工厂/接单/货好时间、订舱与配舱备注。实现前逐项确认业务用途，不能只给表单增加字段却漏保存映射。
- F2：按已确认范围，将主、分空运单从自由文本升级为独立结构化内容，支持制作、预览和 PDF 打印；区分业务引用、印字快照、主/分单归属和版本。具体版式与进口签发语义尚待研究，最终设计仍须审批。
- F3：SI/AI 不套用出口“订舱/签单”的字面流程；换单代理与时间、到货/提货等进口事实需要进一步详情取证和状态设计。
- F4：保留本系统已有重量确认、主单共享、历史及财务来源机制，按需要增加候选/信息表达；不为了外观相似丢失当前正确性。
- F5：长度差异分别决策，不机械向 HG 对齐。至少覆盖业务编号 250 对 100、唛头 10000 对 5000、公共货描对单证货描不同上限，以及后端/打印往返。
- F6：用户已另行批准直单/主分单和进口按服务全链；具体服务模型、直单共享/转换约束、多航段及空运改单/拆拼不能仅由表单观察代为决定。

## 相关规范与外部引用

- `.trellis/spec/server/backend/air-export-contract.md`：主分单身份、重量来源、历史、锁与费用来源。
- `.trellis/spec/server/backend/sea-export-document-contract.md`：海运操作票、主单、运输与单证边界。
- `.trellis/spec/server/backend/sea-hbl-production.md`：版本化打印稿与 PDF 完整输出。
- `.trellis/spec/web/frontend/type-safety.md`：DTO 默认值、optional、映射往返。
- `.trellis/spec/web/frontend/state-management.md`：服务端状态、并发提示和输入保护。
- 外部参考仅为上表既有 REA 证据对应的 `https://eyun.hgj.com/order/edit_{sea_export,sea_import,air_export,air_import}/`，没有新访问、没有引入外部库版本建议。

## 限制与尚未找到的证据

- 只读离线解析四份新增 DOM；未验证计算样式、交互可见性、实际服务端写入、唯一性范围、数据库索引、打印版式及权限组合。
- 未读取请求/响应业务载荷；所有 HG 字段是页面层事实，不能直接当 DTO 名称。
- 没有切换通知人页签、变更重量比例/单位、增加航班、操作引用报价或点击保存；对应结果保持未知。
- 当前进口仍无真实闭环；即使公共 DTO 或单元测试可构造进口字段，也不证明进口产品可用。
- 没有运行任何测试或生成命令；本报告不能用于宣称现有代码验收通过。
