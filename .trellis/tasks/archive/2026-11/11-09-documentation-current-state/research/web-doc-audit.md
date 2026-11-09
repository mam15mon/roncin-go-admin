# 前端文档现状审计（规划阶段）

- Query：对照当前前端实现核查 README、协作入口、web 规范及架构地图，寻找有证据的陈旧说明。
- Scope：internal；只读研究，未修改产品代码或现行文档。
- Date：2026-11-09

## 已确认差异

### W1：两份 web README 仍是上游模板，无法指导当前项目开发

- 文档：`web/README.md:1`、`:22`、`:35`、`:89`、`:104`；`web/README.zh-CN.md:3`、`:21`、`:34`、`:88`、`:103`。
- 当前文字：项目名称 Ant Design Pro，宣称使用 Umi Max/utoopack、内建 AI 助手/区块模板，要求克隆上游并运行 npm，提供 `simple` 删除模板命令。
- 源码：`web/package.json:7-29` 的构建/开发脚本为 Vite，没有 `simple`；`:34-89` 是 React 19、React Router 8、Query 5、antd 6、Vite 7、Vitest 4；`web/config/routes.ts:120-167` 包含海运出口、空运出口及统一新建订单；`web/src/router/adaptRoutes.tsx:13-33` 使用 React Router 与 Vite 页面扫描。
- 命令与包管理依据：`package.json:5-17`、`AGENTS.md` 的前端边界/日常命令。
- 已运行本地 Markdown 目标存在性扫描：`web/README.md:22,24` 的 `./docs/cheatsheet.en-US.md`、`web/README.zh-CN.md:21,23,124` 的 `./docs/cheatsheet.zh-CN.md` 均不存在。
- 建议：将 `web/README.md` 改为 Roncin 前端入口，介绍真实架构、目录、pnpm 启动/定向验证/客户端生成与现行规范链接；`README.zh-CN.md` 用中文短入口转向同一权威说明，避免复制整份介绍。上游来源可保留为来源说明，但不能作为当前使用指南。

### W2：web/CLAUDE.md 的命令、权限来源、测试顺序与当前约束冲突

- `web/CLAUDE.md:11,18-19,41,51-54`：仍要求 `npx`，且把全量 lint 与 antd lint 作为每次提交强制门禁。
- `web/CLAUDE.md:29`：称 access 从 permissions + roleScopes 推导权限；实际 `web/src/access.ts:135-154` 只从 `permissionCapabilities` 按该权限的 dataScope 建立能力，`:157-160` 消费该检查器。`.trellis/spec/web/frontend/type-safety.md:192-241` 已明确不能混用角色范围。
- `web/CLAUDE.md:108-111`：要求先写测试再实现；`AGENTS.md:30-31` 明确禁止 TDD。
- 该文档全部使用英文，与 `AGENTS.md:26-27` 的开发者文档语言约束冲突；`web/CLAUDE.md:60` 还重复了一级标题和整组通用行为说明。
- 依据：`web/package.json:56` 已本地安装 `@ant-design/cli`，可使用 `pnpm --dir web exec antd ...`；`.trellis/spec/web/frontend/quality-guidelines.md:9-87` 已定义分阶段最小检查。
- 建议：改为中文、精简为根 AGENTS + 前端规范的入口，保留项目特有关键定位；用 pnpm exec 查询本地 CLI；权限按 permissionCapabilities；实现后按风险补测试；提交遵循分阶段检查而非再造一套门禁。

### W3：共享航程专项规范仍要求订单主详情持久草稿，已被当前实现退役

- 文档：`.trellis/spec/web/frontend/sea-master-voyage-inheritance.md:129` 说详情初始化后恢复本地草稿；`:136-146` 仍要求持久化/恢复版本、失败保留草稿、成功清草稿。`:152-163` 的“草稿”有些指内存编辑基线，需保留该业务语义而不是机械删除全部“草稿”。
- 源码：`web/src/pages/orders/detail.tsx:1312-1328` 主详情只传 tabKey、draftScope、actionsRef、initialValues，不传 draftPathname；`web/src/components/ui/order-template/OrderFormTemplate.tsx:168-173` 只有 tabKey + draftPathname + draftScope 齐全才生成草稿键，缺少时不读写持久草稿；`:177-181` 仍保留实时 dirty 离开保护。
- 回归证据：`web/src/pages/orders/detail-leave-guard.test.tsx:100-128` 覆盖海运和空运主详情，预埋旧 storage 草稿后仍使用服务端值、离开取消保留当前输入；这里只读测试源码，未执行测试。
- 已更新且一致的规范：`.trellis/spec/web/frontend/state-management.md:192-285`、`organization-switch.md:19`、`capability-navigation.md:20`。
- 建议：明确主详情仅保留当前挂载输入、版本与单证基线；不恢复/写入旧 storage 草稿；保存/显式刷新通过 resetTo 更新；离开确认后放弃，重开读取服务端。新建订单与独立 HBL 编辑的持久化契约保留。

### W4：web/backend 五份规范仍为空模板

- 文档：`.trellis/spec/web/backend/{directory-structure,database-guidelines,error-handling,logging-guidelines,quality-guidelines}.md`，各文件均包含 `(To be filled by the team)` 与模板注释，directory-structure 还展示 `src/...` 假目录。
- 权威现状：`.trellis/spec/web/backend/index.md:3-5` 已明确后端实际在 server，本目录不维护第二套规则；实际实现目录为 `server/internal/{biz,data,service,server}`，完整规范位于 `.trellis/spec/server/backend/`。
- 建议：五份文件替换为中文简短导航，分别链接 `../../server/backend/<同名文件>.md`；保留现有文件入口可避免既有加载路径失效，不复制服务端规则，不新增后端实现。

### W5：组件规范的页面类型表被后续段落拆断

- 文档：`.trellis/spec/web/frontend/component-guidelines.md:9-12` 是表头与前两行；`:14-23` 插入一个完整章节；`:24-25` 的主数据/列表两行已脱离原表，渲染不再属于同一表格。
- 建议：将两行恢复至原表，章节内容留在表后；仅修复 Markdown 结构，不改变业务 UI 规范。

## 覆盖清单与核查深度

以下 25 份文件已纳入路径/关键术语扫描；对照代码重点核实了上述差异及当前依赖、权限、路由、草稿机制。规划研究不等于逐个规范的全部业务矩阵运行验收。

| 文档 | 本轮结论 |
| --- | --- |
| `web/README.md` | W1，需重写项目入口 |
| `web/README.zh-CN.md` | W1，需统一入口及修复链接 |
| `web/CLAUDE.md` | W2，需同步命令/权限/协作要求 |
| `.trellis/spec/web/backend/index.md` | 已正确转向 server，不另建后端规范 |
| `.trellis/spec/web/backend/directory-structure.md` | W4，占位模板 |
| `.trellis/spec/web/backend/database-guidelines.md` | W4，占位模板 |
| `.trellis/spec/web/backend/error-handling.md` | W4，占位模板 |
| `.trellis/spec/web/backend/logging-guidelines.md` | W4，占位模板 |
| `.trellis/spec/web/backend/quality-guidelines.md` | W4，占位模板 |
| `.trellis/spec/web/frontend/index.md` | 现行索引链接均有效；提及的 web/types 目录不存在，建议以 type-safety 生成物清单替换模糊列举 |
| `.trellis/spec/web/frontend/capability-navigation.md` | 已记录详情草稿退役；现有 Markdown 本地目标均存在 |
| `.trellis/spec/web/frontend/column-settings.md` | 本轮未确认过期点；列设置/冻结/调宽细节留执行阶段按风险复核 |
| `.trellis/spec/web/frontend/component-guidelines.md` | W5；patch 引用与 pnpm-workspace.yaml:4 一致 |
| `.trellis/spec/web/frontend/dingtalk-qr-login.md` | 未发现本地链接失效；真实钉钉历史验收记录保留，不伪称本轮真机验证 |
| `.trellis/spec/web/frontend/directory-structure.md` | hooks 路径与路由扫描描述有真实文件；未确认过期点 |
| `.trellis/spec/web/frontend/duplicate-report.md` | 已标业务标签历史内容退役；相关 package.json 命令存在 |
| `.trellis/spec/web/frontend/finance-current-workspace.md` | 当前公司契约明确；未确认过期点 |
| `.trellis/spec/web/frontend/finance-ledger-snapshots.md` | 官方 request/onLoad 采纳口径明确；未确认过期点 |
| `.trellis/spec/web/frontend/hook-guidelines.md` | 本轮未确认过期点 |
| `.trellis/spec/web/frontend/organization-switch.md` | 已同步主详情无持久草稿；未确认过期点 |
| `.trellis/spec/web/frontend/quality-guidelines.md` | 分阶段检查与根脚本一致；本轮未确认过期点 |
| `.trellis/spec/web/frontend/sea-master-voyage-inheritance.md` | W3，需同步持久草稿退役 |
| `.trellis/spec/web/frontend/state-management.md` | 已更新主详情持久草稿退役，作为 W3 的权威关联入口 |
| `.trellis/spec/web/frontend/type-safety.md` | 已正确区分 permissionCapabilities 与角色展示 |
| `.trellis/spec/domain/architecture-map.md` | 海运/空运入口存在；正文 `server/backend/order-fee-correction.md` 是规范相对标识而非实际根目录文件，建议改可点击相对链接明确定位 |

## 外部参考

未访问外网。依赖和功能现状全部以仓库内 package.json、路由、实现和测试为依据，不声称这些依赖是外部最新发布版本。

## 限制与未确认项

- 未启动前后端、浏览器或数据库，未执行 Vitest；这里只核查文档与源码的对应关系。
- Markdown 目标检查覆盖本轮上述文件中的 `](...)` 本地链接，去掉锚点后检查路径存在；尚未验证 heading 锚点、HTML 引用、外部链接可达性或代码块示例。
- 没有将“规范要求”与“现有实现”一概等同。比如主动变动检查是面向协作业务的要求，不应因某页面尚未迁移就削弱规范。遇到实现缺口应另列而非改文档掩盖。
- 没有修改 web 中第三方 CODE_OF_CONDUCT、许可证、字体说明或技能模板；是否另需状态标识由主会话按总盘点范围决定。


## 补充：仓库自带 web 技能文档

按主会话补充要求，进一步核查下列两份仓库内技能。这里只作为文档审计对象读取，没有应用技能或执行其中命令。

### W6：antd 本地技能仍使用 npx 入口

- `web/.claude/skills/antd/SKILL.md:12-14` 的 allowed-tools、`:23` 的 Setup、`:35-192` 命令示例均使用 npx；`:210-211` 的 MCP 示例也以 npx 运行 antd。
- 依据：根 `AGENTS.md` 的前端依赖/脚本约束，`web/package.json:56` 已安装本地 `@ant-design/cli`。
- 建议：技能开发者说明改中文，命令统一在 web 目录使用 `pnpm exec antd ...`（或从仓库根目录 `pnpm --dir web exec antd ...`），同步 allowed-tools 和 MCP 片段；MCP 用 pnpm + `--dir` 指定 web 绝对目录 + `exec antd mcp`，避免根目录找不到 web 依赖。保留用户明确要求才发送外部 issue 的边界，不运行提交动作。

### W7：pro-upgrade 技能仍假设上游 Umi 模板拥有本项目框架与生成物

- `web/.claude/skills/pro-upgrade/SKILL.md:10-14,114-138` 使用 npm/npx。
- `:53-68` 识别“框架文件”的清单仍包括已不存在的 `config/config.ts`、`src/app.tsx` 等；`:68,107` 仅保护 `src/services/ant-design-pro/`，`:75` 反而把其他 services 全当自定义文件，会遗漏真正禁止手改的 `src/services/roncin/`。
- `:87,102` 推荐从 umi 改为 @umijs/max；`:96-99` 建议采用上游依赖版本与模板结构。当前项目已经迁移到 Vite + React Router + 公司工作区治理，直接套用上游会破坏项目架构。
- 依据：`web/package.json:7-29`、`web/src/router/adaptRoutes.tsx:13-33`、`web/src/app/AppProvider.tsx`、`web/src/app/AppLayout.tsx`、根 `package.json:15-19` 的真实生成入口。
- 建议：技能改为针对 Roncin 的受控依赖升级说明：上游模板仅参考，不自动替换现行框架/路由/鉴权/权限文件；真实生成物统一列 `src/services/roncin/`、`src/permissions.generated.ts`、`src/enums.generated.ts`（后者请执行阶段再次核对完整生成物清单）；根目录 pnpm install，按改动风险使用局部检查与需要的构建；生成客户端使用根 `pnpm run generate:web-client`。保留用户业务代码与未提交工作，不能让“升级到最新”暗含恢复 Umi。

补充覆盖后，共 27 份文件进入本研究范围。Trellis 框架技能与本地 web 上游技能不是同一归属；这两份是仓库自带前端协作入口，应按当前项目修正，而不是仅因名称为 SKILL.md 就忽略。
