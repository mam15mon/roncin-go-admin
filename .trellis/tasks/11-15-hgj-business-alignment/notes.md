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
