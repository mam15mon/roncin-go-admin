# 执行计划：费用状态移除

## 启动条件

- [ ] 用户确认本版 PRD/design 的实施范围后才执行 task.py start；本轮仅交付规划。
- [ ] 读取根目录及目标目录 AGENTS.md、JSONL 中规范；检查 git status，保护 `BillCandidateSelectionStep.tsx` 等已有修改，需改同文件时以现有内容为基线。
- [ ] 重新核对研究清单与代码，确认每条消费方的查询/写入位置，不把旧行号当当前事实。

## 1. 源文件与事务实现（R1—R4）

- [ ] 修改 Proto，删除费用状态及其请求/响应依赖、reserve 字段，定义 has_active_bill 和撤销响应 data。
- [ ] data 抽取关联谓词与批量投影；biz 改纯 Go 关联事实；所有费用响应路径完整加载，避免 N+1 和未加载误判。
- [ ] 建账/批量建账/取消移除状态写入，保留锁、版本与行数检查；按「检查→插入账单行→版本更新」实际顺序实现，不能在插入后用 NOT EXISTS 拦住本次写入。
- [ ] 普通删除保留硬删除与审计；单条/批量拒绝补录费用，整批原子性不变。
- [ ] 补录专用撤销完整保留资格、倒序、版本和冲减规则，改硬删除；调整领域结果、service 响应及已删除申请投影。
- [ ] 改造已建账编辑、信用/换币、结案、拆票/改配、单据变更、主单换版、自动锁定、工位、提成及来源指纹消费。

## 2. Schema、生成与数据（R5）

- [ ] 按 `server/migrations/README.md` 生成/编写正式迁移；删 Ent 字段/CHECK/边/索引，检查 User 反向边。
- [ ] 在移除状态前校验并处置 CANCELLED 行，留审计快照，保留历史账单与补录申请/调整；异常时整体失败。
- [ ] 执行 `go -C server generate ./...`、`make -C server api`、`pnpm run generate:web-client`；检查生成差异，禁止手工修补生成文件。
- [ ] 更新 sync-dev 三处费用种子及测试夹具，以真实账单关系表达建账；专用库重复运行验证幂等，不默认修改现有开发库。

## 3. 前端与脚本（R1、R4）

- [ ] 删除 feeConstants/statusMeta 中费用状态，全部费用入口移除状态列与筛选，保留财务进度/账单信息。
- [ ] 编辑/删除/建账按钮消费可靠关联事实及权限；本地新增行不再初始化 status。
- [ ] 补录申请显示「生成费用已删除」、撤销按钮和错误文案；成功后刷新申请/费用/相关汇总，不读取响应 fee。
- [ ] 删除汇总 CANCELLED 排除分支；更新提成费用明细、快照类型与相关验收脚本旧枚举断言。

## 4. 实现后针对性验证

- [ ] A1/A2：订单费用、费用面板、台账筛选/汇总、建账工作台、补录结果、提成明细定向 Vitest；修改文件 Biome；`pnpm --dir web tsc`。
- [ ] A3：PostgreSQL 并发用例验证建账与删除/编辑、取消与重新建账竞争，旧版本 409，无悬挂/重复有效关联，已建账编辑设置与换币相邻路径通过。
- [ ] A4：逐项正反验证结案、拆票、改配、单据变更、主单换版、自动锁定、工位统计与提成来源。
- [ ] A5/A6：单条/批量补录拒删、混合批量零写入、历史账单 SET NULL 与快照、专用撤销权限/倒序/版本/调整曾确认/重复请求/审计/列表回读。
- [ ] A7：仅以有序正式迁移初始化专用 PostgreSQL，覆盖空库与旧 Schema 升级；正常费用不变、CANCELLED 不复活、异常关联整体失败，补录申请及调整仍可查。
- [ ] A8：检查提成指纹变化不会重复计提；种子可重复；保存 EXPLAIN 对比结论。

定向命令按最终修改文件选择：

```bash
go -C server test -p 32 ./internal/biz ./internal/data ./internal/service -run '<受影响测试名>'
pnpm --dir web exec vitest run <相对web目录的测试文件>
pnpm --dir web exec biome check <相对web目录的修改文件>
pnpm --dir web tsc
```

真实集成验证使用已有 `RONCIN_INTEGRATION_DATABASE_SOURCE` 专用库配置，不打印凭据；缺配置造成 SKIP 必须明确记录，不能标记 A3/A5/A6/A7 已通过。涉及 Schema 初始化不得 t.Parallel，不用业务开发库作测试库。

## 5. 完整验收与提交

- [ ] 扫描活动源码、生成客户端、脚本和当前规范中的 OrderFeeStatus、orderfee.Status、FEE_CANCELLED、fee_status、费用 status 引用；历史迁移/历史归档/迁移升级夹具允许作为历史输入保留，不能用全仓零匹配破坏迁移链。
- [ ] 更新 `.trellis/spec/domain/glossary.md` 及受影响财务规范，记录账单事实口径、普通删除与补录撤销边界；不改写历史任务归档。
- [ ] `git diff --check`、相关测试、`pnpm run check:fast`，记录集成测试实际 PASS 与无法运行项。只有构建配置等发生相关变化时追加 build。
- [ ] 契约/生成物/所有调用方与迁移形成同一完整提交，使用中文 Conventional Commit，不提交已有无关修改。
- [ ] 应用现有开发库迁移前提供影响清单并取得明确删除授权；未获授权则交付已验证迁移文件并记录未应用，不清库、不自动修复异常数据。
- [ ] 按 A1—A8 报告结果；完成后按 Trellis finish-work 归档。未通过必要集成验证或仍需已纳入范围的操作时不虚报完成。
