# AI 友好架构核心建设

## Goal

领域地图与术语表 + any 回潮防退门禁；让新 AI 会话以最低成本获得货代领域知识并锁住已有治理成果

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.


## 收尾记录（2026-09-19）

- **领域地图与术语表**：`.trellis/spec/domain/` 三件套（glossary 68 术语+
  通用机制、architecture-map 含数据流/实体关系 mermaid/上手路径），
  AGENTS.md 顶部已挂入口；事实引用逐条经 schema 核对。
- **any 防回退门禁**：noExplicitAny=error + 测试 override；ui 模板层
  39 处逐点豁免（统一理由：模板泛型人体工学，泛型化改造另立后续任务），
  parameter-setting 4 处直接收紧；探针验证新增 any 必被拦截。
- 后续候选（本轮未做）：模板 TFilter 泛型化、ADR 目录、错误码表、
  分层边界 lint。
