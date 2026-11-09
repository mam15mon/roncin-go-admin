# Markdown 本地链接存在性初查

- 日期：2026-11-09。
- 仅校验围栏代码块外的字面量 Markdown 内联链接目标是否存在；不验证远程 URL、锚点、模板占位符和代码中的示例。
- 此报告包含模板和历史文档，失效示例不直接视为现行项目缺陷。

| 类别 | 已检查链接 | 失效目标 |
| --- | ---: | ---: |
| 工具框架文档 | 8 | 0 |
| 项目规范 | 233 | 0 |
| 历史任务 | 31 | 1 |
| 业务与开发文档 | 31 | 5 |

## 失效目标

| 类别 | 文档位置 | 链接目标 |
| --- | --- | --- |
| 历史任务 | `.trellis/tasks/archive/2026-08/08-31-commission-enhancement/prd.md:5` | `../../../docs/plans/提成管理增强计划.md` |
| 业务与开发文档 | `web/README.md:22` | `./docs/cheatsheet.en-US.md` |
| 业务与开发文档 | `web/README.md:24` | `./docs/cheatsheet.en-US.md#faq` |
| 业务与开发文档 | `web/README.zh-CN.md:21` | `./docs/cheatsheet.zh-CN.md` |
| 业务与开发文档 | `web/README.zh-CN.md:23` | `./docs/cheatsheet.zh-CN.md#faq` |
| 业务与开发文档 | `web/README.zh-CN.md:124` | `./docs/cheatsheet.zh-CN.md` |
