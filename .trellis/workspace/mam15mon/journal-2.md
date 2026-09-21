# Journal - mam15mon (Part 2)

> Continuation from `journal-1.md` (archived at ~2000 lines)
> Started: 2026-09-20

---



## Session 80: Pro 规范小修清单：路由导航与 DOM 定位偏离
<!-- trellis-session: v=2 fp=3b7fd74d0d72be0f -->

**Date**: 2026-09-20
**Task**: Pro 规范小修清单：路由导航与 DOM 定位偏离
**Branch**: `fix/pro-convention-small-fixes`

### Summary

审计后小修六项：formErrorUtils 类名常量化+antd 升级哨兵测试；登录回调 4 处站内跳转改 history.push；partner-detail 角色按路由段解析；BusinessTagModal 加载器 ref 转发消除依赖缺项；enterprise-resources 改 useSearchParams；tabCloseGuard 弃静态 Modal.confirm 兜底改原生 confirm 降级（TagsView 测试同步改桥接注入）。check:fast 855 passed/12 skipped。同会话建 09-20-data-layer-direction 任务（规划态，数据层方向待定：React Query vs 沉淀规范 Hook）。

### Git Commits

| Hash | Message |
|------|---------|
| `0b520206` | fix(web): 对齐官方推荐的导航与 DOM 定位细节 |
| `fd939673` | docs(spec): 沉淀 antd 内部细节哨兵测试与站内导航禁令 |

### Status

[OK] **Completed**


## Session 81: React Query v5 选型落地与阶段0基建（随后暂停）
<!-- trellis-session: v=2 fp=0ea5792c2a51f1d7 -->

**Date**: 2026-09-20
**Task**: React Query v5 选型落地与阶段0基建（随后暂停）
**Branch**: `refactor/data-layer-react-query`

### Summary

数据层方向定案 React Query v5：实测推翻 umi 插件方案（底层为2021年 @ahooksjs/use-request v2、vitest 拿不到 '@umijs/max' 注入导出、无卸载保护）；阶段0基建完成并提交（依赖/queryClient 单例/Provider/测试工具/冒烟）。检测到并行会话推进总部业务边界任务且覆盖阶段1文件（MyApplicationHistoryDrawer、commissions 域），用户决策暂停阶段1/2，待其落地后恢复。恢复指引见任务 implement.md 暂停点。

### Git Commits

| Hash | Message |
|------|---------|
| `b4f3147d` | feat(web): 接入 React Query v5 基建 |

### Status

[OK] **Completed**


## Session 82: 总部工作台只读与分公司业务办理边界（收尾验证与提交）
<!-- trellis-session: v=2 fp=1f4a2c9e07b6f1e7 -->

**Date**: 2026-09-20
**Task**: headquarters-business-boundary（总部只读与分公司办理边界）
**Branch**: `refactor/data-layer-react-query`

### Summary

接手并行会话已完成大半的实施并收尾：核心机制为 access 显式经营权限分类 + `Principal.CanOperateBusiness`（启用公司工作台）+ `WorkspaceOrganizationID` 会话锚点（跨组织只读定位不再误判为切换）+ 经营写范围收敛当前公司；混合权限拆分 bill.configure / commission.configure；后台任务重试按业务种类门禁；前端 access.ts `canOperateBusiness/canOperateOrganization` 覆盖订单/往来单位/财务六域与工作台待办。本会话补齐：定向测试（含注入集成库真跑 CommissionApplication Postgres 用例）、生成物三条链幂等核验、开发库存量归属只读核验（41 经营表总部行数 0、非公司归属 0、子根组织错配 0，此前连接失败已补做成功）、check:fast 全绿（WEB 47s / SERVER 115s）。split.tsx 与 SeaDocumentSection.tsx 存量 20 个 noUnusedImports warning 属先前常量拆分任务遗留（HEAD 已存在），未顺带清理。分三组提交后归档任务。

### Git Commits

| Hash | Message |
|------|---------|
| `efe6314a` | feat(server): 经营办理收敛到启用公司工作台 |
| `23459708` | feat(web): 业务办理入口按公司工作台身份受控 |
| `c27876f2` | docs(spec): 收紧经营归属规范的跨组织办理例外 |
| (auto) | chore(task): archive 09-20-headquarters-business-boundary |

### Status

[OK] **Completed**


## Session 83: 数据层迁移 React Query v5：基建+阶段1/2+规范沉淀
<!-- trellis-session: v=2 fp=d36db5429e5e8437 -->

**Date**: 2026-09-20
**Task**: 数据层迁移 React Query v5：基建+阶段1/2+规范沉淀
**Branch**: `refactor/data-layer-react-query`

### Summary

选型实测推翻 umi 插件方案（v2 老包/vitest 不可用/无卸载保护），定案 @tanstack/react-query v5 直装。阶段0 基建（queryClient 单例/Provider/测试工具）；阶段1 迁移 4 个 workbench Drawer 与 CommissionCreateModal（sequenceRef/手写防抖全删）；阶段2 订单详情 8 state+五重竞态收敛为单条聚合 useQuery（消费方零改动）。规范沉淀 state-management.md（key 规范/silent 与 meta.errorMessage 错误提示约定/umi 插件禁用依据）与 quality 禁令。check:fast 869 passed/12 skipped 双零。剩余批次另立任务：use-order-create-options、use-order-fee-options、use-order-lock-state、VerificationWorkbench、BillCreationWorkbench、partner-detail、fees、split、cashflows、invoices、useWorkbenchOverview 等。

### Git Commits

| Hash | Message |
|------|---------|
| `b4f3147d` | feat(web): 接入 React Query v5 基建 |
| `e268ca41` | refactor(web): workbench 抽屉与佣金创建弹窗迁移 React Query |
| `914dfc28` | refactor(web): 订单详情数据聚合迁移 React Query |

### Status

[OK] **Completed**


## Session 84: 数据层迁移第二批：剩余手写请求链全部收敛
<!-- trellis-session: v=2 fp=b7ee62cb89e73913 -->

**Date**: 2026-09-20
**Task**: 数据层迁移第二批：剩余手写请求链全部收敛
**Branch**: `refactor/data-layer-migration-batch2`

### Summary

迁移剩余 12 个手写服务端状态文件：orders 三 hook（lock-state/fee-options/create-options）与 fees 页、VerificationWorkbench 链式拉取、cashflows/invoices/commissions/fees 索引、useWorkbenchOverview、BillCreationWorkbench 预览编排（双令牌+防抖）、partner-detail 聚合与 split 防抖预览。全量 869 passed/12 skipped 双零，tsc 干净，分四笔提交。useMasterDataCrud 保持用户亲改实现不动（后续收敛候选）。剩余事件驱动型请求（tagFilterRequestRef/ProFormSearchableSelect request/useCreditLimitIntervention）已记录待评估。

### Git Commits

| Hash | Message |
|------|---------|
| `04fdfc08` | refactor(web): orders 数据链迁移 React Query |
| `0edffa15` | refactor(web): finance 与工作台页面数据链迁移 React Query |
| `4b487253` | refactor(web): BillCreationWorkbench 预览编排迁移 React Query |
| `ab72886f` | refactor(web): partner-detail 与 split 数据链迁移 React Query |

### Status

[OK] **Completed**


## Session 85: 数据层迁移第三批：useMasterDataCrud 与事件驱动请求收敛
<!-- trellis-session: v=2 fp=158b16ef728e1677 -->

**Date**: 2026-09-20
**Task**: 数据层迁移第三批：useMasterDataCrud 与事件驱动请求收敛
**Branch**: `refactor/data-layer-migration-batch3`

### Summary

useMasterDataCrud（用户亲改的 ref 稳定回调实现）迁移 React Query：API 完全兼容、5 面板与模板零改动、无限重取防御语义以重渲染稳定性断言等价保留；useCreditLimitIntervention 与 fees 标签筛选迁移（防抖关键词进 queryKey + 渲染侧保活合并）；SearchableSelect 判定豁免（pro 原生 request）并沉淀事件驱动处置约定到 spec。全量 873 passed/12 skipped 双零，check:fast 通过。至此手写服务端状态链全部收敛完毕。

### Git Commits

| Hash | Message |
|------|---------|
| `08407069` | refactor(web): useMasterDataCrud 迁移 React Query |
| `4a7df6af` | refactor(web): 事件驱动型请求收敛并沉淀豁免清单 |

### Status

[OK] **Completed**

## 2026-09-20 Umi→Vite+React Router v8 全站迁移（09-20-vite-router-migration）

- 四段式落地：并存基建 → 原子切换（79 文件 codemod + @umijs/openapi 重生成
  25 个 Service）→ 删 Umi → 52 个测试 mock 迁移（4 个并行子代理按配方转换）。
- 关键坑与修法：@umijs/openapi 相对路径按自身包解析须传绝对路径；
  requestErrorConfig↔router 模块求值环导致 errorConfig TDZ 崩溃，history.ts
  改叶子模块经 bindRouter 注入；import.meta.glob 相对路径错级（../../pages）
  tsc/build 均不暴露，已用 adaptRoutes.test.tsx 真实加载 35 页面锁回归；
  dayjs zh-cn locale 数据需显式导入。
- 验收：全量 vitest 871 用例绿、tsc/biome 绿、check:fast 全过（77.8s）、
  vite build 14s、dev 冒烟（首页/模块转换/深链回退）通过；e2e 因沙箱浏览器
  进程被杀（launch ESRCH）无法执行，已记录差异待有浏览器环境补跑。
- 新增约定已落 spec：@/router/history 跳转、renderWithApp 测试包装、
  main.tsx App 包裹、generate-services.mjs 生成链路。

## 2026-09-20 往来单位黑名单视图切换（09-20-partner-blacklist-view）

- 需求：客户/供应商/国外代理三类列表页 Segmented 切换「全部|黑名单」视图；
  追加修复导出超限与拉黑操作人姓名显示。
- 契约：ListPartners/ExportPartners 新增 optional bool blacklisted（角色级过滤）；
  PartnerRole 增 blacklisted_by_name 联查回填；PartnerExportItem 补齐完整角色
  状态/联系人/更新时间。
- 关键决策：黑名单谓词不加 EnabledEQ(true)——SetPartnerRoleBlacklist 不写
  enabled，拉黑与停用正交，停用+拉黑必须仍可见；前端导出从自行调
  ListPartners pageSize=2000（必被 200 上限拒绝）切回 ExportPartners 服务端
  聚合接口，并补透传 keyword/enabled（旧导出无视搜索条件）；发现并修复
  toolBarRender={false} 导致 ProTable 标题栏从未渲染的死代码。
- 验收：trellis-check 0 阻断、PRD 6/6；go test -p 32 全量、vet、tsc、biome、
  vitest 876 用例全绿；无数据库迁移、无新增权限码。
- 提交：e84103ac（feat 黑名单视图）、bc338ccd（docs 契约沉淀）、
  882c6233（fix 导出正统化+操作人姓名）。


## Session 86: AI 友好架构改造：features 分层与依赖边界门禁
<!-- trellis-session: v=2 fp=78f442a21e25bfd3 -->

**Date**: 2026-09-20
**Task**: AI 友好架构改造：features 分层与依赖边界门禁
**Branch**: `main`

### Summary

跨页面共享能力迁入 web/src/features 公开入口，新增 check:architecture 依赖边界门禁（无豁免清单），统一同义状态展示，沉淀能力导航与 ADR 0016；check:web 与 build:web 全绿，任务已归档。

### Main Changes

- 四阶段顺序落地（各阶段实施 + 独立 check 代理核对后提交）：
  A `dd4e5e03` 建账工作台闭包等五个能力迁入 `web/src/features/`（bill-creation/bill-status/credit-control/audit/partners，公开入口 index.ts 最小导出面，无兼容 shim）；费用状态统一到 `statusMeta.ts` 映射（受控展示变更：已开账→已进账单 blue、已确认 green、已作废 default、草稿 gold）。
  B `954d6145` 旧 settings 四组面板按真实消费方迁入 fee-settings/exchange-rates/admin 页内 components；Welcome→`pages/workbench/index.tsx`（URL/权限不变）；删除旧 settings 整页与两条无人用转导出链，`/settings` 重定向保留。
  C `6ba94fff` `web/scripts/check-architecture.mjs`：@babel/parser AST 扫描，六类规则（跨页模块互引/features 反向依赖/绕过公开入口/通用层反依赖/路由精确授权/能力环），18 项正反例测试，接入 `check:web`+CI；无豁免清单。
  D `58fdbd89` 新增 capability-navigation.md 能力导航 + ADR 0016，纠正 Umi/hooks/模板路径漂移，AGENTS.md 补 features 层与边界规则。
- 关键坑：features 禁依赖 pages 迫使 `BillTermsCreditWarnings` 随工作台公开（账单页 BillEditModal 也消费）；编号规则后端在 `biz/orderconfig.go` 不在旧文档写的 number_rule.go；本机 pnpm/node 不在默认 PATH，需 export PATH="$HOME/.local/lib/nodejs/node-v24.21.0-linux-x64/bin:$PATH"。
- 验收：check:web 全绿（含 check:architecture 456 文件 0 违规；Vitest 876 用例通过）、build:web 通过；独立 check 代理判定 R1—R6 全满足、AC1—AC5 全过、无 BLOCKER；server/、OpenAPI/权限生成物零改动。
- 延期：订单候选缓存与 utils/options 领域化、全仓语义重复自动识别。


### Git Commits

| Hash | Message |
|------|---------|
| `dd4e5e03` | refactor(web): 提取建账工作台等跨页能力至 features 并统一费用状态展示 |
| `954d6145` | refactor(web): 设置页面板按真实归属迁移并删除旧设置整页 |
| `6ba94fff` | feat(web): 新增前端依赖边界检查器并接入 check:web 门禁 |
| `58fdbd89` | docs: 新增前端复用能力导航与模块边界 ADR，纠正规范漂移 |

### Status

[OK] **Completed**


## Session 87: 完成重复逻辑扫描与实仓审阅
<!-- trellis-session: v=2 fp=75118253cd539bb3 -->

**Date**: 2026-09-20
**Task**: 完成重复逻辑扫描与实仓审阅
**Branch**: `main`

### Summary

完成第二阶段 TS/Go 函数重复扫描工具、Node 14 与 Go 7 回归及 CI 接入。check:web 通过（160 文件、884 用例），871 源码文件扫描产生 8 组候选并全部人工审阅。输出稳定；全量测试含 localhost:3000 连接噪音，未扩展产品修复。更新能力导航与扫描契约。

### Git Commits

| Hash | Message |
|------|---------|
| `3cc5722a` | feat: 增加前后端疑似重复函数扫描与验证门禁 |
| `64cb1958` | docs: 记录重复扫描契约与八组候选审查结果 |

### Status

[OK] **Completed**


## Session 88: AI 友好架构二期：后端门禁与去重导航
<!-- trellis-session: v=2 fp=a8a073c46e7083fc -->

**Date**: 2026-09-20
**Task**: AI 友好架构二期：后端门禁与去重导航
**Branch**: `main`

### Summary

新增 Go 分层 import 门禁 scripts/layer-check-go（biz 的 ErrorReason 标识符级例外）接入 check:server 与 CI；提取重复报告确认的五组真实重复（候选 8→3 组，剩余为判定合理重复）；新增后端能力导航、补领域地图 features 新入口、修复规范断链并记录首轮审查结论；check:fast 全绿。

### Git Commits

| Hash | Message |
|------|---------|
| `98331d07` | feat: 新增后端分层边界门禁并接入 check:server 与 CI |
| `c77f0b14` | refactor: 提取重复报告确认的五组真实重复实现 |
| `1a72eaef` | docs: 新增后端能力导航并同步领域地图与规范纠错 |

### Status

[OK] **Completed**


## Session 89: 修复重复扫描三项可信性缺陷
<!-- trellis-session: v=2 fp=4576e81f724c7d72 -->

**Date**: 2026-09-20
**Task**: 修复重复扫描三项可信性缺陷
**Branch**: `main`

### Summary

修复TS运行时断言绑定误判、生成注释误排除、基线配置与结构校验，并保护失败扫描不覆盖基线；20 Node测试、Go工具测试、tsc、Biome与独立复查通过。版本2基线仅加元数据，实仓3组无变化。保留并行工作台改动。

### Git Commits

| Hash | Message |
|------|---------|
| `4c92c4f9` | fix: 修复重复扫描绑定误判与基线可比性校验 |

### Status

[OK] **Completed**


## Session 90: 工作台提成模块 UI 结构重构
<!-- trellis-session: v=2 fp=332f25d1fe9bf46d -->

**Date**: 2026-09-21
**Task**: 工作台提成模块 UI 结构重构
**Branch**: `main`

### Summary

调研提成管理全貌后按结构重构方向改造工作台提成 UI：巨石卡拆为 SectionCard 外壳的总览卡（本年/本月已发双指标 hero + 三桶箭头衔接 FlowStat 流程条 + 冲减单行摘要）与月度申请卡（Steps 三阶段状态流 + deriveStage 纯函数 + 去申请主按钮 + 预计可计提脚注）；红线行为（门禁三态、总部仅查看、提交刷新时序、四个下钻抽屉）经 trellis-check 逐条核实等价保留；工作台模块 23 用例与全量 887 用例通过。

### Git Commits

| Hash | Message |
|------|---------|
| `8ea4c421` | feat: 工作台提成模块重构为总览与月度申请双卡 |

### Status

[OK] **Completed**


## Session 91: 统一权限范围与停用仅本人授权
<!-- trellis-session: v=2 fp=5315c89347e60de8 -->

**Date**: 2026-09-21
**Task**: 统一权限范围与停用仅本人授权
**Branch**: `fix/permission-scope-alignment`

### Summary

后端按具体权限投影有效范围，前端权限判断不再跨角色拼接；停用 self 授权保留旧角色提示；修复订单详情和费用页权限门控，保留 lock-only 补录审批。完成契约生成、独立审查与 check:fast（前端917通过12跳过、后端测试及vet漏洞扫描通过）。未修改数据库角色，现存self需管理员显式调整。

### Git Commits

| Hash | Message |
|------|---------|
| `3f6e5d67` | fix(auth): 统一权限范围投影并停用仅本人授权 |

### Status

[OK] **Completed**


## Session 92: 海运提单模式与分单展示简化
<!-- trellis-session: v=2 fp=0de519ddcd1c2aa4 -->

**Date**: 2026-09-21
**Task**: 海运提单模式与分单展示简化
**Branch**: `main`

### Summary

DIRECT 隐藏 HBL 整节与导航，优化提单和签发主体文案；补齐模式切换、草稿恢复与公共分节可见性测试，check:web 全量通过。

### Git Commits

| Hash | Message |
|------|---------|
| `fe7a5650` | fix(web): 简化海运提单模式并在直单下隐藏分单区域 |

### Status

[OK] **Completed**


## Session 93: 往来单位当前公司范围与人员部门展示
<!-- trellis-session: v=2 fp=5f2231240de3e694 -->

**Date**: 2026-09-21
**Task**: 往来单位当前公司范围与人员部门展示
**Branch**: `main`

### Summary

统一伙伴及子资料为当前公司范围；人员候选和岗位保存限定公司及部门团队，不穿透其他公司；新增部门展示契约及生成物。前端63项定向、真实PG12子场景和check:fast全量通过。当前源码未复现旁置公司下拉；同部门同名及既有200人候选限制已记录。

### Git Commits

| Hash | Message |
|------|---------|
| `76aa783e` | feat(partner): 统一当前公司档案范围与人员部门展示 |

### Status

[OK] **Completed**


## Session 94: 全仓死代码与未使用依赖清理
<!-- trellis-session: v=2 fp=2ece12cfcc650b6e -->

**Date**: 2026-09-21
**Task**: 全仓死代码与未使用依赖清理
**Branch**: `main`

### Summary

逐项复核GLM报告后删除72个前端遗留文件，精确清理14个后端文件中的死符号与接口链；移除5个前端直接依赖并整理go.sum。check:fast全部通过，前端937通过12跳过，生产构建通过。保留Sentry、回填工具、测试钩子、重复实现。

### Git Commits

| Hash | Message |
|------|---------|
| `745b7320` | chore(deps): 移除未使用前端依赖并整理 Go 校验和 |
| `effd92f3` | refactor: 清理确认不可达的前后端代码与迁移遗留 |
| `2a7e10c6` | docs: 记录死代码清理复核证据与验收结果 |

### Status

[OK] **Completed**
