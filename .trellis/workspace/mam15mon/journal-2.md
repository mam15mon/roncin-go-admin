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
