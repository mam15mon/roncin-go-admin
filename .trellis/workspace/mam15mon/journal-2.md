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


## Session 95: 修复共享港口导致海运订单列表加载失败
<!-- trellis-session: v=2 fp=932d4730ba68cdad -->

**Date**: 2026-09-22
**Task**: 修复共享港口导致海运订单列表加载失败
**Branch**: `main`

### Summary

创建并完成 shared-location-order-fix：摘要按共享或本公司港口解析，订单港口机场写入复用候选范围及启用校验。真实 PostgreSQL 隔离回归、相邻路径及 check:fast 全部通过；只读核查17条运输执行和34个地点引用，未改业务数据。

### Git Commits

| Hash | Message |
|------|---------|
| `f9b567a2` | fix: 修复共享港口导致海运订单列表加载失败 |

### Status

[OK] **Completed**


## Session 96: 系统管理与公司经营分离及种子验收
<!-- trellis-session: v=2 fp=5bcf1ac93aeff08b -->

**Date**: 2026-09-22
**Task**: 系统管理与公司经营分离及种子验收
**Branch**: `main`

### Summary

完成系统工作台、公司权限和独立配置、公共地点与费用模板迁移。用户批准旧总部费用转换；隔离数据库实际bootstrap及seed两遍成功，正式迁移与权限回归通过，check:fast与生产构建通过。实际开发库未应用迁移或seed，保留已有数据。

### Git Commits

| Hash | Message |
|------|---------|
| `d8370c12` | feat: 分离系统管理工作台与公司经营数据 |
| `792c336f` | fix: 完成系统费用模板迁移与开发种子适配验证 |

### Status

[OK] **Completed**


## Session 97: 订单提成归属真相源切换为订单人员分工
<!-- trellis-session: v=2 fp=a320a1445a70a1bd -->

**Date**: 2026-09-22
**Task**: 订单提成归属真相源切换为订单人员分工
**Branch**: `main`

### Summary

开单被客户档案缺配拦截的问题定位与根治：提成归属（销售/操作/客服三岗）唯一真相源从客户档案责任人切换为订单人员分工。后端创建校验三岗必配（新增 ORDER_COMMISSION_PERSONNEL_MISSING，删除 PARTNER_COMMISSION_ASSIGNMENT_MISSING），快照改从 order_personnels 取数（source_assignment_id 指向订单人员行），草稿换客户仅同步归属行 customer_id；前端海运出口新建表单三岗必填（详情模式不变），客户档案与订单页快建三岗降为选填默认值；sync:dev 种子客户幂等补齐三岗；同步修订提成归属契约/术语表并重新生成错误码目录。check:fast 全绿，trellis-check 有条件通过后两项问题（目录生成物手改、二进制残留）已闭环。

### Git Commits

| Hash | Message |
|------|---------|
| `f59fae24` | feat: 提成归属唯一真相源切换为订单人员分工 |
| `6643334b` | feat: 订单表单三岗必填并放宽客户档案责任人 |
| `b6709633` | feat: sync:dev 种子客户幂等补齐三岗责任人 |
| `1305610b` | fix: 重新生成领域错误码目录与代码对齐 |

### Status

[OK] **Completed**


## Session 98: 清理数据库配置与测试库回退
<!-- trellis-session: v=2 fp=62f3fcd12f124cd7 -->

**Date**: 2026-09-22
**Task**: 清理数据库配置与测试库回退
**Branch**: `main`

### Summary

移除重复的 KRATOS_DATABASE_SOURCE 示例配置与三处订单锁集成测试的硬编码数据库回退；确认未配置专用数据库时六个测试入口明确 SKIP，Go vet 与差异检查通过，真实 PostgreSQL 路径未执行。

### Git Commits

| Hash | Message |
|------|---------|
| `b16070cf` | fix(server): 移除集成测试数据库隐式回退 |

### Status

[OK] **Completed**

## Session 99: 订单费用列设置与关联账单追踪
<!-- trellis-session: v=2 -->

**Date**: 2026-09-22
**Task**: 09-22-order-fee-columns-tracking
**Branch**: `main`

### Summary

完成订单费用录入与财务追踪增强最小闭环：应收/应付共用列设置（搜索/显隐/排序/恢复默认，用户+组织 localStorage 隔离，编辑中禁用），新增税率/税金/不含税总额/折本币金额/费用标签只读快照列；抽取 features/finance/fee-progress 共享进度文案，具备财务读取权限时按订单维度一次查询关联账单，账单号与整账单财务进度列区分未建账/已作废/加载中/失败并提供重试与财务详情入口，写操作后统一失效。工作区存在并行会话未提交改动，采用「HEAD 基底 + 仅本任务补丁」暂存并经隔离 worktree 验证两个提交独立可编译可测试；费用代码列因并行行内预览特性改为默认可见（仍可隐藏）。check:web 全绿（1022 用例）。

### Git Commits

| Hash | Message |
|------|---------|
| `bf32bbac` | feat(web): 订单费用表新增列设置与税额本币标签只读列 |
| `7905f57a` | feat(web): 订单费用页接入关联账单追踪与共享财务进度文案 |

### Status

[OK] **Completed**

## Session 100: 订单费用按账单占用关系物理删除
<!-- trellis-session: v=2 -->

**Date**: 2026-09-22
**Task**: 09-22-order-fee-delete-by-bill-occupancy
**Branch**: `main`

### Summary

承接用户对归档任务的补充澄清：RemoveFee 由软作废升级为物理删除，事务内锁定订单与费用后以「存在未取消账单（含草稿）的活动账单行」为唯一占用判定，冲突码 ORDER_FEE_BILL_OCCUPIED 提示先取消账单；取消账单后恢复可删。finance_bill_lines.order_fee_id 可空化 + SET NULL 保历史快照，费用标签外键改 CASCADE，正式迁移 20260922130000 隔离 Schema 冷启动验证；前端录入页删除确认不再采集原因并过滤历史作废行（数据/最近结果/父级集合/笔数金额同源）。吸收并行会话的 reason 选填契约作为删除无原因的前置必要变更；补丁级提交避免纳入其余未提交改动。新增 7 场景 PostgreSQL 集成测试（含删除与建账并发互斥）；全量单测、check:server、check:web 通过；RoleWorkspaceAnchorBackfill 与 FeeSupplementListAuthorization 两个集成失败经 HEAD 干净复测确认为既有问题。

### Git Commits

| Hash | Message |
|------|---------|
| `d3e87222` | feat(server): 订单费用按账单占用关系物理删除 |
| `677a9b7a` | feat(web): 订单费用录入页改为删除语义并隐藏历史作废行 |
| `0f6a3e5a` | docs(spec): 记录费用删除按账单占用判定与账单行快照独立契约 |

### Status

[OK] **Completed**


## Session 101: 订单费用不含税单价列交付
<!-- trellis-session: v=2 fp=12970c51980933d0 -->

**Date**: 2026-09-23
**Task**: 订单费用不含税单价列交付
**Branch**: `main`

### Summary

完成任务 09-23-order-fee-net-unit-price-and-lock-columns：订单费用表新增「不含税单价」默认可见可排序可选列（紧跟单价），含税行按税率反算两位小数 ROUND_HALF_UP，历史不含税行依 type-safety.md 零值省略规范以 !== true 判定直取单价，行内编辑实时折算预览，移除税率列未税单价 Tag；定向 39 用例、lint/tsc/biome/architecture 全过，任务已归档。发现并修复新代码 === false 死分支（protojson 省略零值，wire 缺失即 DB false），spec 已补记实例。

### Git Commits

| Hash | Message |
|------|---------|
| `d6fcf1af` | feat(web): 订单费用新增不含税单价列并移除税率列未税单价标签 |

### Status

[OK] **Completed**


## Session 102: 订单费用状态简化与批量维护交付
<!-- trellis-session: v=2 fp=6bdf85dfd2b43c0f -->

**Date**: 2026-09-23
**Task**: 订单费用状态简化与批量维护交付
**Branch**: `main`

### Summary

完成任务 09-23-order-fee-bulk-actions：阶段一取消费用确认环节，DRAFT/CONFIRMED 合并为 UNBILLED（proto reserved+UNBILLED=5、Ent 收紧、迁移 20260923090000 双路径幂等、建账/取消/补录/拆票/计提/自动锁单全链路同步，自动锁单改任意方向未建账阻断并新增 FEE_BILLED 触发）；阶段二新增整批事务的批量改结算单位/费用时间/删除与标签批量操作（迁移 20260923110000 修复发生日期分钟精度落库缺陷），前端五项批量入口与建账资格显式拒绝。四个实施子代理分阶段交付，trellis-check 全项 PASS，check:fast 全过，开发库已迁移（UNBILLED 44/CANCELLED 2）。spec 术语三处同步未建账口径。

### Git Commits

| Hash | Message |
|------|---------|
| `ce8e6e6b` | feat: 订单费用状态简化为未建账/已进账单，取消费用确认环节 |
| `2c215d5b` | feat: 订单费用批量维护：整批改结算单位/费用时间、批量删除与标签批量操作 |
| `25698810` | fix: 费用状态迁移 DROP CONSTRAINT 幂等化，同步领域术语为未建账口径 |

### Status

[OK] **Completed**


## Session 103: 计费单位数量规则实施与验收
<!-- trellis-session: v=2 fp=ac8273a4f1594913 -->

**Date**: 2026-09-23
**Task**: 计费单位数量规则实施与验收
**Branch**: `main`

### Summary

由实施与检查子代理完成计费单位整数/小数配置、全链路费用校验、正式迁移及开发库应用。独立 PostgreSQL 回归通过，后端完整门禁通过，前端全量 8 并发 1040 用例通过；记录高并发无关海运测试时序抖动。

### Git Commits

| Hash | Message |
|------|---------|
| `997f8a79` | feat: 按计费单位配置数量规则并统一费用校验 |

### Status

[OK] **Completed**


## Session 104: 汇率管理下沉分公司
<!-- trellis-session: v=2 fp=5765fe817c316eed -->

**Date**: 2026-09-23
**Task**: 汇率管理下沉分公司
**Branch**: `main`

### Summary

公司独立维护并按业务周解析汇率，移除公共基线与历史周继承；同步权限、手工输入、前端提示、生成物和规范。

### Git Commits

| Hash | Message |
|------|---------|
| `6bf0ba57` | feat: 汇率管理下沉分公司并按业务周取值 |

### Testing

- [OK] pnpm run check:fast 通过；Web 1041 通过、12 跳过；Go 测试、vet、Proto、分层与漏洞检查通过。
- [OK] 真实 PostgreSQL 集成测试未运行：未配置 RONCIN_INTEGRATION_DATABASE_SOURCE。

### Status

[OK] **Completed**

### Next Steps

- 费用科目模板种子任务已实现并提交，尚待单独归档。


## Session 105: 费用科目系统模板种子丰富交付
<!-- trellis-session: v=2 fp=30a3b9bcfb7b0eb1 -->

**Date**: 2026-09-23
**Task**: 费用科目系统模板种子丰富交付
**Branch**: `main`

### Summary

完成任务 09-23-fee-catalog-seed：新增迁移 20260923120000 将系统费用模板目录从 10 条丰富至 122 条（清洗用户参考清单，同码冲突拆码、乱码改语义码、同义科目并别名），配套补种费用类别 +5、异常情况 +5、计费单位 DAY，THC/STORAGE 模板类别改挂港口操作，并按同码跳过把模板追加到每个既有公司（应税劳务按公司+名称复用或创建）；search_keywords 由一次性生成器按 searchtext.Build 预计算，生成器未提交。迁移集成测试覆盖全新库计数、公司追加与同码跳过；bootstrap/建组织默认种子改为按 kind+code 幂等跳过。

### Git Commits

| Hash | Message |
|------|---------|
| `8c2d90ad` | feat(server): 费用科目系统模板种子丰富至 122 条并追加既有公司 |

### Status

[OK] **Completed**


## Session 106: 费用补录重提与审批毛利预览
<!-- trellis-session: v=2 fp=1bec573880396c9f -->

**Date**: 2026-09-23
**Task**: 费用补录重提与审批毛利预览
**Branch**: `fix/hide-system-org-chart`

### Summary

费用补录支持撤回后预填重提；审批改为审核视图，服务端只读预览本币毛利和毛利率变化，正式审批仍实时复核。定向与全栈门禁通过。

### Git Commits

| Hash | Message |
|------|---------|
| `d7d968f6` | feat(web): 支持费用补录撤回后预填重提 |
| `eed5e0a5` | feat: 补录审批前预览订单毛利变化 |

### Status

[OK] **Completed**
