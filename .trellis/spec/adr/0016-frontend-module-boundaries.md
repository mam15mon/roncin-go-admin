# ADR 0016: 前端模块边界与 features 领域能力层

- 状态：已采纳
- 日期：2026-09-20（AI 友好架构改造，任务 `.trellis/tasks/09-20-ai-friendly-architecture/`）
- 主题：前端

## 背景

项目此前只有 `pages/` 与 `components/` 两层：跨页面共享的业务能力只能靠
相对路径深引其他页面内部文件。审查确认了三类实际伤害：

- 建账工作台被订单费用、财务费用、账单列表三个入口各自深引账单页私有组件；
  审计展示、往来单位候选标签、信用控制 Hook 也存在跨页面/跨层引用。
- 旧 `pages/settings/` 整页已无路由入口，但其面板被财务与管理页深引，
  另有无人使用的转导出链。
- 同一费用状态在 `constants/statusMeta.ts` 与建账候选列各有一套标签/颜色。

对 AI 会话而言，没有稳定公开入口就无法判断「该复用什么、代码归属哪里」；
没有自动门禁，约定会随迭代漂移。证据见
`.trellis/tasks/09-20-ai-friendly-architecture/research/architecture-audit.md`。

## 决策

- 新增 `web/src/features/<领域>/<能力>/` 层，只容纳**真实跨模块共享**的
  领域能力；公开入口为 `features/<领域>/index.ts` 或
  `features/<领域>/<能力>/index.ts`，内部模块相对互引，不建全局总 barrel。
- 页面模块边界：`pages/<领域>/`；`pages/finance/<模块>/` 为独立工作区边界，
  不同页面模块禁止互相导入。单一页面使用的实现留在页面私有。
- 新增 `web/scripts/check-architecture.mjs`（AST 解析，`@babel/parser`），
  强制：跨页面模块禁互引、features 禁依赖 pages、外部仅经 features 公开
  入口、`components/ui`/`hooks`/`utils`/`constants` 禁反向依赖
  pages/features、路由装配走精确授权清单（`router/adaptRoutes.tsx`）、
  features 公开能力之间有向环报错。以 `pnpm run check:architecture` 接入
  `check:web` 与 CI，正反例测试覆盖别名/相对路径、重导出、类型导入、
  字面量动态导入等绕过形态；**无豁免清单机制**。
- 同义状态展示统一到既有 `constants/statusMeta.ts` 映射；账单状态独立为
  `features/finance/bill-status`，不与费用状态合并。受控展示变更（建账候选
  「已开账」→「已进账单」及颜色对齐）在任务文档记录。
- 复用发现路径收敛为单一条款：先查
  [能力导航](../web/frontend/capability-navigation.md) → 页面私有就近 →
  真实跨模块使用再提取 → `check:architecture` 验证。

## 理由

- 公开入口让「复用什么、从哪进」变成可枚举的事实；AI 与新人按导航即可
  找到真实入口，不需要读全仓调用链。
- AST 而非正则：别名、`export from` 重导出、`import type`、字面量动态
  导入都能绕过文本匹配；显式声明 `@babel/parser` 为 web 开发依赖避免依赖
  偶然提升。不引入大型架构框架，保持上线前最小闭环。
- 无豁免清单：历史豁免会让边界检查退化为形式；当前仓库零违规是迁移一次
  做干净的结果，后续违规一律改源码归属而不是加白名单。
- 测试文件豁免产品边界检查：单测访问被测私有实现是合法内聚，不是依赖
  越界；但不对生产代码建立任何基线豁免。
- `import.meta.glob` 路由装配是合法机制，静态扫描不判定运行时拼接路径，
  由路由回归测试兜底。

## 后果

- 跨模块复用必须走 features 公开入口，页面间直接深引会被 CI 拒绝；
  提取新能力需同步更新能力导航，否则 AI 发现不到（导航链接须解析到
  真实文件）。
- 公开入口是契约面：修改 features 导出需检查全部消费方，删除导出按
  破坏性变更一次性同步，不留兼容 shim。
- 旧 `pages/settings/` 已删除，`/settings` 仅保留重定向；后续配置类页面
  一律落在真实归属（财务配置进 `pages/finance/`，平台管理进 `pages/admin/`）。
- 订单候选缓存与 `utils/options` 领域化等更深的语义重复治理明确延期，
  不在本决策内（后续于 2026-09-20 完成：候选能力已迁入 `features/orders/options`、
  `features/master-data/{currencies,shipping-lines}` 等领域入口，旧聚合已删除）。
