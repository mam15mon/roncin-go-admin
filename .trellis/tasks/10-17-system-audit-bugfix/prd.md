# 系统体检缺陷修复

## 目标与用户价值

修复本轮体检确认的全部三类缺陷及同根因调用路径，让操作结果、工作台数据和财务导出可信；补齐真实 PostgreSQL 验收，验证后提交、合回 `main` 并安全删除本次任务分支。

## 已确认事实与需求

| 编号 | 优先级 | 证据与现状 | 期望行为 |
| --- | --- | --- | --- |
| R1 | P1 | `web/src/utils/requestClient.ts:98` 在全局处理错误后返回 undefined。浏览器模拟重置密码 409，`web/src/pages/admin/components/users/ResetPasswordModal.tsx:38` 仍提示成功、关闭并刷新。同类见角色保存、账户、编号规则、账单取消、对冲动作。 | 所有受此根因影响的命令式写操作，只有实际成功报文才显示成功、关窗、清草稿、导航或刷新。失败保留输入及可重试身份；拒绝、undefined、success:false、必要实体缺失均不算成功。 |
| R2 | P1 | `web/src/pages/workbench/MyCommissionDrawer.tsx:78` 等缓存键缺少用户/公司；`web/src/components/OrganizationSwitcher.tsx:36` 只清订单主数据，QueryClient 为全局单例。浏览器复现重新挂载抽屉先显示 A 数据，B 返回才替换。 | 所有业务查询缓存以已认证用户及当前工作台隔离。用户/公司变更及重新登录不能显示旧身份结果；旧在途响应不能污染新工作台。同工作台翻页/刷新缓存、公开钉钉配置预热和 Cookie 串行契约保持正确。 |
| R3 | P2 | `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:112` 按原始 dataIndex 导出；`web/src/pages/finance/bills/components/billColumns.tsx:131` 只在 render 拼币种。 | 共用财务台账导出明确字段及币种；金额无千分位且两位，汇率/比例/数量保留精度，时间本地可读，状态中文，文本转义并防公式执行。导出范围清楚，不把当前页冒称全量，不循环翻页。 |
| R4 | P1 | `.github/workflows/ci.yml:68` 普通 Go 测试未注入真实库配置，仅两个钉钉测试注入。`server/internal/data/integration_test_helper_test.go:28` 缺连接直接 SKIP，迁移另需开关。 | CI 持续执行完整真实 PostgreSQL data/migration 验证，显式配置隔离连接及开关；本次核对 PASS/FAIL/SKIP，修复启用后暴露的相关失败，跳过不算通过。 |
| R5 | 交付 | 初始工作区干净，main 基线 `851a3403`。 | 按验证批次提交；最终 main 包含修复和归档记录；只删除已合入的本次分支，保留其他分支、工作树及用户数据。 |

## 验收标准

- A1（R1）：真实请求层与组件回归证明重置密码、角色/客商及账单/对冲代表路径失败不报成功、不丢输入；全站同类写调用完成审计，成功相邻路径通过。
- A2（R2）：同一应用实例换用户、公司及 A→B→A，不显示旧缓存/迟到响应。提成、在途回款、申请历史及其他业务查询均受边界保护；公开登录配置测试通过。
- A3（R3）：六类台账（费用、账单、发票、资金流水、核销、对冲）验证币种、金额、状态、时间、特殊字符、下载清理和范围；覆盖多币种、负金额和小数汇率；空结果不生成文件。
- A4（R4）：完整 data/migration 包在一次性 PostgreSQL 实际执行，关键真实库用例不跳过，记录其他 SKIP 及原因，CI 覆盖同样要求。只清理本次拥有的测试资源，不重置开发库。
- A5（R1—R4）：定向回归、类型检查、最终 `pnpm run check:fast` 通过；浏览器复核此前两类已复现缺陷，公共模板与状态边界集成复核。
- A6（R5）：main 包含全部工作提交，任务归档，工作区干净；以 `git branch -d` 删除 `codex/system-audit-bugfix`，不强删或清理无关分支。

## 范围外与后续能力

附件真实上传、客户 PDF/Excel 账单发送确认、新运输业务、备份平台是体检建议的功能缺口，不属于本次 bug 修复。生产备份需部署侧核实，不借此次修复扩建这些能力。

## 约束

禁止 TDD，先实现后补回归；不手改生成客户端，不隐式改变全局请求错误返回契约。文档/开发注释中文；真实库用本次隔离资源，凭据不入库。实现和检查经 Trellis 子代理，主会话负责集成、规范、提交、合并及收尾。
