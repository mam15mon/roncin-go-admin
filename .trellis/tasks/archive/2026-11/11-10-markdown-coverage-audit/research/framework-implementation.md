# 工具与框架文档实施结果

- 日期：2026-11-10。
- 任务：Markdown逐份补查与覆盖证据。
- 实施范围：已批准的 FW01–FW13、FW15 文案错误；FW14 已由研究复核撤销，没有按误报改动。
- 本组共修正 **38 份 Markdown**：**25 份受版本管理、13 份本地忽略技能**。未改业务代码、运行脚本、配置、模板哈希、数据库或第三方许可；未进行 Git 暂存/提交/分支操作。

## 修正内容与依据

| 发现 | 已实施 | 复核依据 |
| --- | --- | --- |
| FW01 | 创建示例补必填 description，删除不存在的 create-pr 命令，明确空JSONL与旧占位行区别；hooks读config.yaml，列真实本地状态脚本/平台钩子 | task.py --help、common/task_store.py、common/task_context.py:128、common/config.py:get_hooks、common/task_utils.py:run_task_hooks |
| FW02–FW06 | channel统一无tag参数、_global、实际forum/thread/context命令；消息与上下文不冒充工作者待处理inbox；完成用done/turn_finished | 当前安装CLI 0.6.17各命令--help；已安装包store/schema、supervisor/inbox和interrupt实现 |
| FW03/FW06细化 | CLI只登记interrupt_requested，运行中的supervisor接收后才可能追加interrupted；不再称CLI必定写一对事件或直接完成提供方中断 | dist/commands/channel/interrupt.js调用requestInterrupt；supervisor/inbox.js消费后记录method/outcome |
| FW07/FW08 | 规范扫描范围明确session.spec_scope并给嵌套示例，不能写成顶层 | 本地common/config.py:get_spec_scope读取session.spec_scope |
| FW09/FW10 | 上游契约/模板目录明确不属于本仓库；复盘只回写真实server/web/guides规范，不向不存在目录同步 | 本仓库目录、实际本地任务脚本与根AGENTS |
| FW11 | GitNexus命令改pnpm exec；说明工具可选、已安装可用才运行，未安装时直接源码分析；安装示例不使用隐式latest升级 | 根/web package.json未声明工具；AGENTS的pnpm要求；本组未安装或执行这些工具 |
| FW12/FW13 | mem无--task，改按cwd/关键词定位真实会话；OpenCode已可读取，但phase切片不支持，不能泛称全能力退化 | trellis mem help；不读取任何实际用户会话以验证说明 |
| FW15 | 本地前端技能以真实permissionCapabilities+公司工作区作最小权限示例；详情useBlocker使用DataRouter工具并明确fixture要求 | web/src/access.ts:135-185、permissions.generated.ts、web/tests/renderWithApp.tsx、detail-leave-guard.test.tsx |

## 工作流与多平台保护

- `.trellis/workflow.md` 的 **8 个 workflow-state 块逐字节不变**，不改变创建同意、规划审阅、实施批准、代理默认模式或提交要求。
- 只读执行 `get_context.py --mode phase --step 1.4 --platform codex` 与 `--step 2.1` 均能解析原阶段；未调用task状态变更或运行hook。
- 12组 `.claude` / `.zcode` / `.agents` 对应副本内容相等。其余待确认的外部宿主说明保留，不因文档同名而宣称运行通过。
- 未改 `.dsh` 或 `.trellis/agents`：原范围中没有待修正的确证事实，不为增加变更数而修改。

## 忽略技能交付

13份 `.agents/skills` 已在当前本地目录修正，仍遵守Git忽略，不强制纳入版本管理。

- `local-skill-updates.patch`：按修改前真实字节形成的统一补丁；当前目录已应用，不要重复套用。
- `framework-change-hashes.json`：38份文档各自修前/修后SHA-256；Git跟踪身份单列。
- 补丁在临时原始目录执行dry-run并重放，13份重放结果与当前本地文件逐字节相等。临时目录已清理，未覆盖仓库其他文件。

## 验证

- `trellis --version`：0.6.17；send/wait/messages/interrupt/forum/thread/context及mem help只读核对。
- 38份改动的围栏命令扫描：未保留可执行的--tag、create-pr、forum list、thread show、mem list --task示例。否定说明中的旧参数可保留以解释错误。
- 本组38份文档没有字面量Markdown本地链接（路径主要以内联代码描述），不虚报链接数量；新增明确本地源码引用均存在。
- 权限测试范本经当前 @babel/parser 的TypeScript/JSX语法解析通过，真实能力字段、权限键和DataRouter导出静态核对；没有执行产品测试或完整tsc。
- `git diff --check -- .claude .zcode .trellis/workflow.md`通过。
- 本组纯文档，没有运行业务lint/类型检查、Go/Vitest、构建、生成器、迁移、数据库、真实频道创建/派发、中断或任何外部消息。

## 最终台账

`framework-final-ledger.jsonl`保留全部150条路径及原始摘要：**33份已核实无需修改、38份已修正、79份待确认**。待确认项涉及未运行的外部宿主/上游行为；每条保留具体深度、证据和限制，不新增“所有平台已验收”声明。

## 修改清单

- `.claude/skills/trellis-break-loop/SKILL.md`
- `.claude/skills/trellis-channel/SKILL.md`
- `.claude/skills/trellis-channel/references/command-reference.md`
- `.claude/skills/trellis-channel/references/progress-debugging.md`
- `.claude/skills/trellis-channel/references/workers.md`
- `.claude/skills/trellis-channel/references/workflows.md`
- `.claude/skills/trellis-meta/SKILL.md`
- `.claude/skills/trellis-meta/references/customize-local/change-spec-structure.md`
- `.claude/skills/trellis-meta/references/customize-local/change-task-lifecycle.md`
- `.claude/skills/trellis-meta/references/local-architecture/spec-system.md`
- `.claude/skills/trellis-session-insight/references/triggering-patterns.md`
- `.claude/skills/trellis-spec-bootstrap/references/mcp-setup.md`
- `.zcode/skills/trellis-break-loop/SKILL.md`
- `.zcode/skills/trellis-channel/SKILL.md`
- `.zcode/skills/trellis-channel/references/command-reference.md`
- `.zcode/skills/trellis-channel/references/progress-debugging.md`
- `.zcode/skills/trellis-channel/references/workers.md`
- `.zcode/skills/trellis-channel/references/workflows.md`
- `.zcode/skills/trellis-meta/SKILL.md`
- `.zcode/skills/trellis-meta/references/customize-local/change-spec-structure.md`
- `.zcode/skills/trellis-meta/references/customize-local/change-task-lifecycle.md`
- `.zcode/skills/trellis-meta/references/local-architecture/spec-system.md`
- `.zcode/skills/trellis-session-insight/references/triggering-patterns.md`
- `.zcode/skills/trellis-spec-bootstrap/references/mcp-setup.md`
- `.trellis/workflow.md`
- `.agents/skills/roncin-web-stack/SKILL.md`（本地忽略）
- `.agents/skills/trellis-break-loop/SKILL.md`（本地忽略）
- `.agents/skills/trellis-channel/SKILL.md`（本地忽略）
- `.agents/skills/trellis-channel/references/command-reference.md`（本地忽略）
- `.agents/skills/trellis-channel/references/progress-debugging.md`（本地忽略）
- `.agents/skills/trellis-channel/references/workers.md`（本地忽略）
- `.agents/skills/trellis-channel/references/workflows.md`（本地忽略）
- `.agents/skills/trellis-meta/SKILL.md`（本地忽略）
- `.agents/skills/trellis-meta/references/customize-local/change-spec-structure.md`（本地忽略）
- `.agents/skills/trellis-meta/references/customize-local/change-task-lifecycle.md`（本地忽略）
- `.agents/skills/trellis-meta/references/local-architecture/spec-system.md`（本地忽略）
- `.agents/skills/trellis-session-insight/references/triggering-patterns.md`（本地忽略）
- `.agents/skills/trellis-spec-bootstrap/references/mcp-setup.md`（本地忽略）
