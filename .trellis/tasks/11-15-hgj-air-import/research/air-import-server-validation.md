# 空运进口后端实施与实际验证

日期：2026-11-16（UTC）。本报告只记录本代理实际执行的服务端与生成验证；浏览器、非生成前端、独立复核及最终门禁由主会话另行汇总。未提交Git、未推送。

## 实施结果与契约

- `AI + IMPORT` 使用同一 `Order`、航空DIRECT/HOUSE、MAWB/可选HAWB及结构化正文；普通保存、已有主单ID建票和创建幂等均复用航空校验与指纹。没有新增航空或服务模型、Ent列、SQL迁移。
- HTTP路径和载荷保持现有 `/api/v1/air-export/...`、`/api/v1/air-weights/...` 及 `airExport` 字段。候选必须显式传 `businessType=AE|AI`，不能默认AE。
- 两航空Proto服务的权限注解改为既有 `ACCESS_MODE_ORDER_PERMISSION` 的read/update，并明确组织scope。HTTP/gRPC共用Authorization集中解析航空目标：候选按请求类型，其他按当前公司内真实Master/Order稳定身份；恢复另要求同类型transition。
- Service再次校验真实目标类型对应权限，重量仓储按ORDER/MASTER的真实目标逐次授权；完整成员公司、方向与类型错误显式拒绝，不过滤掉错误成员后继续计算。
- 创建、共享修改、读取聚合、终止与显式恢复支持AI。固定 `Order集合主键排序 → MAWB → Link → House/重量`；审核入口复用完整航空成员锁，不能仅锁本票再锁MAWB。恢复复验历史HAWB公司，历史Link保持ENDED，新建原主单/原模式Link。
- AI审核与AE一样绑定航空模式、Master/House/Link身份版本和独立正文哈希，固化航空快照；不填写海运快照外键。主单正文变更即使Order.version不变也能拒绝旧审核指纹。
- 计划/实际、ORDER/MASTER各自独立确认及历史继续使用既有算法；本票费用只采本票确认，过期确认保留旧值与旧费用。
- AI列表机场/日期、详情与报表复用权威航空主单查询。经营日取ETA并换算中国自然日；共享改期同步成员日期和可更新系统率费用。手工率、发票/核销/对冲保护费用保持原值；已建账无保护事实仍更新费用，账单行快照不动，沿既有公共率联动合同。
- 固定五进口服务直接使用前置实现，Order.version唯一；服务摘要、完成/撤回原因、审核锁、终止、完结与显式重开均在AI真实票验证。
- 费用台账主单号读取条件扩展到海/空进口，保持活动/本次终止关系语义。

## 修改文件

产品源：

- `server/api/order/v1/air_export.proto`、`air_weight.proto`：真实目标的动态品类权限声明。
- `server/internal/biz/{air_export,order_types,order_usecase}.go`：航空类型判定、候选显式类型、轻量目标身份与AI校验/幂等。
- `server/internal/service/{air_export,order_write}.go`：实际类型与已有主单共享写权限。
- `server/internal/server/{auth,http,grpc}.go`：集中航空路由目标授权及依赖装配。
- `server/internal/data/{air_export,air_order_sync,air_order_query,air_weight,fee_ledger_master_no,order_convert,order_write_lifecycle,order_lock,order_lock_apply}.go`：AI航空事实/完整成员/恢复/审核/权威查询与重量。

生成：`air_export.pb.go`、`air_weight.pb.go`、`server/internal/server/access_rules_gen.go`、`server/cmd/server/wire_gen.go`。已执行OpenAPI/web客户端/Proto常量生成，HTTP形态未变化，因此这些前端生成物和OpenAPI没有新增差异；Manifest未改，不需要新增权限键。

验证：新增 `server/internal/server/air_import_route_test.go`、`server/internal/data/air_import_integration_test.go`；将既有航空DIRECT、共享、正文、机场、重量测试按AE/AI同时运行。同步Go候选签名、Authorization/装配调用方、AI真实审核夹具与SI迁移隔离Schema断言，未删除既有业务断言。

## 命令与实际结果

### 普通Go、传输与静态检查

```bash
go -C server test -p 32 ./internal/biz ./internal/data ./internal/service ./internal/server -count=1
go -C server test -p 32 ./internal/server -run 'TestAir.*HTTP' -count=1
go -C server test ./cmd/server -run '^$' -count=1
go -C server vet ./internal/biz ./internal/data ./internal/service ./internal/server
buf lint server
git diff --check
```

结果：四包普通Go最终PASS（biz 2.154s/data 0.239s/service 1.540s/server 0.181s）；HTTP矩阵最终PASS（0.099s）；主入口编译PASS（0.046s，无测试用例）；vet、buf lint与diff检查PASS。普通Go会跳过未配置连接的PG用例，不用其证明PG通过。

HTTP矩阵保留真实生成绑定、Authorization、Service/Usecase，只替换仓储：AE/AI各11种ID接口 × 真实权限/只读权限/另一方向权限/跨公司/恢复缺transition，共110场景；候选显式类型另10场景。覆盖MASTER/ORDER重量预览/确认/历史、主单读/预览/写、候选/原票恢复、分单历史；伪造查询businessType不能覆盖ID真实身份。只有AI对应权限可达，AE借权、只读写入及跨公司均拒绝。

### 真实PostgreSQL

每次都通过以下原样包装运行，仅把 `.env.local` 的连接在子进程环境中传给测试，不打印连接串：

```javascript
import { spawnSync } from 'node:child_process';
const result = spawnSync('go', [
  '-C', 'server', 'test', '-p', '32', './internal/data',
  '-run', '<下列实际正则>', '-count=1', '-v',
], {
  env: { ...process.env, RONCIN_INTEGRATION_DATABASE_SOURCE: process.env.DATABASE_SOURCE },
  stdio: 'inherit',
});
process.exit(result.status ?? 1);
```

入口为 `node --env-file=.env.local --input-type=module - <<'JS'`。沿已有helper随机创建隔离Schema并顺序执行正式SQL；不使用Schema.Create、不清理开发业务表。下表PG运行均核对无SKIP。

| 实际 `-run` 正则 | 结果 | 证据范围 |
| --- | --- | --- |
| `TestAir(ExportShared\|Direct\|Waybill\|WeightConfirmation\|OrderLocation)\|TestAirMasterIdentity` | PASS，54.826s，0SKIP | AE/AI两模式、并发直单与历史模式、共享/私有写、结构化正文省略/清空/独立版本/历史、旧审核指纹、机场筛选与终态关联、ORDER/MASTER及PLANNED/ACTUAL/过期/费用来源与并发，航空旧迁移相邻 |
| `TestAirImport\|TestAirDirectCreate\|TestAirWaybillIndependent\|TestOrderLockAndUnlockPostgres` | PASS，22.721s，0SKIP | 当前修正后的DIRECT/HOUSE五服务/完结重开/审核解锁/审计回滚/终止恢复、外公司HAWB恢复拒绝、完整方向异常成员、ETA中国午夜跨周/缺率/锁回滚，直单及正文审核相邻；其中最后一个名称无匹配，不算通用审核验收 |
| `TestAirImport\|TestOrderLock_PostgresFlows\|TestFinanceReportAuthoritative\|TestFeeRateRefreshMatrix\|TestFeeLedgerMasterNo\|TestSeaImport` | 43.686s；其余全部PASS，仅SI旧迁移隔离断言误报，原因与补跑见下 | AI专项补列表ETA范围及报表，通用六类型审核/解锁/并发/幂等/审计/钉钉相邻、费用率保护/账单快照、台账号、AE/AI/SI报表权威日期、SI业务/服务/费用/共享/迁移 |
| `^TestSeaImportMigrationRejectsUnverifiableSnapshotsPostgres$` | 修正隔离Schema断言后PASS，2.590s，0SKIP | 真实失败迁移完整回滚DDL、原flow及revision，不误算public同名表 |

重点断言：

- AI DIRECT真实零HAWB、独立ORDER/MASTER确认及费用来源；同模式第二当前票、混历史模式和并发加票拒绝；HOUSE有独立正文、版本和永久占号。
- AI两模式五服务的承接子集与并行完成、纳秒时间、姓名投影、撤回原因、旧版本409、未完成阻断完结、已完结必须重开、审核锁与终止门禁、审计故障全回滚；恢复后本票服务与历史保留。
- AI同号AE主单身份独立；跨方向/公司已有主单ID创建不留半票。混方向成员使主单/重量/审核显式冲突；AI重量不能借AE权限。
- ETA从 `2026-10-25T15:59:00Z` 改到 `16:01:00Z`，中国业务日由10月25日到26日并切周汇率，ETD不变；另一成员缺率时主单、两Order、费用与历史全回滚；系统率更新、手工率保留；锁成员拒绝无部分更新。列表ETA范围、详情、报表和费用日一致。
- 报表额外制造AI订单旧ETA冗余投影，验证以同号AI主单的权威ETA命中中国日期，不借AE主单、订单旧值或ETD。
- 公共费用率矩阵验证已建账无保护事实联动费用且账单行不变，以及发票/核销/对冲保护费用保留，避免为AI另造重算/拒绝规则。

### 生成

```bash
make -C server api
pnpm run generate:web-client
go -C server generate ./cmd/server
```

结果：全部PASS。为不打断并行浏览器，最终幂等检查将 `buf generate --template buf.gen.yaml --output <临时目录>` 输出的91个Proto/OpenAPI文件逐字节比较仓库，并将 `go run ./cmd/generate-access-rules -proto-root . -output <临时文件>` 与当前权限规则比较：全部一致，未重写仓库文件。

## 验证中修正的问题

1. Proto动态权限初次生成缺scope，生成器显式拒绝；源Proto补组织scope后重生成成功，没有手补生成代码。
2. HTTP重量确认测试最初给了短指纹和空操作者，真实biz校验返回400；测试改为合法64字节指纹和UUID后矩阵PASS，未放宽产品校验。
3. 新AI服务测试引入解锁历史，原航空夹具清理顺序缺少先清UnlockRequest；补齐外键顺序后最新PG通过。
4. 普通data单测原SQL预期只写AE；同步到AE/AI权威日期表达式后四包全测PASS。
5. 更广SI旧迁移失败用例在public已经应用1600之后首次触发误报：`to_regclass('order_import_handlings')`沿search_path命中public表。断言改为 `pg_catalog.pg_tables` 精确 `schemaname=current_schema()`，定向真实PG补跑PASS；无产品/迁移改动。
6. 静态复核补齐审核完整航空成员锁序与恢复HAWB公司复验，最新真实PG明确覆盖，无已知剩余产品失败。

## 边界与移交

- 本报告不冒称真实浏览器、AWB制作/PDF或完整CSV验收；PDF/CSV和全局工作台/通知/待办横切仍属于父后续组。
- 横切剩余AE/SE限定入口包括 `biz/notification.go`、`service/workbench.go`、`data/workbench.go`、`data/workbench_todo.go`；本组没有据“AI可建票”宣称这些已接通。
- 本组沿用并实际执行已有正式迁移链；没有需要另加Schema迁移的模型变化，没有重置开发库、读取外站业务凭据或写入外站。
- 主会话继续独立服务端复核、浏览器记录及最终风险匹配门禁；本代理完成后释放服务端与生成写权。
