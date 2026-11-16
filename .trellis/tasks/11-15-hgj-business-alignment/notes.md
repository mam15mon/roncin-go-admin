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
