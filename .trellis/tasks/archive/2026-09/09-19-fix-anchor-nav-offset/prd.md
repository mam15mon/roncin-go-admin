# 修复表单锚点导航落点被吸顶栏遮挡

## Goal

锚点导航两条路径缺吸顶补偿：错误分节兜底 scrollIntoView block:start 落点被吸顶按钮栏遮挡；展开折叠节与平滑滚动存在布局竞态。统一为实测吸顶高度+布局稳定后滚动

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
