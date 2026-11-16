# SI 服务端独立复核

日期：2026-11-16。复核基线为 `df217ccd` 后本组 `server/` 源码、Proto、Ent、正式迁移和生成物，包括未跟踪新文件。未提交、未归档；未覆盖其他代理的前端改动。本轮服务端与生成物写权已释放给主会话。

## 发现并修复

1. `service/order_import_handling.go` 完成时间响应使用 `RFC3339`，丢失合法输入的小数秒。已完成服务保存后原样读取再提交会被误判为更正事实并要求原因。改用 `RFC3339Nano`，整秒表现不变；`TestImportHandlingCompletedTimeRoundTrip` 实际执行领域值→ProtoJSON→解析→领域校验，覆盖整秒和带 `+08:00` 的九位小数秒，均不误要求原因。
2. 浏览器联调定位的共同航程权限断点：两个航程更新 RPC 固定要求 `reassign`，而该权限仅存在于 SE，SI 正常编辑权限无法进入。经主会话确认，在 `internal/access/manifest.go` 声明路由动作 `OrderTransportUpdate`，由唯一权限入口映射 SE→既有 `reassign`、SI→既有 `update`，其他类型拒绝；不新增权限码，不开放 SI 整票改配。Proto 两 RPC 改为 `transport.update` 并实际生成绑定及访问规则。
3. 聚焦检查时发现 `server/auth.go` 的主单候选鉴权仍固定为 SE，导致 SI 请求不能按真实 SI 权限查询，并可能借 SE read 进入 SI 候选。改为严格读取显式 SE/SI `businessType`；缺失、AE/AI 等拒绝。
4. 补充三项服务请求的品类及公司范围回归；补充真实 HTTP 注册、鉴权中间件、Service、Usecase 的航程和候选测试，防止仅仓储测试通过而 HTTP 无法进入。

## 合同复核结果

- 五项服务完整快照、非 optional false 与 optional 完成时间、任意子集及并行派生状态已核对。当前 JSON 仅保存服务事实，唯一提交版本为 Order.version，事件保留当时前后版本。
- 服务保存先锁 Order 并核对公司、版本、进口类型、OPEN、审核锁与终止门禁，再验证本公司供应商/人员、本票且本公司附件；事实、Order+1、flow、事件和审计使用同一统一事务。普通整票保存不清服务；完结只检查承接服务及既有异常/待审费用；无出口 flow 自动补齐。
- SI 真实 DIRECT/HOUSE、MBL/航程/HBL、同方向候选/直接 ID、共享完整成员和 Order→MBL→Link→后续记录锁序已核对。关联已有主单仅加票，不改已审核成员或正文；共享写入复验成员集合与方向，公司过滤不隐藏异常成员。
- SI 审核固化真实单证版本，终止恢复新建原主单/航程 Link，放货使用本票真实单证，外部 HBL 派生 EXTERNAL_ARCHIVE 与真实签发方。SE 拆改、审核和自签单证合同继续保留。
- 复核了鉴权按请求类型解析的共享入口：箱货、HBL、服务和审核锁均通过 `GetOrderId` 加 `FindAuthorized` 的真实类型/公司范围；除已修候选硬编码外，未发现同类 SI 断点。未开放 AI 创建。
- 正式 SQL 与 Ent 的服务表、事件不可变字段/外键、进口 flow 值集/适用性及 SI 审核 CHECK 同步。旧不可验证 SI 快照明确失败；未在本轮改写迁移或处置开发库记录。

## 实际验证

以下命令均为本轮实际执行，退出码 0。无数据库的定向命令均未选择需跳过的 PostgreSQL 用例。

```bash
make -C server lint
go -C server vet ./...
go -C server vet ./internal/service ./internal/server
go -C server vet ./internal/access ./internal/server
git diff --check

go -C server test -p 32 ./internal/biz ./internal/service ./internal/data/ent/migrate \
  -run 'TestImportHandling|TestOrderLockSchemaMetadataSupportsAllBusinessTypes' -count=1 -v
go -C server test ./internal/data -run '^TestSeaImportSchemaMetadata$' -count=1 -v
go -C server test -p 32 ./internal/server \
  -run 'TestGeneratedAccessRulesMatchProto|TestRequestOrder|TestHasPermissionKeepsOrderBusinessTypesIsolated|TestMatchSeaMasterBillCandidate|TestSharedContainerRequests' -count=1 -v

# 最终源码：包括时间回环、权限映射、HTTP外层门禁、候选类型与Schema元数据。
go -C server test -p 32 ./internal/access ./internal/biz ./internal/service ./internal/server ./internal/data/ent/migrate \
  -run 'TestImportHandling|TestSeaTransport|TestSeaCandidateHTTP|TestMatchSeaMaster|TestGeneratedAccessRules|TestOrderOperationWrites|TestOrderLockSchemaMetadata' -count=1 -v
```

最终 HTTP 测试的仓储边界为明确 stub，保留真实生成路由、Authorization、参数绑定、Service 与 Usecase。航程预览/执行共覆盖 SI.update、SE.reassign、SE 仅 update、SI 借 SE、SI 仅读、跨公司；SI 整票改配拒绝另覆盖四场景。候选 HTTP 覆盖 SE/SI/缺失/AE/AI 各与 SE/SI 授权组合，共十场景。所有场景实际 PASS。此证据不冒称同一 HTTP 测试连接了真实 PostgreSQL。

真实 PostgreSQL 定向相邻回归使用 `.env.local` 中连接串仅在子进程环境映射，未输出凭据：

```bash
node --env-file-if-exists=.env.local - <<'JS'
const {spawnSync} = require('node:child_process');
const result = spawnSync('go', ['-C', 'server', 'test', './internal/data',
  '-run', '^TestSeaImportSharedDocumentsCargoAndArrivalPostgres$', '-count=1', '-json'], {
  stdio: 'inherit',
  env: {...process.env, RONCIN_INTEGRATION_DATABASE_SOURCE: process.env.DATABASE_SOURCE},
});
process.exit(result.status ?? 1);
JS
```

该用例实际 2.96 秒、包耗时 3.012 秒，1 PASS / 0 FAIL / 0 SKIP。覆盖 SI 同方向共享、独立服务、外部 HBL 及版本、真实共享航程 ETA 修改、箱货/分配、真实放货引用与异常方向拒绝。执行使用随机独立 Schema 和有序正式 SQL，仅清理自身 Schema。此前核心及扩展批次的证据见 `sea-import-core-validation.md`，本轮未无因重跑这些整批。

生成：实际执行 `make -C server api`、`pnpm run generate:web-client`、`pnpm run generate:permission-keys`，随后完整重复同一流程并比较 SHA256。125 份绑定/访问规则/OpenAPI/客户端/枚举/错误原因/权限生成物零变化；权限键仍为 282，无新增权限码。未手改生成物。

## 未在本代理修复与验证边界

- 服务端范围内无剩余确认缺陷。前端 Dayjs 表单对外部九位小数秒转换成毫秒的问题已给主会话及前端检查者，属于其写权；本代理未越界改前端。服务端响应已完整保留 RFC3339 小数精度。
- 浏览器真实服务的权限回归、PDF实际渲染、最终全栈 `check:fast` 与生产 build 由主会话统合；本轮未把上述 HTTP stub 或单项 PG 结果当作它们的替代。
- 新权限动作及时间合同已通知主会话同步规范，规范由主会话独占维护。
- 开发库孤立主单的明确归 SE 处置由主会话另行执行。本轮仅提供精确 ID 的一次性临时迁移副本建议，不改正式迁移、不伪造 Link、不删除主单/历史；未实际执行开发库迁移。
