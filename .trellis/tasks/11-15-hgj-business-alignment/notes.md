# 工作检查点

- 北京时间 2026-11-16：任务创建已批准；本期四类进出口实施，其他五类仅调研。
- 当前 planning，未运行 task.py start，产品代码未修改，最终设计未审批。
- 已完成：现有系统代码盘点和横切影响；九类列表与新增；四类真实详情；空运进口列表/详情/字段设计/流程配置及费用DTO采样；MAWB/HAWB空制作表单；交付前端静态规则。研究覆盖/局限见research报告，尚无写操作、真实PDF、并发验证。
- 本地 Chrome 服务 rea-browser，CDP http://127.0.0.1:19322，当前目标 E09581C2013FA87BA5A4F1B491B6AF43，已登录。不要重复登录或读取输出凭据。
- CLI 需 rea-local-browser 绕过回环代理。REA capture-browser-scenario connect 每次重新导航 start_url，会重置未保存输入；可只打开空表单，不提交真实业务单据。
- 原始目录 /home/vince/.local/share/rea/hgj/order-research/，敏感数据不提交。主页面已开多个业务页签，官网目标 A4C7F2F3F2839A5B34A1F26F2776159E 不参与调研。
- 研究脚本 research-orders.py、research-create.py 等位于本地 hgj 目录；选择器基于当前页面取证，不复用失败选择器。新手引导“我来试试”遮罩已处理，是否出现以当前页面为准。
- 本轮用户新决定：主、分空运单均结构化并制作/预览/PDF打印；进口到货/换单/清关/提货/派送按票服务适用；AE/AI同时支持无HAWB直单和主分单。
- 后续用户已回答：真正多航段、复杂空运改配/拆拼后续；直单一MAWB一操作票不与HOUSE混挂；完整筛选CSV本期、批量导入后续。范围问题已收敛，不要重问。
- 最终稿提案已写父prd/design：航空创建选模式，保存后互转不在本期；一套A4航空版式加续页，真实主体/归档语义；进口固定五环节，Order.version与独立事件；business_type方向索引、SI快照CHECK正式迁移。均待最终摘要之后用户明确实施批准。
- 六个planning子任务已用task.py create --no-start --parent建立并有prd/design/implement/非空上下文；父current保持不变。下一步：等待用户对本轮最终规划摘要的明确批准，然后按依赖start子任务、派trellis-implement/check，不把自动goal续行当实施批准。产品代码仍未改。
- 本轮最重要纠正：*-detail-direct为展示号路由，实际空详情，排除；真实有效详情是*-detail-record。号码span单击只选行，序号td普通div可冒泡到row导航。AI runtime样本 ai-detail-runtime-dto.evidence.json已读真正order/detail。
- 某次 query返回体233754字节，REA最初body-size/allocation限制；最终 ai-query-recorded-dto.evidence.json（ev_8fc4fe467632277c210e6ec8192bc899bc171bf4d4baf98ed1ab4780fdb2186f）显式256KiB/4MiB取得响应。所有运行记录依然有missing events，不能声称全流量覆盖。
- 主分AWB空制作路径：air-import-record-document-maker.evidence.json → mawb-empty/hawb-empty。只打开单个新增表单(id=0)不提交，不走批量新增。源代码已复核初始化无保存，运行只观察读取。
- 静态前端捕获 loaded-sources-expanded.evidence.json（正确CLI用重复 --allowed-origins 参数），107脚本；来源/指纹/字节范围与REA部分语义图见frontend-bundle-rules.md。
- 未获批准操作：在海管家创建/修改真实单据、发送通知、调用税务/报关/收费动作。

## 本轮追加取证与就绪检查

- 四类箱货只读页、SI单证目录和空MBL/HBL制作页、四类模板弹窗已取证，见cargo-and-sea-document-observations.md。
- 全导航响应体抓取耗尽allocation丢事件，不能用这些记录证明模板响应；AI同页hash关闭/重开弹窗成功缩窄捕获 ai-export-template-isolated.evidence.json（ev_18d61bbd94c9e783d7957ae7ce4a1a9b3cb98b87c3325bd21280482b553c419a），保留模板/账户DTO，值只在本机。
- 新研究cargo-document-navigation.md区分读取/本地createBill/真实保存与导出生成。正常打开编辑器不是保存，预览/下载/邮件仍未执行。
- 子任务路径均 .trellis/tasks/11-15-hgj-{order-direction,air-documents,sea-import,air-import,air-waybill-pdf,cross-module-acceptance}。
- 父/六子task.py validate均0；占位符、敏感值模式、22个可解析证据ID/digest与git diff --check通过。最终planning独立复核已通过（coverage-audit追加节），修复F34长度冲突、V5模式与V20 CSV覆盖及PDF依赖摘要；可展示最终规划摘要，但必须等待其后的用户明确实施批准，产品测试未运行。

- 当前等待：用户对最终规划摘要明确批准。下一turn自动goal续行不是批准；若没有用户答案，允许必要只读核对，但不得start子任务/实施。规划门禁不要重复重启建任务。

## 实施批准

用户在最终方案摘要及确认请求之后明确回复“可以”，批准2a96d89e最终方案。解除此前等待实施确认的阻塞，按既定六组依赖开始。不得再次请求相同批准，也不得将既有不清库/外站不写入的边界扩大。

- 实施当前分支codex/hgj-business-alignment；父和hgj-order-direction已in_progress。direction_core代理负责server方向身份/正式迁移/生成，air_common_fields已完成AE公共字段，check_air_common正独立检查；未跑全期最终门禁，其他子任务未start。

## 2026-11-16 当前实施检查点（窗口01a1dff8-3ebc-7380-9095-81092e5cc406结束前）

### 用户授权与完整目标

- 最新真实用户明确消息“可以”：回答最终方案审批，批准提交2a96d89e对应的父prd/design/implement及六组实施。此前blocked/自动goal续行不是批准；现在已真正批准，目标active，不要重复问。
- 完整目标：对标HGJ可观察前后端规则格式，补齐SE/AE，实现SI/AI，其他5类仅调研。已确认主分AWB结构化/制作预览PDF、AE/AI直单一MAWB一票不混HOUSE、进口五服务全链按票适用、完整筛选CSV；高级空运多航段/拆拼改配与批量导入后续。细节以父最终design为已批准合同。
- 用户消息与工具输出在本环境没有可见item id；以本窗口末的“可以”及提交2a96d89e之后的历史定位，不重新收敛需求。

### 仓库当前状态

- cwd /home/vince/roncin-go-admin；分支codex/hgj-business-alignment；从main创建，无worktree。
- 父任务11-15-hgj-business-alignment、当前子任务11-15-hgj-order-direction均in_progress；task.py current指子。其余五子planning，按依赖继续，不因为当前子没完提前start其他。
- 最新提交c2c7435b feat: 补齐空运本票经营字段及显式清空映射。它包含5前端文件、对应spec和用户批准/任务状态记录。之前2a96d89e是已审最终规划。
- 工作区大量server未提交改动全部由direction_core负责，切勿重置或当用户无关改动删除。主会话未改server。前端5文件已提交，server方向API生成/调用方仍可能改前端，但不应覆盖新公共字段。

### 代理状态与责任

- direction_core（trellis-implement）仍运行，独占方向隔离server/Ent/正式迁移/生成物及必要候选调用参数。prompt要求不派子agent、不commit；先实现后测试。需后续等待其真实结果，再spawn trellis-check审查自修，然后主会话规范/提交/记录。
- direction_core已加Sea/Air master和SeaMasterBillVersion的Immutable business_type及方向唯一键、一些biz/data/fixture/seed；正在贯穿Proto/candidate/history/恢复/共享集合。已消息提醒DB CHECK只允许SE/SI、AE/AI、版本快照也保存身份，未填候选类型不能默认SE/AE，不提前放开SI/AI创建假闭环。
- air_common_fields已完成并检查，check_air_common独立修复两个CurrencyAmountInput漏requireAmountWhenCurrency；两定向17 tests、tsc、5文件Biome、antd lint、diff全部通过，已提交c2c7435b。无真实浏览器/API保存验证，仍需最终整组验收。
- acceptance_commands只读research代理已写当前子research/validation-entrypoints.md（123行），涵盖精确Go/Vitest入口、隔离Schema和12files20直接fixture影响；主会话已读。代理因上下文切换忘派工，已消息告知文件满足派工、直接final，无需再探索。该研究未提交。
- 其他早期调研代理completed，无产品实现工作。不必重复调研。

### 本组已实现前端与规范

5文件：web/src/pages/orders/templates/components/air/AirCommercialFields.tsx新，templates/air-template.tsx、templates/air-create-layout.test.tsx，order-kinds/air-export/form-adapter.ts及test。
补shipmentMode、条款/业务付款、业务代理/合约、货值保费币种、receivedAt/cargoReadyAt、工厂和备注。金额原始字符串/零不丢，Dayjs回显ISO提交，更新显式空字符串/枚举零值清空（后端merge支持）；新建optional省略。付款字段可见仅注册一次，readonly全传，本票字段不写主单。危险品原省略语义未改，正文/模式/进口还未实现。
新规范.trellis/spec/web/frontend/air-order-commercial-fields.md（7节）已挂frontend index；记录到当前子research/air-common-validation.md，均已提交。

### 数据库/验证环境

- PostgreSQL 127.0.0.1:5432已ready。当前shell没有DATABASE_SOURCE/RONCIN_INTEGRATION_DATABASE_SOURCE；.env.local有DATABASE_SOURCE和KRATOS_DATABASE_SOURCE。禁止输出值/真实密码。
- 现有server/internal/data/integration_test_helper_test.go用RONCIN_INTEGRATION_DATABASE_SOURCE，CREATE随机roncin_test_iso_* Schema，在其内执行全部正式server/migrations，t.Cleanup只drop自己schema；不Schema.Create、不重置开发数据。
- 已告诉direction_core可用node --env-file-if-exists=.env.local进程内取DATABASE_SOURCE传给go子进程的RONCIN_INTEGRATION_DATABASE_SOURCE，禁止shell打印/命令拼接连接串。无需migrate:dev改现有库。
- Ent生成：go -C server generate ./internal/data/ent。仓库无Atlas diff脚本，正式SQL遵现有迁移惯例并与Ent声明一致。
- 当前未跑check:fast或build，等阶段/最终风险匹配门禁；不能每组无因全量重复。

### 下一步

1. 继续等待direction_core具体实现/验证结果；可只读检查当前diff，不和它争写server/生成物。所有长等待保持每60秒以内用户更新。
2. 若它完成，spawn trellis-check（Active task首行+责任范围+已跑结果）检查方向身份/迁移回填/索引/CHECK/候选必填/直接ID/完整成员恢复/生成一致和真实DB测试，不把SKIP算过。修复后同步实际规范并提交同组源+生成物+正式迁移+tests。
3. 当前子独立验收完成再记录完成/归档，按六子依赖start下一组air-documents（大工作，需继续持久推进）及SI等；主会话协调规格/提交，实现/check默认用Trellis子代理，原生context优先。
4. 父全目标远未完成，不update_goal complete，不再次请求最终规划批准。只有真实新用户业务决策未明才问，已批准的锁序/实体边界不重问。

## 2026-11-16 00:52 UTC 实施检查点（窗口 01a1e1b3-ee77-7b40-8710-4f71df016cda）

### 仍有效的目标与授权

- 最终规划2a96d89e后的用户明确“可以”已经批准六组完整实施；不再问相同实施/提交批准。完整目标仍是SE/AE完善+SI/AI闭环+主分AWB制作预览PDF+四类完整筛选CSV，其他五类仅调研。父prd/design为已批准合同。未完成整个父目标，goal保持active。
- 用户消息item id在当前工具环境不可见；目标线程01a1dff8-3ebb-77c1-b55b-385aacecba50。原始HGJ资料/真实单据不写入、不外发；不清开发数据。

### 已完成并提交

- c2c7435b：AE公共经营字段及双向映射/显式清空（前窗口已提交，17定向用例/tsc/Biome/antd通过）。
- 375a20c2：方向身份整组108文件，海空主单/海运版本的必填Immutable business_type、方向唯一/查询索引/CHECK/正式迁移、DTO/候选显式businessType、ID/恢复/共享/拆票改配门禁、主单hash身份、源/生成物/种子夹具和针对性测试，含规范和验证记录。
- 5342fd33：已归档方向任务至 `.trellis/tasks/archive/2026-11/11-15-hgj-order-direction`，父子完成计数1/6。不要继续使用旧活跃目录作为可写任务目录；验证记录均在归档research。
- 新规范 `.trellis/spec/server/backend/order-master-business-identity.md`，已挂index并更新海空旧唯一键/签名描述，后续五组implement/check JSONL已加入此规范。

### 方向组真实验证（均有归档记录）

- 主会话新增 `order_direction_migration_integration_test.go`：3主测试、13异常及2约束子场景，最终36.05s PASS无SKIP，正式SQL前态→截至目标迁移，原列全行比较、四类与Ended关系/版本回填、异常DDL/revision回滚、方向公司唯一/NOT NULL/CHECK无默认。只随机Schema，不清开发数据。
- 实现代理完整方向+出口相邻PG首轮144.383s含2FAIL（404语义、旧唯一索引映射），不能称该轮PASS；修复后受影响5主项补跑16.608s全PASS、0FAIL、0SKIP。其余所选新方向/迁移/共享/单证/终止恢复/审核锁/重量/拆票改配均PASS无SKIP，精确命令见 direction-core-validation.md。
- 检查代理自修2项：共享正文预锁Order集合后MBL锁内完整Link集合比对、去逆序Link锁；拆票/改配Preview和Execute都验证原/目标已有MBL完整成员身份。新增2主PG测试（其中1含4子场景）实际PASS。最终404/索引名小修静态复核PASS。
- Root5前端定向文件27tests PASS；7文件Biome、tsc、3组件antd lint PASS；biz/service/ent-migrate完整纯包PASS；相关vet PASS；Ent/API/web再生成指纹0变化。
- 额外高风险 `pnpm run check:fast` 真实PASS，227.8s（WEB227.8 / SERVER225.5）。前端479文件/2576用例PASS，既有 sea-template.test.tsx 1文件/11用例SKIP（HEAD已有，未算通过）；普通Go全量没数据库环境，不能代替专用PG验收。全量存在既有Biome警告/环境getComputedStyle/localhost3000噪音；未改无关文件或关规则。无build，PDF生产入口组后续执行。

### 开发库真实阻断（保留数据，未问用户处理）

- Root只读实际开发库：1张海运孤立主单，有1条SeaVersion，无任何历史Link/HBL/锁/其他Order引用；其余海空未知/混向/跨公司/版本错误计数均0。不能从关联确定该记录方向；正式迁移按已批准合同应拒绝。不要猜SE/放宽迁移、不要删记录或清库。
- 当前没有在开发库apply新迁移。使用隔离Schema继续代码/验收；整套代码具体就绪后，应用开发库前需要用户明确这条记录业务归属或明确批准具体处置。原始业务标识不写Git；必要时重新只读定位，本checkpoint不保存标识。
- 本地 `.env.local` 包含DATABASE_SOURCE，shell没有；用 `node --env-file-if-exists=.env.local` 在进程内取source传Go子进程RONCIN_INTEGRATION_DATABASE_SOURCE，绝不打印源/凭据。helper自动迁移随机Schema并只drop自己Schema。postgres127.0.0.1:5432可用。

### 现在正在进行的第二组

- 当前任务已start：`.trellis/tasks/11-15-hgj-air-documents`（in_progress），branch codex/hgj-business-alignment。当前未提交仅该任务task.json（start状态）；其余产品干净。父仍in_progress，其他四子planning。
- 已新派 `air_documents_core`（trellis-implement，fork none），独占所有server/Proto/Ent/正式迁移/生成物/种子/后端测试夹具，含生成web client。它不改非生成前端、不commit、不派子代理。
- 主会话负责本组所有非生成前端UI/adapter/相关测试及spec/任务记录。不要抢写server或重复生成。它会先定Proto/领域/模式/内容shape并生成客户端，消息给主会话后再由主会话接前端。
- 后端prompt明确AE显式DIRECT/HOUSE创建：DIRECT无HAWB、单当前成员不混HOUSE；HOUSE真实HAWB永久占号。已保存不互转/不换主单。AirWaybillContent独立结构化/JSONB、主/分独立hash/版本历史、可选house存在性、普通更新没改正文不清空、旧houseContent迁备注保全、不兼容双读；保留重量与费用来源/快照/锁序/原子性。本组不开放SI/AI创建、不做PDF（后续组）。内容字段和长度按父design+observations，decimal/声明文本/AS ARRANGED分清。候选应给实际成员模式/阻断或按请求模式过滤，前端不凭数量猜混挂。
- Root尚未开始本组前端代码。下一步可读取当前前端air adapter/templates/AirMaster页等来准备；等air_documents_core发具体合同后实施模式选择、optional house、结构化正文/页面读取保存和定向测试；已有AE公共字段不要覆盖。按roncin-web-stack/antd规范查询API与检查，无关模式不扩展。
- 方向旧代理 direction_core/check_direction 已完成并释放所有写权，无遗留产品工作。

### 后续推进

1. 持续和air_documents_core协调前后端责任，合同出来后主会话完成非生成前端。不要再跑方向已过全套；新改动按风险定向验证。
2. 当前航空子组实现/真实PG/前端验证后派独立trellis-check全范围自修，更新规范，同组Conventional Commit+归档；再start SI固定进口服务组（依赖方向已满足），再AI、PDF、横切/完整CSV整体验收。
3. 最终父AC1–AC8、F01–F45、V1–V20全对应真实证据（业务财务链、双空运模式、越权/并发、实际PDF长文分页、>200行CSV、check:fast/PDFbuild同域路径）。不把研究、枚举或基础模型当完成。
4. 全部已授权工作应继续，不因单个子任务完成就结束父goal；只有真正新用户业务信息不明才澄清，已批准锁序/字段设计不重问。任务创建/实施/提交授权均持久有效。

- air_documents_core 最新锁定待生成合同：AirOrderInput.documentMode:string DIRECT/HOUSE（创建必填、更新如传须一致）；house?:AirHouseWaybillInput {houseNo,status,content?:AirWaybillContent,expectedVersion}。AirOrderInfo {documentMode,master,house?:AirHouseWaybill{id,houseNo,status,version,content,contentHash},linkId,linkVersion,active}，删除旧扁平house字段。Master.content仍运输Input，新增waybillContent?:AirWaybillContent（省略保持，{}显式清空）和contentHash。候选Master返回documentMode（依据历史固定模式，空主单无值）、canAttach、attachBlockedReason；DIRECT仅无既有关系可新加票，原票恢复须单独合法放行（已提醒代理），HOUSE不混DIRECT。精确正文字段和生成就绪待它后续消息；Root已确认将按此接前端，尚未写代码。
