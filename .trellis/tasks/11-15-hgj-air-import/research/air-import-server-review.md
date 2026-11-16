# 空运进口服务端独立复核

日期：2026-11-16（UTC）。基线：`74f86b3e` 后全部服务端变更及新增文件。复核者为独立 `trellis-check`；本轮只新增本报告，未修改产品代码、测试、生成物、正式迁移、开发库或非生成前端，未提交/推送。

## 发现与处理

未发现需要本轮修复的确认缺陷，无未修复的服务端阻断项。已按实际路径核对以下合同，而非仅根据测试名或类型枚举判断功能存在。

- AI+IMPORT 创建、已有主单ID创建及幂等进入同一航空规范化/指纹路径；AE保持EXPORT。普通保存仍禁止变更主单、模式与既有HAWB号码，DIRECT没有虚拟分单，HOUSE保存独立正文/版本/历史。
- 航空候选使用显式AE/AI与对应read权限；主单ID、分单历史和ORDER/MASTER重量由当前公司内目标Order/MAWB稳定身份解析，不能用请求的另一类型覆盖。Proto声明、生成权限规则、Authorization、Service和重量仓储相符，HTTP/gRPC都装配 `airUsecase`，Wire调用已同步。
- 共享成员读取与写入复验完整成员数、公司、业务类型和方向，不以AE过滤后漏掉AI。共享写入先排序锁Order集合，再锁MAWB和Link，集合变动拒绝，不在主单锁后追加未知Order。
- 特别核对新增审核锁入口：`LockOrder` 在授权父锁之后复用 `lockAirLifecycleTarget`，航空锁完整Order集合，再进入MAWB/Link/可选HAWB快照；目标预期Order版本、费用版本/指纹与航空单证指纹仍在同一事务检查。非航空入口继续锁本票，不改变SE真实海运快照。
- 恢复仍要求原主单及原模式、Order/Master预期版本和目标同类型update/transition；HOUSE额外复验历史HAWB公司、主单和原号码。恢复新建Link，不复活ENDED历史；DIRECT不要求不存在的HAWB。
- AI的列表机场/日期、详情与报表取航空权威主单。共享ETA修改同步成员日期和系统周率；经营日按中国自然日，缺率回滚主单、订单、费用和历史。手工率/受保护费用与既有账单快照沿用公共合同，不新增进口算法。
- AI复用前置五服务，Order.version为唯一基线，完成/撤回原因、OPEN/审核锁/终止门禁与审计事务保持；完结不补出口flow，未承接不强制完成。
- 本组没有Schema变更，沿用正式迁移；SI迁移测试改为精确current_schema表存在性断言，避免public同名表误报，未放宽产品约束。

## 本轮实际验证

```bash
go -C server vet ./...
make -C server lint
git diff --check
go -C server test -p 32 ./internal/server ./internal/service ./internal/biz \
  -run 'TestAir.*HTTP|TestAirExport|TestMasterCandidatesRequireExplicitBusinessIdentity|TestGeneratedAccessRulesMatchProto' -count=1
```

结果全部PASS，退出码均0。三个定向包实际耗时：server 0.114s、service 0.034s、biz 0.036s。该测试正则执行现有航空HTTP真实路由/Authorization/Service/Usecase矩阵：两真实航空类型的11项ID接口×5组授权，共110场景；显式候选类型另10场景，共120场景。仓储边界为stub，不能称作同一HTTP请求连接真实PostgreSQL。生成权限规则另与Proto实际描述符对照通过。

Go类型/编译由上述测试及全仓vet覆盖；前端tsc与最终高风险全栈门禁由主会话统合。本轮未发现新增修复所需的数据库场景，按调度未重复执行已有54.826s/22.721s等广泛PG批次，也未重复生成打断浏览器。真实PG与91份Proto/OpenAPI临时生成、访问规则/Wire生成的实际证据属于实施者，已核对 [air-import-server-validation.md](./air-import-server-validation.md)，本报告不将它们冒称本轮重新执行。

## 范围与移交

- 服务端、Proto及生成物当前冻结，写权释放给主会话；可以执行最终 `check:fast`。
- 非生成web由独立前端检查者负责；真实浏览器、PDF、CSV及完整四类财务链不由本报告替代。工作台/通知/待办横切仍依父后续任务，不能因为AI建票可用而标记全部完成。
- 当前规范已覆盖本组真实品类、双模式、重量、权威ETA与五服务航空基线，无需本代理另外调整spec。已批准开发库孤立主单处置不是本次代码变更，本代理未重复执行或改写迁移。
