# 前端 any 类型逃逸治理

## Goal

清理 web/src 中 328 处 as any/: any 类型逃逸：模板基座泛型收紧 + 业务页面表单值类型化，行为零变更

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.


## 收尾记录（2026-09-19）

- 全仓（非测试/非生成物）`as any` / `: any`：**328 → 3**。
- components/ui：42 → 3，保留 3 处均为 antd 类型缺口（Select option value
  不含 boolean ×2、strictFunctionTypes 逆变边界 ×1），代码内有注释。
- pages 全域：101 → 0（settings/finance/partners/orders/admin/master-data
  表单值类型化 + unknown 收窄 + 类型守卫）。
- 实际 pages 总量 101 处而非预估 280（预估口径混入了并行范围与历史状态）；
  尾部域子任务与 pages 全量任务范围重叠，已裁撤归档（无代码产出）。
- 最终验证：合并态全量 vitest 852 通过/12 跳过、tsc 零错误、
  biome 与基线一致；行为零变更（全部改动限于类型层）。
