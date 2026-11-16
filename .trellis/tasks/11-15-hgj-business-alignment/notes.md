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

## 2026-11-16 01:38 UTC 实施检查点（当前窗口01a1e1d8-4376-7580-b937-5ae93203a364）

- 最终方案2a96d89e后用户明确“可以”已批准六组完整实施；不要重复请求同一批准。父目标仍active：四类进出口闭环+主分AWB制作/PDF+完整筛选CSV，其他五类仅调研。禁止外站写单/真实员工通知/清开发数据。
- 已完成第二组：df217ccd feat航空直单与独立结构化主分运单（128文件），e1f816a5归档至tasks/archive/2026-11/11-15-hgj-air-documents。第一组375a20c2/5342fd33也已归档；父当前2/6完成。航空真PG全套+方向/审核107.544s无SKIP，额外财务10.213s、AE拒AI重量4.525s、checker显式master版本3.326s。check:fast239.7s PASS（489files2625tests+既有1suite11tests跳过），vet/962生成文件幂等/tsc/Biome/antd通过。无浏览器/PDF声称。本期已提交两组不重跑无因全套。
- 当前task .trellis/tasks/11-15-hgj-sea-import 已start/in_progress，工作区该组未提交（包含Root spec/Panel、前端代理、server代理和e2e代理）。分支codex/hgj-business-alignment，无worktree，其他AI/PDF/横切子仍planning。
- root负责所有import-handling目录（Panel/model/7testsfixtures）、spec/任务文档/最终门禁提交；sea_import_front独占其余非生成web；sea_import_core独占server/Proto/Ent/生成webclient；sea_import_browser只web/tests/e2e/sea-import.e2e.ts和/tmp私有harness及research。都不能提交/派子代理。Root下一步需要两独立trellis-check，先等front/core释放写权（已消息询问是否仅tests收尾）。
- API已生成可用：orderServiceGetImportHandling/SaveImportHandling/ListImportHandlingEvents；API.OrderImportHandling {orderId,orderVersion(=Order.version),exchangeAgentId?,exchangeAgentName,steps[5]}；Step {kind枚举ARRIVAL/EXCHANGE/CUSTOMS/PICKUP/DELIVERY,undertaken,occurredAt?,responsibleUserId?,responsibleUserName,referenceNo,notes,attachmentIds}；Event {actorId/actorName/occurredAt/reason/before/after}。PUT全五项snapshot+expectedVersion+reason，建票无承接；普通整票保存不写服务。completed=occurredAt存在，任意子集并行；已完成事实任一字段变/EXCHANGE代理变需reason；取消承接清时间保资料，前端确认。flow四个IMPORT_*原子派生，SI不能出口flow；CLOSED服务变更先重开。引用供应商/人员/附件须本公司/本票。
- Root Panel已完成：当前公司/真实品类权限门控、只读/闭案/终止、官方ModalForm、完整五步、前台15s与回焦/隐藏离页、固定打开快照、候选按scopegeneration拒迟到、服务保存onSaved不借版本、取消确认/原因、失败/在途新输入保留、A→B→A代际，事件展开前后表。Props {order,expectedVersion(主Form基线),readonly?,blockedReason?,onSaved({orderId,previousVersion,saved})}。4files11tests实际PASS17.94s，每文件<20s；7filesBiome/tsc/antd0问题。报告research/import-handling-front-validation.md；日志/tmp/si-import-root-final-tests.log、/tmp/si-root-final-tsc.log。
- Front已SI注册/route/共用海运字段/真实单证箱货放货/状态摘要/严格读取businessType缺失0错配全拒；独立服务回调仅Order恰+1且MBL/TE/Link/HBL编辑身份全部未变才推进Form.version，不reset草稿。SE专用拆票/改配不额外给SI。最近补F08海运可见业务付款只注册一次、F07tradeTerm清空枚举0及可清公共字段显式空。最新10files64tests PASS9.10s；最后payment相关7files71tests有1仅测试API查询错误（代理修），未最终释放。
- SI外部HBL最薄EXTERNAL_ARCHIVE由真实SI+外部issuerPartner派生（无新DB模板配置），本公司RONCIN_SELF四版保留；前端既有hbl模块加归档A4资料页+续页/同Blob，真实主体不借公司英文名/不伪签。PDF首轮布局测试过但实际lineHeight正文不可见，front已改为fontSize+固定行boxheight14并实际核查：普通1页、极值7页、地址/货描/末箱全文完整。Root已view_image正常样张字形/零值/日期/issuer可见；文件/tmp/roncin-si-hbl-archive.pdf与-long.pdf，图片-page1/long-page2/7.png。front正在最终验证及research，build未跑。
- Core已贯通SI真DIRECT/HOUSE/MBL/TE/HBL/箱货/费用/审核/终止恢复/放货及五服务。新Schema/正式迁移20261116160000_sea_import_handling.sql，修SI审核CHECK，旧进口无服务回填空承接而不猜完成。SI专用真PG首轮15.543s无SKIP；追加ETA费用/竞争/审计失败原子后扩展相邻PG正在跑，命令包含TestSeaImport|TestOrderLock_PostgresFlows|TestSeaMasterBillCreatePreservesSharedContent|TestOrderLockGrantScopes|TestSettlementDoesNotReview|TestFinanceReport|TestSeaChangeMasterIdentity|TestFeeCorrectionCommission|TestCommission.*Correction。它修共享财务旧SI+EXPORT+DRAFT无单证夹具为SI+IMPORT+IMPORT_NOT_APPLICABLE+真DIRECT MBL/TE，同步明确SE对照，不改真实数据。生成/vet及最终报告待回传。
- Browser已准备web/tests/e2e/sea-import.e2e.ts，tsc/Biome/list发现1条。/tmp/roncin-si-browser-acceptance/run.mjs只运行单一次性DB+role/正式迁移/bootstrap/server8010+9010/vite8002/指定e2e/严格cleanup，显式.env.local进程内作admin连接，不输出凭据；8001已有未知服务不停止，8010/9010空闲，选8002。尚未运行，Root需确认core/front产品稳定后明确放行。PyMuPDF可用PYTHONPATH=/tmp/roncin-hbl-pdf-tools import pymupdf，其他系统python无fitz；可pdftotext/pdftoppm。用户管理员均随机伪测试值，证据仓库外0700。
- 新spec .trellis/spec/server/backend/order-import-handling.md（7节）已写并挂server/webindex/capability；更新sea-hbl-production与sea-export开头适用SI说明；AI/横切jsonl已加入规范。其他规范如需同步在最终review后补，Root独占spec。
- 真实开发库仍未apply方向/航空/SI迁移。之前只读确定1张孤立海运主单号1、创建2026-10-27、无Order/Link/HBL引用但有1版，无法推断方向；不得猜SE/SI或删。后续整套可验收后，应用该开发库前需用户明确归属或具体处置。此笔原始ID仅本机不进Git；隔离Schema/DB可继续全部验收。
- 下一步：完成SI两个独立check（server/web全scope），准许一次性真实browser，最终相关门禁+PDF生产build/同域检查，提交SI并归档；再start AI→AWB PDF→横切CSV/四类业务财务完整验收，不因当前组结束就停父goal。当前未完成父目标，不update_goal complete。

### 01:44 UTC 追加（同窗口结束前，优先于上述01:38状态）

- SI核心已final并释放server全写权。最新PG45项含子83.938s/0SKIP0FAIL、扩展97.950s/0SKIP，纯biz/service/data/server/ent-migrate/vet/buf lint/diff全部PASS。974+3生成指纹幂等PASS。SaveImportHandling已独立SaveImportHandlingResponse（wire字段不变），当前服务JSON及正式SQL只存exchangeAgentId+steps，不含orderId/orderVersion/name镜像；API实时投影Order.version，事件快照保版本。报告research/sea-import-core-validation.md已落。禁止无因再生成引起HMR。
- SI前端已final并释放所有非生成产品写权；新付款测试仅选项文案查询错误已修，最终2files44tests PASS，其他6files70tests PASS；详细8files106tests+HBL19tests+相邻10files64tests报告research/sea-import-front-validation.md。留sea-export/form-adapter.ts:435格式折行交checker，不再原代理改prod。
- 当前两独立checker都已spawn并运行：check_sea_import_web独占所有非生成web（排除browser e2e单文件）；check_sea_import_server独占server/Proto/Ent/迁移/生成物。Root不再写产品，继续spec/记录/最终门禁/提交。两check派单要求全scope及定向自修，不重跑无因已过PG；生成如需改先协调browser。
- webchecker已确认准备自修：Panel保存await onSaved/invalidateQueries之后还需serial最新校验防关新稿；归档PDF固定续页页眉/页脚加入有界HBL号+保存版本；HBL后续读取和费用/详情读取需严格目标Order ID+组织+类型，不能只初始读/if-present；HBL查询key初始带kind而Editor不带kind需统一同shape。Root已明确这些必要局部修复授权。它正在改与定向验证，完成后需Root重build（这些源码在首轮build之后改）。serverchecker刚接手，未报告发现。
- Root pnpm run build已真实PASS：web29.65s、serverexit0（exec session85264已完成），日志/tmp/hgj-sea-import-build.log。只有既有大chunk warning和本机未配置Sentry上传跳过，未忽略编译失败。由于checker随后改web/PDF，需要修后再build/stage嵌入产物，勿把第一次build当最终修复版本验收。SI check:fast尚未跑，等checker完成后跑；别误用航空组239.7s记录。
- browser agent sea_import_browser仍运行但暂停新一轮，等checker稳定和新build后Root明确再放行。已准备e2e/harness/research。前几轮失败都已准确分类且每轮临时DB/role/进程全清：dev连接角色无CREATEROLE→本机sudo -n -u postgres仅创建清理随机验收资源；旧成员helper路径405/空body→按真实batchCreate修；一次生成器重写client导致Vite HMR/reload；最新海运备注选择器需先展开折叠再按placeholder（非产品缺陷）。最新实际阶段证据：真实API SI DIRECT/HOUSE创建PASS、DIRECT无HBL、EXTERNAL_ARCHIVE真外部主体确认，详情/personnel/sea-docs/lock/services API均200，无page/consoleerror；整例仍未PASS，不把阶段当完流。
- browser当前应使用/tmp/roncin-si-browser-acceptance/run.mjs，显式8002；8001未知服务不动；server8010/9010仅自己追踪进程。PyMuPDF路径PYTHONPATH=/tmp/roncin-hbl-pdf-tools（import pymupdf），上一01:38精确子目录说明作废。它已加8010生产同域SPA/assets/font/404及真实同域HBL检查，待最终轮执行。Rootbuild的新embed是同域证据的先决。不得改开发角色/库或停止未知服务。
- Root服务面板最后又加optional occurredAt显式空字符串拒绝和回归，model4tests已单独PASS0.826s；4files11tests全组PASS17.94s、7filesBiome、全tsc均PASS。报告research/import-handling-front-validation.md已落。生产现归webchecker负责自修，不抢写。
- Root同步新order-import-handling spec+server/webindex/capability及旧sea-hbl-production/sea-export开头适用SI；刚修旧order-lock-and-document-version.md第41行CHECK说明为SE/SI四海运引用必填，其他AE/AI/LAND/RAIL空。工作台spec仍SE/AE属后续横切本组不假称接通。AI/横切implement/checkJSONL已注入新规范。
- 下一步：接checker真实结果，修必要跨层发现，重build和真实browser最终轮+check:fast风险门禁；AC证据齐后SI同组Conventional Commit/归档（目前任务170多修改文件均本组AI，尚未提交）；再AI→AWB PDF→横切/CSV/最终四类业务财务完整验收。父仍active/2of6完成，不停目标或重问授权。

## 2026-11-16 01:59 UTC 追加（窗口01a1e205-bb18-7551-9ff5-c62a991467ef）

- 最新用户针对开发库孤立主单异步问题明确回复“随便弄，这个是开发库”。已获具体处置授权，Root选择归SE并保留主单及1历史版本；此前“仍需询问归属”的阻断已解除，不再重复问。
- Root实际完成开发库迁移到20261116160000：私有备份+精确ID/全部FK引用/全字段锁内断言，临时副本仅准该ID显式SE；正式仓库方向迁移未加默认/兼容。首次到1600因开发库缺旧orders_flow_status_check失败该文件回滚；按20260827160000原CHECK补齐后，正式migrate+开发checksum repair成功。事后master及version除business_type新字段外深比较完全相等；1200/1400/1600 revision checksum与仓库SHA256全部true。私有证据/tmp/roncin-approved-dev-direction-nctd3T/，原始ID/凭据不进Git。操作脚本/tmp/roncin-approved-dev-direction.mjs因前次部分迁移已完成不要再运行。
- SI两checker已完成全scope复核且释放全部产品/生成写权，报告research/sea-import-{server,web}-review.md。Backend修：RFC3339Nano精度；两个共同航程RPC路由动作transport.update集中映射SE.reassign/SI.update无新增权限码；auth候选从硬编码SE改显式真实SE/SI。真实HTTP26矩阵/PG共享航程1PASS0SKIP/最终vet/buf/diff/125生成指纹幂等PASS。
- Web修：服务onSaved/invalidate awaits后serial复验；详情/费用/HBL每次读取严格ID+公司+类型；HBL queryKey一致；归档续页32字单号摘要+版本；正文/印字SHA与模板修订入指纹；服务未改纳秒原wire保留、真实改ISO；SI共同航程update权限及收回关闭旧modal；代理历史前后姓名。最终69文件Biome、3组件antd0、tsc、3定向批次20/22/31各PASS（非累加独立计数）。Root实际view_image新版长文page2字形/边界/页脚正常。
- Root修后最终pnpm run build PASS（Web29.45s+Goexit0，日志/tmp/hgj-sea-import-build-final.log，exec4293已完成），在两checker产品停止后开始构建；此后只有tests/fixtures改，无需再build。
- SI首次check:fast（exec74460已exit1，日志/tmp/hgj-sea-import-check-fast.log）serverPASS，web5文件10失败/499文件2694通过/既有1suite11tests跳过。后续check_web followup已核对全是新路由与严格身份后的旧夹具/期望，只改5tests：adaptRoutes、TagsView、order-abnormal-entry-data、orders-breadcrumbs、SeaTransportExecutionUpdateModal.test。无产品变更/无放宽边界。5files57tests PASS16.08s，航程独立6tests14.50s，Biome/tsc/diffPASS；写权再释放。
- 当前最终pnpm run check:web正在exec session82347运行（日志/tmp/hgj-sea-import-check-web-final.log）。这次通过后与首轮serverPASS组合为最终门禁，不必再无因重跑server。
- browser agent sea_import_browser仍运行并独占web/tests/e2e/sea-import.e2e.ts及/tmp私有harness/research。现在已放行最终新build轮。真实已达：SI两模式、共同航程enabled、直单保存刷新、HOUSE2/5子集并行完成、服务保存保主表dirty且其后主表提交200、撤回缺原因拦截/有原因保存与前后历史、外部版本变动保输入禁提交、外部HBL长文85行货描60箱保存重读、真实PDF预览字体200+iframe/blob及下载打印同href。尚无完整PASS；此前轮次日期清除/图标前缀accessible name/exact locator残留等测试定位问题均清理所有自建DB/role/process后重跑，不算产品失败/整例通过。最新轮/tmp/roncin-si-browser-bWwK5L/，需等真实下载/生产同域/逐页图/最终cleanup报告。
- Root新增research/sea-import-root-validation.md草稿、父research/implementation-acceptance-progress.md并更新validation-matrix标题；准确记录失败/剩余父验收。规范order-import-handling已同步精度、transport.update权限、指纹与页脚。后续AI任务research/entrypoints.md已有定位并加入implement/check JSONL；尚未start，代码未实施。已修后续文档两已归档前置路径。
- 下一步：等check:web与完整browser真实PASS（必要修产品需重build；仅tests无需），更新root报告/PRD与implement勾选、确认全部改动属本组，按已获授权同组中文Conventional Commit后archive SI，父完成3/6；再start AI，按Trellis派server/非生成web两个实现代理（责任隔离、不可派子/不可commit），Root协调spec/最终门禁。之后AWB PDF→横切/CSV/四类全财务验收。父goal仍active，不停止或误标complete。

### 02:09 UTC 更新

- 上述SI待办已全部完成：最终check:web exit0（504files2704tests、既有11skip），browser实际整例1PASS1.4m且PDF4页完整/生产同域/cleanup都PASS。Root独立看真实下载PDF末页末箱和保存版本清晰。SI提交74f86b3e（216files）、归档299fb32a、会话209日志8d833b98。父3/6，开发库迁移到1600仍保持数据已核验。
- 当前已start `.trellis/tasks/11-15-hgj-air-import` in_progress；根只改规范/任务元数据。AIR三代理现在运行：air_import_server（trellis-implement）独占server/所有生成物；air_import_web（trellis-implement）独占非生成web排除e2e；sea_import_browser复用旧代理准备air-import.e2e.ts/research及/tmp私有AI harness，尚不准运行长轮，等产品/生成稳定。均不commit/派子代理，Root所有spec/任务主文档/最终门禁。
- 浏览器计划仅AI业务权限+必要公共候选（无AE能力），DIRECT真实UI建票，HOUSE/API共享真实票，主单/重量ORDER-MASTER/进口服务与版本输入保护。后端共享API沿用/api/v1/air-export名称（协议不为命名美化复制），前端新/orders/air-import/masters/:id。server若必要签名变动须早发前端；当前暂无新生成ready。
- Root发现旧order-lock-and-document-version头部仍“非SE不可海运快照”矛盾，已仅改SE/SI真实海运适用、恢复/审核和验证措辞，未改产品。待本组AI真实实现后同步AE/AI与入口/权威日期/权限合同，不能提前将AI产品断言写成已验。

### 02:31 UTC AI实施检查点（同窗口）

- 本组AI所有实现/两独立check均完成释放产品和tests/生成写权。air_import_server报告research/air-import-server-validation.md：无新Schema/迁移；Proto航空权限改动态ORDER_PERMISSION，HTTP/gRPC+Wire加入航空真实目标身份解析；repo候选AE/AI显式、ID/ORDER-MASTER按实体类型授权、双模式/成员/恢复/审核/重量/权威ETA/费用日。最终补审核LockOrder完整集合锁序与HOUSE恢复HAWB公司。PG54.826s0SKIP航空全相邻；22.721s0SKIP AI五服务/完结锁恢复/跨身份/ETA午夜周率缺率回滚；广43.686s仅SI旧to_regclass误命public同名表FAIL，其余PASS，改精确current_schema断言后单例2.590s0SKIPPASS。120场真实HTTP中间件/Service/Usecase矩阵（repo stub）PASS、四包Go/cmd编译/vet/buf/91临时生成幂等PASS。
- web实现报告research/air-import-web-validation.md：AI全路由/adapter/主单/候选/重量费用分支/五服务航空基线；37suite317tests PASS29.54s、47文件Biome、tsc、3antd0/diffPASS。主单探测owner已加身份A→B→A/同ID重进保护。
- 独立webchecker报告air-import-web-review.md：修主单编辑/恢复modal官方onFinish迟到关新稿风险，serial+revision+存活每await复验，恢复固定打开master版本/模式，写响应严格identity；旧测试invalidate错误air-export key→真air-orders并断言二次读取。5files26testsPASS12.7s、最终tsc/Biome49files/architecture18checks560files/antd0通过。产品冻结已通知后才最终browser。
- 独立serverchecker报告air-import-server-review.md：全scope实码/锁序/权限/ETA复核无确认缺陷，无产品自修/生成；独立全vet/buf/diff/120HTTP矩阵定向PASS。未无因重跑已过PG，报告准确来源。
- browser最终稳定真实轮PASS（所有产品冻结后）：RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-ai-browser-acceptance/run.mjs exit0，Playwright1PASS1.3m；onlyAI角色/auth/me无AE+AE候选403，DIRECT真实UI提交AI/IMPORT无HAWB+私有改刷新，真实货物+ORDER计划150/实际180、MASTER210独立，fee quantity来源150/180（没造费用不声称汇率）、HOUSE2票共享候选/主单ETA中国午夜/完整汇总/继续建票AI路由，五服务2/5+主表draft随后提交，真实15s轮询和回焦各有输入保护/pageerror空。证据/tmp/roncin-ai-browser-5xhpwo，独立DB/role0/ports8010/9010/8002清理通过；Root独立看polling输入保护截图。报告air-import-browser-validation.md完整，e2e代理已final释放。中间operator必填/选择竞态/更多菜单定位失败都tests问题，未算整例通过。
- 正在运行本组最终pnpm run check:fast exec session42621（/tmp/hgj-air-import-check-fast.log），约02:29:05启动，未结束。产品已冻结可等结果；若仅旧测试合同漂移由checker最小修，server已过不无因重跑；必要prod改需相应重新browser/build（本组无build/PDF声明）。
- Root写新spec air-import-contract.md7节并挂索引/导航、同步air-export-contract AE/AI、air-waybill scope、air-order-commercial-fields、主单候选类型、锁spec AE/AI完整Order集合、finance-reports权威日期。AI implement/check及后续PDF/横切JSONL已加入新spec。Root在后续两task research分别加PDF entrypoints与最终acceptance-entrypoints，只定位未start/未编码。
- 当前仍AI in_progress、父3/6；下一步等check:fast真实结果，补root report/PRD/implementation勾选、同组中文feat提交后archive+日志（已有持久提交授权勿重复问），再start PDF任务。PDF建议双实现责任拆分：core负责制作页/编辑保存/必要后端输出元数据+生成与routes；renderer负责AWB版式/中立PDF共用字体工具及必要HBL同步；需先共同锁模型shape和真实主体（不为PDF引入重平台）。browser随后只e2e私有harness。不把当前航空正文存在当PDF可用，父目标仍active，之后还有横切/CSV/四类完整财务链。

## 2026-11-16 02:39 UTC 当前检查点（窗口01a1e205-bb18-7551-9ff5-c62a991467ef）

- AI上述待办全完成：check:fast exit0，总237.0s（WEB237/SERVER144）；510files2761testsPASS、既有1suite11skip不计通过。提交734bb814（129files）、归档50d6bf44、会话210日志ac55b257。父当前4/6。不要无因重跑SI/AI整组。父objective仍active，没完成最终PDF+横切CSV/四类财务。
- 已start当前 `.trellis/tasks/11-15-hgj-air-waybill-pdf` in_progress；其前置AE正文/AI/SI实际完成。工作区全为本组变动，包括Root spec和任务元数据，保留不reset。分支codex/hgj-business-alignment，无worktree。
- 当前三代理active：awb_editor_core（trellis-implement）负责全部server/生成物、web非PDF制作页/模型/hook/routes/按钮；不得改awb/pdf、中立PDF工具、hbl/pdf和e2e；awb_pdf_renderer（trellis-implement）独占web/pages/orders/awb/pdf、真正中立PDF工具与hbl/pdf必要同步；sea_import_browser复用旧代理只准备web/tests/e2e/air-waybill.e2e.ts+research/awb-browser-validation.md及/tmp私有harness，尚未允许运行长轮。Root独占spec/任务文档/门禁/提交，各代理不commit/不派子代理。
- Core与renderer已冻结最薄模型（renderer持awb/pdf/types.ts）：target MAWB|HAWB，businessType AE|AI，documentId/version/contentHash/masterNo/houseNo可选，issuerDisplayName，outputKind SELF_ISSUED|EXTERNAL_ARCHIVE，organizationId，完整content。Preview默认{open,onClose,model,fingerprint?}，模板修订+完整model+fingerprint重建usePDF。
- Root已技术确认主体：AE HAWB SELF_ISSUED实际目标主单经营公司id/nameEn；MAWB及AI主分EXTERNAL_ARCHIVE仅取已保存content.issuedBy，缺必需主体只阻PDF允许维护，不能自动从workspace/航司猜补、不改已存issuedBy。独立公司抬头与正文印字分别输出，不伪签/条款/放货。本期一套A4完整分区、续页，非复刻外站原PDF（未取原PDF）。
- Core新增Proto AirMasterWaybill OUTPUT_ONLY organization_name_en=16，经真实master.organizationId+KindCompany与Data.client读取，无新DB列/迁移/权限。make api+webgen已在跑/部分生成更新可见，未报全部ready。HAWB保存最薄UpdateOrder只airExport.house+Order/House/Link/Master原编辑基线、不带经营/运输副本；MAWB保存现Preview→Update完整打开时transportinput仅变waybillContent，不能用后台新版本借旧稿。
- 路由计划已告诉browser：/orders/{air-export|air-import}/:id/awb/{target}（小写target具体待core确认），独立MAWB /orders/{kind}/masters/:masterId/awb。需要AE/AI双模式适用主分，DIRECT无HAWB入口。制作页/保存/签发/打印独立。
- Renderer已开始移同域字体/字形覆盖/宽度JSON到中立orders/documents/pdf，HBL原业务wrapper保留以最小化相邻；已见hbl/pdf/fontCoverage及fontGlyphWidths删除+新目录，为正常移动不是误删。最终必须实际PDF PyMuPDF提字+view_image，普通+极值/200rows/零/未知/AS ARRANGED/声明文本/末行/页界+HBL原4自签/SI归档回归。不要只布局树或PDF头。
- Root新spec `.trellis/spec/server/backend/air-waybill-production.md`7节已写并挂server/webindex，加入当前PDF及后续横切implement/checkJSONL；涵盖冻结主体/API/版本/主体变更指纹/输入保护/全字段分页/实际输出。两实现代理已通知，Root尚需路径具体后更新capability。不要改他们产品文件。公司nameEn变不推进Master.version，15s/回焦/生成前重读须捕获并使旧Blob失效、保稿。
- 最终PDF组必须两独立check全scope，再pnpm check:fast + pnpm build（新生产入口/字体按需理由充分），然后browser等Root稳定build放行，真实主分保存/预览/实际下载同Blob/productionSPA-js-font-mime-404+HBL相邻，逐页实图。循环生成会触发HMR，早期可集中生成，最终browser冻结后不要无因再生成。
- 环境继续本机PG、开发库已在上一SI用户“随便弄这个是开发库”授权后精确归SE保全数据并迁至1600（私有证据/tmp/roncin-approved-dev-direction-nctd3T，脚本不要重跑）；本组专项仍随机Schema正式SQL，浏览器随机DB/role，source只进程env不打印。端口8001未知不碰；自己的8010/9010/8002 cleanup全核验。PYTHONPATH=/tmp/roncin-hbl-pdf-tools import pymupdf可用。
- 下步持续推进当前PDF，不再次询问已批准实施/提交/锁模型。必要新技术问题Root协调代理，不借用户最新开发库回复缩减原全目标。全部6组真实AC1–AC8+F01–F45+V1–V20未齐不能complete父goal。下一组横切专用完整CSV>200行、工作台/通知/台账号/报表和四类完整业务财务链尚未实施，入口只读盘点在其research/acceptance-entrypoints.md。

## 2026-11-16 03:00 UTC 当前检查点（窗口01a1e239-0bc6-76b0-9737-fbf534d49546）

- 用户批准六组完整实施及此前开发库处置仍有效，不再问同一批准；父goal active、前4组已提交归档，当前PDF组未提交/未归档，最终横切/CSV/四类完整财务仍未实施，不能complete父goal。
- PDF两实现全部结束冻结；core报告research/awb-editor-validation.md（真实公司投影Proto16+正规生成幂等，PG AE/AI两子0SKIP，核心27tests及相邻）；renderer报告awb-pdf-validation.md（29tests/12files，64页真实PDF全文/字框/逐页实图、各scale suite<20s）。原字体coverage/width JSON与HEAD逐字节一致移到orders/documents/pdf，中立Blob组件；HBL业务wrapper/四版/归档保持。
- 两独立check全部结束冻结：check_awb_server无缺陷，GoAir/vet/buf/diffPASS，真实PG和生成沿用实现证据；check_awb_web修Master合法模式及link/embedded一致性、合法DIRECT夹具+回归、5个真实PDFtests局部act、字体README新再生路径，12files47tests/tsc/Biome44/architecture18+575AST/antd4files零问题。两报告awb-{server,web}-review.md已读。Root spec/导航/公司/HBL文档已同步，Root不改产品。
- MAWB订单路径初次验票归属后规范独立master身份（AwbEditor和路由均处理），避免保存后order字段误判；固定原master snapshot继续Preview→确认→Update不借后续Order基线。HAWB最薄house+各原版本；主体/版本/nameEn/dirty/并发/在途输入与回焦15s均受保护。
- Root已独立看普通主分、双200行末3页、长party14及HBL4版、真实browser长HAWB第6页与409输入保护截图。long-cargo末页曾整页显示页眉疑点，经同PDFheader bbox+1198暗像素+原/重raster SHA和裁剪实图证明无确认产品问题，勿盲改。Root独立复跑/tmp/roncin-awb-pdf-validation/verify.py exit0，64页全文/行尾/bbox0复验。
- 最终pnpm run build实际exit0：Web27.21s+Go生产，日志/tmp/hgj-awb-build.log，exec55298已完成；后无产品变化。最终pnpm run check:fast实际exit0总245.3s（WEB245.3/SERVER147.8），521files2799tests PASS、既有1suite11skip不计通过，日志/tmp/hgj-awb-check-fast.log，exec30605已完成。既有TimeoutNaN/getComputedStyle/localhost3000拒连stderr在前AI通过日志亦存在，非本组确认失败。不要无因重复门禁/build。
- browser代理sea_import_browser已复用followup运行。首轮/tmp/roncin-awb-browser-hjYJnb：Playwright整例1PASS1.9m、六份真实PDF16页、真实保存/409/在途新输入/15s+focus/同Blob/生产SPA-js-font-MIME-404+lazy/生产AWB及HBL相邻全部PASS，pageerror=[]；原私有PyMu postprocess因第一列表格跨行和其他列穿插误判末标记导致harness总exit1，不是PDF缺字。修私有verify_pdfs.py按真实列提字后6份16页全部PASS并逐页实图，DB0/role0/三端口cleanup均通过。
- 目前browser正跑同一修后最终完整轮证明runner总exit0，证据/tmp/roncin-awb-browser-Oczm1d，约03:00前启动；agent独占e2e+research/private harness，不改产品/生成，Root等最终结果与cleanup。8001未知不动，自己的8010/9010/8002全核验。不要把第一轮后处理错误漏记，也不因修private verifier重build。
- Root草稿research/awb-root-validation.md已写定向/独立复核/实际PDF/build/checkfast真实结论但浏览器总结果仍pending。implement前4项已勾，PRD3AC仍待最终browser。父implementation-acceptance-progress空运进口条已更新734bb814归档路径，PDF仍不算5of6。
- 最后横切组research/acceptance-entrypoints.md追加精确只读发现：Workbench service遍历SE/AE、data/workbench_todo谓词仅两出口、前端MyTodosDrawer/SideCards两出口；biz/notification.go和notification_test_usecase只两出口。OrderListTemplate目前request无onLoad采纳导出快照；ListOrders proto全筛选sort+200page上限，需新无page ExportOrders+withReadSnapshot(在data/transaction.go)完整DTO/count/time，前端复用CSV且不循环分页。scripts/acceptance-finance-fixtures已有prepareAcceptanceCompany/verifyCompositeFinanceWorkflow可借真实财务API但其建票SE不可照搬。尚未start最终子，不提前修改产品。
- 下一步：等browser最终整轮exit0+cleanup真实证据，更新Root/父报告与AC、taskvalidate/diff/秘密扫描后中文Conventional Commit本组全部源+生成+spec+报告（工作区全本组，无其他用户改动），archive PDF并记会话；再start cross-module-acceptance，按Trellis拆server+web两实现（各责任及生成归属固定）加受控四类财务/CSV验收准备。最终CSV>200/四类应收应付建账收付提成/工作台通知报表AC1-8+F01-45+V1-20全齐才能归档父goal。

## 2026-11-16 03:34 UTC 检查点（窗口01a1e239-0bc6-76b0-9737-fbf534d49546）

- PDF组全部收口完成：b3b5b407 feat、65370c82归档、2dd58fe2会话211。父5/6，当前最后子 .trellis/tasks/11-15-hgj-cross-module-acceptance 已start in_progress，父仍active，未complete。Root在本窗口03:03后已更新父进度表/PRD和旧前置路径，不重复询问实施/提交授权。
- 三实现全部完成冻结：cross_module_server所有server/API/Ent源及全部生成物（报告cross-module-server-validation.md）；cross_module_web非生成web（报告web-validation）；复用sea_import_browser负责新的scripts/acceptance-hgj-{client,cross-module}.mjs+web/tests/e2e/hgj-cross-module.e2e.ts及当前research/private harness。Root始终spec/任务/门禁/提交。当前所有dirty全本组AI，无其他用户未提交改动起点。
- 最终server：ExportOrders GET /api/v1/orders/export，独立无page请求、必填SE/SI/AE/AI、其余ListOrders全部filter/sort同内核，Order[]/total/generatedAt，静态route在Get前；RR snapshot完整数据/海空/资源投影+授权裁剪，normal<=200不变。OrderContainerRequest新增OUT_ONLY container_spec_code7/name8，真实Kind批量投影Get/List/Export，不按enabled滤历史，前端删额外字典。工作台所有范围/本人IDs/近期/费用/异常四类；通知正式动作四类，system测试页canSelectOrder=false只示例首页、真实order拒绝、零网络发送。
- Root F表审计发现并收口：Order中文biz按runes但Ent字节上限→本期自由文本MaxRuneLen原尺寸；航空master notes同；SeaTransportExecution+不可变Version vessel_name128/voyage_no64同链。机器ID/currency/ISO/UN不机械替换，Ent迁移schema与HEAD无差异，无新SQL迁移/权限/DB列。航空取消DG原undefined省略保旧→web更新UN/CLASS/date显式''、PG0，创建省略；service真实wire optional merge+PG领域更新持久化都过，最终脚本再HTTP衔接。
- server真实PG：205完整导出/四类真实建票/zero/6.7183/排序/normal201拒绝/跨公司/写txCtx拒绝/RR外连接改客户名和箱型名旧快照新请求，四类ledger主单搜索/权威ETD出口ETA进口中国日/本人workbench交集全部0SKIP PASS。中文Order/AirNotes+DG清空2.68s0SKIP；最终Export箱型RR+SE/SI中文船名不可变历史5.751s0SKIP PASS。早期错夹海运字段到AE/AI、immutable code setter、fixture cleanup漏FK都准确记FAIL并修夹具/Schema统一cleanup后真实exit0，非忽略清理。正规Ent/API/webgen1139文件重复SHA0。
- 两独立check全scope最终完成冻结：check_cross_server无确认缺陷/无产品改，全Go定向/vet/buf/gofmt/diff PASS，沿用真实PG/生成不重跑；check_cross_web三修：同RR箱型projection缺真实字段拒绝，现有导出会话key含真实container/attachment access位防只撤子权限旧列下载，queryOrderList/Export/fees detail严格success===true（省略/false旧data拒绝）。旧mock在收紧后29fail6error已按真实success封套更新，最终6files81tests PASS23.57s，页面18.032/旧list16.401单文件<20s，tsc/29Biome/577AST/antd0都过。两review报告已落，不把实现report旧版本当check后版本。
- Root写新order-cross-module-export.md七节并挂索引/JSONL，更新workbench/通知/capability/领域图/air-common字段及cross-layer guide；research final-research-audit AC1/5/V19、field-delivery-audit F01–45仍有待最终实证项、cross-layer-boundary-analysis及root-validation。注意箱型code既有immutable，RR测试只真实改name，不能声称改code成功。
- 最终pnpm run build已实际exit0，Web27.12s+Go，/tmp/hgj-cross-build.log，exec6666完成；所有prod/generated此前冻结。现在pnpm run check:fast正在exec73566（/tmp/hgj-cross-check-fast.log），03:33:43左右启动；等待真实结果，若仅旧mock漂移test修不需build，prod变动才重build/必要重browser。不要无因重复已过PG/生成。
- browser新整轮现已明确放行，agent sea_import_browser followup正在运行：RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-hgj-browser-acceptance/run.mjs。新schema randomDB/role/Go run+Vite，通知OSS关，8010/9010/8002 cleanup追踪，未知8001不动；不清开发库。准备脚本语法/Biome/list已过，不算业务PASS。会真实四类各业务/应收100应付40→workbench异常/进口服务2of5→完结/人工锁→建账→先收付确认核销→应收登记发票/应付免票→利润60的10%提成6确认/支付，报表业务日/交易日/账龄分别核验。四类各205匹配CSV+单类角色/两公司；Asia/Shanghai Intl独立核对UTC跨午夜+createdAt/ETD/ETA/本地文件名。新增AE/AI DG设→省略保持→显式清空HTTP每次精确version与读回；额外生产8010 SPA精确新dist/JS MIME/404+实际205CSV。
- 下一步：持续收checkfast和browser进度/失败分类，必要仅协调确认prod缺陷，确保整轮API+浏览器+CSV+cleanup0全部PASS，Root独立看CSV/实图/facts。更新F01–45/V1–20/AC1–8实际矩阵，最后风险门禁/秘密扫描/diff并本组中文Conventional Commit，archive最后子+journal；父整体验收全齐才能归档父并update_goal complete。当前还未达成父目标，不结束。
