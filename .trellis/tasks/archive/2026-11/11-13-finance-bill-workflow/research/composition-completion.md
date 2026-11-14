# 阶段 F：建账、已有发票登记与核销组合（2026-11-14）

## 实现边界

用户已批准父任务整套财务模块实施。阶段 F 在既有逐组建账基础上消费发票多记录、逐费用金额/手工税额及差额核销契约，不串联独立写请求，不新增公司选择或财务动作权限旁路。

- `CreateBillBatchGroupInput.invoices[]/verifications[]` 为平铺信封；尚未存在的账单 ID、初始版本、公司、方向、结算单位和币种由组产物内部填入。
- 普通模式组内顺序为建账→同组多票登记→逐笔核销/差额登记，任一步错误整体回滚；组与组独立。NETTING 保留整批原子，任一组附带操作失败时双方账单及草稿对冲同时回滚。
- 每步校验对应权限及公司范围。组失败返回 `step/item_index/group_key`，前端显示中文记录序号；边界金额解析失败归入本组，不阻断普通模式的另一合法组，非法原始意图的稳定摘要仍参与幂等比较。
- `CreateWithinTransaction`、`CreateWithCashflowWithinTransaction` 只写入/回放身份；整个原子单元的既有资金按 UUID 排序预锁。资金仓储在共享事务中返回已写入对象，避免读取未提交的完整响应；完整账单进度在提交后重读，同键回放亦然。读响应失败视为未知结果，不能宣称整个操作回滚。
- 子步骤键由批次键、组键、步骤和位置派生；组合意图纳入组摘要。未知响应冻结原请求，使用相同载荷及幂等键确认，失败不会退化为只建账。
- 工作台按组复用 `features/finance/invoice-registration` 编辑器，核销支持多笔既有资金分配及明确填写的差额。资金候选绑定当前公司及本组方向/单位/币种，搜索其他关键词保留已选资金快照。差额不会随余额变化自动扩大。

## 已运行验证

1. 正式迁移链初始化的独立 PostgreSQL：`TestFinanceBillCompositionAtomicPostgres` 十场景全通过（2026-11-14 17:01 UTC，约 3.6 秒），无 SKIP；日志 `/tmp/bill-composition-pg.log`：
   - 多票、多次核销及同键重试；无发票先核销；既有资金加差额；多笔既有资金锁序；同键并发无重复。
   - 第二张发票失败整组回滚；第二条核销失败整组回滚；缺附带权限整组拒绝；某组失败不影响另一组；NETTING 任一步失败整批回滚。
   - 第一轮发现相邻旧 `active_bill` 唯一索引阻断同账单多票，已由发票任务修正正式迁移；清理夹具同步纳入新发票行，最终第三轮十场景完整通过，没有删除业务断言。
2. `go -C server test -p 32 ./internal/biz ./internal/service ./internal/server` 全部通过；含金额边界坏值不污染另一组的 service 回归。日志 `/tmp/bill-composition-go-final.log`。
3. `pnpm --dir web exec vitest run` 定向四文件、十一例通过，各文件不超过 20 秒：
   - `BillVerificationFields.test.tsx`：真实 Form 新增两条核销与差额、删除行触发输入变更、保留其他配置；资金搜索空结果仍保留选择与金额；无核销/资金权限的入口与候选门禁。
   - `billCompositionPayload.test.ts`：两张发票和两笔核销映射、临时账单身份不外传、外组费用/金额不符拒绝、中文步骤错误映射。
   - `BillCreationWorkbench.partial-result.test.tsx`：部分成功及原请求重试、失败组调整、未知结果冻结。
   - `BillCreationWorkbench.submission-session.test.tsx`：旧初始化和提交响应不能覆盖新来源会话。
   日志 `/tmp/bill-composition-web-final.log`。此前既有 Workbench 与 GroupCard 两文件十三例也通过（`/tmp/bill-composition-web-tests.log`）。
4. `pnpm --dir web tsc` 通过；改动文件 Biome、`git diff --check` 与 `BillVerificationFields.tsx` 的 `antd lint` 通过。
5. 四份 acceptance 脚本已同步免确认及正式发票登记 RPC；费用 E2E 同步整表保存、统一订单锁、当前公司无 Select、逐组令牌及发票登记入口。四脚本 `node --check` 通过；本地语法验证不等于实际 HTTP/浏览器验收。

## 交父任务统一验收

`acceptance-finance-foreign-currency.mjs` 调用组合链路夹具：故意第二笔核销超额、无费用占用遗留、两票与两笔差额成功、原请求重试无重复、利润/收付/两个方向账龄查询。实际收付检查交易日、币种及原本币合计。四报表桌面/窄屏截图、生产同域静态/API 路径、发票登记入口及完整门禁由父任务在一次性独立环境统一执行，尚未执行的项目不记为通过。

共享 Proto、Ent/API/OpenAPI/Web 生成物与发票、报表实施互相依赖，须由父任务按完整契约组提交，不能只提交本组调用方留下不可编译中间态。零额/负数账单仍遵守已批准的本期边界：只验证纯进度判断，不放开创建和数据库金额约束。
