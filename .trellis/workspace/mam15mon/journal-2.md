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
