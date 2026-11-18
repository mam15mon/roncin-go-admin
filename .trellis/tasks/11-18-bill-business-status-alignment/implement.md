# 实施与验证计划

## 阶段门禁

- [x] 用户同意创建任务并规划，明确要求限制未开票核销。
- [x] 初始git工作区干净；读取适用规范、当前实现与HGJ只读证据。
- [x] PRD收敛，设计及验证范围写入文档，研究与上下文清单就绪。
- [ ] 向用户展示最终方案，取得其随后明确批准；之前不得task.py start或修改产品。

## 实施顺序

1. [ ] 批准后执行task.py start，按trellis-implement子代理协议分配契约／服务端和前端组；明确文件责任，不覆盖其他改动。先完成契约生成再让前端消费，统一生成物责任。
2. [ ] 新增biz业务状态类型、解析、筛选校验及Proto请求／投影，接入service与data同源谓词、汇总和候选。
3. [ ] 在现有账单锁内新增核销准入，覆盖独立／差额流水／建账同次路径，保留原事务及幂等语义。
4. [ ] make -C server api；pnpm run generate:web-client；检查源与生成差异。
5. [ ] 账单多选、服务端投影显示与CSV、筛选设置／恢复接入；建账核销前置提示和保留草稿的提交防护。
6. [ ] 功能实现后补定向测试与合法夹具；同步现行规范，不修改历史研究作为当前规则。
7. [ ] 按组验证并Conventional Commit；由trellis-check子代理独立检查／修复，最终check:fast及真实环境定向验收，归档收尾。

## 验证矩阵

- biz：五态优先级、部分票＋部分核销、先结清但票不齐、免票／零额、正负核销、纯对冲；非法枚举与多选边界；核销开票前置。
- service／HTTP：repeated枚举查询实际绑定、多选参数与权限／公司转换；ProtoJSON省略UNSPECIFIED、false／0及合法已投影业务状态；必要时增加生成路由真实HTTP请求测试。
- data真实PG：五态单选／OR／AND、全五态覆盖有效集合、取消／跨公司排除、分页与完整汇总一致、纯对冲及带符号核销；逐行投影与SQL筛选集合一致。
- 核销真实PG：无发票独立提交拒绝、多账单混入一张无发票整体回滚；部分／票齐／免票成功；旧候选在发票或免票撤销后拒绝；差额流水不残留；建账＋发票＋核销成功，无票组失败且不改变其他组既定语义；反核销／正常对冲回归。
- 前端：五态多选请求、重置、隐藏清值、恢复与公司隔离；当前页CSV消费已采纳快照与服务端状态；未投影状态不猜测。建账核销前置、移除发票仍保留草稿并阻止提交、登记部分发票后允许、权限不足场景；候选失败不当空成功。
- 搜索现有正向核销fixture及scripts验收中的先核销假设，只更新受规则影响的调用，不靠跳过测试或放宽门禁通过。

## 命令与风险

先运行受影响biz/service/data定向Go测试及前端单文件Vitest、变更文件Biome、pnpm --dir web tsc、git diff --check。真实PG使用既有隔离RONCIN_INTEGRATION_DATABASE_SOURCE和正式SQL迁移，不在业务开发库清表或Schema.Create；环境不可用时明确说明，不把SKIP当通过。

契约和跨入口财务规则风险需要最终执行pnpm run check:fast；普通UI不要求build，本任务也不改构建／依赖。定向测试正则与文件按实际新增名称完善，现有入口包括：

```bash
go -C server test -p 32 ./internal/biz ./internal/service -run 'Test.*(FinanceBill|Verification|BillBatch)'
go -C server test -p 32 ./internal/data -run 'Test(FinanceBillProgressProjection|VerificationCreateSharedTransaction|VerificationShortfallAtomic|Verification.*Candidates).*Postgres' -count=1 -timeout=5m
pnpm --dir web exec vitest run src/pages/finance/bills/bill-query-state.test.tsx src/pages/finance/bills/filter-settings.test.tsx src/pages/finance/bills/components/billProgressStatus.test.tsx src/pages/finance/bills/ledgerExport.test.ts src/features/finance/bill-creation/BillVerificationFields.test.tsx src/pages/finance/verifications/VerificationWorkbench.candidates.test.tsx
pnpm --dir web tsc
git diff --check
pnpm run check:fast
```

禁止测试先行。重型组件suite超过10用例或20秒须按功能拆分。没有数据库结构变化，不生成无意义迁移。
