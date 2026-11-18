# 独立检查记录

日期：2026-11-18（北京时间）。角色：trellis-check。检查范围为本任务全部前后端、Proto及生成物、现行规范、受影响验收脚本和测试差异；未提交Git，未修改其他业务模块。

## 结论

最终补查发现详情状态仍显示实体“正常”的遗漏，已最小修复为业务五态并保留取消终态，可提交本任务变更。完整仓库check:fast存在已在改动前基线复现的20例失败，因此不能称为全仓门禁通过。

## 已核实的实现

- 五态biz解析与data SQL组合一致：有效账单且completed优先，开票NONE/PARTIAL优先，票齐或免票后按结算及实际核销金额显示核销待办。仅部分对冲且实际核销为0不被称为部分核销；非有效或未完整投影为UNSPECIFIED。
- 多选OR组合加入既有统一predicates，再由同一过滤计算分页total、matchedSummary及顶部指标；保留公司与实体有效状态边界，其他筛选为AND。
- 新Proto枚举与字段为追加，服务层只转换不下沉Ent；未知和UNSPECIFIED筛选由biz显式拒绝。真实HTTP绑定用例覆盖camelCase、snake_case、枚举名和浏览器Axios的businessStatuses[]数组形式。
- 候选在Count/LIMIT之前复用开票资格谓词；新增核销在既有账单锁内读取最新有效发票关联／免票事实并拒绝无票目标。未新增事务、锁序、Schema或权限旁路；差额资金与建账同次核销走既有共享事务，幂等回放及反核销继续适用。
- 前端与CSV只映射businessStatus，不根据旧双进度或金额回退重算。多选沿现有查询／恢复／隐藏清值和采纳快照链路。
- 建账同次核销使用Form.useWatch读取本组发票草稿、禁止无票新增记录，Form.List自定义校验和最终payload校验共同防止删除最后发票后绕过；仍保留已有核销草稿供补齐后继续。权限检查延续原有能力，部分开票即可，不人为增加开票额度限制。
- scripts中原先先核销后开票的链路已同步为先发票／免票；无票直接API拒绝与组合差额回滚检查加入。现行spec记录用户2026-11-18新决定，没有把HGJ未知算法称为已复刻。

## Findings（已修复）

- `BillDetailModal.tsx` 原先“状态”仍消费实体status，有效账单显示“正常”，与列表／CSV五态不一致。改为消费共享businessStatus元数据，取消记录仍显示明确终态，缺投影不回退；新增真实详情渲染测试，覆盖部分开票→取消→未投影。没有为了全量通过而修改无关模块。

## Findings（未修复）

1. 全仓前端Vitest现存4个未改文件共20例失败：commissions/index权限mock，以及orders/new、new-create-idempotency、new-source的数据路由／来源夹具。主会话在git archive HEAD产品改动前基线已原样复现相同20失败，见validation.md及仓库外baseline日志。属于本任务外既有测试维护，未修复、未关闭规则；不是本次实现的可归责缺陷。
2. 完整finance-bill-batch页面E2E初轮存在旧入口／异步候选夹具失败，前端代理已最小校准测试；主会话随后实际完整海运财务E2E通过（1.4分钟），包括真实发票手税、先票后核销、提成和四报表。该PASS由主会话执行，本检查只引用证据，见validation.md及仓库外e2e-business.log。
3. 正式数据库CHECK要求账单total_amount/base_currency_amount及核销分摊为正，不能构造零／负账单或负核销分摊的真实PG记录。纯Go规则覆盖零额完成与带符号核销；PG验证免票完成、混合核销／对冲结清与纯对冲。没有删除正式约束制造样本，后端会在backend-validation.md说明证据边界。

## 独立执行的验证

- `pnpm --dir web lint`：PASS。Biome仅报告已有3个warning及2个info，随后tsc通过；未禁用规则。
- `pnpm --dir web tsc`：PASS。
- 对6个本任务产品源文件执行Biome check：PASS（无修复）。
- 6个关键Vitest文件共24例：PASS，耗时10.51秒，覆盖五态显示／CSV、多选查询／恢复／隐藏、建账核销前置与移除发票保留草稿。
- `go -C server test ./internal/biz -run 'TestResolveFinanceBill(BusinessStatus|Progress)' -count=1`：PASS。
- `go -C server test -p 32 ./internal/biz ./internal/service ./internal/server -run 'Test(FinanceBillBusiness|BillBusinessStatus|ListBillsBusiness|CanVerifyFinanceBill)' -count=1`：service/server PASS；该正则在biz无匹配，biz实际解析测试由上一条独立执行。
- `git diff --check`：PASS。

## 其他执行者证据与限制

后端定向biz/service、真实PG五态与核销专项、整包PG结果由backend-validation.md给出；本检查未重复执行全部真实数据库用例。主会话check:fast的服务端门禁通过，前端全量20例基线失败如上，不能用本检查的定向PASS替代完整门禁结果。主会话独立运行真实浏览器五态多选／重置及API应收应付、外币和组合原子核销，证据在/tmp/roncin-bill-status-qa/，原始业务报文／凭据不进Git。

## 最后文案补充复核

主会话最后同步 `BillCreationResultTable.tsx:158` 为“账单无需确认；核销前须登记发票或标记免票。”，已复读当前源码，与本次准入规则一致，避免旧成功提示让人误以为无票即可核销。该项仅文案调整，主会话Biome通过，本检查不重复业务suite。

## 详情五态修复的验证

- `BillDetailModal.test.tsx`：6例全部PASS（2.15秒），新增状态渲染测试使用真实详情组件，沿原测试基建断言渲染内容，不依赖JSDOM的Modal动画可见性。
- 上述详情源与测试Biome check：PASS；补改后tsc：PASS；无E2E文件变更。
