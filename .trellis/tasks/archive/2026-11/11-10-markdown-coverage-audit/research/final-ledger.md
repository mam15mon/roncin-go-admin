# Markdown逐份核查台账

- 核查日期：2026-11-10；冻结基线：2aeaeeee。
- 基线1,126份Git Markdown与额外267份本地文件分别列出；每路径一条，不包含本任务新建产物。
- 已核实无需修改仅对应记录的核查深度，历史保留不表示业务事实已重新验收。外部宿主、实现差异及用户未提交任务如实待确认。
- 完整摘要、源码依据和备注见 [机器台账](final-ledger.jsonl)，统计见 [汇总](final-ledger-summary.json)。

## 会话记录

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.trellis/workspace/mam15mon/index.md` | Git基线 | 历史保留 | 逐文件全文解析、结构及历史定位检查；未重跑会话验收 | `baseline-ledger.jsonl` |
| `.trellis/workspace/mam15mon/journal-1.md` | Git基线 | 历史保留 | 逐文件全文解析、结构及历史定位检查；未重跑会话验收 | `baseline-ledger.jsonl` |
| `.trellis/workspace/mam15mon/journal-2.md` | Git基线 | 历史保留 | 逐文件全文解析、结构及历史定位检查；未重跑会话验收 | `baseline-ledger.jsonl` |
| `.trellis/workspace/mam15mon/journal-3.md` | Git基线 | 历史保留 | 逐文件全文解析、结构及历史定位检查；未重跑会话验收 | `baseline-ledger.jsonl` |

## 前端领域和架构说明

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.trellis/spec/adr/0001-contract-single-direction-flow.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0002-permission-manifest-single-source.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0003-ent-schema-and-sql-migrations.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0004-unified-transaction-wrapper.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0005-pessimistic-plus-optimistic-locking.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0006-pagination-limit-and-masterdata-strategy.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0007-pinyin-search-keys.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0008-frontend-template-system.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0009-server-state-react-query.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0010-order-kind-registry.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0011-antd6-testing-conventions.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0012-same-origin-deployment.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0013-order-lock-and-immutable-document-history.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0014-monthly-commission-application.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0015-trellis-workflow-and-spec-knowledge.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/0016-frontend-module-boundaries.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/adr/index.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/domain/architecture-map.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/domain/error-catalog.md` | Git基线 | 已核实无需修改 | 生成器只读一致性校验；非人工逐行语义核验 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/domain/glossary.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/domain/index.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/backend/database-guidelines.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/backend/directory-structure.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/backend/error-handling.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/backend/index.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/backend/logging-guidelines.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/backend/quality-guidelines.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/capability-navigation.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/column-settings.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/component-guidelines.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/dingtalk-qr-login.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/directory-structure.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/duplicate-report.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/finance-current-workspace.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/finance-ledger-snapshots.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/hook-guidelines.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/index.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/organization-switch.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/quality-guidelines.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/sea-master-voyage-inheritance.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/state-management.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `.trellis/spec/web/frontend/type-safety.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/.claude/skills/antd/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/.claude/skills/pro-upgrade/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/.github/ISSUE_TEMPLATE/bug_report.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/.github/ISSUE_TEMPLATE/feature_request.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/.github/ISSUE_TEMPLATE/question.md` | Git基线 | 已修正 | 本轮全文阅读与关键源码静态对照，文案修改后复读、差异和链接目标/锚点核查；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/CLAUDE.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/CODE_OF_CONDUCT.md` | Git基线 | 待确认 | 全文阅读＋上游来源/本地用途边界核对；不验证外部负责人 | `web-domain-final-ledger.jsonl` |
| `web/README.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/README.zh-CN.md` | Git基线 | 已核实无需修改 | 全文阅读＋关键源码/契约静态对照；未运行业务矩阵 | `web-domain-final-ledger.jsonl` |
| `web/public/fonts/roncin-hbl/README.md` | Git基线 | 已核实无需修改 | 全文阅读＋SHA256/字符表计数＋fc-scan名称静态实测 | `web-domain-final-ledger.jsonl` |

## 历史任务

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.trellis/tasks/archive/2026-08/00-bootstrap-guidelines/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-bill-lock-order/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-bill-lock-order/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-bill-lock-order/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-cny-snapshot/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-cny-snapshot/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-cny-snapshot/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-cny-ui/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-cny-ui/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-cny-ui/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-enhancement/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-enhancement/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-enhancement/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-export/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-export/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-export/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-08/08-31-commission-static-route-order/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/08-31-commission-postgres-transaction-test/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/08-31-commission-postgres-transaction-test/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/08-31-commission-postgres-transaction-test/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/08-31-commission-postgres-transaction-test/research/transaction-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-finance-end-to-end-test-audit/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-finance-end-to-end-test-audit/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-finance-end-to-end-test-audit/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-finance-end-to-end-test-audit/research/audit-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-finance-end-to-end-test-audit/research/preliminary-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-foreign-currency-finance-e2e/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-foreign-currency-finance-e2e/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-foreign-currency-finance-e2e/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-foreign-currency-finance-e2e/research/acceptance-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-foreign-currency-finance-e2e/research/agy-plan-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-01-foreign-currency-finance-e2e/research/current-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-builtin-shipping-service-types/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-builtin-shipping-service-types/research/root-cause.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-cargo-allocation/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-cargo-allocation/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-cargo-allocation/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-cargo-allocation/research/current-model.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-document-content/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-document-content/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-document-content/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-document-content/research/current-model.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-document-versioning/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-document-versioning/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-document-versioning/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-finance-allocation/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-mbl-foundation/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-mbl-foundation/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-mbl-foundation/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-split-reassignment/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-split-reassignment/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-split-reassignment/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-transport-document-model/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-transport-document-model/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-transport-document-model/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-02-sea-export-transport-document-model/research/current-model.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-fix-insecure-context-random-uuid/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-fix-sea-export-p1-issues/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-fix-sea-export-p1-issues/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-fix-sea-export-p1-issues/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-fix-sea-export-p1-issues/research/current-defect-analysis.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-review-tagsview-order-breadcrumbs/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-review-tagsview-order-breadcrumbs/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-revise-sea-export-readiness/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-sea-export-readiness-audit/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-sea-export-readiness-audit/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-sea-export-readiness-audit/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-sea-export-readiness-audit/research/capability-matrix.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-sea-export-readiness-audit/research/current-capability-inventory.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-sea-export-readiness-audit/research/readiness-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-sea-transport-readonly-hbl-display/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-unify-order-breadcrumbs/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-unify-order-breadcrumbs/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-unify-order-breadcrumbs/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-unify-order-breadcrumbs/research/current-breadcrumb-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-unify-tags-view-reuse/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-unify-tags-view-reuse/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-unify-tags-view-reuse/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-unify-tags-view-reuse/research/current-routing-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-universal-order-lock/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-universal-order-lock/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-universal-order-lock/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-05-universal-order-lock/research/current-lock-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-diagnose-mobile-blank-screen/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-fix-mobile-layout-display/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-fix-mobile-layout-display/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-fix-mobile-layout-display/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-fix-order-cache-org-isolation/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-fix-order-cache-org-isolation/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-fix-order-cache-org-isolation/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-optimize-order-create-loading/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-optimize-order-create-loading/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-optimize-order-create-loading/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-simplify-order-list-header/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-simplify-order-list-header/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-simplify-order-list-header/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-simplify-sea-mbl-issuer/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-simplify-sea-mbl-issuer/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-simplify-sea-mbl-issuer/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-simplify-sea-mbl-issuer/research/data-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-use-shipping-lines-for-sea-carrier/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-use-shipping-lines-for-sea-carrier/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-use-shipping-lines-for-sea-carrier/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-06-use-shipping-lines-for-sea-carrier/research/current-model-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-07-simplify-sea-order-house-bill-model/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-07-simplify-sea-order-house-bill-model/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-07-simplify-sea-order-house-bill-model/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-07-simplify-sea-order-house-bill-model/research/current-model-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-dual-row-page-header/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-fix-order-partner-quick-add/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-fix-order-partner-quick-add/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-fix-order-partner-quick-add/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-multi-org-foundation-architecture/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-multi-org-foundation-architecture/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-multi-org-foundation-architecture/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-order-partner-quick-add/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-order-partner-quick-add/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-order-partner-quick-add/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-polish-breadcrumb-ui/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-remove-order-loading-terms/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-remove-order-loading-terms/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-remove-order-loading-terms/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-remove-stale-sea-list-edit-entry/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-unify-breadcrumb-ui/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-unify-breadcrumb-ui/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-08-unify-breadcrumb-ui/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-cleanup-order-draft-defenses/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-cleanup-order-draft-defenses/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-cleanup-order-draft-defenses/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-consolidate-order-draft-lifecycle/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-consolidate-order-draft-lifecycle/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-consolidate-order-draft-lifecycle/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-consolidate-order-draft-lifecycle/research/evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-finance-launch-readiness/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-finance-launch-readiness/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-finance-launch-readiness/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-order-kind-registry-foundation/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-order-kind-registry-foundation/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-order-kind-registry-foundation/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-09-order-kind-registry-foundation/research/evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-10-system-contradictions-and-safety-hardening/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-10-system-contradictions-and-safety-hardening/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-10-system-contradictions-and-safety-hardening/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-casual-partner-loop/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-casual-partner-loop/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-casual-partner-loop/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-exchange-rate-simplification-and-bugfixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-exchange-rate-simplification-and-bugfixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-exchange-rate-simplification-and-bugfixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-partner-terms-credit/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-partner-terms-credit/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-11-partner-terms-credit/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-auth-org-switcher/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-auth-org-switcher/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-auth-org-switcher/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-dingtalk-invitation-transfer/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-dingtalk-invitation-transfer/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-dingtalk-invitation-transfer/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-dingtalk-registration-approval/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-dingtalk-registration-approval/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-dingtalk-registration-approval/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-exchange-rate-cross-calc/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-exchange-rate-cross-calc/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-exchange-rate-cross-calc/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-netting-commission-and-followups/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-netting-commission-and-followups/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-12-netting-commission-and-followups/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-13-frontend-error-idempotency/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-13-frontend-error-idempotency/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-13-frontend-error-idempotency/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-13-remove-role-org-access/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-13-remove-role-org-access/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-14-master-data-governance/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-14-master-data-governance/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-14-master-data-governance/migration-conflicts.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-14-master-data-governance/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-15-role-workspace-invariant/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-15-role-workspace-invariant/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-15-role-workspace-invariant/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-16-mbl-autolink-hbl-unique/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-16-mbl-autolink-hbl-unique/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-16-mbl-autolink-hbl-unique/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-locked-order-fee-policy/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-locked-order-fee-policy/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-locked-order-fee-policy/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-partner-role-blacklist/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-partner-role-blacklist/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-partner-role-blacklist/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-prevent-commission-attribution-loss/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-prevent-commission-attribution-loss/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-17-prevent-commission-attribution-loss/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-18-my-workbench-and-commission/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-18-my-workbench-and-commission/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-18-my-workbench-and-commission/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-adr-directory/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-ai-friendly-core/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-ai-friendly-phase2/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-any-regression-gate/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-cleanup-antd6-deprecation-noise/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-cleanup-antd6-deprecation-noise/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-cleanup-antd6-deprecation-noise/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-domain-map/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-error-catalog/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-fix-anchor-nav-offset/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-fix-data-test-baseline/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-fix-port-airport-required/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-monthly-commission-application/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-monthly-commission-application/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-monthly-commission-application/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-admin-enterprise-web/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-any-escape/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-any-pages/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-any-tail-pages/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-any-ui-templates/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-commission-biz-app/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-commission-ledger-data/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-drawer-workbench-web/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-fee-supplement/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-finance-bill/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-large-files/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-order-split-page/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-order-write-lock/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-sea-document-change/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-sea-document-section/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-sea-order-change-data/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-refactor-web-hygiene/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-tfilter-generics/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-ui-finance-ledger-shell/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-ui-modal-drawer-sizes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-ui-tag-semantics/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-19-ui-unification/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-ai-friendly-architecture/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-ai-friendly-architecture/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-ai-friendly-architecture/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-ai-friendly-architecture/research/architecture-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-ai-friendly-phase2/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-ai-friendly-phase2/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-ai-friendly-phase2/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-direction/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-direction/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-direction/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-migration-batch2/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-migration-batch2/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-migration-batch2/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-migration-batch3/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-migration-batch3/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-data-layer-migration-batch3/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-domain-options-duplication/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-domain-options-duplication/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-domain-options-duplication/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-domain-options-duplication/research/duplicate-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-domain-options-duplication/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-duplicate-scan-correctness/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-duplicate-scan-correctness/research/review-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-duplicate-scan-correctness/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-headquarters-business-boundary/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-headquarters-business-boundary/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-headquarters-business-boundary/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-headquarters-business-boundary/research/entry-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-headquarters-business-boundary/research/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-partner-blacklist-view/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-partner-blacklist-view/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-partner-blacklist-view/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-pro-convention-small-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-vite-router-migration/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-vite-router-migration/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-vite-router-migration/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-workbench-commission-ui-refactor/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-workbench-commission-ui-refactor/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-20-workbench-commission-ui-refactor/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-dead-code-cleanup/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-dead-code-cleanup/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-dead-code-cleanup/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-dead-code-cleanup/research/confirmed-cleanup.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-dead-code-cleanup/research/dependencies.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-dead-code-cleanup/research/server-cleanup.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-dead-code-cleanup/research/web-cleanup.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-fix-order-create-entry-guard/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-fix-order-detail-fees-route-guard/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-partner-personnel-workspace/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-partner-personnel-workspace/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-partner-personnel-workspace/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-permission-scope-alignment/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-permission-scope-alignment/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-permission-scope-alignment/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-permission-scope-alignment/research/backend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-permission-scope-alignment/research/check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-21-sea-document-mode-ui/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-database-config-test-isolation/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-fee-columns-tracking/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-fee-columns-tracking/follow-up-cancelled-fees.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-fee-columns-tracking/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-fee-columns-tracking/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-fee-columns-tracking/research/field-matrix.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-fee-delete-by-bill-occupancy/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-fee-delete-by-bill-occupancy/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-fee-delete-by-bill-occupancy/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-personnel-commission-source/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-personnel-commission-source/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-order-personnel-commission-source/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-sea-export-detail-status-transition/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-sea-export-detail-status-transition/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-sea-export-detail-status-transition/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-shared-location-order-fix/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-shared-location-order-fix/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-shared-location-order-fix/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-system-company-data-boundaries/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-system-company-data-boundaries/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-system-company-data-boundaries/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-system-company-data-boundaries/research/authorization-implementation-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-system-company-data-boundaries/research/fee-catalog-company-defaults.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-system-company-data-boundaries/research/headquarters-dependencies.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-system-company-data-boundaries/research/system-workspace-authorization.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-22-system-company-data-boundaries/research/test-fixture-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-branch-exchange-rate-ownership/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-branch-exchange-rate-ownership/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-branch-exchange-rate-ownership/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-catalog-seed/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-catalog-seed/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-quantity-unit-validation/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-quantity-unit-validation/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-quantity-unit-validation/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-quantity-unit-validation/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-profit-preview/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-profit-preview/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-profit-preview/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-resubmit-profit-preview/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-resubmit-profit-preview/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-resubmit-profit-preview/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-resubmit/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-resubmit/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-fee-supplement-resubmit/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-hide-system-in-org-chart/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-order-fee-bulk-actions/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-order-fee-bulk-actions/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-order-fee-bulk-actions/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-23-order-fee-net-unit-price-and-lock-columns/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-24-exchange-rate-4-decimal/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-24-unified-table-headers/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-24-unified-table-headers/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-24-unified-table-headers/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-24-unified-table-headers/research/entry-inventory.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-24-unified-table-headers/research/settings-vs-headers-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-column-settings-popover/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-column-settings-popover/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-column-settings-popover/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-column-settings-ui-redesign/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-column-settings-ui-redesign/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-column-settings-ui-redesign/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-search-filter-chips/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-search-filter-chips/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-25-search-filter-chips/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-26-fee-hard-delete/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-26-fee-hard-delete/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-26-fee-hard-delete/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-26-fee-hard-delete/research/baseline-failure-registry.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-26-fee-hard-delete/research/status-consumer-inventory.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-fee-status-review-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-fee-status-review-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-fee-status-review-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-fee-status-review-fixes/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-acceptance-closure/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-acceptance-closure/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-acceptance-closure/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-acceptance-closure/research/closure-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-acceptance-closure/research/evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-acceptance-closure/research/execution-progress.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-live-acceptance/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-live-acceptance/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-live-acceptance/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-live-acceptance/research/environment-readiness.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-live-acceptance/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-live-acceptance/research/ui-live-flow.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-finance-live-acceptance/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-integration-baseline-four-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-integration-baseline-four-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-integration-baseline-four-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-integration-baseline-four-fixes/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-27-integration-baseline-four-fixes/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-28-dead-code-cleanup/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-28-dead-code-cleanup/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-28-dead-code-cleanup/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-28-dead-code-cleanup/research/audit-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-28-dead-code-cleanup/research/check-verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-28-dead-code-cleanup/research/server-verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-28-dead-code-cleanup/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-09/09-28-dead-code-cleanup/research/web-verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-dingtalk-directory-provisioning/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-dingtalk-directory-provisioning/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-dingtalk-directory-provisioning/planning.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-dingtalk-directory-provisioning/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-dingtalk-directory-provisioning/research/current-state.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-dingtalk-directory-provisioning/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-fix-header-coverage/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-fix-header-coverage/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-layout-dimensions/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-layout-dimensions/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-layout-dimensions/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-layout-dimensions/research/layout-consumers.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-layout-dimensions/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-retire-wecom/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-retire-wecom/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-retire-wecom/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-09-retire-wecom/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-column-fixed-preferences/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-column-fixed-preferences/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-column-fixed-preferences/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-column-fixed-preferences/research/browser-notes.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-column-fixed-preferences/research/check-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-column-fixed-preferences/research/fees-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-column-fixed-preferences/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-column-fixed-preferences/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-dingtalk-optional-profile/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-dingtalk-optional-profile/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-dingtalk-optional-profile/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-dingtalk-optional-profile/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-dingtalk-sync-feedback/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-dingtalk-sync-feedback/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-dingtalk-sync-feedback/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-dingtalk-sync-feedback/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-table-responsive-scroll/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-table-responsive-scroll/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-table-responsive-scroll/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-table-responsive-scroll/research/audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-table-responsive-scroll/research/coverage.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-table-responsive-scroll/research/entry-inventory.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-10-table-responsive-scroll/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/research/amount-contract-validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/research/browser-validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/research/check-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/research/frontend-validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-fee-review-fixes/research/lock-order-validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/implementation-progress.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/backend-approval-design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/check-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/frontend-approval-design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/implementation-core.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/initial-findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/lock-overlap-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/payment-contract-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/recommended-flow.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-order-fee-approval/research/supplement-commission-design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-interest-scope/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-interest-scope/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-interest-scope/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-interest-scope/research/interest-check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-interest-scope/research/scope-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-keyboard-access/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-keyboard-access/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-keyboard-access/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-keyboard-access/research/keyboard-check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-keyboard-access/research/keyboard-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-list-state/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-list-state/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-list-state/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-list-state/research/list-check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-list-state/research/list-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-list-state/research/list-preflight.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-list-state/research/verification-status.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-narrow-layout/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-narrow-layout/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-narrow-layout/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-narrow-layout/research/narrow-check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-narrow-layout/research/narrow-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-narrow-layout/research/narrow-preflight.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-save-protection/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-save-protection/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-save-protection/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-save-protection/research/save-check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-save-protection/research/save-contract.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-save-protection/research/save-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/audit-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/check-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/research/forms-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/research/navigation-tables-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/research/skill-method.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-fixes/final-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-fixes/planning-check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-fixes/research/execution-progress.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-fixes/research/final-acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-fixes/research/integration-check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-unified-column-resizing/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-unified-column-resizing/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-unified-column-resizing/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-unified-column-resizing/research/browser-notes.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-unified-column-resizing/research/check-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-unified-column-resizing/research/initial-findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-12-unified-column-resizing/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-auto-lock-settlement/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-auto-lock-settlement/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-auto-lock-settlement/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-auto-lock-settlement/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-auto-lock-settlement/research/implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-background-task-notify/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-background-task-notify/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-background-task-notify/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-background-task-notify/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-background-task-notify/research/check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-background-task-notify/research/current-state.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-background-task-notify/research/implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-batch-user-memberships/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-batch-user-memberships/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-batch-user-memberships/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-batch-user-memberships/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-closure-pending-supplement/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-closure-pending-supplement/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-closure-pending-supplement/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-closure-pending-supplement/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-closure-pending-supplement/research/implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-company-admin-provisioning/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-company-admin-provisioning/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-company-admin-provisioning/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-company-admin-provisioning/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-login-memberships/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-login-memberships/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-login-memberships/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-login-memberships/research/current-state.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-login-memberships/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-qr-login/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-qr-login/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-qr-login/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-dingtalk-qr-login/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/research/accounting-design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/research/check-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/research/commission-design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/research/commission-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/research/core-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/research/correction-boundaries.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/research/finance-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-exception-closure/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-exception-closure/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-exception-closure/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-exception-closure/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-fee-exception-closure/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-remove-primary-organization/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-remove-primary-organization/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-remove-primary-organization/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-13-remove-primary-organization/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-14-pending-user-multi-organizations/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-14-pending-user-multi-organizations/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-14-pending-user-multi-organizations/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-14-pending-user-multi-organizations/research/root-cause.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-14-pending-user-multi-organizations/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-dingtalk-html-frame/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-dingtalk-html-frame/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-dingtalk-html-frame/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-dingtalk-html-frame/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-dingtalk-login-switch/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-dingtalk-login-switch/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-dingtalk-login-switch/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-dingtalk-login-switch/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-login-ui-card/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-login-ui-card/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-login-ui-card/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-15-login-ui-card/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-16-permission-discovery/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-16-retire-role-data-scope/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-16-retire-role-data-scope/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-16-retire-role-data-scope/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-16-retire-role-data-scope/research/scope-retirement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-16-retire-role-data-scope/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/research/audit-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/research/browser-verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/research/cache-isolation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/research/commission-fixtures.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/research/final-verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/research/finance-export.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/research/independent-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-17-system-audit-bugfix/research/write-call-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-cards/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-cards/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-cards/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-cards/research/code-map.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-cards/research/protocol.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-cards/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-markdown/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-markdown/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-markdown/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-markdown/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-preview/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-preview/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-preview/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-preview/research/current-capabilities.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-preview/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-sso/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-sso/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-sso/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-sso/research/backend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-sso/research/frontend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-dingtalk-notification-sso/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-fix-password-reset-dev-migration/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-fix-password-reset-dev-migration/root-cause.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-fix-password-reset-dev-migration/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-mobile-login-dingtalk-password/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-mobile-login-dingtalk-password/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-mobile-login-dingtalk-password/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-mobile-login-dingtalk-password/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-mobile-login-dingtalk-password/research/current-state.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-19-mobile-login-dingtalk-password/research/delivery-status.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-admin-role-max-permissions/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-admin-role-max-permissions/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-admin-role-max-permissions/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-admin-role-max-permissions/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-admin-role-max-permissions/research/root-cause.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-admin-role-max-permissions/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/batch-operations.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/competitor-findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/competitor-questions.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/research.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/weight-model.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-billing-unit-quantity-assist/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-billing-unit-quantity-assist/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-billing-unit-quantity-assist/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-billing-unit-quantity-assist/research/current-state.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-billing-unit-quantity-assist/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-organization-switch-retain-page/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-organization-switch-retain-page/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-organization-switch-retain-page/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-organization-switch-retain-page/research/domain-forms-coverage.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-organization-switch-retain-page/research/forms-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-organization-switch-retain-page/research/guards-implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-organization-switch-retain-page/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-20-organization-switch-retain-page/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-21-air-export-dangerous-goods/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-21-air-export-dangerous-goods/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-21-air-export-packing-group/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-21-air-export-packing-group/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-21-air-export-packing-group/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-21-air-export-ui-align-sea/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-21-air-export-ui-align-sea/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-21-air-export-ui-align-sea/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-order-list-lifecycle-badges/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-order-list-lifecycle-badges/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-table-column-layout/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-table-column-layout/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-table-column-layout/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-table-column-layout/research/check-report.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-table-column-layout/research/review-scope.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-table-column-layout/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-unified-order-review-lock/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-unified-order-review-lock/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-unified-order-review-lock/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-unified-order-review-lock/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-workbench-todo-details/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-workbench-todo-details/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-workbench-todo-details/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-23-workbench-todo-details/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-24-order-detail-fee-refinements/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-24-order-detail-fee-refinements/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-24-order-detail-fee-refinements/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-order-reopen-lock-guard/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-order-reopen-lock-guard/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-continuous-house-order/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-continuous-house-order/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-continuous-house-order/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-continuous-house-order/research/entry-and-field.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-continuous-house-order/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-master-voyage-inheritance/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-master-voyage-inheritance/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-master-voyage-inheritance/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-master-voyage-inheritance/research/backend-voyage.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-26-sea-master-voyage-inheritance/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-port-chinese-seeds/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-port-chinese-seeds/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-port-chinese-seeds/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-port-chinese-seeds/research/port-name-findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-port-chinese-seeds/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-shipping-agent-masterdata/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-shipping-agent-masterdata/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-shipping-agent-masterdata/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-shipping-agent-masterdata/research/backend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-shipping-agent-masterdata/research/frontend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-27-shipping-agent-masterdata/research/source.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-28-sea-order-unified-save/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-28-sea-order-unified-save/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-28-sea-order-unified-save/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-28-sea-order-unified-save/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-28-sea-order-unified-save/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-29-sea-export-entry-experience/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-29-sea-export-entry-experience/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-29-sea-export-entry-experience/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-29-sea-export-entry-experience/research/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-29-sea-export-entry-experience/research/browser-acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-29-sea-export-entry-experience/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-console-warnings/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-console-warnings/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-order-list-default-columns/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-order-list-default-columns/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-order-list-review-entry/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-order-list-review-entry/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-order-milestones/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-order-milestones/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-order-milestones/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-order-milestones/research/backend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-order-milestones/research/frontend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-order-milestones/research/schema.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-order-milestones/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-quick-fee-drawer/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-remove-quick-fee-drawer/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-container-entry-permission/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-container-entry-permission/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-container-entry-permission/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-container-entry-permission/research/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-container-entry-permission/research/auth-root-cause.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-container-entry-permission/research/browser-acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-container-entry-permission/research/review-shared.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-container-entry-permission/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-order-port-save-version/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-order-port-save-version/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-order-port-save-version/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-order-port-save-version/research/live-change-check.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-order-port-save-version/research/save-boundary.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-order-port-save-version/result.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-order-port-save-version/review-checklist.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-port-invalid-argument/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-port-invalid-argument/research/location-validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-port-invalid-argument/result.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-shared-leg-house-destination/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-shared-leg-house-destination/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-shared-leg-house-destination/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-shared-leg-house-destination/research/backend-boundaries.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-shared-leg-house-destination/research/frontend-experience.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-shared-leg-house-destination/result.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-10/10-30-sea-shared-leg-house-destination/review-checklist.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-03-order-attachments-aliyun-oss/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-03-order-attachments-aliyun-oss/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-03-order-attachments-aliyun-oss/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-03-order-attachments-aliyun-oss/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-03-order-attachments-aliyun-oss/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-company-english-name-hbl/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-company-english-name-hbl/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-company-english-name-hbl/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-company-english-name-hbl/research/current-state.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-company-english-name-hbl/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-bill-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-bill-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-bill-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-bill-fixes/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/research/review-bills.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/research/review-commission.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/research/review-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/research/review-integration.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/research/review-rates.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-business-fixes/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-commission-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-commission-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-commission-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-commission-fixes/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-rate-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-rate-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-finance-rate-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-business-fixes/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-business-fixes/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-business-fixes/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-business-fixes/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-business-fixes/validation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-export-hbl-production/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-export-hbl-production/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-export-hbl-production/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-export-hbl-production/research/current-state.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-export-hbl-production/research/pdf-output.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-04-sea-export-hbl-production/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-06-sea-operation-blockers/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-06-sea-operation-blockers/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-06-sea-operation-blockers/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-06-sea-operation-blockers/research/document-version-lifecycle.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-06-sea-operation-blockers/research/reassignment-browser.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-06-sea-operation-blockers/research/runtime-acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-06-sea-operation-blockers/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-commission-rule-netting-acceptance/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-commission-rule-netting-acceptance/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-commission-rule-netting-acceptance/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-commission-rule-netting-acceptance/research/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-commission-rule-netting-acceptance/research/netting-chain.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-commission-rule-netting-acceptance/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-commission-rule-netting-acceptance/research/rule-form.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-commission-rule-netting-acceptance/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-restore-recheck-commission-company/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-restore-recheck-commission-company/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-restore-recheck-commission-company/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-restore-recheck-commission-company/research/final-gate.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-restore-recheck-commission-company/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-restore-recheck-commission-company/research/implementation.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-restore-recheck-commission-company/research/live-acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-restore-recheck-commission-company/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/research/backend-verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/research/frontend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/research/live-repair.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/research/restore-options.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/research/review-progress.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-order-restore-link/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-terminated-order-detail/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-terminated-order-detail/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-terminated-order-detail/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-terminated-order-detail/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-sea-terminated-order-detail/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-master-history/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-master-history/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-master-history/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-master-history/research/live-evidence.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-master-history/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-master-history/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-shipping-line-fee-approval/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-shipping-line-fee-approval/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-shipping-line-fee-approval/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-shipping-line-fee-approval/research/fee-approval.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-shipping-line-fee-approval/research/findings.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-shipping-line-fee-approval/research/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-08-terminated-shipping-line-fee-approval/research/verification.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/adr-doc-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/adr-update-result.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/backend-doc-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/backend-update-result.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/check-result.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/document-inventory.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/general-doc-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/general-update-result.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/local-link-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/validation-summary.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/web-doc-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/web-update-result.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-finance-current-workspace/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-finance-current-workspace/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-finance-current-workspace/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-finance-current-workspace/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-finance-current-workspace/research/contracts.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-finance-current-workspace/research/dialogs.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-finance-current-workspace/research/lists.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-finance-current-workspace/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-order-unsaved-close-guard/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-order-unsaved-close-guard/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-order-unsaved-close-guard/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-order-unsaved-close-guard/research/order-drafts.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-order-unsaved-close-guard/research/tab-close.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-order-unsaved-close-guard/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-protojson-response-contract/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-protojson-response-contract/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-protojson-response-contract/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-protojson-response-contract/research/independent-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-protojson-response-contract/research/protojson-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/acceptance.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/research/contracts.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/research/frontend.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/research/local-data-audit.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/research/storage.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-retire-business-tags/review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-shared-ui-contracts/design.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-shared-ui-contracts/implement.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-shared-ui-contracts/prd.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-shared-ui-contracts/research/finance-summary.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-shared-ui-contracts/research/independent-review.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |
| `.trellis/tasks/archive/2026-11/11-09-shared-ui-contracts/research/title-and-reassignment.md` | Git基线 | 历史保留 | 逐文件UTF-8全文解析、标题/内联和引用式链接目标扫描、归档元数据核对；未逐条复验历史业务结论 | `baseline-ledger.jsonl` |

## 工具与框架说明

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.claude/agents/trellis-check.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/agents/trellis-implement.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/agents/trellis-research.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/commands/trellis/continue.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/commands/trellis/finish-work.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-before-dev/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-brainstorm/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-break-loop/SKILL.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-channel/SKILL.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-channel/references/command-reference.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-channel/references/forum.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-channel/references/progress-debugging.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-channel/references/workers.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-channel/references/workflows.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-check/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/SKILL.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/change-agents.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/change-context-loading.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/change-hooks.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/change-spec-structure.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/change-workflow.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/customize-local/overview.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/bundled-skills.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/context-injection.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/generated-files.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/overview.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/spec-system.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/task-system.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/workflow.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/local-architecture/workspace-memory.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/platform-files/agents.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/platform-files/overview.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/platform-files/platform-map.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-meta/references/platform-files/skills-and-commands.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-session-insight/SKILL.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-session-insight/references/cli-quick-reference.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-session-insight/references/triggering-patterns.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-spec-bootstrap/SKILL.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-spec-bootstrap/references/mcp-setup.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-spec-bootstrap/references/repository-analysis.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-spec-bootstrap/references/spec-writing.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.claude/skills/trellis-update-spec/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.dsh/DSH.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.dsh/skills/trellis-continue/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.dsh/skills/trellis-finish-work/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.dsh/skills/trellis-start/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.trellis/agents/check.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.trellis/agents/implement.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.trellis/workflow.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/agents/trellis-check.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/agents/trellis-implement.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/agents/trellis-research.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/commands/trellis/continue.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/commands/trellis/finish-work.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-before-dev/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-brainstorm/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-break-loop/SKILL.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-channel/SKILL.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-channel/references/command-reference.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-channel/references/forum.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-channel/references/progress-debugging.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-channel/references/workers.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-channel/references/workflows.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-check/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/SKILL.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/change-agents.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/change-context-loading.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/change-hooks.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/change-spec-structure.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/change-workflow.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/customize-local/overview.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/bundled-skills.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/context-injection.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/generated-files.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/overview.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/spec-system.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/task-system.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/workflow.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/local-architecture/workspace-memory.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/platform-files/agents.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/platform-files/overview.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/platform-files/platform-map.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-meta/references/platform-files/skills-and-commands.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-session-insight/SKILL.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-session-insight/references/cli-quick-reference.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-session-insight/references/triggering-patterns.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-spec-bootstrap/SKILL.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-spec-bootstrap/references/mcp-setup.md` | Git基线 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-spec-bootstrap/references/repository-analysis.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-spec-bootstrap/references/spec-writing.md` | Git基线 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.zcode/skills/trellis-update-spec/SKILL.md` | Git基线 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |

## 服务端说明与规范

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.trellis/spec/server/backend/air-export-contract.md` | Git基线 | 已核实无需修改 | 全文人工阅读；HTTP签名、显式主分单及重量有理数/来源指纹关键实现对照 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/auth-session-org-switch.md` | Git基线 | 已核实无需修改 | 全文人工阅读；候选、会话轮转锁序、滑动续期及400天Cookie关键实现对照 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/background-task-notify.md` | Git基线 | 已核实无需修改 | 全文人工阅读；LISTEN、独立回调、退避、四类worker接线与迁移入口对照 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/billing-unit-quantity-rule.md` | Git基线 | 已核实无需修改 | 全文人工阅读；权威单位属性、新增整数校验、decimal数值等价与条件追加规则对照 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/business-tag-retirement.md` | Git基线 | 已核实无需修改 | 全文人工阅读；退役迁移与现行Proto reserved关键编号名称对照 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/capability-navigation.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/commission-scheme-assignment.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/company-admin-provisioning.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/company-english-name.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/database-guidelines.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/dingtalk-embedded-login.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/dingtalk-notification-markdown.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/dingtalk-notification-test.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/dingtalk-qr-protocol.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/dingtalk-registration-approval.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/directory-structure.md` | Git基线 | 已核实无需修改 | 人工全文；分层约束及Proto→Go/OpenAPI→Web生成配置对照 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/error-handling.md` | Git基线 | 待确认 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/exchange-rate-single-rate.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/finance-bill-currency.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/finance-commission-lock.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/index.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/logging-guidelines.md` | Git基线 | 待确认 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/mobile-login-password-reset.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/operating-company-commission-attribution.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/order-attachment-oss.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/仓储代码对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/order-fee-correction.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/仓储代码对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/order-fee-quantity-preview.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/仓储代码对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/order-fee-review.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/仓储代码对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/order-lock-and-document-version.md` | Git基线 | 已修正 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/order-milestone-retirement.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/Schema/迁移对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/order-personnel-editor-retirement.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/Schema/迁移对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/order-shared-enum-field.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/Schema/迁移对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/organization-shared-masterdata.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/Schema/迁移对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/partner-casual-contract.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/Schema/迁移对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/partner-role-blacklist.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/仓储入口对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/passkey-authentication.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/仓储入口对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/quality-guidelines.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/retire-role-data-scope.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/仓储入口对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/role-workspace-ownership.md` | Git基线 | 已核实无需修改 | 人工阅读全文、关键Proto/领域/仓储入口对照；未执行业务矩阵 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/sea-document-change-history.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/sea-export-document-contract.md` | Git基线 | 已修正 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/sea-hbl-production.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/sea-order-restore.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/sea-shared-write-consistency.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/sea-terminated-order-read.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/user-membership-batch.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/wecom-retirement.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `.trellis/spec/server/backend/workbench-todos.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `server/AGENTS.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `server/README.md` | Git基线 | 已核实无需修改 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `server/internal/biz/README.md` | Git基线 | 已核实无需修改 | 本轮人工全文阅读职责说明并核对根协作规范及后端分层索引 | `backend-final-ledger.jsonl` |
| `server/internal/data/README.md` | Git基线 | 已核实无需修改 | 本轮人工全文阅读职责说明并核对根协作规范及后端分层索引 | `backend-final-ledger.jsonl` |
| `server/internal/service/README.md` | Git基线 | 已核实无需修改 | 本轮人工全文阅读职责说明并核对根协作规范及后端分层索引 | `backend-final-ledger.jsonl` |
| `server/migrations/README.md` | Git基线 | 已核实无需修改 | 人工全文；对照advisory锁、逐文件事务、校验和开发修复与非线性保护实现 | `backend-final-ledger.jsonl` |
| `server/seeds/README.md` | Git基线 | 已修正 | 人工阅读全文并静态对照关键源码/契约；未执行全部业务矩阵或真实环境验收 | `backend-final-ledger.jsonl` |
| `server/seeds/ports.zh-CN.md` | Git基线 | 已核实无需修改 | 人工全文；CSV只读计数/国家及类别统计，命令、同步及测试入口对照 | `backend-final-ledger.jsonl` |
| `server/seeds/shipping-agents.md` | Git基线 | 已核实无需修改 | 人工全文；源快照SHA-256直接计算，CLI与事务补缺实现对照 | `backend-final-ledger.jsonl` |
| `server/seeds/shipping-lines.zh-CN.md` | Git基线 | 已核实无需修改 | 人工全文；CSV只读计数及分类、CLI与同源覆盖/停用实现对照 | `backend-final-ledger.jsonl` |

## 根说明指南与工作区入口

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.trellis/spec/guides/code-reuse-thinking-guide.md` | Git基线 | 已修正 | 人工全文阅读、根规范及实际目录/脚本对照后项目化修正；已完成本地链接/锚点及命令脚本对照复核 | `main-fixed-ledger.jsonl` |
| `.trellis/spec/guides/cross-layer-thinking-guide.md` | Git基线 | 已修正 | 人工全文阅读、根规范及实际目录/脚本对照后项目化修正；已完成本地链接/锚点及命令脚本对照复核 | `main-fixed-ledger.jsonl` |
| `.trellis/spec/guides/index.md` | Git基线 | 已修正 | 人工全文阅读、根规范及实际目录/脚本对照后项目化修正；已完成本地链接/锚点及命令脚本对照复核 | `main-fixed-ledger.jsonl` |
| `.trellis/workspace/index.md` | Git基线 | 已修正 | 人工全文阅读、根规范及实际目录/脚本对照后项目化修正；已完成本地链接/锚点及命令脚本对照复核 | `main-fixed-ledger.jsonl` |
| `AGENTS.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `PLAN.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `README.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/README.md` | Git基线 | 已修正 | 原文与已深审版本一致；新增逐份核查总览入口后检查目标与锚点 | `main-implementation.md` |
| `docs/airport-port-sync.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/cross-repository-transactions.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/eyun-config-management.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/local-development.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/migration-matrix.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/order-attachment-oss.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/plans/业务标签建设计划.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/plans/企业资源备忘录建设计划.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/plans/客户页面.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/plans/提成管理增强计划.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/plans/海运出口订单闭环验收.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |
| `docs/plans/费用明细批量转账单.md` | Git基线 | 已核实无需修改 | 复用上轮逐份内容/源码核对证据，验证与已复核版本逐字节一致；本轮补查链接和关键维护入口；未重复执行业务矩阵 | `main-ledger.jsonl` |

## 本地备份或工具构建产物

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.agy-staff/jobs/implement-mtirok0xp.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtitinwv2.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtitw2208.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtiu3vhch.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtiui4v3h.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtiurkp9n.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtiuy6im2.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtiv6q4bx.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtivdwvil.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtivvipsr.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtiw6n36l.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtiwbf2wi.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtjvrfrl0.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtjwmqd77.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtjx15tjp.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtjxube25.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/implement-mtjzh4bcq.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.agy-staff/jobs/research-mtipthdjd.result.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-before-dev/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-brainstorm/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-break-loop/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-channel/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-channel/references/command-reference.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-channel/references/forum.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-channel/references/progress-debugging.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-channel/references/workers.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-channel/references/workflows.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-check/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-continue/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-finish-work/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/change-agents.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/change-context-loading.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/change-hooks.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/change-spec-structure.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/change-workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/customize-local/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/context-injection.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/generated-files.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/spec-system.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/task-system.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/platform-files/agents.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/platform-files/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/platform-files/platform-map.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-session-insight/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-session-insight/references/cli-quick-reference.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-session-insight/references/triggering-patterns.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-spec-bootstrap/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-spec-bootstrap/references/mcp-setup.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-spec-bootstrap/references/spec-writing.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-start/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.agents/skills/trellis-update-spec/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.trellis/agents/check.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.trellis/agents/implement.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.trellis/workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/agents/trellis-check.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/agents/trellis-implement.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/agents/trellis-research.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/commands/trellis/continue.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/commands/trellis/finish-work.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-before-dev/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-brainstorm/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-break-loop/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-channel/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-channel/references/command-reference.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-channel/references/forum.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-channel/references/progress-debugging.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-channel/references/workers.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-channel/references/workflows.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-check/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/change-agents.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/change-context-loading.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/change-hooks.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/change-spec-structure.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/change-workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/customize-local/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/context-injection.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/generated-files.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/spec-system.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/task-system.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/platform-files/agents.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/platform-files/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/platform-files/platform-map.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-session-insight/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-session-insight/references/cli-quick-reference.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-session-insight/references/triggering-patterns.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-spec-bootstrap/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-spec-bootstrap/references/mcp-setup.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-spec-bootstrap/references/spec-writing.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/.zcode/skills/trellis-update-spec/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-05T15-04-28/AGENTS.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-before-dev/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-brainstorm/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-break-loop/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-channel/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-channel/references/command-reference.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-channel/references/forum.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-channel/references/progress-debugging.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-channel/references/workers.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-channel/references/workflows.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-check/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-continue/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-finish-work/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/change-agents.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/change-context-loading.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/change-hooks.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/change-spec-structure.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/change-workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/customize-local/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/context-injection.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/generated-files.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/spec-system.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/task-system.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/platform-files/agents.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/platform-files/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/platform-files/platform-map.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-session-insight/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-session-insight/references/cli-quick-reference.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-session-insight/references/triggering-patterns.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-spec-bootstrap/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-spec-bootstrap/references/mcp-setup.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-spec-bootstrap/references/spec-writing.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-start/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.agents/skills/trellis-update-spec/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.dsh/DSH.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.dsh/skills/trellis-continue/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.dsh/skills/trellis-finish-work/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.dsh/skills/trellis-start/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.trellis/agents/check.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.trellis/agents/implement.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.trellis/workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/agents/trellis-check.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/agents/trellis-implement.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/agents/trellis-research.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/commands/trellis/continue.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/commands/trellis/finish-work.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-before-dev/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-brainstorm/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-break-loop/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-channel/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-channel/references/command-reference.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-channel/references/forum.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-channel/references/progress-debugging.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-channel/references/workers.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-channel/references/workflows.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-check/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/change-agents.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/change-context-loading.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/change-hooks.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/change-spec-structure.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/change-workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/customize-local/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/context-injection.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/generated-files.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/spec-system.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/task-system.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/workflow.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/platform-files/agents.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/platform-files/overview.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/platform-files/platform-map.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-session-insight/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-session-insight/references/cli-quick-reference.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-session-insight/references/triggering-patterns.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-spec-bootstrap/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-spec-bootstrap/references/mcp-setup.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-spec-bootstrap/references/spec-writing.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/.zcode/skills/trellis-update-spec/SKILL.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `.trellis/.backup-2026-09-17T05-45-35/AGENTS.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |
| `server/internal/webassets/dist/fonts/roncin-hbl/README.md` | 本地额外 | 历史保留 | 仅路径和所属备份/产物目录识别，未做内容语义核对 | `baseline-ledger.jsonl` |

## 本地忽略技能

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.agents/skills/roncin-web-stack/SKILL.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-before-dev/SKILL.md` | 本地额外 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-brainstorm/SKILL.md` | 本地额外 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-break-loop/SKILL.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-channel/SKILL.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-channel/references/command-reference.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-channel/references/forum.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-channel/references/progress-debugging.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-channel/references/workers.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-channel/references/workflows.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-check/SKILL.md` | 本地额外 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-continue/SKILL.md` | 本地额外 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-finish-work/SKILL.md` | 本地额外 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/SKILL.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/add-project-local-conventions.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/change-agents.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/change-context-loading.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/change-hooks.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/change-skills-or-commands.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/change-spec-structure.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/change-task-lifecycle.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/change-workflow.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/customize-local/overview.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/bundled-skills.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/context-injection.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/generated-files.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/multi-agent-channel.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/overview.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/spec-system.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/task-system.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/workflow.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/local-architecture/workspace-memory.md` | 本地额外 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/platform-files/agents.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/platform-files/hooks-and-settings.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/platform-files/overview.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/platform-files/platform-map.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-meta/references/platform-files/skills-and-commands.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-session-insight/SKILL.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-session-insight/references/cli-quick-reference.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-session-insight/references/triggering-patterns.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-spec-bootstrap/SKILL.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-spec-bootstrap/references/mcp-setup.md` | 本地额外 | 已修正 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes；本轮按当前CLI 0.6.17帮助/本地源码修正文案，复核同内容副本、链接与命令边界 | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-spec-bootstrap/references/repository-analysis.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-spec-bootstrap/references/spec-task-planning.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-spec-bootstrap/references/spec-writing.md` | 本地额外 | 待确认 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-start/SKILL.md` | 本地额外 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |
| `.agents/skills/trellis-update-spec/SKILL.md` | 本地额外 | 已核实无需修改 | 全文阅读（相同 SHA-256 共享语义核查）+ 本地入口/命令/配置对照；运行验收范围见 notes | `framework-final-ledger.jsonl` |

## 用户未提交任务

| 文件 | 版本范围 | 结论 | 核查深度 | 依据记录 |
| --- | --- | --- | --- | --- |
| `.trellis/tasks/10-30-sea-order-page-layout/prd.md` | 本地额外 | 待确认 | 仅路径、字节数、SHA-256完整性记录 | `baseline-ledger.jsonl` |

