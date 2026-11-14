# 收付核销独立审查（2026-11-14）

## 范围

审查本轮差额流水原子核销、分步权限与公司范围、幂等/并发/回滚，以及工作台输入会话和三列表筛选。只补确定的局部缺陷，保留所有其他代理正在实施的账单、免票及费用检索改动；未提交 Git。

## 已修复

1. **勾选变化没有使在途提交失效**：`VerificationWorkbench` 原本只有金额、备注等编辑推进提交会话；新增或移除资金、移除账单、全清候选后，旧请求成功仍可关闭已经改变选择的弹窗。增加事件内同步 `invalidateSubmitSession`，覆盖方向/对象/币种与两侧候选选择；从已有分配变为空分配也推进会话。查询范围选择仍沿原行为，不单独标记为已编辑业务草稿；真正分配/金额编辑继续使用离开保护。
2. **无序分配仅重排会生成新幂等键**：服务端 `sameVerificationIntent` 按资金、账单与金额集合比较，但前端使用原数组顺序生成 JSON 指纹。未知结果后重选相同候选可能只改变分配顺序，却取得新键并重复核销。现在仅对意图指纹排序，保留实际提交的原分配、已规范化金额和可选字段；同一分配重排继续使用原键。
3. **对冲操作列测试缺少新增筛选导出**：`nettings/index.test.tsx` 局部 UI mock 没有 `SearchFilterTemplate`，导致原有4个状态/版本操作测试在渲染时报错。补全本文件的聚焦 mock，保留原状态、权限和请求断言；真实筛选交互仍由独立 `financial-filter-settings.test.tsx` 覆盖，没有屏蔽该测试。

新增行为回归包括资金新增、资金移除、账单移除、全清并生成空分配后旧成功不得关闭/刷新，以及未知结果后仅重排相同分配仍复用原键。同步更新 `cashflow-create-idempotency.md` 的可执行契约。

## 后端核对

- 差额金额必须精确等于引用本次流水的分摊之和；无差额载荷的空资金ID仍拒绝。
- 核销创建、流水创建、流水确认（`cashflow.update`）及显式手工汇率权限分别执行，同一公司范围进入仓储。
- 流水创建/确认与核销共用外层事务；核销仓储仍按订单闭包→排序资金→排序账单加锁、锁内复核余额；新增流水在本事务提交前不可见。
- 组合子流水键由主核销键派生；成功重试先比较既有流水意图，核销同键不同内容拒绝。唯一约束失败的重放在整个外层事务回滚后，仅当原流水及原核销均匹配才返回。
- 本轮静态核对未确认新的金额、权限、并发余额或回滚缺陷；没有修改后端产品代码。既有真实PG验证见 `implementation-verification.md`，本审查没有把未重复执行的PG测试列为本代理的新验收结果。

## 尚未修复，已交主会话续做

- **核销候选缺少分页/关键字**：当前 `ListCreationCandidates` 固定取两类候选的第1页、最多200条；超过该数量的同对象/币种数据无法在工作台选到，也缺少服务端候选关键字。这需要公开契约与UI分页改造，非局部机械修复。主会话已重新派实施代理补齐，不能以当前原子核销通过宣称R13全部验收。
- **阶段F的只写编排接口**：独立 `CreateWithCashflow` 的最后完整读取位于自身事务提交后，但将来建账F传入外层 `txCtx` 时仍须由F的纯写入口避免在大事务内组装完整响应。主会话已接受该集成约束，当前F尚未实施，本审查不擅自改用例公共接口。

## 本代理验证

- `pnpm --dir web exec vitest run src/pages/finance/verifications/VerificationWorkbench.test.tsx`：PASS，16例，组件用例本体1.90秒；包含新增5例。
- `pnpm --dir web exec vitest run src/pages/finance/nettings/index.test.tsx`：PASS，4例。
- 修改3个TSX文件的 `biome check`：PASS；工作台 `antd lint --format json`：零问题。
- `pnpm --dir web tsc`：PASS。
- `git diff --check`：PASS。

日志位于 `/tmp/financial-atomic-review-workbench.log`、`/tmp/financial-atomic-review-nettings.log`、`/tmp/financial-atomic-review-tsc.log`、`/tmp/financial-atomic-review-biome.log`、`/tmp/financial-atomic-review-antd.log`。根会话的中间全量Web门禁曾读取到本代理写入中的初版范围标脏变更而失败；上述结果针对最终修正状态。
