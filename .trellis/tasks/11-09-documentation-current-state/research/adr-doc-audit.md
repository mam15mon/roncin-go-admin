# ADR、文档导航与生成文档审计

- 日期：2026-11-09。
- 范围：`.trellis/spec/adr/` 的索引及 16 份决策、领域索引和自动生成错误目录、全仓字面量 Markdown 本地链接。
- 本文件只记录研究；正式文档尚未修改。

## 已确认差异

| 文档位置 | 当前源码依据 | 建议修正 |
| --- | --- | --- |
| `adr/0001-contract-single-direction-flow.md:18-20` 将 server/openapi.yaml 更新归给前端命令 | `server/Makefile:16-19` 的 api 目标调用 Buf；`server/buf.gen.yaml:14-21` 输出 OpenAPI；`web/scripts/generate-openapi-input.mjs:5-11` 只读取 YAML 写入 JSON；根 `package.json:15` 包含前端生成与 proto 常量生成 | 明确 make api 同时生成 Go 与服务端 OpenAPI，前端命令消费 YAML 生成输入、客户端及 proto 常量；生成目录按脚本实查 |
| `adr/0009-server-state-react-query.md:22-25` 未限定持久草稿适用范围 | `web/src/pages/orders/detail.tsx:1312-1328` 不传 draftPathname；`web/src/components/ui/order-template/OrderFormTemplate.tsx:168-173` 完整身份才持久化；`state-management.md:224-251` 已给出新契约 | 详情只保留内存编辑和未保存离开保护；新建和独立 HBL 编辑仍持久化，引用现行规范 |
| `adr/0010-order-kind-registry.md:9` 使用当前只交付海运出口口径 | `web/src/pages/orders/order-kinds/registry.ts:9-12` 注册 sea-export 与 air-export；`web/config/routes.ts:140-151` 存在空运入口 | 保留当时决策背景，并补注当前 SE/AE 开放状态，其他品类不作已交付描述 |
| `adr/0013-order-lock-and-immutable-document-history.md:19-24` 把未结案作为内容门禁 | `server/internal/data/order_lock.go:39-68` 明确 ACTIVE 且未业务锁定，完结本身不冻结内容 | 现行门禁删除未结案条件，说明人工审核锁与生命周期操作各自职责；费用仍按逐行能力判断 |

## 检查通过

- 读取其余 ADR：0002、0003、0004、0005、0006、0007、0008、0011、0012、0014、0015、0016；保留历史背景，不因日期较旧机械改写。具体实现入口仍应在实施和复核时按变更点再次确认。
- `node scripts/generate-error-catalog.mjs --check` 通过：403 个唯一错误码，未手工修改生成目录。生成目录中仍定义但未调用的旧错误不等同于现行业务行为，不据此恢复退役功能。
- 字面量 Markdown 链接初查覆盖 303 个目标：现行项目规范 233 个均存在；业务与开发文档 31 个中 5 个目标失效，均为前端 README 模板引用不存在的 cheatsheet。
- 历史任务另有 1 个失效目标，仅记录，不重写历史任务。框架模板存在占位路径时不按业务导航失败处理。
- Claude/ZCode/DSH 下 100 份 Markdown 共有 56 种内容，其中 44 组是逐字相同的多平台副本；不批量翻译或升级这些框架文档。

## 范围限制

- 本地链接检查不访问网络，不验证页面锚点、Markdown 引用式链接、内联代码路径或真实业务端到端行为。
- `registry.ts:37` 还存在 AE 未注册的旧源码注释。只记录为范围外，不能扩展为代码改动；文档当前状态以注册表实际值为准。
- 被忽略的 `.agents/` 属本地技能文档，另作盘点；指向仓库外部的全局技能软链接不作为本仓库可提交文件。
