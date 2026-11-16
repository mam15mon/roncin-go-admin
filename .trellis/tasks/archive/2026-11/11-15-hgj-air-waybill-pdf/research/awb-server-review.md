# 航空制作服务端独立复核

## 范围与结论

2026-11-16，独立检查本组全部服务端源、Proto/生成物及新增验证文件，未发现需要修复的确认缺陷，没有修改产品代码或生成物。当前报告不替代前端制作页、真实PDF、浏览器或最终全栈门禁。

已读当前任务PRD/design/implement及check上下文，结合根/server AGENTS、server/backend与web/frontend索引质量检查、航空正文/方向/进口合同、航空制作与公司英文名规范。包发现仅列web/backend时仍显式核对实际server规范。

## 跨层核对

- 契约唯一增量为 `AirMasterWaybill.organization_name_en = 16`，标注 `OUTPUT_ONLY`；输入消息没有可写公司抬头。生成Go字段及getter、OpenAPI `readOnly`、前端OpenAPI输入及TypeScript投影一致。
- `biz.AirMasterWaybill.OrganizationNameEn` 为纯Go展示投影。service只将该字段传到API，不访问Ent或重新选择主体。
- `decorateAirMaster` 使用调用方传入客户端，按目标主单自身 `OrganizationID` 与 `kind=company` 读取；不从当前工作区、系统节点、其他公司或固定名补造。空英文名照常投影为空，不修改 `issuedBy`。
- `attachAirOrders`（详情、列表和幂等重读）、主单候选/列表及 `readAirMasterDetail` 均走上述装饰入口；普通仓储读取经 `Data.client(ctx)`，主单完整详情在原 `WithTx` 的 `tx.Client()` 中装饰。没有新增连接池直读或事务旁路。
- 主单查询及Order关联已按当前公司和真实AE/AI身份约束；新增名称查询进一步固定同一个主单公司。原目标权限解析与只读/写入校验未改变。
- 名称不进入 `AirMasterWaybillInput`、正文哈希或主单历史内容。预览/执行指纹和航空审核快照使用未装饰的主单/正文，管理员更名不推进Master版本或改写不可变正文；前端需按制作合同另将真实投影纳入输出变化指纹。
- AE HAWB自签主体与MAWB/AI外部 `issuedBy` 的选择属于前端保存快照模型，本组后端只提供真实公司事实，不替调用方篡改单证印发人。
- 无Schema、迁移、Manifest权限或静态资源配置变化，未引入历史回退、日志输出或秘密。

## 本人实际执行

| 命令 | 结果 |
| --- | --- |
| `go -C server test -p 32 ./internal/biz ./internal/service -run 'Air' -count=1` | PASS，biz 0.028秒、service 0.026秒；包括新增真实ProtoJSON名称/空值省略及正文独立投影用例 |
| `go -C server vet ./...` | PASS，包含服务端及测试编译/静态检查 |
| `buf lint`（server目录） | PASS |
| 对全部本组非生成Go源码/测试运行 `gofmt -l` | 无输出，PASS |
| `git diff --check` | PASS |

## 沿用实现证据与未重复项

实现代理已实际执行 `make -C server api`、`pnpm run generate:web-client`，原样再次生成SHA256零差异；本复核静态核对源及全部生成差异，没有再次触发生成。

实现代理已实际执行 `TestAirWaybillOutputCompanyProjectionPostgres`，通过进程环境注入真实PostgreSQL连接，在随机Schema顺序应用正式SQL迁移，AE/AI两个子场景无SKIP通过。用例覆盖Order embedded与Get投影、跨公司读取拒绝、系统/其他公司名不能借用、真实公司更名后名称更新且版本/hash/正文不变、缺英文名仍可读取且保留主分 `issuedBy`。本复核已阅读该用例及真实调用路径，没有无因重跑已通过的数据库专项；数据库结论明确来自实现证据，不能由普通Go测试的数据库跳过推导。

前端全量TypeScript、保存/轮询/输入保护、PDF解析/逐页实图、生产同域和最终check:fast由相应检查与主会话执行。本报告不宣称这些阶段已经通过。

## 发现处理与交接

- 已修复：无。
- 未修复确认问题：无。
- 服务端与所有生成物写权已冻结并交回主会话；没有重生成、未提交Git。
