# 架构决策记录（ADR）目录

> 记录「为什么这么设计」：每个文件固定四段——背景、决策、理由、后果。
> 查现行规则请看 [../../../AGENTS.md](../../../AGENTS.md) 与对应层
> spec；本目录只补历史动因与取舍依据。
> 新增决策时新建编号文件并更新本索引；日期不确定的决策用月份粒度，不
> 编造精确日期。

## 契约与生成物

| 编号 | 标题 | 状态 | 一句话结论 |
|------|------|------|-----------|
| [0001](./0001-contract-single-direction-flow.md) | 契约单向流：proto 是接口唯一真相源 | 已采纳 | 接口改动只从 `.proto` 出发，经两条生成命令单向流出，生成物禁止手改 |
| [0002](./0002-permission-manifest-single-source.md) | 权限清单单一真相源 manifest.go | 已采纳 | 权限码只在后端清单定义，前端键名由清单生成、编译期对齐，中间件默认拒绝 |
| [0003](./0003-ent-schema-and-sql-migrations.md) | Ent Schema 同源，正式 SQL 迁移是生产唯一真相 | 已采纳 | 声明真相在 Ent Schema、执行真相在有序 SQL 迁移；禁运行期建表与并发 Schema.Create |

## 事务与并发

| 编号 | 标题 | 状态 | 一句话结论 |
|------|------|------|-----------|
| [0004](./0004-unified-transaction-wrapper.md) | 统一事务封装 WithTx / WithinTransaction | 已采纳 | 事务只走统一封装与共享事务上下文，禁止手写分散事务模板 |
| [0005](./0005-pessimistic-plus-optimistic-locking.md) | 悲观锁 + 乐观锁双层并发防护 | 已采纳 | ForUpdate 锁行、version 比对、状态机检查、版本 +1 四步固定；按实体分层取舍 |

## 接口与主数据

| 编号 | 标题 | 状态 | 一句话结论 |
|------|------|------|-----------|
| [0006](./0006-pagination-limit-and-masterdata-strategy.md) | 分页统一 200 上限与主数据全量/分页二分 | 已采纳 | pageSize 上限全站 200；小数据集全量接口 + 前端分页，大数据集服务端分页 + keyword |
| [0007](./0007-pinyin-search-keys.md) | 选择器拼音检索键 | 已采纳 | 服务端 keyword 覆盖代码/中英文/别名/无声调全拼/首字母，检索键预计算成列 |

## 前端

| 编号 | 标题 | 状态 | 一句话结论 |
|------|------|------|-----------|
| [0008](./0008-frontend-template-system.md) | 前端模板复用体系与满屏流式布局 | 已采纳 | 公共模板统一由 `@/components/ui` 导出，满屏 Fluid 布局，禁局部 maxWidth 居中 |
| [0009](./0009-server-state-react-query.md) | 服务端状态归 React Query | 已采纳 | 接口数据不镜像进全局 store，直接消费服务端状态层 |
| [0010](./0010-order-kind-registry.md) | 订单类型注册表与三类真相边界 | 已采纳 | 类型元数据、操作能力、表单生命周期三类真相各有唯一所有者，未知 kind 显式 404 |
| [0011](./0011-antd6-testing-conventions.md) | antd 6 测试交互惯例 | 已采纳 | antd 6 基线下 DatePicker 用 click、submit 点按钮本体、modal.confirm 用 okButtonProps |
| [0016](./0016-frontend-module-boundaries.md) | 前端模块边界与 features 领域能力层 | 已采纳 | 跨页面共享能力经 features 公开入口消费，依赖越界由 check:architecture 自动失败，无豁免清单 |

## 部署

| 编号 | 标题 | 状态 | 一句话结论 |
|------|------|------|-----------|
| [0012](./0012-same-origin-deployment.md) | 生产同域部署 | 已采纳 | Go 服务同时提供 `/api/*`、`/health/*` 与 React 静态资源，开发代理/生产同域双路验证 |

## 业务设计

| 编号 | 标题 | 状态 | 一句话结论 |
|------|------|------|-----------|
| [0013](./0013-order-lock-and-immutable-document-history.md) | 订单业务锁与不可变单证历史 | 已采纳 | 锁定事实只追加、统一内容写门禁、解锁走审批分流、单证版本不可变 |
| [0014](./0014-monthly-commission-application.md) | 月度提成申请 | 已采纳 | 自然月汇总申请、财务整单批准/驳回、驳回显式重提、迟到提成顺延 |

## 工作流

| 编号 | 标题 | 状态 | 一句话结论 |
|------|------|------|-----------|
| [0015](./0015-trellis-workflow-and-spec-knowledge.md) | Trellis 工作流与 spec 知识沉淀 | 已采纳 | 规范靠注入不靠回忆，研究与决策持久化到文件，任务结束回写 spec |

## 维护约定

- 决策被替代时将状态改为「已废弃」并指向替代 ADR，不删除文件。
- 每个 ADR 引用现行代码/文档的相对路径（自本目录出发 `../../../` 指向
  仓库根，`../` 指向 spec 兄弟层）。
- 素材来源：`.trellis/spec/` 各层规范、`AGENTS.md`、git 历史（关键节点
  在各 ADR 的「日期」行标注了提交依据）。
