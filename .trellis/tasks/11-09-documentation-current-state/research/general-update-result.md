# 通用文档更新结果

- 日期：2026-11-09。
- 责任范围：根 README.md、PLAN.md 与 docs/**，共更新 14 份已有文档，新增 docs/README.md。
- 未修改产品代码、数据库、契约、依赖或其他代理负责文件；未执行 Git 暂存、提交或分支操作。

## 已完成

| 审计项 | 处理 |
| --- | --- |
| G01/G02 | README 同列 SE/AE 已开放；移除标签现行能力，链接退役契约；SE 验收明确 AE 另行验收 |
| G03/G04 | Go 下限按 go.mod 同步 1.26.10，本站会话改为随机令牌摘要的数据库会话 |
| G05/G06 | 本地说明纠正 bootstrap 幂等补齐、专用集成库隔离 Schema 按正式 SQL 初始化与清理；SKIP 不计通过 |
| 新发现 | Air 监听 SQL 但只重编译启动 Go，不会每次重载迁移；新增迁移须 migrate:dev 或重启 pnpm dev |
| G07/G08 | 机场港口文档按全局公共目录重写；去组织目标，默认归档港口与匹配中文文件，区别机场保留中文和港口写入对照；保留原 HQ 写入数量为历史记录 |
| G09 | 跨仓储说明要求统一 Data.client(ctx)，补 withSQLTx 禁入共享事务、提交后重读；批量路径已由 FinanceBillUsecase.CreateBatch 共享事务实现 |
| G10/G11 | SE 清单与批量建账说明改为统一人工锁、无需先审核、逐源费待审阻断、活动账单行投影；NORMAL/NETTING 和 grouping_policy/group_configs 同步 |
| G12 | PLAN 和标签/企业资源/竞品笔记明确历史状态及现行入口；迁移矩阵当前摘要更新角色范围退役、标签退役、SE/AE、通知 Worker、幂等和公共目录 |
| G13 | 企业图片删除在原决策附近改为事务删资源并登记任务、提交后 Worker 删除对象；不只加页头免责声明 |
| G14 | 提成计划说明 CNY/归属日期/导出已有代码，并保持原设计基线；不将旧计划当清空重建授权 |
| G15 | 客户计划区分原问题与后续实现，保留未落实的静态省市区缺口，不把文档更新当代码修复 |
| G16 | 迁移矩阵改为金额 numeric(28,8)、汇率 numeric(18,8)、费用数量/单价 numeric(18,4)、税率 numeric(5,2)，展示层按格式规范 |
| 配置边界 | OSS 说明将历史 Bucket/IP 改为目标环境核对说明，不声称真实云端已验证 |
| 统一导航 | docs/README.md 提供开发、现行专项规范、历史计划和维护边界，不复制第二套业务规则 |
| 参数转发 | 按主会话无副作用 pnpm 10 实测，根 README 与港口换包命令移除多余 --，同时要求匹配中文对照 |

## 追加源码复核

- `server/.air.toml` build.cmd 只有 go build，scripts/dev.mjs 仅启动阶段 migrate:dev。
- `server/internal/data/industry_reference_sync.go`：全局查询；机场更新不写中文字段，港口新增/更新 SetNameZh。
- `server/cmd/sync-unlocode/chinese_names.go`：完整代码集合、英文身份、中文名及摘要校验。
- `server/internal/biz/finance_bill_batch.go`：NORMAL 单方向、NETTING 配对、费用用途和更正身份分组；CreateBatch 建立共享事务。
- `server/internal/data/ent/schema/{order_fee,finance_bill,finance_bill_batch}.go`：当前精度、批次字段与 NORMAL/NETTING。
- `web/src/pages/partners/partner-detail.tsx`：独立查询与内嵌 settlementRule 单次创建/更新。
- `web/src/pages/partners/components/BasicInfoSection.tsx:32,252` 仍引用 `pcaCascaderOptions`，`web/src/utils/chinaDivision.ts` 从 china-division 静态 JSON 生成选项；这是剩余实现缺口，未通过修改文案掩盖。

## 本范围验证

- `git diff --check`：通过。
- 15 份根/docs 文档中围栏代码块外字面量 Markdown 本地链接目标：85 个，失效 0；没有验证远程 URL 和真实环境状态。锚点及引用式链接由主会话统一复核。
- 提取上述文档中的 `pnpm run` 命令，对照根/web package.json：22 种命令均存在。
- 逐项核对同步 CLI flags 与示例；没有执行会下载、写数据库或访问外部系统的命令。
- 无 Vitest、Go 业务测试、全量 lint、check:fast、构建、迁移或业务验收；本组纯文档，按批准的风险匹配验证范围执行。

## 限制

本轮修复有代码依据的文档现状差异，不保证早期计划的每一字段/竞品观察已变成现行功能。
历史验收记录保留原事实，新的验收表仍标未执行。正式域名、DNS/HTTPS、OSS 和钉钉状态不能从源码推定。
