# 航空模式与正文：服务端独立复核

## 范围与协作边界

复核航空 Proto 与 DTO、纯 Go 正文与校验、Ent 源与生成元数据、正式迁移、主分版本历史、普通保存、共享修改、审核指纹及重量事实。以父任务已批准设计和 `air-waybill-content-and-mode.md` 为准；本次不开放 SI/AI、制作页或 PDF。

先只读检查，待主会话明确释放 server 写权后再自修。未提交 Git、未归档、未改主会话维护的 spec，未重新生成或手改任何生成物。

## 已修发现

- 文件：`server/internal/data/air_order_sync.go`。
- 问题：普通保存取得 MAWB 行锁后未校验调用方显式提交的 `ExpectedMasterVersion`。主单正文变更不推进 Order.version，因此旧页面仍能携过期主单基线成功保存本票，违背预期版本合同。
- 修复：在 MAWB `ForUpdate` 之后、Link/HAWB 写入之前，`ExpectedMasterVersion > 0` 且不等于权威 MAWB.version 时返回 `ErrAirWaybillConflict`。省略值仍保留纯私有保存语义，没有强制所有更新携带主单版本。
- 回归：在 `air_waybill_content_integration_test.go` 新增 `TestAirWaybillOrdinarySaveChecksExplicitMasterVersionPostgres`。先通过正式主单命令改正文，确认旧主单基线的普通保存被拒，Order/House/Link 版本、正文、主分历史保持原子；再确认不携主单版本的私有保存保持主分正文及子实体版本，显式当前主单版本保存正常。

## 静态复核结论

- DIRECT 不创建 HAWB 或分单历史；HOUSE 创建真实分单。关联门禁检查全部历史模式，DIRECT 当前成员归零允许同模式新挂；并发活动 DIRECT 由部分唯一索引兜底。恢复读取原票历史模式及原主单，不提供普通保存切模式、换主单或改分单号。
- 共享写按排序 Order 集合 → MAWB → Link 顺序加锁并复验完整成员；普通私有保存持有目标 Order 后取得 MAWB/Link/HAWB，不反向追加其他成员锁。公司与真实 AE 类型在写入与 MASTER 重量入口检查，AI 不由 AE 接口提前放行。
- House 正文消息省略保持、`{}` 清空；真变化推进 House/Link 并追加历史。Master 正文独立写入与留史。审核指纹包含模式、主单、可选分单及 Link 版本，最终在相应锁内比较。
- 正文是纯 Go 结构与 JSONB 分层，声明文本与数值分开，AS ARRANGED 保持未知；合计使用 decimal。MASTER 重量指纹使用 Link 身份与模式，未把正文修订版本当货量变更。
- 正式迁移在数据改写前拒绝错误旧关系，完整保留旧正文与分单版本为 remarks，并以与 Go 编码一致的哈希回填；主单运输备注及原历史字节保留。Ent 与 SQL 的模式 CHECK、JSONB、哈希和活动 DIRECT 唯一索引同步。生成幂等已有实施证据，本次仅改仓储局部逻辑及回归，不触及源契约或生成链路。

## 未修发现

无已确认而未修复的本组服务端问题。主会话负责把显式主单预期版本的精确语义同步到规范；后续进口及 PDF 属已规划的其他组，不列为本组缺陷。

## 本次独立执行的验证

| 检查 | 结果 |
| --- | --- |
| 本组全部修改/新增 Go 文件 `gofmt -l`，修复后对两处改动 `gofmt -w` | PASS，检查输出为空 |
| `go -C server vet ./...` | PASS，退出码 0 |
| 下述正式迁移隔离 PostgreSQL 定向测试 | PASS，包耗时 3.326s，用例耗时 3.26s，1 PASS / 0 FAIL / 0 SKIP |
| `git diff --check -- server/internal/data/air_order_sync.go server/internal/data/air_waybill_content_integration_test.go` | PASS |

实际 PostgreSQL 命令如下，连接串仅从 `.env.local` 在进程内传递，未打印、未拼入命令，也未向现有开发业务 Schema 应用迁移：

```bash
node --env-file-if-exists=.env.local --input-type=module -e 'import {spawnSync} from "node:child_process"; const r=spawnSync("go",["-C","server","test","-p","32","./internal/data","-run","^TestAirWaybillOrdinarySaveChecksExplicitMasterVersionPostgres$","-count=1","-v"],{stdio:"inherit",env:{...process.env,RONCIN_INTEGRATION_DATABASE_SOURCE:process.env.DATABASE_SOURCE}}); process.exit(r.status ?? 1);'
```

测试 helper 创建随机隔离 Schema、顺序执行正式 SQL 迁移，并只删除自己的 Schema；未调用运行期 `Schema.Create`，未清空或重置开发数据。Go 类型编译由该 data 包测试与全服务端 vet 通过验证；无单独 TypeScript 改动。

## 采用的已有实施验证

`research/backend-validation.md` 已记录完整 biz/service 测试、航空双模式/正文/迁移/重量真实 PG（24.699s，0 SKIP），以及 Proto/Ent/OpenAPI/web 客户端 962 个生成文件重跑前后哈希一致。上述结果属于实施代理证据，本次未无因重复；主会话继续统合航空、方向、通用审核相邻 PG 与最终全栈门禁。
