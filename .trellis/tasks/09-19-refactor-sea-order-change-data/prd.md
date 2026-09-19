# 拆分 sea_order_change.go 数据层大文件

## Goal

5185 行按变更类型拆为 actions/split/transport-update/reassignment/events 同包文件，纯移动零行为变更

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
