# ADR 0015: Trellis 工作流与 spec 知识沉淀机制

- 状态：已采纳
- 日期：2026-08-31（接入 Trellis 并沉淀前后端开发规范）
- 主题：工作流

## 背景

本项目的主要开发者是 AI 会话（ZCode + Trellis），上下文随会话压缩消失。
规范若靠「模型记忆」必然漂移：上个会话学到的教训，下个会话不知道。研发
过程本身也需要文件化——研究结论、决策理由、执行计划若只留在对话里，
无法审阅也无法归档。

## 决策

- 采用 Trellis 工作流（[../../../.trellis/workflow.md](../../../.trellis/workflow.md)），
  核心原则：**规范靠注入不靠回忆**（specs injected, not remembered）、
  **一切持久化到文件**（conversations get compacted, files don't）。
- 知识分四层存放：
  - `AGENTS.md`：协作规范唯一真相源；
  - `.trellis/spec/domain|server|web/guides/`：面向 AI 任务执行的浓缩规范，
    冲突时以 `AGENTS.md` 为准；
  - `.trellis/tasks/<task>/`：任务级 prd / design / implement 与过程记录；
  - `.trellis/spec/adr/`（本目录）：架构决策的「为什么」。
- 每完成一个任务，把可复用的教训回写 spec（trellis-update-spec /
  break-loop 机制），而不是留在会话里。
- 文档与面向开发者的注释统一中文；spec 只写「导航级」契约，字段级真相
  留给 schema 与 proto。
- 交付节奏遵循 AGENTS.md：每组可验证修改即提交，Conventional Commits；
  禁止 TDD，功能实现后按风险补针对性验证。

## 理由

- 对 AI 协作而言，「文件是唯一跨会话记忆」；注入式规范把遵循成本降到
  最低（会话不需要自己去发现规则）。
- 分层避免单文件膨胀：AGENTS.md 管底线，spec 层管领域细则，ADR 管
  历史动因——三者引用同一批代码真相源。
- 回写机制使规范库随实际踩坑增长（如 antd 6 惯例、CHECK 同源事故），
  是「Capture learnings」原则的落地。

## 后果

- 新会话的上手路径固定：AGENTS.md → domain 层（architecture-map +
  glossary）→ 对应层 spec →（需要动因时）本 ADR 目录。
- 术语与代码冲突时以代码为准并回改文档；spec 与 AGENTS.md 冲突时以
  AGENTS.md 为准。
- 新增重要架构决策应补 ADR 文件并更新
  [./index.md](./index.md)，保持索引同步。
