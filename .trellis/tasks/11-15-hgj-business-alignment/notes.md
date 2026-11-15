# 工作检查点

- 北京时间 2026-11-16：任务创建已批准；本期四类进出口实施，其他五类仅调研。
- 当前 planning，未运行 task.py start，产品代码未修改，最终设计未审批。
- 已完成：现有系统代码盘点和横切影响；九类列表与新增；四类真实详情；空运进口列表/详情/字段设计/流程配置及费用DTO采样；MAWB/HAWB空制作表单；交付前端静态规则。研究覆盖/局限见research报告，尚无写操作、真实PDF、并发验证。
- 本地 Chrome 服务 rea-browser，CDP http://127.0.0.1:19322，当前目标 E09581C2013FA87BA5A4F1B491B6AF43，已登录。不要重复登录或读取输出凭据。
- CLI 需 rea-local-browser 绕过回环代理。REA capture-browser-scenario connect 每次重新导航 start_url，会重置未保存输入；可只打开空表单，不提交真实业务单据。
- 原始目录 /home/vince/.local/share/rea/hgj/order-research/，敏感数据不提交。主页面已开多个业务页签，官网目标 A4C7F2F3F2839A5B34A1F26F2776159E 不参与调研。
- 研究脚本 research-orders.py、research-create.py 等位于本地 hgj 目录；选择器基于当前页面取证，不复用失败选择器。新手引导“我来试试”遮罩已处理，是否出现以当前页面为准。
- 本轮用户新决定：主、分空运单均结构化并制作/预览/PDF打印；进口到货/换单/清关/提货/派送按票服务适用；AE/AI同时支持无HAWB直单和主分单。
- 待用户回答：本期是否纳入真正多航段及复杂空运改配/拆拼（已异步提问，未答不能擅自选）。
- 仍待最终设计复核：直单是否一主单一票且不混HOUSE、建票后模式转换范围；具体AWB纸面/份联及进口维护签发语义；受控外站写规则测试方案。
- 下一步：根据 coverage-audit.md 补阻塞规划的取证；合并尚待范围答案与技术方案，拆可独立验证子任务，完成最终PRD/设计/实施复核后等待明确实施批准。产品代码仍未改。
- 本轮最重要纠正：*-detail-direct为展示号路由，实际空详情，排除；真实有效详情是*-detail-record。号码span单击只选行，序号td普通div可冒泡到row导航。AI runtime样本 ai-detail-runtime-dto.evidence.json已读真正order/detail。
- 某次 query返回体233754字节，REA最初body-size/allocation限制；最终 ai-query-recorded-dto.evidence.json（ev_8fc4fe467632277c210e6ec8192bc899bc171bf4d4baf98ed1ab4780fdb2186f）显式256KiB/4MiB取得响应。所有运行记录依然有missing events，不能声称全流量覆盖。
- 主分AWB空制作路径：air-import-record-document-maker.evidence.json → mawb-empty/hawb-empty。只打开单个新增表单(id=0)不提交，不走批量新增。源代码已复核初始化无保存，运行只观察读取。
- 静态前端捕获 loaded-sources-expanded.evidence.json（正确CLI用重复 --allowed-origins 参数），107脚本；来源/指纹/字节范围与REA部分语义图见frontend-bundle-rules.md。
- 未获批准操作：在海管家创建/修改真实单据、发送通知、调用税务/报关/收费动作。
