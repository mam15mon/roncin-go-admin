# 执行计划

## 进入实施前
- [x] 用户批准创建任务和规划。
- [x] 完成实际代码、路由、现有规范与检查入口取证。
- [x] PRD 收敛，设计、执行计划与研究证据落盘。
- [ ] 展示最终规划摘要并取得后续明确批准；之前不运行 `task.py start`。
- [ ] 加载 trellis-before-dev 与 roncin-web-stack；组件实现涉及 Ant Design 时加载相应技能；主会话按 Trellis 自动模式派发 implement/check 子代理并注入 JSONL。
- [ ] 确認 Git 状态，保留 `.trellis/tasks/09-20-partner-blacklist-view/` 等其他工作。
- [ ] `pnpm install --frozen-lockfile` 建立依赖；记录环境问题，之后增加解析器直依赖只用 pnpm。

## A. 领域共享能力与状态
- [ ] 实施代理负责 `features` 初始能力、建账与审计原文件及调用方、候选标签与信用 Hook、对应测试。
- [ ] 按设计迁移闭包，逐一更新生产与测试 mock 引用，避免旧路径重导出。
- [ ] 费用状态复用已有映射；账单状态提取并复用；记录展示差异。
- [ ] 定向运行迁移测试、订单费用、财务入口、审计相关回归；检查迁移后静态测试中的源码路径。
- [ ] 运行修改文件 Biome、`pnpm --dir web tsc`、`git diff --check`；独立 check 代理核对行为/入口；主会话提交。

## B. 页面归属收敛
- [ ] 实施代理负责旧 settings 面板迁移、admin/finance 消费者及测试、workbench 入口与 routes 配置。
- [ ] 再次确认旧设置页及转导出无调用再删除；测试与注释路径同步。
- [ ] 验证管理中心、汇率、费用设置、工作台及路由适配测试；URL/权限不变。
- [ ] 定向 Biome、tsc、diff 检查后提交。

A、B 都涉及 admin 与 finance 的消费者；采用顺序执行，禁止两个实现代理同时写这些文件。

## C. 自动依赖边界
- [ ] 实施代理负责 `web/scripts/check-architecture*`、直接解析器依赖、lockfile 与根门禁入口。
- [ ] 实现检查器后补正反例测试（禁止 TDD），涵盖设计各规则与路径绕过。
- [ ] 增加并执行 `pnpm run check:architecture`，当前源码零违规，不使用遗留豁免清单。
- [ ] 确保 check:web/CI 链路执行新检查；定向检查与 diff 通过后提交。

## D. 导航和规范
- [ ] 主会话更新 AGENTS.md、前端指南、能力导航、领域架构图与 ADR，所有入口基于最终文件位置。
- [ ] 核对新建能力发现路径、公开导出与“何时提取”的说明，删除本次涉及指南的过时 Umi/hooks 描述。
- [ ] 检查规范链接与 `git diff --check` 后提交。

## 最终验收
- [ ] check 子代理加载 check.jsonl，审查 R1—R6、AC1—AC5 与边界规则实际覆盖。
- [ ] 执行一次 `pnpm run check:web`，包含依赖边界、权限/枚举一致性、Biome、tsc 与全量 Vitest。
- [ ] 执行 `pnpm run build:web`：本任务更改路由组件位置与开发依赖，因此验证生产入口与打包。
- [ ] 扫描旧导入路径、全部差异与 Git 状态，确认生成物、未提交用户工作、业务权限未被意外改动。
- [ ] 主会话按 finish-work 规范归档任务、记录提交/验证结果与延期项，不自行推送。

## 延期项
订单候选缓存与 `utils/options` 全量领域化、所有页面内部业务逻辑外提、全仓重复语义自动识别均不在本期。能力导航记录现有入口，未来出现实际跨模块需求再提取。
