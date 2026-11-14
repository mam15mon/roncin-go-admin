# 发票直接登记与修订独立审查

日期：2026-11-14。按发票子任务 `check.jsonl`、PRD、设计、实施计划及后端事务/前端当前工作区规范独立审查；实现代理冻结后接管范围内修复。未提交Git，最终完整门禁、生产构建与一次性HTTP/浏览器验收由父任务统一执行。

## 已修复发现

1. **发票列表丢失真实账单关联。** 原 `ListScoped` 只预加载组织，账单内列表读取的 `billLinks` 永远为空，导致金额0、关联单号空、有效票错误显示已释放。现按发票主实体分页/汇总，独立预加载真实关联；账单内场景再加载逐费分配，普通台账只取轻量关联。跨账单票据不倍增计数或汇总。与实现代理的“只累计active、历史不累加、关联单号去重”修复同步。
2. **登记账单候选没有兑现完整检索。** 原关键字只有账单号和历史中文结算名，页面却提示拼音。现复用 `financePartyKeywordPredicate`，覆盖代码、中英文、别名、全拼与首字母，并保留账单号、源订单号及历史结算名；过滤在COUNT/LIMIT之前，公司范围固定，不因多个匹配别名重复行。
3. **直接修订并发可混用两版响应。** 原详情头、关联、行及历史分别读取，没有一致快照。现详情、列表完整汇总/关联和登记候选统一使用 `Data.withReadSnapshot`，通过 `Data.client(readCtx)` 查询，现有快照复用。精确回归在读头后、加载关联前从另一连接提交真实修订：当前请求仍读旧头/旧分配，下请求完整读新版本，未使用等待或重试掩盖。
4. **修订历史丢失被覆盖的汇率经济事实。** 原快照只保存原币与分配，日期修订重算后原汇率、来源、日期及折本币金额不可恢复。现保存本币及完整前后汇率经济字段，前端按业务字段结构化呈现，金额/汇率/日期分别复用 `formatAmount/trimDecimal/formatDate`，来源明确区分SYSTEM、MANUAL、WEEKLY、BOC_SYNC及既有历史来源。
5. **合法移除被显示成损坏历史。** 原移除后的 `Links=nil` 编码为 `bills:null`，前端按数组合同明确拒绝。现快照初始化空数组，真实移除后输出 `[]`，保留经济事实及已移除状态，不渲染原始JSON/内部UUID/版本。
6. **500中文备注/反向原因被Ent按字节误拒。** `FinanceInvoice.note/cancellation_reason/red_flush_reason` 与领域/界面500字符合同不一致。现Schema统一 `MaxRuneLen(500)` 并正式重生成Ent，仅改变校验，不新增SQL列类型变更或迁移；真实PG验证500中文原样保存、501领域拒绝且不推进版本/释放事实。
7. **金额巨大正指数缺少早期上界。** `invoiceAmountValid` 原先仅限制小数位，后续比较可能展开极大正指数。现先短路指数 `-8..19` 再比较numeric值域，登记/修订规范化仍先于指纹序列化。新增极大正负指数、最大合法金额、零/负/精度边界验证。
8. **退役调用的测试与权限说明未同步。** 发票筛选测试仍mock已删除CreateModal/IssueModal，现改mock共享登记能力；权限 `FinanceInvoiceUpdate` 描述由“开具”改“修订”，键名及Requires不变。检索确认Proto、服务端、前端产品和验收脚本已无CreateInvoice/IssueInvoice双流程调用。

主要修复文件：`data/finance_invoice.go`、`data/finance_invoice_registration.go`、`biz/finance_invoice_registration.go`、`data/ent/schema/finance_invoice.go`及生成校验、`access/manifest.go`、共享`InvoiceRevisionHistory`及相关测试。新增专项测试为 `finance_invoice_review_integration_test.go`、`finance_invoice_text_length_integration_test.go`、`finance_invoice_review_test.go`。

## 已核查的相邻合同

- 逐票→逐账单→逐费用金额守恒、税额由财务明确填写、部分余额及同一次多记录共同基线；未税相减，不反改费用/提成。
- 订单闭包→账单→发票固定锁序，账单/票版本复核、事务失败无半条命令、持久命令同键重放与不同意图冲突；组合内部入口只写，完整响应在提交后读取。
- 原票按各自ID保留抬头快照，同profile不同旧票不互相覆盖；新增票使用当前启用主档，NORMAL改SPECIAL仍校验原票完整快照。
- 修订新增票需要同公司create权限；修订/作废/红冲共用既有update权限，不能跨当前公司、隐式反核销或反对冲。有任一有效核销或已确认对冲均拒绝直接更正，完整影响读取全部跨账单关联。
- 原分配只追加/停用，当前进度依活动事实派生；部分发票不当票齐，免票不伪装开票，最后财务事实解除不解除人工订单锁。
- 来源税率未设与显式零的领域/Schema/Proto/wire/展示存在性；正式迁移只从原账单快照补齐可确定事实，缺失明确拒绝，失败迁移不删除票据或留下半结构。
- 共享前端出口、当前公司与最低业务权限、前台15秒/回焦探测、原提交基线不被探测覆盖、失败/迟到输入保护、未知结果原键确认、结构化前后历史与金额格式化。

## 验证结果

| 检查 | 实际结果 |
| --- | --- |
| 新增列表投影、完整候选检索、详情/列表精确并发修订3个PG顶层测试 | PASS，11.237s，`/tmp/invoice-final-review-pg.log` |
| 追加完整经济快照、真实移除空数组的详情/列表精确并发PG | 最终PASS，7.752s，`/tmp/invoice-final-review-snapshot-final.log` |
| 受影响Register部分手税与终态释放PG相邻路径 | 最终PASS，6.501s，`/tmp/invoice-final-review-adjacent-final.log` |
| 中文500/501登记备注、修订备注/原因、作废与红冲原因PG | 最终PASS，3.074s，`/tmp/invoice-final-review-text-length-final.log`（含修订备注501拒绝及原版本保留） |
| 领域巨大指数及numeric边界9个子场景 | PASS，`TestInvoiceRegistrationAmountExponentBounds` |
| History与旧筛选mock定向2文件6例 | PASS，7.72s，`/tmp/invoice-final-review-web.log` |
| 最终实际来源标签及经济/移除History4例 | PASS，7.07s，`/tmp/invoice-final-review-history-final.log` |
| `pnpm --dir web tsc` | PASS，`/tmp/invoice-final-review-tsc.log` |
| 修改前端3文件Biome、`git diff --check` | PASS |
| `go vet ./internal/biz ./internal/data` | 最终PASS，`/tmp/invoice-final-review-vet-final.log` |
| `pnpm run check:permission-keys` | PASS，282键；描述调整不改变键名生成内容 |
| Ent生成 | PASS，官方 `go generate ./internal/data/ent`，未手修生成物 |

追加经济快照PG断言首轮错误地把同币种来源预期写成`BASE_CURRENCY`；现行 `ResolveRate` 返回 `SYSTEM`。该轮不是全绿；改为核对原登记事实后单独重跑快照及受影响邻接路径全部通过，未放松产品规则。父任务最终全量的具体失败与补跑另由总验收记录维护，不以本组定向结果冒称单次全量全绿。

## 规范与交接

新增可执行七节规范 `.trellis/spec/server/backend/finance-invoice-registration.md`，覆盖RPC/三层载荷、optional来源税率、金额/指数、事务/幂等、原快照/结算依赖/反向、正式迁移拒缺、错误矩阵及真实测试入口，已接后端index与能力导航。前端导航已存在并核对四个共享组件及最小公开导出，未反向依赖页面。

当前发票范围未发现已知未修缺陷。产品源已冻结，所有日志保存在仓库外且设为600；数据库验证使用专用隔离Schema，未清理或重建开发库。父任务负责最终门禁合并证据、生产入口/HTTP/浏览器及提交，不在本组声明这些已完成。


## 最终HTTP验收追加：抬头候选被详情路由截获

父任务真实浏览器验收在票据编辑器请求抬头时捕获400，后端operation为`GetInvoice`，并非`ListInvoiceProfilesForParty`；证据为仓库外私有`/tmp/roncin-finance-final-runtime-seventh.log`。此前service直调与编辑器mock通过不能证明真实HTTP分发正确。根因是Kratos按注册顺序匹配，旧候选`GET /api/v1/finance/invoices/party-profiles`注册在动态`GET /api/v1/finance/invoices/{id}`之后，静态片段被当成详情ID。

最小修复：Proto唯一源将该候选改为`GET /api/v1/finance/invoice-creation/party-profiles`，独立创建候选命名空间避开动态记录路由；不保留被遮蔽的旧静态路径。执行官方`make -C server api`与`pnpm run generate:web-client`同步绑定、OpenAPI和生成客户端，权限/RPC名/JSON语义不变，没有手改生成物，也未修改其他UI产品或Ent。规范中的HTTP签名与回归入口同步。

新增`server/internal/server/invoice_creation_route_test.go`：嵌入Unimplemented服务，仅覆写目标候选和详情方法，注册完整生成HTTP路由并通过httptest真实GET，验证camelCase与snake_case查询绑定、候选正确方法及Operation、正常UUID详情仍正确、旧静态路径不再注册。新增service最小权限/当前公司回归，确认仅invoice.create即可读此候选、只有invoice.read与系统工作区均不查询仓储。

本组验证：

- 新真实HTTP3个子场景、service候选scope3个子场景和既有`TestInvoiceCreateCandidatesUseWritableOrganizationScope`全部PASS（server0.062s、service0.030s，`/tmp/invoice-party-profile-route-go.log`）。
- `pnpm run lint:proto` PASS，`/tmp/invoice-party-profile-route-buf.log`。
- 重新生成后`pnpm --dir web tsc` PASS，`/tmp/invoice-party-profile-route-tsc.log`。
- 原共享Editor定向3例PASS，5.63s，`/tmp/invoice-party-profile-route-editor.log`。
- `git diff --check` PASS，产品/生成源码已冻结。最终完整server、生产构建与真实浏览器重验仍由父任务统一完成，本组不以定向代替这些检查。

## 最终截图追加：成功关闭后的查询取消误报

父任务完整真实业务验收通过后，审阅成功截图`04-invoice-registration.png`仍发现“请求失败／请联系系统管理员查看服务日志。”通知；同轮浏览器API未出现HTTP400以上错误，抬头候选HTTP返回200。独立复查证实新增抬头和修订初始影响查询携带AbortSignal，却未声明由React Query处理错误；关闭/卸载取消axios请求产生CanceledError，经requestClient全局errorHandler被当作普通失败通知，绕过了React Query对正常取消的隔离。它也使真实失败同时出现全局notification和query.meta消息。

最小修复仅在`InvoiceRegistrationEditor.tsx`的抬头查询和`InvoiceRegistrationModal.tsx`的初始影响查询传`{signal,skipErrorHandler:true}`。不改全局请求层；保留两查询原有`meta.errorMessage`，真实异常继续抛给QueryCache显示明确提示，取消则由React Query正常处理。既有探测查询已经使用相同边界。

新增`InvoiceRegistrationQuery.request.test.tsx`走真实组件、生产createQueryClient、生成API客户端、真实requestClient/errorConfig与axios实例，仅替换网络adapter。adapter监听实际AbortSignal并抛真实CanceledError；未修代码时，两查询卸载都证实发生abort并精确复现与截图相同的notification（`/tmp/invoice-query-cancellation-before.log`，4例按预期暴露错误，不当作通过）。修复后验证两种卸载取消不提示、两种真实HTTP500失败仍只显示一次各自query.meta明确消息，未吞异常；原Editor与修订相邻路径一并3文件9例PASS，5.92s（`/tmp/invoice-query-cancellation-after.log`）。最终`pnpm --dir web tsc`、修改3文件Biome和diff检查PASS。源码已冻结，父任务负责最新构建及最终空运浏览器/截图复验。
