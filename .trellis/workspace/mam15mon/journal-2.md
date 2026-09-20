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
