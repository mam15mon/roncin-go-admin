# 领域知识层（Domain）

面向 AI 会话与新成员的货代业务领域知识，目标是「不读业务代码也能建立正确图景」。

| 文档 | 内容 |
| --- | --- |
| [glossary.md](./glossary.md) | 海运/财务/往来单位/平台术语表，附代码定位（唯一真相源指引） |
| [architecture-map.md](./architecture-map.md) | 技术栈结构、业务域→代码地图、核心数据流与链路约束、实体关系图、新会话上手路径 |
| [error-catalog.md](./error-catalog.md) | biz 层业务错误码全集（HTTP 类别、中文消息、定义位置、关联 proto ErrorReason），由 `scripts/generate-error-catalog.mjs` 自动生成 |

维护约定：

- 新增业务概念或改口径时同步更新本层；术语与代码冲突时以代码为准并回改文档。
- 文档只写「导航级」关系与约束，字段级细节留给 schema 与 proto（真相源）。
