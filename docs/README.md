# 项目文档导航

> 现状核对：2026-11-09。现行说明按仓库代码、契约和配置维护；历史计划与验收记录保留原时间语义。
> 文档中的实现状态不代替真实环境验收，也不代表正式域名、云存储或钉钉已配置生效。

## 上手与开发

| 文档 | 用途 |
| --- | --- |
| [项目说明](../README.md) | 当前能力、技术栈、初始化、开发和同域部署入口 |
| [协作约束](../AGENTS.md) | 分层、生成、依赖、权限、事务、验证及按组提交要求 |
| [本地开发](local-development.md) | PostgreSQL、环境变量、幂等初始化、热重载与隔离集成库 |
| [服务端说明](../server/README.md) | Go 开发、生成与服务入口 |
| [前端说明](../web/README.md) | Vite/React Router 开发、目录与最小验证 |
| [主数据同步](airport-port-sync.md) | 全局机场/海港目录、源文件和中文对照、预览与写入区别 |
| [归档数据说明](../server/seeds/README.md) | 船代、船公司、航司、海港、机场及行政区划来源 |
| [订单附件 OSS](order-attachment-oss.md) | 专用私有 Bucket、签名上传、访问与共享引用删除 |
| [跨仓储事务](cross-repository-transactions.md) | 共享事务、提交后响应、类型化外部副作用任务 |
| [正式迁移](../server/migrations/README.md) | SQL 迁移执行、校验和与开发/生产差异 |

## 现行业务与工程规范

按[领域术语](../.trellis/spec/domain/glossary.md)和[架构地图](../.trellis/spec/domain/architecture-map.md)
定位模块，再阅读[服务端规范](../.trellis/spec/server/backend/index.md)、
[前端规范](../.trellis/spec/web/frontend/index.md)及[架构决策](../.trellis/spec/adr/index.md)。
字段细节以 Proto/Ent 为准，生成文档按生成器更新，不手改生成物。

| 近期演进 | 现行入口 |
| --- | --- |
| SE 海运与 AE 空运均已开放 | [海运契约](../.trellis/spec/server/backend/sea-export-document-contract.md)、[空运契约](../.trellis/spec/server/backend/air-export-contract.md) |
| 订单资料与费用一次人工审核，建账无需先审核 | [审核与建账](../.trellis/spec/server/backend/order-fee-review.md)、[费用补审](../.trellis/spec/server/backend/order-fee-correction.md) |
| 财务列表、候选和写入绑定当前公司 | [当前工作区](../.trellis/spec/web/frontend/finance-current-workspace.md)、[列表快照](../.trellis/spec/web/frontend/finance-ledger-snapshots.md) |
| 角色范围配置退役，成员及工作台决定访问边界 | [角色范围退役](../.trellis/spec/server/backend/retire-role-data-scope.md)、[工作台切换](../.trellis/spec/server/backend/auth-session-org-switch.md) |
| 订单主详情无持久草稿，新建/HBL 保留草稿 | [状态与离开保护](../.trellis/spec/web/frontend/state-management.md) |
| 已付提成保留实际支付事实，费用变动经济重算 | [提成经济重算](../.trellis/spec/server/backend/finance-commission-lock.md) |
| 自由业务标签、独立里程碑和独立人员维护入口退役 | [标签](../.trellis/spec/server/backend/business-tag-retirement.md)、[里程碑](../.trellis/spec/server/backend/order-milestone-retirement.md)、[人员入口](../.trellis/spec/server/backend/order-personnel-editor-retirement.md) |

## 验收与历史材料

[海运出口闭环验收](plans/海运出口订单闭环验收.md)维护当前待验证标准及既有脚本限制。
脚本存在、测试跳过和旧任务通过均不能代替本版本全链路验收；运行写数据的命令前明确目标环境和样例保留方式。

| 历史材料 | 阅读边界 |
| --- | --- |
| [重构阶段计划](../PLAN.md) | 原阶段与交付记录；页头列当前差异，不作为当前完整待办 |
| [迁移矩阵](migration-matrix.md) | 旧模型范围、当前摘要与取舍；不是旧数据自动迁移工具 |
| [客户档案计划](plans/客户页面.md) | 早期设计，旧问题清单不能直接当现存缺陷 |
| [企业资源计划](plans/企业资源备忘录建设计划.md) | 原设计/验收；标签已退役，图片删除改为事务任务 |
| [业务标签计划](plans/业务标签建设计划.md) | 已退役能力的历史记录，不指导恢复 |
| [批量建账说明](plans/费用明细批量转账单.md) | 当前准入与分组摘要，具体字段以现行契约为准 |
| [提成增强计划](plans/提成管理增强计划.md) | CNY/日期/导出设计基线，后续演进见专项规范 |
| [易云调研](eyun-config-management.md) | 2026-08-29 的竞品观察，不声称第三方当前状态 |

`.trellis/tasks/archive/` 与 `.trellis/workspace/` 分别保留历史任务及会话日志，不追改当时事实。
项目使用的 Trellis 框架文档、第三方许可及社区模板按各自来源维护，不随业务状态机械重写。

## 文档维护

[Markdown 逐份核查记录](markdown-audit.md)列出本轮基线、每份结论、核查深度和待确认边界，避免将路径盘点视为全部业务验证。

功能、命令、路由、权限或生成流程变化时，在同一组变更中同步对应现行说明、规范和导航。
已有文档声称与实现不同，先追查契约与代码；规范要求尚未落实时明确记录实现缺口，不通过降低规范掩盖问题。
无法从仓库确认的部署和外部服务状态明确标注，不补造日期、库存或通过结果。历史决策被替代时在相关位置链接现行约束。
