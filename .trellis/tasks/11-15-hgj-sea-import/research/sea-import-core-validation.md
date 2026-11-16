# SI 后端核心交付与验证

日期：2026-11-16。执行分支：`codex/hgj-business-alignment`。本报告仅覆盖本组服务端、Proto、Ent、正式迁移及生成客户端；前端交互、PDF实际呈现、浏览器联调和最终门禁由主会话统合。

## 已实现合同

- SI 必须使用 IMPORT，复用现有海运 MBL／航程／可选 HBL 实体；DIRECT 不伪造 HBL、单成员独占，HOUSE 可以同公司同方向显式共享。SE／SI 同号各自独立，候选类型显式传递，直接 ID 与共享成员身份在仓储复验。
- 进口服务采用独立 `OrderImportHandling` 和追加事件，五项固定 ARRIVAL／EXCHANGE／CUSTOMS／PICKUP／DELIVERY，可承接任意子集、并行完成。完成存在实际时间；撤回、取消或纠正已有完成事实，以及已完成换单后更换代理必须有原因。
- `Order.version` 是唯一提交基线，当前服务 JSON 只保存 `exchangeAgentId` 和 `steps`，不保存镜像版本；正式迁移回填只保存五项无承接服务。读取响应实时投影订单版本，事件快照保留当时前后值及版本、操作者和原因。
- 普通订单保存与独立服务保存互不清空。服务保存锁定 Order，验证公司、预期版本、终止／审核锁及 OPEN，保存事件与审计后原子推进摘要／订单版本。CLOSED 服务变更必须先重开。
- 进口摘要为 IMPORT_NOT_APPLICABLE／IMPORT_NOT_STARTED／IMPORT_IN_PROGRESS／IMPORT_COMPLETED，客户端不能通过出口流转命令直写。完结只检查适用服务及既有活动异常／待审费用门禁，进口不自动补齐出口“已放单”，服务完成不自动审核、放货、建账或核销。
- 换单代理必须是本公司有效且未列入黑名单的供应商，与订舱／国外代理独立；责任人验证本公司人员范围；附件 ID 同时验证本票归属及资产公司。
- SI 审核保存真实 MBL／航程及适用 HBL 不可变快照；终止结束活动 Link，恢复新建原 MBL／航程 Link，历史不复活。放货只引用本票真实海运单证。
- SI 外部签发 HBL 保留真实 `issuerPartnerId/Name`，读取及保存响应派生 `EXTERNAL_ARCHIVE`，不冒用公司自签抬头。普通保存继续生成完整 HBL 历史及印字快照。
- 进口费用经营日沿用权威 ETA；共享航程改期事务内同步订单日期并联动系统周汇率，缺汇率回滚订单、航程、费用及历史。

## API

沿用 `OrderService`，按实际订单类型复用现有 read／update 权限，无新增权限码：

- `GET /api/v1/orders/{order_id}/import-handling` → `GetImportHandlingResponse`。
- `PUT /api/v1/orders/{order_id}/import-handling` → `SaveImportHandlingResponse`。完整五项 steps、`expectedVersion`、可选 `exchangeAgentId`、必要时 `reason`。
- `GET /api/v1/orders/{order_id}/import-handling/events`，`page >= 1`、`1 <= pageSize <= 200`，复用领域分页校验。

每项支持 `kind/undertaken/occurredAt/responsibleUserId/referenceNo/notes/attachmentIds`；人员名称与事件操作者名称为 OUTPUT_ONLY 投影。未承接和未完成由布尔值／optional 时间明确区分，不能按 TypeScript 可选字段猜测完成。

## 正式迁移与生成

迁移：`server/migrations/20261116160000_sea_import_handling.sql`，Ent Schema 与生成元数据同步维护两个新表、事件操作者外键、进口 flow 值集及适用性 CHECK、SI 真实审核快照 CHECK。无法核验的旧 SI 空审核快照显式失败，事务不留下表、摘要更新或迁移 revision。

全部 PostgreSQL 测试使用 `.env.local` 的 `DATABASE_SOURCE` 仅在子进程环境映射为 `RONCIN_INTEGRATION_DATABASE_SOURCE`；既有 helper 创建随机隔离 Schema，顺序执行正式 SQL，结束只删除自己的 Schema。未对开发业务库执行迁移，未删除／修复其中孤立主单，也未使用 Ent Schema.Create。

已执行：

```bash
make -C server api
pnpm run generate:web-client
go -C server generate ./...
make -C server lint
```

结果均 PASS。同一完整生成流程再次执行，974 个绑定／Ent／Wire／权限路由／OpenAPI／客户端文件内容指纹零变化；另对 `web/config/openapi.generated.json`、`web/src/enums.generated.ts`、`web/src/errorReasons.generated.ts` 单独复跑生成确认三份指纹零变化。生成物未手工修改。

## 已通过验证

纯包及服务器入口：

```bash
go -C server test -p 32 ./internal/biz ./internal/service ./internal/data ./internal/server ./internal/data/ent/migrate
go -C server vet ./...
git diff --check
```

均 PASS。纯包命令未配置 PostgreSQL，其跳过的集成测试不计作数据库验证；实际数据库证据来自以下显式注入的命令。

真实 PostgreSQL 扩展相邻批次（97.950 秒，0 SKIP，PASS）：

```bash
node --env-file-if-exists=.env.local - <<'JS'
const {spawnSync} = require('node:child_process');
const pattern = 'TestSeaImport|TestOrderLock_PostgresFlows|TestSeaMasterBillCreatePreservesSharedContent|TestOrderLockGrantScopes|TestSettlementDoesNotReview|TestFinanceReport|TestSeaChangeMasterIdentity|TestFeeCorrectionCommission|TestCommission.*Correction';
const result = spawnSync('go', ['-C', 'server', 'test', '-p', '32', './internal/data', '-run', pattern, '-count=1', '-v'], {
  stdio: 'inherit',
  env: {...process.env, RONCIN_INTEGRATION_DATABASE_SOURCE: process.env.DATABASE_SOURCE},
});
process.exit(result.status ?? 1);
JS
```

该批实际通过 SI 服务／事件／关闭重开／锁及恢复、合法本票附件与外票拒绝、同版本双写一成功一409、审计触发器失败回滚、ETA 费用及缺汇率回滚、正式迁移保留／回滚，以及 SE 人工审核／授权／共享加票、费用更正提成、经营报表和结清不自动锁单。

旧测试中将 SI 当作“无海运单证的通用品类”的夹具已同步：费用场景生成合法 SI／IMPORT／真实 DIRECT 主单和 ETA；纯通用单证／提成更正测试使用真实非海运类型；SI 审核断言改为真实海运快照。没有通过放松产品约束保留非真实夹具。

最新源码的最终 PostgreSQL 补充批次（83.938 秒，45 个测试含子用例，0 SKIP，0 FAIL，PASS）使用同样的进程内环境映射，Go 参数为：

```bash
go -C server test -p 32 ./internal/data -run '^TestSeaImport|^TestSeaChangesReject|^TestSeaMasterContent|^TestSeaShared|^TestSeaHouseBill|^TestOrderMasterBillDownstream' -count=1 -json
```

覆盖新增当前 JSON 无镜像版本断言、真实跨公司主单目标拒绝、跨方向异常摘要读取、全部 SI 新测试，以及拆改完整成员身份、并发加票冲突、共享箱成员锁与成员变化、独立最终目的港、HBL 草稿版本／正式升级及下游事实门禁。测试 JSON 输出统计按 `Action=pass/skip/fail` 实际计算。

## 边界与后续

- 本组不开放 AI 创建，不扩大 SE 拆票／整票改配的业务范围；共享航程、箱、真实海运正文和历史按 SI 适用性复用。
- 全筛选 CSV、工作台／通知等横切内容由父后续组完成；本组只保证本票权威 ETA 与费用链。
- `EXTERNAL_ARCHIVE` 的前端实际 PDF 呈现、浏览器四类操作与最终整体验收由主会话负责，本报告不以服务端投影或文件存在替代打印验收。
- 服务范围未完成取消由前端显式确认；服务端对已完成事实的撤回／更正保留原因及不可变历史。
- 当前未运行全量 check:fast／build；按调度交由主会话在独立审查后统合。未提交、未归档，服务端产品写权已释放给主会话及审查代理。
