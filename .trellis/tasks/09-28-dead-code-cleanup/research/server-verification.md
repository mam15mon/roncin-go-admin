# 服务端无用代码清理核查与验证

## 最终引用与取舍

- `IsBusinessOperationPermission`、`businessOperationPermissions`、`isBusinessOperationPermissionDefinition`：全仓 Go 引用只在三者之间，没有 `service`、`data`、`server`、`cmd` 或测试消费者；删除分类链。`manifest` 权限定义、`Requires`、`IsCompanyBusinessPermission` 和工作台授权入口未改，因此不需要重生权限键。
- `RequireBaselineWrite`：全仓只存在定义；现行全局主数据写入使用 `RequireGlobalMasterDataWrite`，B 型配置工作台另有身份判定。删除未接入函数，保留 `requireSystemPermission` 及现有真实写入路径。
- `OrderFeeSupplementStatus.CanTransitionTo`：只被 `order_fee_supplement_test.go` 的自身断言调用。实际审批、驳回、撤回都在事务中先锁定订单及申请行，再校验 `expectedVersion` 和 `PENDING`，并返回同一个 `ErrFeeSupplementTransition`。删除误称“唯一规则”的孤立函数及自证断言；保留 `Valid`、`Terminal` 测试和三条生产状态检查。真实库决策用例继续覆盖重复审批、终态拒绝与审批/撤回并发，并补充驳回后重复驳回、撤回的同一错误断言，以及并发输家的具体错误断言。
- `ComputeFingerprint(content)`：全仓只被 `TestSeaOrderChange_Fingerprint` 自身测试调用；前端 `split.tsx` 使用 canonical SHA-256 生成 `split-fp:` 指纹和拆票幂等键，服务端 `ExecuteSplit` 使用请求指纹与历史事件直接比对。删除孤立 Go helper 与测试，不改 `ComputeAttachmentFingerprint`、请求结构、前端算法及拆票幂等路径。真实库拆票用例仍覆盖相同指纹重放、不同指纹冲突、附件指纹冲突与并发竞争。

## 执行结果

1. `go -C server test -p 8 ./internal/access ./internal/biz ./internal/data -run 'TestManifest|TestOrderFeeSupplement|TestBusinessWorkspace|TestWorkspaceSeparatesSystemManagementAndCompanyBusiness|TestPublicLocationWriteRequiresSystemWorkspaceAndPermission|TestSeaOrderChange_' -count=1`：三个包均 PASS。
2. `go -C server vet ./internal/access ./internal/biz ./internal/data`：PASS。
3. 一次性 PostgreSQL 库中执行 `go -C server test -p 8 -v ./internal/data -run '^(TestFeeSupplementDecisionPostgres|TestSeaOrderSplitAndReassignment_PostgresIntegration)$' -count=1`：2 个顶层、23 个子项 PASS，0 FAIL、0 SKIP。日志为 `/tmp/roncin-deadcode-server-a045e5b9.log`。库和角色在命令退出时清理，随后查询确认均不存在；未使用开发库。
4. `git diff --check` 与修改 Go 文件的 `gofmt -l`：PASS，无输出。

全量 `pnpm run check:fast` 及整体任务收口由主会话执行。未改 Proto、Ent Schema、正式迁移、生成物或权限码。
