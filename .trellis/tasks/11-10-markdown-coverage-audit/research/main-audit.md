# 主会话补查：指南、历史记录与覆盖边界

- 日期：2026-11-10；规划基线 `2aeaeeee`。
- 本阶段只写核查与规划记录，没有修改现行说明或业务代码。

## 全量覆盖的定义

逐份记录不能等同于逐条业务结论通过。基线 1,126 份 Git 跟踪 MD 逐文件读取 UTF-8 全文并解析标题、Markdown 内联/引用式链接；机器检查逐项登记。现行文档另由分域人工对照源码，历史文件仅核其归档身份、结构、引用和时间语义，不冒充重跑历史场景。

本轮额外路径发现 267 份 MD：47 份本地忽略技能、1 份用户未提交任务、219 份历史备份/工具结果/内嵌构建产物。路径均单独登记；备份/产物只做目录定位并保留，不标内容最新；不读取外部软链接或把忽略文件强行入库。

## 已确认遗漏

| 文档 | 证据 | 处理建议 |
| --- | --- | --- |
| guides/index.md | 英文模板说明，示例优先grep，与中文/rg约定不一致 | 用中文导航本项目代码复用和跨层指南，指向具体领域规范 |
| guides/code-reuse-thinking-guide.md | 193–223行要求 src/templates/trellis、packages/cli 并运行 rsync --delete，目标目录在此项目不存在 | 按真实 pages/features 与生成入口项目化，移除上游源码维护指令 |
| guides/cross-layer-thinking-guide.md | Cross-Platform Template Consistency等段落重复；docs-site、src/templates、斜杠check-cross-layer不在仓库 | 按现行proto/DTO/biz/data/前端映射、事务/版本/组织范围组织检查表，链接权威规范 |
| workspace/index.md | 全局工作区入口写英文为强制；Active Developers none yet；tasks错误置于workspace | 同步实际mam15mon工作区、当前脚本和中文要求，历史记录原貌保留 |

## 其余主会话范围证据

根三份说明及 docs/13 份在前轮已全文审阅与源码交叉核对，本轮验证其字节与 `2dd784b2` 的最终复核版本完全一致；复用对应逐份报告，并重新检查链接目标。该复用深度在每行标注，不宣称又重新跑过全部业务矩阵。

## 历史记录检查

- 889 份归档 MD 来自 233 个已归档任务，逐份关联其task.json并登记状态/完成日；JSON状态均completed。
- 历史文件有 1 个本地链接目标失效：08-31-commission-enhancement/prd.md 指向旧相对位置。原历史事实保留，问题在独立列表可查；现行入口不依赖它。
- 开发者索引与三份journal按历史记录保留；不因出现旧实现/英文标题就改写原会话事实。全局workspace/index是现行入口，因此单独判待修正。
- 用户已有未跟踪任务只记路径和SHA完整性，未修改其正文，也不以其 planning 状态宣称已完成。

## 证据文件

- baseline-ledger.jsonl：基线1,126份 + 额外267份的路径、SHA、结构与初始深度记录。
- historical-link-findings.json：逐文件已解析的失效目标。
- main-ledger.jsonl：主会话20份现行文档结论，分域台账由对应代理补充。

## 限制

本记录不涵盖远程网站当前状态、部署事实、被排除依赖缓存内容或历史业务结论的重新验收。需要外部源码或实际平台才能核实的工具声明列待确认，不用模板同名或文件存在性掩盖不确定。
