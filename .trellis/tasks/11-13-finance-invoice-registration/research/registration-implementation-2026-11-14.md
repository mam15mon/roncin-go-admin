# 发票多记录登记与修订实施记录

日期：2026-11-14。产品代码已交独立发票检查代理复查；本记录不是父任务完整验收声明。

## 本组实施

- 用 `RegisterInvoices` 多条已有票据直登记替代 Create/Issue 双步骤；一次提交具备命令持久幂等与请求指纹，多记录共享事务，组内 `RegisterWithinTransaction` 只返回身份，完整响应在提交后读取。
- 明确发票总额→账单（版本/金额）→费用（含税额/财务手填税额）三层，逐层相等、不超逐费余额、禁止重复/异公司/异方向/对象/币种/本币；不按来源税率计算本次税、不补尾差、不反写费用及提成税基。
- 新不可变逐费分配、命令和修订历史实体及正式迁移；移除旧每账单一条活动发票唯一索引。来源税率在领域、Schema、Proto、生成物及前后快照中保留未设与显式零的区别。
- `ReviseInvoiceRegistrations` 锁内校验全部来源及票据版本；无有效核销和已确认对冲才可直接修改或移除，原因必填；原分配停用而不删除，追加新事实，保留业务前后快照、人、时、原因。已有结算时影响接口展示所有关联，修订/作废/红冲均拒绝隐式回退；原始余额只释放一次。
- 逐票保留原抬头快照及专票完整性；同一 profile ID 的不同旧票不能互用历史快照，修订夹带新增票始终读取当前启用主档。修订新增票须同时具备 invoice.create 权限；原票更新只需 invoice.update。
- 新 `ListInvoiceProfilesForParty` 在登记权限下提供本公司客商抬头，组合建账不依赖 partner.read。候选支持 exact bill_id，登记变动探测不依赖 bill.read；修订使用完整影响接口探测，不覆盖提交基线。
- 共用 `features/finance/invoice-registration` 编辑器、弹窗、分配追溯和结构化修订历史；发票页及账单详情入口已接入。逐记录/账单/费用编辑，失败保留输入；意图→键映射在页面实例内跨关闭保留，A→B→A复用原键，旧提交不得关闭已编辑或重开的新会话。
- 账单侧显示当前有效分配，历史修订不能累加成当前金额，关联账单名称去重；历史逐费分配仍可展开核对。费用台账开票信号按有效分配足额计算，免票不伪装已开票。

## 已执行定向验证

- `go -C server test ./internal/biz ./internal/service -run 'TestInvoiceRegistration|TestNormalizeInvoice|TestFinanceInvoice|TestInvoiceRevision|TestInvoiceAllocation'`：通过，覆盖金额精度、税额边界、逐票原抬头/专票要求、修订新增权限及 wire optional。
- 专用隔离 PostgreSQL：`TestFinanceInvoiceRegistration*` 与 `TestFinanceCorrectionInvoiceEconomicContentPostgres` 通过（12.132s）；部分手税/余额、原费用不变、多记录回滚、不同命令并发超开、持久重放、修订历史/移除一次释放、全跨账单影响与结算阻断。
- `TestFinanceInvoiceRegistrationMigrationPostgres`：通过（7.093s），正式迁移前态验证完整已登记/取消历史、来源税率 null 与零、不完备草稿拒绝、已删来源无法确定事实时拒绝；失败不删原票、不留下半套新结构。真实 DRAFT 受旧 CHECK 约束不可能带完整登记事实，迁移不伪造登记人/时间。
- 旧发票终态/免票 PostgreSQL 定向回归通过（首轮旧草稿反向用例按已批准新契约迁移后，其他有效状态、版本、原子、并发断言保留）。
- 新前端测试：4文件20例通过（9.74s），额外3文件6例通过（7.15s）；最终 Editor 新旧抬头区分后3文件10例重跑通过（11.53s）。覆盖真实 Form/Modal 提交、当前公司、最小权限、税零/金额正零、精度、三层守恒、失败保留、A→B→A、关闭重开迟到、在途编辑、跨账单影响、结构化历史及修订后账单金额。
- 旧发票相关4文件12例通过（7.05s）；旧 CreateModal/IssueModal 与其废弃流程用例已替换为共享登记行为测试，未保留废弃业务入口兼容。
- `go -C server test -run '^$' ./...`：全后端编译通过。
- `pnpm --dir web tsc`：最终通过。
- 修改文件 Biome：通过。`antd lint web/src/features/finance/invoice-registration --format json`：0问题。
- `pnpm run check:architecture`：18个检查用例通过，542个产品文件无边界违规。
- `git diff --check`：通过。

## 交接与限制

- 独立检查代理现负责 `data/finance_invoice.go` 的候选拼音检索、列表关联加载与读取快照一致性三处复查修复，及其专用真实库回归；应以其最终报告为准。
- 根会话负责整棵树全量门禁、生产构建、HTTP/真实浏览器与跨任务端到端验收，本组不以定向通过冒称完整交付。
- 按根协调要求未单独提交，避免把并行的报表/组合核销中间变更混成提交；最终由根统一提交。
