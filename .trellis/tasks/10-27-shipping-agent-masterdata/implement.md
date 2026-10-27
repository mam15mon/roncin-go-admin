# 船代公共主数据执行计划

## 阶段与前提
当前 planning。研究完成，等待用户对最终方案的下一条明确确认后执行 task.py start，不提前修改产品代码。保留 eyun.txt，不读出其内容、不纳入 Git；与其他会话共享工作区，操作前核对 git status。

## 顺序与责任
- [x] 建立任务、保存无凭据来源快照和研究报告，核查旧引用数量与组织规范。
- [x] 收敛 PRD、技术设计、实现计划及 implement/check 上下文。
- [ ] 用户确认最终规划摘要后启动任务；创建本任务 codex/ 分支（不重置工作区）。
- [ ] 派发 trellis-implement 负责后端契约、领域枚举、仓储全量接口、种子/迁移、订单引用/名称投影和后端定向测试；前端 worker 等 API 及客户端生成后再接入，避免共享生成物冲突。
- [ ] 后端按源文件依次生成 Ent、API/OpenAPI、前端客户端/枚举；禁止手改生成物。权限 Manifest 未变无需权限键重生成。将 29 条船代写入 server/seeds/ 来源快照及正式 SQL 种子，验证初始化幂等且不覆盖管理员修改。
- [ ] 派发 trellis-implement 负责前端船代维护页、页签真相源、订单查询 helper 与字段远程过滤、相关前端测试；保留现有详情seed和相邻代理查询。
- [ ] 运行相关最小单测、修改文件 Biome、类型检查、diff --check，审阅契约/迁移/种子29条与源快照一致。
- [ ] 专用隔离 PostgreSQL 执行正式迁移/种子/检索/订单/旧引用拒绝测试，检查 PASS/FAIL/SKIP。不得在开发库执行测试清理或重置。
- [ ] 实施完成后派发 trellis-check 按 check.jsonl 独立 review，检查并修正契约、权限、生成物、seed、迁移、前后端调用与测试缺口。
- [ ] 全任务验收执行 pnpm run check:fast；失败修复后仅按失败与新增风险重跑必要检查。
- [ ] 更新公共主数据 code-spec、能力导航（仅新增公开能力时）并核查中文文档。
- [ ] 按可验证组 Conventional Commit 提交相关代码及生成物，不包含凭据或临时文件。
- [ ] 确认目标完成后按 finish-work 归档本任务并记录会话日志；不归档其他会话任务。
- [ ] 审核本次分支的提交与 main 差异，合并到本地 main；确认合并后的提交可达、工作区无未提交任务改动后，安全删除已合并的本地 codex 分支。保留 eyun.txt，不删除 main，不强制删除未合并分支；未获推送指令不推送远端。

## 实现位置
精确锚点见 research/backend.md、research/frontend.md。核心必要变更：
- server/api/masterdata/v1/masterdata.proto 和必要 order.proto 语义注释；对应合法生成物。
- server/internal/biz/masterdata.go、service/masterdata.go、data/masterdata.go、Ent master_data_item 源与生成物。
- 新有序迁移、server/seeds/船代数据与来源说明；order_sync.go / order_query.go 及其针对性测试。
- web/src/pages/master-data/components/ShippingAgentsPanel.tsx、master-data/index.tsx、router/workspaceTabs.ts。
- orders/common.ts、new.tsx、detail.tsx、SeaBasicInfoSection.tsx 与必要测试。

## 验证命令
```bash
go -C server generate ./internal/data/ent
make -C server api
pnpm run generate:web-client
go -C server test ./internal/biz ./internal/service -run 'TestMasterData|TestOrder' -count=1
go -C server test ./internal/data -run 'TestValidateOrder|TestOrderCreateRollsBack|TestShippingAgent|TestMigrationManifestCoversEntSchema' -count=1 -timeout 25m
pnpm --dir web exec vitest run src/pages/master-data/index.test.tsx src/pages/master-data/components/ShippingAgentsPanel.test.tsx src/router/organizationSwitchDestination.test.ts
pnpm --dir web exec vitest run src/pages/orders/common.test.ts src/pages/orders/new.test.tsx src/pages/orders/templates/components/sea/SeaBasicInfoSection.partner-mapping.test.tsx src/pages/orders/order-kinds/sea-export/form-adapter.test.ts
pnpm --dir web test:changed
pnpm --dir web tsc
pnpm run check:architecture
git diff --check
pnpm run check:fast
```
新增测试名称以实际实现为准。Biome 只针对改动源码，集成测试显式注入专用连接，不在命令输出或记录中泄露连接串。最终不因为普通UI修改要求build。

## 迁移审阅与回滚点
- 审阅 kind CHECK 与 Ent 注解一致、合法集合不含退役值、SQL拼音键与Build一致。
- 验证非目标订单/组织/经营资料无变化，种子重复执行不改管理员配置；原单编辑与拆票继承可保留停用旧引用，普通复制或主动更换必须选启用项。
- 如旧引用或名称冲突拒绝，保持数据库原状并报告，不能删除旧记录或猜配。
- 本期仅追加字典与改变引用契约，无运行时旧版兼容；部署必须同步前后端与迁移，不单独回滚任一层制造语义不一致。
