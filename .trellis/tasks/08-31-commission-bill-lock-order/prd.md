# 修复提成账单多行加锁缺排序

## Goal

finance_commission.go 账单多行 ForUpdate 缺显式主键排序，存在多行锁顺序死锁窗口；补排序与真实 PostgreSQL 并发测试（Codex 复核遗留 P2，进入生产前修复）

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
