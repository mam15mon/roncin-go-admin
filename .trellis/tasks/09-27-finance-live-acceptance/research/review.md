# 财务实弹验收早期静态复核

## 范围与状态

- 复核时间：2026-09-27；读取时 HEAD 为 `57d7b1a0`，验收脚本、夹具、Playwright 配置及同订单 E2E 尚未提交，实施代理仍在修正运行问题。
- 使用 Trellis 检查上下文与当前 Proto、Biz、Data 代码交叉核对；本轮只写此报告，未修改实施代理拥有的文件。
- 本报告不是最终验收结论。未重复执行全量集成或浏览器；最终需针对稳定代码重新检查并采信实际运行证据。

## 已交实施代理修正的问题

1. **应付开单缺三岗**：`scripts/acceptance-finance-payable.mjs` 的 CreateOrder 原始请求缺 `personnelAssignments`。当前开单契约要求销售、操作、客服齐备；须补合法当前公司员工，不能关闭业务校验。
2. **外币核销断言仍为旧契约**：`scripts/acceptance-finance-foreign-currency.mjs` 仍配置 WRITE_OFF，并断言 `verification.exchangeRate`、来源、日期和设置 ID；Proto `settlement.proto:847` 已 reserved 这些字段。核销不产生第三种汇率，40 EUR 对应流水 1.25、账单 1.2 的固定输入应为流水本币 50 USD、账单本币 48 USD、汇差 2 USD，核销单头 `baseAmount` 为 50 USD。单头 `billBaseAmount`、`cashflowBaseAmount` 在 Proto 中仍保留，应继续断言，并补唯一分配行的来源 ID 和对应金额。
3. **浏览器金额与来源证据不足**：早期 E2E 已使用独立订单及专用销售员工，但主要只断言订单 ID 和总提成。应核对账单唯一行 `orderFeeId`、提成唯一订单行的费用 ID、125 收入、0 成本、125 毛利、12.50 提成及来源核销 ID；核销唯一分配连接本轮流水和账单，刷新重读后余额应为零。不能仅通过 `lines[0]` 和全页第一处金额判断。
4. **页面错误证据不完整**：早期 E2E 仅监听 `pageerror`，未记录 `requestfailed` 或意外业务 4xx/5xx。A4 要求无未处理请求失败，需监听并区分明确预期负例与意外失败，避免把取消导航导致的正常请求中止计入业务失败。
5. **费用占用 UI 相邻路径**：HTTP 辅助样本已覆盖草稿占用、重复建账、普通修改/删除拒绝、取消释放及同费用重建。浏览器早期只有草稿占用正例；建议补精确费用行重复建账拒绝和删除入口隐藏证据。取消重建保留独立辅助订单，不触碰主链已核销账单。
6. **敏感 trace 的保存口径**：失败 trace 可能记录登录密码、API 新用户密码、Cookie/请求头。Playwright 默认输出被 Git 忽略，新配置允许自定义目录；执行必须使用私密仓库外目录并控制权限。只公开脱敏链路摘要、截图或经审阅的记录，报告明确 trace 原件不能直接提交或作为可公开附件。

## 已确认符合当前边界

- 公司准备走正式成员资格及切换 API，不放开 system 工作台经营写权限；`allowCreate: apply` 保证无 `--apply` 时不创建 Membership。无 `--apply` 仍可能切换登录会话工作台，这是认证准备，不应记录为真实业务闭环通过。
- 客户责任人及订单三岗请求不带旁置组织；提成预览/创建改用员工、身份、公司，由服务端解析方案，未恢复 `ruleId` 旁路。
- HTTP 客户端在切换公司响应后更新 Cookie；服务 `SwitchOrganization` 的确调用 `setCookie`，因此更新是必要的。当前服务只有会话 Cookie，未发现需要多 Cookie 合并的新契约。
- 同订单浏览器关键财务写入均由真实页面点击触发，未发现 `route` mock 或 API 预写本轮费用、账单、流水、核销、提成。
- 一次性编排仍以本轮创建资源集合和任务前缀校验销毁对象；新 Web 端口校验范围并加 `--strictPort`，防止误连占用端口的其他服务。
- 编排改为完整 data/migration 两包、`-p 32`、40 分钟超时，并检查顶层 PASS/SKIP/FAIL；最终仍需运行日志证实执行数和零 SKIP。

## 核实后撤回的初步判断

早期曾怀疑浏览器 CreateUser 未显式建立成员资格。实际 `server/internal/data/admin_user.go` 的 CreateUser 在同一事务自动创建当前组织启用主 Membership，故不是缺陷；实施代理计划加只读成员资格断言，不应重复创建成员。

## 验证状态

- Lint：本轮未执行；文件仍由实施代理修改，待稳定后的最终检查。
- TypeCheck：本轮未执行；同上。
- Tests：本轮未执行；真实集成与浏览器由实施代理运行，本报告不宣称 PASS。
- 最终复核须确认以上发现的最终处理、实际精确金额、同链来源、刷新结果、失败原件秘密保护、资源回收以及运行代码 SHA。

## 最终静态审阅追加（代码仍在运行修复期）

审阅范围：`scripts/acceptance-finance-*.mjs`、一次性编排、同订单 Playwright 用例、`package.json` 和 Playwright 配置；只读交叉核对当前页面与账单工作台实现。本轮没有运行 HTTP 或浏览器，没有修改实施代理负责的代码。以下结论针对审阅时的未提交工作树，不等于运行验收 PASS。

### 已确认覆盖

- `package.json` 的 `acceptance:finance` 顺序运行应收 `--apply`、应付 `--apply`、指定财务 UI 用例；一次性编排 Stage A 实际调用此入口，Stage B 单独运行外币脚本。`--runtime-only` 跳过数据库包时日志明确标识，不能将其当完整门禁。
- 浏览器场景 API 仅准备公司可用的用户、客户、账户、订单和方案；费用、账单、收款流水及确认、核销、提成预览与生成由真实 UI 完成。脚本未注册关键财务路由 mock。
- 同订单用例保留 125.00000000 费用→唯一账单行 `orderFeeId`→唯一核销分配 `cashflowId`/`billId`→唯一提成订单行和费用 ID；收入 125、成本 0、利润 125、提成 12.50。刷新后读取提成详情和账单/流水金额，并检查页面不是原始 ISO 时间；主链有四张成功节点截图。
- 独立 HTTP 辅助样本覆盖草稿账单占用、重复建账拒绝、普通修改/删除拒绝、取消释放与同费用重建；浏览器还检查已建账行无删除入口及再次建账提示。主账单未被辅助样本取消。
- 外币脚本已按当前 Proto 移除独立核销汇率断言，固定输入仍检查费用 1.10、账单 1.20、发票 1.22、流水 1.25 的快照及核销 48/50/2 的行级金额；核销单头本币为流水分摊 50。应付与外币订单已补创建时三岗，提成由员工、身份、公司解析方案。
- 编排使用显式管理员连接串创建随机任务前缀 PostgreSQL 角色及数据库，Stage A/B 的迁移、服务与验收请求均显式指向本次 `connectionSource`。清理前查本轮创建集合与数据库 owner，结束后验证删除；预检 8010/9010/Web 端口并加 Vite `--strictPort`，不接管未知服务。集成门禁打开 `RONCIN_POSTGRES_MIGRATION_TEST=1`，两包完整执行并检查 SKIP 数。

### 已发主会话与实施代理的待修项

1. **账单创建异步竞态**：Playwright 用例在抽屉出现后立刻点击创建并等待 POST。`BillCreationWorkbench.createBatch` 会在结算账户尚未异步带入、`configurationComplete` 为 false 或预览令牌不可用时只提示并返回，不发 POST。应先等待默认账户的具体值及最后一次有有效 token、完整配置的预览，再点创建；不能使用固定 sleep。
2. **失败 trace 原件的私密目录**：当前 `web/playwright.config.ts` 允许 `RONCIN_ACCEPTANCE_ARTIFACT_DIR`，一次性编排未创建或校验该目录，默认 `web/test-results` 只受 Git 忽略，没有目录权限保证。失败 trace 可包含页面登录密码、API 创建用户密码及 Cookie。编排应强制使用仓库外 0700 私密路径并传递给 Playwright；报告仅列脱敏摘要，原件不得提交。
3. **请求中止判定过宽**：Playwright 用例将任意 API `net::ERR_ABORTED` 从 `errors` 排除，仅写附件。关键财务 POST 中止必须让验收失败；仅确认为导航造成的只读中止可另记。
4. **Biome 格式**：审阅时 `pnpm --dir web exec biome check playwright.config.ts tests/e2e/finance-bill-batch.e2e.ts` 失败，仅测试文件第 53 行 heading 断言应按格式器换行；实施代理已收到，不并行编辑。

### 本轮静态命令

| 命令 | 结果 |
|---|---|
| 五个验收/编排 `.mjs` 逐个 `node --check` | PASS |
| `pnpm --dir web tsc` | PASS，退出码 0 |
| 修改的 Playwright 配置与 E2E `biome check` | FAIL，上述单处格式问题 |
| `git diff --check` | PASS |

本次未重复实施代理正在执行的完整集成与浏览器用例。待其修正并冻结文件后应复查以上三项实质问题和 Biome，最终采信实际运行结果与资源回收日志。

## 三项修复复核（2026-09-27）

本节更新上节「待修项」的状态；审阅时运行修复仍在继续，因此仍不能据静态检查单独宣告全链 PASS。主会话另提供 R14 运行记录：真实页面建账与确认请求均返回 POST 200；本复核未重复启动全量脚本或浏览器。

1. **账单预览竞态：已修正，静态通过。** 用例先等待「页面验收结算账户」可见，通过 UI 点击「刷新快照」，预先注册响应监听并断言返回 `previewToken`、唯一叶子及 `configurationComplete=true`，随后才点击「原子生成 1 张账单」并捕获真实 POST。该顺序满足 `BillCreationWorkbench.createBatch` 的完整配置/令牌守卫；R14 的实际建账和确认 POST 200 进一步证明当前运行场景走到了写路径。没有加入固定时间等待。
2. **trace 私密位置：已修正，静态通过。** 编排在启动前生成本轮证据子目录：父路径取显式 `RONCIN_ACCEPTANCE_ARTIFACT_DIR` 或系统 tmp；经 `realpath` 拒绝仓库内路径；`mkdtemp` 生成随机子目录并 `chmod 0700`；Stage A 把绝对目录注入 Playwright `outputDir`。默认情况下原件位于系统 tmp 下的私有子目录，仓库内不保存 trace。证据原件仍需按报告说明保留为私密材料，只发布脱敏摘要。
3. **关键写入中止：已修正，静态通过。** `requestfailed` 现在仅把 GET `net::ERR_ABORTED` 单列，POST/PUT/DELETE 等写请求中止进入 `errors`，结尾断言为空；所有 HTTP 4xx/5xx 也进入 `errors`。导航造成的 GET 中止保留在证据附件供审阅。

### 复核发现的剩余事项

- **Biome 格式尚未绿。** 当前 `pnpm --dir web exec biome check playwright.config.ts tests/e2e/finance-bill-batch.e2e.ts` 失败位置已从旧第 53 行变为新增的第 261 行 `persistedFeeAfterReload` 换行格式；已通知实施代理，不并行修改其文件。
- **自定义证据父目录的权限副作用。** `createPrivateBrowserArtifactDir` 对显式指定且已存在的父目录执行 `chmod 0700`。这会改变用户提供的目录本身的权限，超出本轮随机子目录的管理范围。建议校验父目录归属/可用性，只为本轮新建子目录设 `0700`；若父目录过宽，报错或要求指定私有父目录，避免修改既有目录。默认系统 tmp 路径未执行该 `chmod`。

### 本次定向检查

- `node --check scripts/run-acceptance-finance-disposable.mjs`：PASS。
- `git diff --check`：PASS。
- Biome：上述第 261 行格式 FAIL。当前实现文件仍在变更，待实施代理处理后需最后再查一次。

## 冻结版本最终复核（2026-09-28）

### 范围与结果

本轮针对 R29 完成后的验收脚本、夹具、一次性编排、指定财务 E2E、根脚本入口及 Playwright 配置做只读复核。没有修改实施文件，没有重复真实 HTTP、浏览器或完整集成套件。已核验 R29 实际日志和附件；最后总结文案已按模式修正，当前无遗留源码审阅问题。A5 最终版本 SHA、统一报告和资源收口由主会话记录，不以本节替代最终报告。

此前 Biome 格式问题、自定义证据父目录 `chmod` 副作用均已处理。编排不再改变已有父目录权限，并在进程入口设置 `umask 077`，确保 Playwright 清空重建 `outputDir` 后仍为私有权限。

### A1–A5 对照

| 标准 | 冻结代码与实际证据 |
|---|---|
| A1 三类 HTTP 实跑 | 默认 Stage A 顺序运行应收 `--apply`、应付 `--apply` 及指定 UI；Stage B 执行外币。R29 日志明确应收、应付和外币成功；R29 采用 `--runtime-only`，完整 data/migration 门禁来自 R2，二者必须分别注明 |
| A2 同订单浏览器闭环 | UI 写费→批次建账及确认→登记/确认收款→核销→预览/生成提成→刷新打开同提成；账单唯一行链接 feeId，核销唯一分配链接 billId/cashflowId，提成唯一订单行及唯一费用明细链接本轮 orderId/feeId。费用/账单/核销125、收入125/成本0/利润125/提成12.50，R29 同链摘要与断言一致 |
| A3 占用正反例 | HTTP 独立样本保留草稿占用、不同幂等键重复建账409、普通修改409且原因正确、删除409且原因正确、取消释放、金额不变、同费用进入新账单；UI 精确已建账行无删除按钮、重复建账警告且不开抽屉。辅助样本未取消主链核销账单 |
| A4 页面与持久化 | UI 金额检查125.00/12.50 CNY、抽屉不含原始ISO时间、刷新后同提成和费用占用保持、流水已核销125且余额0；pageerror、HTTP错误和非GET中止使失败。GET导航中止附在私密摘要中，不能混同关键写入成功。R29 Playwright 1 passed，主会话另记录人工走查未执行 |
| A5 证据与边界 | R29有同链所有ID/单号和输入期望/实际金额，目录内五张节点截图；根与后代权限实测私有。完整门禁、运行SHA、各阶段日期、人工状态和资源清理需在主会话最终报告统一汇总 |

### 汇率契约核对

- EUR→USD 四个业务节点的来源断言为 `WEEKLY`，与 `data.exchangeRateRepo.ResolveRate`/`weeklySnapshotSource` 的当周公司直接汇率行一致；不是旧按类型多表配置，也不回读公共基线。
- 外币输入仍为费用100 EUR @1.10=110 USD，账单100 @1.20=120，发票100 @1.22=122，流水40 @1.25=50；更新同一自然周行后，已落库费用/账单/发票/流水快照重读保持。
- 核销不带独立汇率：40 EUR分配对应账单本币48、流水本币50、汇差2，单头本币50；唯一分配的账单/流水ID和行级48/50/2仍断言。
- USD提成按账单实现收入48、成本0、利润48、比例10%计4.80 USD。旧 `cny*` 传输字段当前是原币记账的恒等快照；`biz.ResolveCommissionCNYRate` 明确返回rate1、source `BASE_CURRENCY`、无settingId、金额4.80。因此本次不是额外USD→CNY折算验证，报告不得宣称证明了该转换。脚本对预览、创建、详情重读保留上述恒等断言，没有修改生产计算逻辑。

### 隔离与敏感材料

- `--stage-b-only` 仅显式提供时跳过 A，未提供时仍依次A/B；`--runtime-only` 仅显式提供时跳过集成包，默认仍完整运行两包且拒绝SKIP。
- 正式迁移、测试和服务均显式使用本轮随机前缀隔离库；业务脚本指向本轮8010，Vite代理同目标。销毁检查本轮创建集合和owner，不重建日常开发库、不终止未知服务。
- 公司切换和公共主数据维护按正式system/company权限办理；HTTP客户端读取轮换Cookie，不放宽业务守卫。
- 证据目录经 `realpath` 拒绝仓库内位置，随机子目录0700且子进程继承umask077；自定义父目录只校验归属，没有修改它的权限。
- R29实测目录 `/tmp/roncin-finance-browser-evidence/roncin-finance-browser-TvOneA` 权限700，所有后代对组/其他用户权限均为0；有01-fee、02-bill、02-billed-guard、03-verification、04-commission五张图。通过运行没有保留失败trace。
- 只输出链路ID、单号、金额和非敏感来源；按模式扫描R29日志，含凭据PostgreSQL URL、Cookie/Authorization头、JSON密码字段均为0。该扫描不是完整秘密证明，失败trace原件仍必须保持私密、不能进Git。
- R29日志有CNY/USD库与角色删除确认和最终全局回收成功；主会话创建的外层管理员/环境文件继续由主会话独立清理。

### 最后一项审阅发现（已解决）

编排 `main` 原先无条件输出「双环境财务全链路验收全部成功」，可能夸大 `--stage-b-only` 或 `--runtime-only` 的执行范围。实施代理已改为优先判断 `--stage-b-only`，仅报告 USD Stage 2；其次判断 `--runtime-only`，报告双环境 HTTP/UI 成功且完整数据库门禁须另行关联；未提供上述参数时保留默认完整成功文案。只改变结束报告，不改变阶段执行顺序、业务断言或清理流程。

追加定向复核：实际读取末尾分支并重算文件 SHA256；`node --check scripts/run-acceptance-finance-disposable.mjs` 与单文件 `git diff --check` 均 PASS。没有重复大测试，R29 运行日志中的旧尾部文案仍作为原始记录保留，最终报告采用明确模式说明。

### 最终定向检查与冻结指纹

- 五个 `.mjs` 逐个 `node --check`：PASS。
- 修改的Playwright配置和E2E `biome check`：PASS（2文件）。
- `pnpm --dir web tsc`：PASS，退出0。
- `git diff --check`：PASS。
- R29日志：HTTP三类与Playwright PASS，CNY/USD资源回收成功；数据库完整门禁采用R2独立证据，不能用R29替代。`check:fast`日志末尾显示WEB/SERVER门禁通过；前端184文件/1089例PASS，1文件/12例既有SKIP，不能称全套零SKIP。

| 文件 | 审阅时SHA256 |
|---|---|
| `acceptance-finance-bill-batch.mjs` | `b567119b8f763f810e4bc5e0b0e7eadba012f393fa4bf1a056a2fb16091a74a2` |
| `acceptance-finance-fixtures.mjs` | `f7cf5c784ed944c2921a9e1352abc9cb98e4ef88cbf0d439c1f3509ae3813707` |
| `acceptance-finance-foreign-currency.mjs` | `ea62bb173ad7b83675b6d7d2a3b5d3764f59afe9873a307d6f09e75e8dfcadba` |
| `acceptance-finance-payable.mjs` | `562c3b7e0b705dfe389768881b23c71eaf793b41b436a79e60fbe315ba382f07` |
| `run-acceptance-finance-disposable.mjs` | `75a6773b81a8f9580b7485222f44055abef82cdad57942edbda1bb95005043ea` |
| `finance-bill-batch.e2e.ts` | `ddadbf93e3dc2b1f1d0e1c63e2e5578420b7778c6204d54ffae6d09213aadbfd` |
| `web/playwright.config.ts` | `f2ffd1ab5695ec0792609e6cda46c44c12cb38911e14bea1c7a641529980a868` |
