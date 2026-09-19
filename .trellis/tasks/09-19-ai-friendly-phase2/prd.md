# AI 友好架构二期

## Goal

台账模板 TFilter 泛型化、ADR 决策记录目录、biz 错误码目录三项收尾

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.


## 收尾记录（2026-09-19）

- **TFilter 泛型化**：模板 any 豁免 41 → 7（消除 34 处）；TFilter 泛型 +
  FinanceLedgerRequestParams 交叉类型根治索引读取报错；22 个消费文件类型化。
  保留 7 处均有实证理由；**新发现存量契约缺口**：港口/机场表单字段
  required:false 与 OpenAPI CreatePort/AirportRequest 必填标记不一致
  （useMasterDataCrud 泛型化受阻点，建议另立任务修正契约或表单）。
- **ADR 目录**：15 条决策记录（契约/并发/分页/前端/部署/业务/工作流），
  git 断代 + 代码实位核验。
- **错误码目录**：369 码七域分布 + 生成脚本（--check 防漂移，暂未接入
  check 门禁）。
