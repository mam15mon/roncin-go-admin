# 前端、领域与 ADR 补查实施记录

- 日期：2026-11-10；任务已获用户批准，状态 in_progress。
- 责任范围：规划台账的52份文件；10条确证待修正已实施。
- 仅更新Markdown及本任务研究记录，没有修改业务代码、API、权限、依赖、配置、生成物或数据库；未作Git暂存、提交或分支操作。

## 已修改文件与证据

| 文件 | 修正 | 源码依据 |
| --- | --- | --- |
| `.trellis/spec/adr/0014-monthly-commission-application.md` | 明确application_month是提交月、coverage_to为上月末，同公司员工每提交月一份；覆盖全部未申请合格历史来源，可跨多个归属月；重提只刷新原明细来源事实 | `server/internal/biz/finance_commission_application.go:173-185`、`server/internal/data/ent/schema/finance_commission_application.go:42-52`及`:88`、`server/internal/data/finance_commission_application_candidates.go:178-232`、`server/internal/data/finance_commission_application_submit.go:113-155`及`:222` |
| `.trellis/spec/adr/0015-trellis-workflow-and-spec-knowledge.md` | 列出真实domain/server/backend/web/frontend/guides兄弟目录，移除含混路径 | 实际`.trellis/spec/`目录和各索引 |
| `.trellis/spec/adr/0016-frontend-module-boundaries.md` | 任务及原架构审计指向真实归档目录，保持原决策日期和动因 | `.trellis/tasks/archive/2026-09/09-20-ai-friendly-architecture/research/architecture-audit.md` |
| `.trellis/spec/domain/architecture-map.md` | 月度申请流程和完整候选说明同步提交月、上月末及历史归属月；列表筛选不改变提交集合 | 同ADR0014的period/candidates实现 |
| `.trellis/spec/domain/glossary.md` | 月度申请术语同步；常用单证预设改为六类主体/文本的企业资源投影 | `server/internal/biz/partner_shipping_preset.go:26-31`、`server/internal/data/partner_shipping_preset.go:27-46` |
| `.trellis/spec/web/frontend/component-guidelines.md` | 权限示例使用真实公开access.canExportFinanceCommissions | `web/src/access.ts:277-278` |
| `.trellis/spec/web/frontend/type-safety.md` | “总部”切换场景改为系统管理/公司工作台，权限判断规则不变 | `web/src/access.ts`与角色范围退役契约 |
| `web/.github/ISSUE_TEMPLATE/bug_report.md` | 改为Roncin缺陷和当前React/Vite工程环境信息，去Umi/上游项目要求 | `web/package.json`、`web/src/router/adaptRoutes.tsx` |
| `web/.github/ISSUE_TEMPLATE/feature_request.md` | 改为Roncin业务场景与可观察验收标准 | 当前业务项目范围与根协作规范 |
| `web/.github/ISSUE_TEMPLATE/question.md` | 改为Roncin使用/开发问题及当前环境信息 | 当前前端架构与公司工作区 |

两批修改各5份后及时记录；没有为增加改动数量改写其余正确说明。问题模板保留既有title/labels/assignees字段，不推断嵌套web/.github已被远端仓库启用，不执行提交或发布问题。

## 最终逐份台账

`web-domain-final-ledger.jsonl`包含原52条路径：**41条已核实无需修改、10条已修正、1条待确认**。每条保留基线sha256并增加baseline_sha256/current_sha256；未修改文件逐字节不变，已修正文件确有差异。各条核查深度和源码证据继承规划的实际阅读记录，已修正条目增加修改后复核，不声称重新运行全部业务矩阵。

## 验证

- 52份文件的MarkdownIt内联/引用式链接和标题锚点检查：**209个本地目标，0失效**；跳过远程URL及模板占位，不访问外网。
- 10份修改文档的关键描述断言：真实access字段、申请唯一键/完整历史候选入口、归档路径均存在；旧错误权限键与禁止跨归属月描述已移除。
- `git diff --check -- <本组52文件>`通过。
- `node scripts/generate-error-catalog.mjs --check`通过：403个唯一错误码，自动生成目录未手改。
- 机器检查结果另存`web-domain-validation.json`。

## 保留与限制

- `web/CODE_OF_CONDUCT.md`的上游个人邮箱归属仍待确认。保留原文、不猜负责人、不联系外部，不将上游公约视为本项目已确认支持入口。
- 字体资产及其文档、其余前端/领域/ADR内容维持规划研究中已记录的核查结论，不扩大到真实PDF视觉或全部前端交互验收。
- 纯文档没有运行业务lint、类型检查、Vitest/Go测试、构建、真实数据库、OSS、钉钉或正式部署验证，不将未运行项写为通过。
- Git提交、全范围独立check、总台账合并与任务归档由主会话完成。
