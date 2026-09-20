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
