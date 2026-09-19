# 拆分订单写入与锁数据层文件

## Goal

order_write.go(1590) 与 order_lock.go(1589) 同域配对拆分，保持悲观锁+乐观锁范本引用指向

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
