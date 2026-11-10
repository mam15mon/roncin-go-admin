# 工具与框架 Markdown 逐份补查

- Query：补查 .claude/.zcode/.dsh/.trellis/agents/workflow 及忽略的 .agents 本地技能，给每份路径记录核查深度和依据。
- Scope：internal；仅研究，没有改现行文档、配置、脚本、生成物或 Git 状态。
- Date：2026-11-10。

## 覆盖与方法

- 受版本管理路径 103 份，本地忽略技能 47 份，共 150 份；按 SHA-256 形成 64 个不同内容组。每组通读，再用 SHA 对多平台副本复用阅读证据；平台差异文件单独读取。
- 台账逐条列路径、状态、深度、来源、限制、SHA-256、代表文件和发现编号；源文件 [framework-ledger.jsonl](framework-ledger.jsonl)。
- 状态「待修正」是规划研究结论，尚未实施，不能对外当作「已修正」。无确定错误也不自动视为全部最新：外部平台/上游发布实现仍标「待确认」。
- 本地证据以 `.trellis/scripts`、`.trellis/config.yaml`、平台注册文件、实际 CLI 0.6.17 的只读帮助为准。为确认事件名只读取已安装包，不修改、升级或下载包。

## Findings：已确认差异

| 编号 | 代表文件 | 依据 | 差异与处理边界 |
| --- | --- | --- | --- |
| FW01 | `.trellis/workflow.md` | .trellis/scripts/task.py:638-695；.trellis/scripts/common/task_store.py:348,578-587；.trellis/scripts/common/config.py:262-299；.trellis/scripts/common/task_utils.py:381-414 | 创建示例漏必填 --description；列已不存在 create-pr；空 JSONL/legacy seed 说明互相矛盾；hooks 实际读取 config.yaml，非 task.json.hooks；把不存在的上游契约路径写成本地入口。只修命令和事实，不改流程门禁。 |
| FW02 | `/trellis-channel/SKILL.md` | trellis channel {send,wait,messages} --help（0.6.17）；安装包 trellis-core/dist/channel/internal/store/schema.js:2 | 当前 CLI 无 --tag；不能称 interrupt 是保留标签。global 桶实际为 _global，非 __global__。 |
| FW03 | `/trellis-channel/references/workers.md` | trellis channel send/wait --help；trellis-core/dist/channel/api/interrupt.js:5-50；internal/store/events.js | 普通发送无 --tag；中断事件为 interrupt_requested，实际尝试后有 interrupted，不存在 interrupt kind。 |
| FW04 | `/trellis-channel/references/workflows.md` | trellis channel messages --help（0.6.17） | messages 无 --tag；完成通过 done/turn_finished 与实际 message 内容读取，不发明 final_answer 过滤旗标。 |
| FW05 | `/trellis-channel/references/progress-debugging.md` | trellis channel {wait,messages,forum,thread,context} --help | 移除不存在标签旗标；forum/thread 直接接名称，无 list/show 子命令；context 只有 add/delete/list，不是工作者 inbox 查询入口。 |
| FW06 | `/trellis-channel/references/command-reference.md` | 安装包 trellis/dist/commands/channel/interrupt.js:11；trellis-core/dist/channel/api/interrupt.js:5-50；internal/store/events.js | 本文件其他旗标较新，但 interrupt 行为仍称旧 interrupt kind，需与事件白名单一致。 |
| FW07 | `/trellis-meta/references/customize-local/change-spec-structure.md` | .trellis/scripts/common/config.py:462-480 | spec_scope 位于 session.spec_scope，不是顶层。 |
| FW08 | `/trellis-meta/references/local-architecture/spec-system.md` | .trellis/scripts/common/config.py:462-480 | 同上，规范扫描配置路径需要明确 session.spec_scope。 |
| FW09 | `/trellis-meta/references/customize-local/change-task-lifecycle.md` | .trellis/scripts/common/{active_task,task_store,task_utils}.py；本仓库 .trellis/spec 无 cli/ | 上游契约文章不存在于本仓库；保留为上游背景须明确来源范围，本地事实指当前脚本。 |
| FW10 | `/trellis-break-loop/SKILL.md` | 本仓库无 src/templates/markdown/spec；AGENTS.md；.trellis/spec/{server/backend,web/frontend}/index.md | 复盘后强制同步 Trellis 上游模板不适用本仓库，应限定只有真实维护上游模板仓库时同步；本地只回写真实 spec。 |
| FW11 | `/trellis-spec-bootstrap/references/mcp-setup.md` | AGENTS.md:185-186；package.json；web/package.json；trellis-spec-bootstrap/SKILL.md（工具为可选） | GitNexus 安装/查询示例使用 npx，与项目 pnpm 约束冲突；工具未安装或未获准时用现有源码分析，不自动安装或访问网络。 |
| FW12 | `/trellis-session-insight/references/triggering-patterns.md` | trellis mem help（0.6.17） | mem help 无 --task 参数；改成有支持的 cwd 范围列表及已知会话查询，不访问实际对话来验证文档。 |
| FW13 | `/trellis-meta/SKILL.md` | trellis mem help（0.6.17） | 本版已列 OpenCode 数据读取，只有 phase slicing 明确不支持，不能泛称整个 0.6 系列 OpenCode 读取退化；其他上游发布实现未完整验证。 |
| FW15 | `.agents/skills/roncin-web-stack/SKILL.md` | web/src/access.ts:135-154；web/tests/renderWithApp.tsx:23-49；web/src/pages/orders/detail.tsx；.trellis/spec/web/frontend/type-safety.md | 旧测试示例仅填展示权限且用不存在权限码，不能开启真实 access；需注入真实 permissionCapabilities/公司工作区，含 useBlocker 的详情应使用 dataRouter/renderWithDataRouter。此文件忽略，不强制入 Git。 |

### 命令与类型依据

- `task.py --help` 的子命令清单没有 create-pr；create 要求非空 title/description（`common/task_store.py:348`），创建空 implement/check 清单（`:578-587`）。start 的空上下文拦截与 --allow-empty-context 存在，不能把“旧 seed 可容忍”当运行现状。
- 生命周期 hooks 由 `.trellis/scripts/common/config.py:262-299` 从 config.yaml 读取，`task_utils.py:407-414` 调用并注入 TASK_JSON_PATH；未查到 task.json.hooks 接入。
- `get_spec_scope` 在 `common/config.py:462-480` 读取 session.spec_scope。
- `trellis channel send/wait/messages --help` 无 --tag；forum/thread/context 帮助也没有旧 list/show/按工人 inbox 子命令。
- 实际安装 `@mindfoldhq/trellis` 0.6.17 的 `dist/commands/channel/interrupt.js` 调用核心 requestInterrupt；其依赖 `trellis-core/dist/channel/api/interrupt.js:18,50` 写 interrupt_requested，事件类型白名单没有 interrupt。
- 实际 `trellis-core/dist/channel/internal/store/schema.js:2` 的 GLOBAL_PROJECT_KEY 为 `_global`；channel 主技能写 `__global__` 不符，meta 的 multi-agent-channel 文档该处反而正确，不要机械同改。
- `trellis mem help` 无 --task，列出 OpenCode，phase slicing 对 OpenCode 警告并返回全部。未实际打开或输出用户会话记录。

## Related Specs

- 根 `AGENTS.md`：中文、pnpm、禁止 TDD、保留工作区与按验证风险执行。
- `.trellis/spec/domain/`、`.trellis/spec/{server/backend,web/frontend}/index.md`：现行项目规范归属。
- `.trellis/workflow.md`：保留本项目阶段与审批政策；本任务只修错误事实和命令，不替用户改变工作流。

## External References

没有访问外网、打开用户会话、创建频道、派发外部消息、执行安装升级或数据库。版本相关断言以当前本机发布包及只读 CLI 帮助为证据，不宣称外部最新发布版本。

## Caveats / Not Found

- 其他平台（未配置的 Pi/Kimi/Grok/Snow 等）真实 hooks、原生工具和安装产物未验收；广义平台矩阵保留待确认，不能把文件路径相似当作支持已验证。
- 上游源码目录 `packages/cli/...` 只在明确说明上游实现时保留；不能要求本仓库向不存在的上游模板目录同步。没有去改安装包或模板哈希。
- 模板英文或历史“v0.6.0”字样本身不是错误；修复需要实际行为、版本边界或仓库规则冲突的依据。
- 忽略的本地技能只登记，实施是否修改及如何交付需主会话明确；不得强制加进 Git 或改仓库外部软链接。
- 深度说明是全文阅读和关键本地入口核对，不代表文中的每条外部 SDK、全部平台和所有业务矩阵均实际运行。

## 逐文件结论

| 路径 | 状态 | 内容组 | 核查依据与限制 |
| --- | --- | --- | --- |
| `.claude/agents/trellis-check.md` | 已核实无需修改 | `.claude/agents/trellis-check.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/agents/trellis-implement.md` | 已核实无需修改 | `.claude/agents/trellis-implement.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/agents/trellis-research.md` | 已核实无需修改 | `.claude/agents/trellis-research.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/commands/trellis/continue.md` | 已核实无需修改 | `.claude/commands/trellis/continue.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/commands/trellis/finish-work.md` | 已核实无需修改 | `.claude/commands/trellis/finish-work.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-before-dev/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-before-dev/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-brainstorm/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-brainstorm/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-break-loop/SKILL.md` | 待修正 | `.claude/skills/trellis-break-loop/SKILL.md` | FW10 |
| `.claude/skills/trellis-channel/SKILL.md` | 待修正 | `.claude/skills/trellis-channel/SKILL.md` | FW02 |
| `.claude/skills/trellis-channel/references/command-reference.md` | 待修正 | `.claude/skills/trellis-channel/references/command-reference.md` | FW06 |
| `.claude/skills/trellis-channel/references/forum.md` | 待确认 | `.claude/skills/trellis-channel/references/forum.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-channel/references/progress-debugging.md` | 待修正 | `.claude/skills/trellis-channel/references/progress-debugging.md` | FW05 |
| `.claude/skills/trellis-channel/references/workers.md` | 待修正 | `.claude/skills/trellis-channel/references/workers.md` | FW03 |
| `.claude/skills/trellis-channel/references/workflows.md` | 待修正 | `.claude/skills/trellis-channel/references/workflows.md` | FW04 |
| `.claude/skills/trellis-check/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-check/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/SKILL.md` | 待修正 | `.claude/skills/trellis-meta/SKILL.md` | FW13 |
| `.claude/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/customize-local/change-agents.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-agents.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/customize-local/change-context-loading.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-context-loading.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/customize-local/change-hooks.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-hooks.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/customize-local/change-spec-structure.md` | 待修正 | `.claude/skills/trellis-meta/references/customize-local/change-spec-structure.md` | FW07 |
| `.claude/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | 待修正 | `.claude/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | FW09 |
| `.claude/skills/trellis-meta/references/customize-local/change-workflow.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-workflow.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/customize-local/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/local-architecture/context-injection.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/context-injection.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/local-architecture/generated-files.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/generated-files.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/local-architecture/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/local-architecture/spec-system.md` | 待修正 | `.claude/skills/trellis-meta/references/local-architecture/spec-system.md` | FW08 |
| `.claude/skills/trellis-meta/references/local-architecture/task-system.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/task-system.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/local-architecture/workflow.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/workflow.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 已核实无需修改 | `.claude/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/platform-files/agents.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/agents.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/platform-files/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/platform-files/platform-map.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/platform-map.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-session-insight/SKILL.md` | 待确认 | `.claude/skills/trellis-session-insight/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-session-insight/references/cli-quick-reference.md` | 待确认 | `.claude/skills/trellis-session-insight/references/cli-quick-reference.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-session-insight/references/triggering-patterns.md` | 待修正 | `.claude/skills/trellis-session-insight/references/triggering-patterns.md` | FW12 |
| `.claude/skills/trellis-spec-bootstrap/SKILL.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-spec-bootstrap/references/mcp-setup.md` | 待修正 | `.claude/skills/trellis-spec-bootstrap/references/mcp-setup.md` | FW11 |
| `.claude/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-spec-bootstrap/references/spec-writing.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/spec-writing.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.claude/skills/trellis-update-spec/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-update-spec/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.dsh/DSH.md` | 待确认 | `.dsh/DSH.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.dsh/skills/trellis-continue/SKILL.md` | 已核实无需修改 | `.dsh/skills/trellis-continue/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.dsh/skills/trellis-finish-work/SKILL.md` | 已核实无需修改 | `.dsh/skills/trellis-finish-work/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.dsh/skills/trellis-start/SKILL.md` | 已核实无需修改 | `.dsh/skills/trellis-start/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.trellis/agents/check.md` | 已核实无需修改 | `.trellis/agents/check.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.trellis/agents/implement.md` | 已核实无需修改 | `.trellis/agents/implement.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/agents/trellis-check.md` | 已核实无需修改 | `.zcode/agents/trellis-check.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/agents/trellis-implement.md` | 已核实无需修改 | `.zcode/agents/trellis-implement.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/agents/trellis-research.md` | 已核实无需修改 | `.zcode/agents/trellis-research.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/commands/trellis/continue.md` | 已核实无需修改 | `.zcode/commands/trellis/continue.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/commands/trellis/finish-work.md` | 已核实无需修改 | `.claude/commands/trellis/finish-work.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-before-dev/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-before-dev/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-brainstorm/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-brainstorm/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-break-loop/SKILL.md` | 待修正 | `.claude/skills/trellis-break-loop/SKILL.md` | FW10 |
| `.zcode/skills/trellis-channel/SKILL.md` | 待修正 | `.claude/skills/trellis-channel/SKILL.md` | FW02 |
| `.zcode/skills/trellis-channel/references/command-reference.md` | 待修正 | `.claude/skills/trellis-channel/references/command-reference.md` | FW06 |
| `.zcode/skills/trellis-channel/references/forum.md` | 待确认 | `.claude/skills/trellis-channel/references/forum.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-channel/references/progress-debugging.md` | 待修正 | `.claude/skills/trellis-channel/references/progress-debugging.md` | FW05 |
| `.zcode/skills/trellis-channel/references/workers.md` | 待修正 | `.claude/skills/trellis-channel/references/workers.md` | FW03 |
| `.zcode/skills/trellis-channel/references/workflows.md` | 待修正 | `.claude/skills/trellis-channel/references/workflows.md` | FW04 |
| `.zcode/skills/trellis-check/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-check/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/SKILL.md` | 待修正 | `.claude/skills/trellis-meta/SKILL.md` | FW13 |
| `.zcode/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/customize-local/change-agents.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-agents.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/customize-local/change-context-loading.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-context-loading.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/customize-local/change-hooks.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-hooks.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/customize-local/change-spec-structure.md` | 待修正 | `.claude/skills/trellis-meta/references/customize-local/change-spec-structure.md` | FW07 |
| `.zcode/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | 待修正 | `.claude/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | FW09 |
| `.zcode/skills/trellis-meta/references/customize-local/change-workflow.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-workflow.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/customize-local/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/local-architecture/context-injection.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/context-injection.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/local-architecture/generated-files.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/generated-files.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/local-architecture/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/local-architecture/spec-system.md` | 待修正 | `.claude/skills/trellis-meta/references/local-architecture/spec-system.md` | FW08 |
| `.zcode/skills/trellis-meta/references/local-architecture/task-system.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/task-system.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/local-architecture/workflow.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/workflow.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 已核实无需修改 | `.claude/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/platform-files/agents.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/agents.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/platform-files/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/platform-files/platform-map.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/platform-map.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-session-insight/SKILL.md` | 待确认 | `.claude/skills/trellis-session-insight/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-session-insight/references/cli-quick-reference.md` | 待确认 | `.claude/skills/trellis-session-insight/references/cli-quick-reference.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-session-insight/references/triggering-patterns.md` | 待修正 | `.claude/skills/trellis-session-insight/references/triggering-patterns.md` | FW12 |
| `.zcode/skills/trellis-spec-bootstrap/SKILL.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-spec-bootstrap/references/mcp-setup.md` | 待修正 | `.claude/skills/trellis-spec-bootstrap/references/mcp-setup.md` | FW11 |
| `.zcode/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-spec-bootstrap/references/spec-writing.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/spec-writing.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.zcode/skills/trellis-update-spec/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-update-spec/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.trellis/workflow.md` | 待修正 | `.trellis/workflow.md` | FW01 |
| `.agents/skills/roncin-web-stack/SKILL.md` | 待修正 | `.agents/skills/roncin-web-stack/SKILL.md` | FW15 |
| `.agents/skills/trellis-before-dev/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-before-dev/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-brainstorm/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-brainstorm/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-break-loop/SKILL.md` | 待修正 | `.claude/skills/trellis-break-loop/SKILL.md` | FW10 |
| `.agents/skills/trellis-channel/SKILL.md` | 待修正 | `.claude/skills/trellis-channel/SKILL.md` | FW02 |
| `.agents/skills/trellis-channel/references/command-reference.md` | 待修正 | `.claude/skills/trellis-channel/references/command-reference.md` | FW06 |
| `.agents/skills/trellis-channel/references/forum.md` | 待确认 | `.claude/skills/trellis-channel/references/forum.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-channel/references/progress-debugging.md` | 待修正 | `.claude/skills/trellis-channel/references/progress-debugging.md` | FW05 |
| `.agents/skills/trellis-channel/references/workers.md` | 待修正 | `.claude/skills/trellis-channel/references/workers.md` | FW03 |
| `.agents/skills/trellis-channel/references/workflows.md` | 待修正 | `.claude/skills/trellis-channel/references/workflows.md` | FW04 |
| `.agents/skills/trellis-check/SKILL.md` | 已核实无需修改 | `.claude/skills/trellis-check/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-continue/SKILL.md` | 已核实无需修改 | `.agents/skills/trellis-continue/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-finish-work/SKILL.md` | 已核实无需修改 | `.agents/skills/trellis-finish-work/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/SKILL.md` | 待修正 | `.claude/skills/trellis-meta/SKILL.md` | FW13 |
| `.agents/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/customize-local/change-agents.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-agents.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/customize-local/change-context-loading.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-context-loading.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/customize-local/change-hooks.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-hooks.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/customize-local/change-spec-structure.md` | 待修正 | `.claude/skills/trellis-meta/references/customize-local/change-spec-structure.md` | FW07 |
| `.agents/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | 待修正 | `.claude/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | FW09 |
| `.agents/skills/trellis-meta/references/customize-local/change-workflow.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/change-workflow.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/customize-local/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/customize-local/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/local-architecture/context-injection.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/context-injection.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/local-architecture/generated-files.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/generated-files.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/local-architecture/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/local-architecture/spec-system.md` | 待修正 | `.claude/skills/trellis-meta/references/local-architecture/spec-system.md` | FW08 |
| `.agents/skills/trellis-meta/references/local-architecture/task-system.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/task-system.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/local-architecture/workflow.md` | 待确认 | `.claude/skills/trellis-meta/references/local-architecture/workflow.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 已核实无需修改 | `.claude/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/platform-files/agents.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/agents.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/platform-files/overview.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/overview.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/platform-files/platform-map.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/platform-map.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 待确认 | `.claude/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-session-insight/SKILL.md` | 待确认 | `.claude/skills/trellis-session-insight/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-session-insight/references/cli-quick-reference.md` | 待确认 | `.claude/skills/trellis-session-insight/references/cli-quick-reference.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-session-insight/references/triggering-patterns.md` | 待修正 | `.claude/skills/trellis-session-insight/references/triggering-patterns.md` | FW12 |
| `.agents/skills/trellis-spec-bootstrap/SKILL.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-spec-bootstrap/references/mcp-setup.md` | 待修正 | `.claude/skills/trellis-spec-bootstrap/references/mcp-setup.md` | FW11 |
| `.agents/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-spec-bootstrap/references/spec-writing.md` | 待确认 | `.claude/skills/trellis-spec-bootstrap/references/spec-writing.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-start/SKILL.md` | 已核实无需修改 | `.agents/skills/trellis-start/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |
| `.agents/skills/trellis-update-spec/SKILL.md` | 已核实无需修改 | `.agents/skills/trellis-update-spec/SKILL.md` | 本地入口已核对；未验证外部运行的部分见台账 notes |

### 避免误报的复核

`workspace-memory.md` 的 `--no-commit` 与空提交值说明已再读 `add_session.py:321-327`。空值被规范化为空提交集合，默认值为 `-`；原用法可运行，撤销 FW14 初判，不列待修正。`--no-commit` 只控制是否自动提交日志，说明可以更精确但不是错误 CLI 参数。
