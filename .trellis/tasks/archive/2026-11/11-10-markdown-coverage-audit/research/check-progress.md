# 独立复核进度

- 日期：2026-11-10。仅文档检查，不做Git写操作、产品/配置/生成物改动、写库或外部消息。
- 已读取保存的完整角色注入，并核对任务PRD、设计、执行计划和规划摘要。

## 已完成范围

已复核前端/领域10份及guides三份、workspace全局入口，共14份已完成文档差异。

- 月度申请：对照CommissionApplicationPeriod、Schema唯一键、resolveApplicationCandidates和Resubmit刷新原明细路径，提交月/上月末/多归属月累计与当前实现一致。
- 单证预设：对照biz六类枚举及data的EnterpriseResource关联投影，未误删仍存在的业务用例。
- 权限与路径：canExportFinanceCommissions为access公开字段；ADR指向的归档路径真实存在；知识目录按实际兄弟层列出。
- 三份问题模板：已改为Roncin/React/Vite环境，未声称嵌套web/.github已被远端采用，也未发送问题。
- 工程指南：当前职责、生成入口、锁内复核、工作区和输入基线与根规范一致；已移除上游不存在目录操作，未削弱审批/权限或并发约束。
- 工作区入口：tasks与workspace实际分离；个人自动索引保持单一动态统计，语言和命令符合当前根规范。

本范围暂未发现需要自修的问题。框架、后端和最终汇总/台账尚在并行写入，收到主会话完成通知后再做全范围复核及最终结论。

## 中途检查

- 280份现行文档/本地技能的435个Markdown目标和标题锚点：0失效。
- git diff --check：通过。
- node scripts/generate-error-catalog.mjs --check：403个唯一错误码通过。
- 纯文档按批准计划不运行业务lint、类型检查、测试、构建或外部/真实库验收，均不记为通过。

## 框架只读预审

已读取框架差异（仍由对应代理实施，不并发修改）；对照本地trellis channel wait/messages/interrupt --help及trellis mem help：无--tag、正确interrupt_requested/interrupted区分、mem无--task、OpenCode phase切片不支持等修正符合当前帮助。config.py的get_hooks读取config.yaml.hooks、get_spec_scope读取session.spec_scope，与文案一致。

尚待最终复核：框架所有修改完成后的重复副本一致性、CLI中断路由的具体字段过滤、workflow审批/调度状态块完全保持、13份本地技能patch；后端完成后检查真实待确认原因及全量final-ledger/稳定汇总入口。

主会话已通知：review先完成前端/领域及guides/workspace；框架与后端未收到最终完成消息前只读，不覆盖它们。主会话将提供final-ledger.jsonl与docs/markdown-audit.md，收到后才能写research/final-check.md并结束。禁止Git写操作、产品/配置/生成物/数据库修改；不运行业务lint/typecheck/test/build，已批准纯文档验证。

## 框架最终复核批次一

- 38条改动摘要逐一实算：38份当前SHA与交付hash完全一致；25份tracked修前SHA与HEAD字节相同。
- 12组`.claude/.zcode/.agents`三份副本逐字节相同；workflow 8个状态块（含示例块）与HEAD逐字节相同。
- 已复核channel四份references与meta四份差异；对照安装CLI的requestInterrupt、supervisor/inbox.js、wait.js，确认request只登记请求、supervisor才可能输出结果，默认事件过滤与消息路由说明未再把请求当完成。
- config层级、上游目录不存在事实及mem功能修正符合已核源码。尚未运行真实派发、中断或外部消息。

## 框架最终复核批次二

- 已复核其余channel技能、break-loop、mem触发、MCP说明及本地前端技能最小权限范本。真实capability+公司工作区判定与renderWithDataRouter入口存在，示例没有用退役权限字段绕过判断。
- 13份忽略技能补丁在独立临时树反向还原后逐一比对修前SHA，再重放并比对修后SHA；全部一致，仓库忽略目录未被覆盖或暂存。
- 框架150条最终记录唯一且状态计数33/38/79准确，当前SHA逐一对应；外部或上游能力明确保留待确认。
- 框架复核暂未发现需自修问题。后端与公共总览/合并台账仍待主会话完成通知，未把收尾中状态写成最终通过。

## 后端最终复核

- 后端完成后读取58条最终记录与完成说明；集合和规划两份台账完全相同，唯一性、53无需改/3修正/2待确认及当前SHA均独立核对。
- 三份修改对照真实代码：恢复caller在同事务调用createSeaOrderRestoreLink且保留历史Link；唯一HBL适配器保留原houseNo；SMDG首列代码写SCACCode、项目prefix映射和重点航司先注入事实一致。
- 两项待确认均核对真实路径：ent_error集中helper仍用strings.Contains约束名，ExportScoped参数/计数/批读早退无失败审计；规范仍保留要求，未通过降低约束掩盖缺口。
- 未修改产品代码，未运行业务测试或数据库命令。最终合并台账与public docs仍等待主会话完成。

## 最终合并台账复核批次一

- 1393条路径与原baseline-ledger集合完全一致且唯一；1126 tracked集合使用git ls-tree -z（避免中文引号转义误计）与2aeaeeee完全一致。
- 1126个Git原blob逐一实算SHA；1393个当前文件逐一实算SHA，baseline/current/change标志与四状态均一致，depth/evidence/notes不缺。
- tracked计数134无需改/43修正/893历史/56待确认；local8/13/219/27，与summary和public总览一致。43份正式现有文档修改恰与已修正行一致。
- 总览没有声称全部历史业务或外部工具已验收；两项后端实现差异保留原规范要求。归档前的三个未来台账链接将单列核对，不能当当前目标已存在。

## 最终复核完成

完整结果已写final-check.md。1393条JSONL/人工台账、全部基线与当前SHA、原用户四文件、436普通链接、403错误码和diff检查通过；3个未来归档目标尚不存在，明确留主会话归档后无别名扫描。未新增现行MD自修，未运行业务lint/typecheck/test/build、数据库、Git写入或外部操作。
