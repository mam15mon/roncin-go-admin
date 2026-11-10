# 开发思考指南

本目录提供 Roncin 项目的开发前检查表，帮助定位真实规则和复用入口。具体业务契约由对应规范、Proto 与实现维护，协作要求遵循[根 AGENTS.md](../../../AGENTS.md)。

| 指南 | 使用场景 | 后续入口 |
| --- | --- | --- |
| [代码复用检查](./code-reuse-thinking-guide.md) | 新增函数、常量、配置、重复映射或跨页面能力 | [前端能力导航](../web/frontend/capability-navigation.md)、[后端能力导航](../server/backend/capability-navigation.md) |
| [跨层数据流检查](./cross-layer-thinking-guide.md) | 接口、持久化、权限、异步状态或多层调用变化 | [后端规范](../server/backend/index.md)、[前端规范](../web/frontend/index.md) |

## 修改前

- 明确本次行为差异、所属层、影响文件和验证方式，先检查 Git 状态并保留已有工作。
- 用 `rg` 搜索将要修改的符号、字段、文案或配置，沿真实调用链确认所有消费方。
- 多页面使用或多个入口处理同一数据时，先查领域能力；不要按文件名相似就新建抽象。
- 核对代码评审发现的来源和触发条件：当前实现、设计要求、历史记录及外部假设分别说明。

```bash
rg --files web/src/features server/internal/biz
rg -n '待核对的符号或字段' web/src server/internal
```

## 文档与验证

实现功能后按风险补验证，禁止 TDD。业务缺陷至少覆盖触发场景和相邻路径；未执行的真实环境验收明确写未执行。

文档陈旧时先追查源码；规范要求尚未实现时记录缺口，不通过降低要求掩盖问题。文档核查应逐份记录范围、依据、深度和未确认项，不把全量路径盘点称为全量内容验证。

本仓库是 Trellis 的使用方。维护本地规范时修改实际 `.trellis/spec/`，工具自定义按 `trellis-meta` 指引核对当前本地脚本和配置，不执行 Trellis 上游源码目录的同步命令。
