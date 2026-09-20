# AI 友好架构改造

## 目标与价值
让 AI 开发时能发现现有能力、判断代码归属并复用稳定入口；用自动检查阻止已识别的依赖越界，减少跨页面耦合和同义展示漂移。交付包含真实代码迁移、门禁与可定位的开发导航。

## 背景
项目已有 OpenAPI/权限唯一真相源和 `components/ui` 模板体系，但尚无 features 层与依赖边界门禁。审查发现建账工作台、审计展示跨页面依赖，旧设置页遗留面板被多个领域导入，以及 AI 指南与 Vite/现有 hooks 目录不一致。完整证据见 `research/architecture-audit.md`。

## 需求
- R1：迁移已识别的共享业务能力。三处建账入口共同消费独立工作台（`web/src/pages/orders/fees.tsx:17`、`web/src/pages/finance/fees/index.tsx:22`、`web/src/pages/finance/bills/index.tsx:34`）；审计展示独立于管理页（`web/src/pages/partners/components/AuditLogSection.tsx:19`）；往来单位候选标签和信用控制 Hook 按领域归属。
- R2：只服务单一活动页面的设置面板归入真实使用页面。移除确认无调用的旧设置整页及转导出（`web/src/pages/settings/index.tsx:21`、`web/src/pages/master-data/components/NumberRulesPanel.tsx:1`、`web/src/pages/finance/fee-settings/index.tsx:18`）。工作台入口与私有组件同目录（`web/src/pages/Welcome.tsx:6`），URL 与权限不变。
- R3：同一费用状态使用已有统一展示映射（`web/src/constants/statusMeta.ts:60`），替换建账候选中的分散实现（`web/src/pages/finance/bills/components/billWorkbenchFeeColumns.tsx:35`）；账单映射提取为财务公开能力。不同实体状态不强行统一。
- R4：依赖越界可自动失败。覆盖跨页面模块导入、features 反向依赖 pages、外部绕过领域公开入口、通用 UI/Hook/工具反向依赖 pages/features；识别别名、相对路径、重导出、类型导入和字面量动态导入。路由装配与测试采用精确规则。
- R5：开发规范明确“搜索已有能力→判断归属→消费公开入口→验证”；导航列出真实已有能力及入口，纠正 Umi、hooks 与模板路径的过时描述。公共模板继续复用。
- R6：保留已有未提交改动，分组验证并提交；完成完整前端门禁、检查器正反例与路由/构建验证。

## 验收标准
- AC1（R1/R2）：所有审查确认的跨模块产品导入消除，不保留旧路径兼容转导出；工作台、账单、费用、审计、费用设置、汇率与管理中心既有入口正常。
- AC2（R3）：建账候选与费用台账相同状态的标签/颜色一致；账单列表与建账结果共享账单状态映射。业务状态流转、权限和数据提交语义不变。
- AC3（R4）：合法模块内部与公开入口引用通过；每类越界的独立违规样例失败，并输出文件、行号与原因；别名和相对路径不能绕过，现有仓库零边界违规。接入 `check:web` 后 CI 自动执行。
- AC4（R5）：AI 从前端规范索引能找到目录规则、能力导航和检查命令，所列文件真实存在；无新建全局万能业务组件或空领域目录。
- AC5（R6）：针对性回归、`git diff --check`、完整 `pnpm run check:web` 与 `pnpm run build:web` 通过；无法运行项明确记录原因，不声称通过。

## 范围外
不改服务端业务规则、数据库、API、权限清单、部署架构；不实现微前端或跨端平台；不批量重写所有页面与请求 Hook；不重做订单候选缓存；不改全站视觉。仅 R3 已列同义状态展示允许统一标签/颜色。

## 约束与规划状态
遵循上线前最小闭环、禁止 TDD、不加兼容分支、生成物不手改。用户已同意本任务的审查与规划；本稿与设计、执行计划待最终摘要批准后进入实施。没有阻塞性的仓库事实待确认。
