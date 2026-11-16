# 四类最终真实财务与CSV验收

## 当前状态

2026-11-16：最终第十轮真实整轮PASS，执行会话`73125`正常exit0。正式迁移、四类HTTP财务/危险品/完整导出/权限/公司/通知示例、实际浏览器文件及生产同域验证全部通过；Playwright为`1 passed (1.3m)`、0 SKIP。最后本次随机库、角色、追踪进程组和8010/9010/8002端口完整回收，本代理没有修改产品、生成物、spec或提交代码。

最终私有证据目录：`/tmp/roncin-hgj-browser-uA0yfn`。主会话在两实现及两独立检查冻结、本组新build成功后明确放行；本轮Go服务嵌入该新构建。此前九轮失败只涉及验收夹具或浏览器定位，均单独完整清理；下文保留错误与修正记录，不把失败轮称为通过。

## 最终真实数值与业务闭环

| 类别 | 本票 | 应收账单/金额 | 应付账单/金额 | 提成 | 最终状态 |
| --- | --- | --- | --- | --- | --- |
| SE | SE2026111600001 | BI2026111600001 / 100 | BI2026111600002 / 40 | 6 | 两账单completed；提成PAID(3) |
| SI | SI2026111600002 | BI2026111600003 / 100 | BI2026111600004 / 40 | 6 | 两账单completed；提成PAID(3) |
| AE | AE2026111600003 | BI2026111600005 / 100 | BI2026111600006 / 40 | 6 | 两账单completed；提成PAID(3) |
| AI | AI2026111600004 | BI2026111600007 / 100 | BI2026111600008 / 40 | 6 | 两账单completed；提成PAID(3) |

四类均用真实海/空DIRECT关系创建；公共COLLECT、货值15362.5/CNY、保费125.67/CNY、工厂/合约、接单/货好两时刻及四项独立备注保存重读。每类真实应收100/应付40零税，经过本人未建账与异常待办、解决异常、进口承接EXCHANGE和DELIVERY并分别完成（其余三环节不承接）、业务完结、费用版本/指纹与航空正文指纹审核锁、逐组真实建账、确认收款与付款、全额核销后才登记应收发票/应付适用免票，最后已实现利润60按10%生成提成6、确认和实际标记支付。

经营日为`2026-11-16`，出口使用中国时区ETD、进口使用中国时区ETA；各自费用exchangeRateDate与利润60/未税利润60匹配该日。真实资金交易日和核销日另取`2026-11-17`；现金流报表精确对应本票两笔已确认流水ID，没有将经营日代作收付日。应收应付账龄各在核销前总数1，结清后总数0；按真实海空主单号的双方向费用台账均能查到本票。旧正版本请求均409、notes不变且version不增。

AE/AI另外分别真实HTTP设置UN3481/CLASS9/截申报/PGII，省略四字段更新普通文本后保持，随后显式三个空字符串及PG0清空。每次成功写入和重读版本严格一致，阶段版本各为`1→2→3→4`；无货类推断或后续版本借用。

四个只读身份分别只能读本类详情/205行完整导出，另三类详情/导出均403，写入本类403，箱/附件字段按权限裁剪；本人工作台返回类型始终在对应权限交集。公司B真实客户和订单按自己的客商档案业务/客服配置建立，A不能读取B票且导出不混公司，A会话显式请求B财务报表403。系统治理会话通知目录的canSelectOrder语义false，预览仅首页示例、真实orderId拒绝；没有调用任何通知发送接口。

## 最终实际文件与生产同域

文件均在`/tmp/roncin-hgj-browser-uA0yfn/playwright/hgj-cross-module.e2e.ts-四类完整筛选CSV真实205行、采纳排序与授权裁列/`。

| 实际文件 | 数据行 | 列 | 首单→末单 | 说明 |
| --- | --- | --- | --- | --- |
| se-complete-205.csv | 205 | 54 | SE2026111600005→SE2026111600209 | 客户业务号筛选、订单号升序 |
| si-complete-205.csv | 205 | 54 | SI2026111600210→SI2026111600414 | 同上 |
| ae-complete-205.csv | 205 | 54 | AE2026111600415→AE2026111600619 | 同上 |
| ai-complete-205.csv | 205 | 54 | AI2026111600620→AI2026111600824 | 同上 |
| si-production-same-origin-complete-205.csv | 205 | 54 | SI2026111600414→SI2026111600210 | 8010新生产SPA、当前默认顺序，与同次API逐行一致 |
| si-read-only-authorized-columns.csv | 206 | 52 | SI2026111600414→SI2026111600002 | 该只读会话未筛选，包含205夹具及1条业务样票；无箱号/附件列 |

四类开发态文件真实解析后核对完整205条及官方采纳的筛选/排序；下载请求只有专用`/api/v1/orders/export`、不带page/pageSize，没有分页拼接。文件包含UTF-8 BOM、CRLF、中文/引号/多行、公式防护`'=SUM(1,2)`；货值为`15362.50`（两位、不带千分位），毛重`12.345`和体积`1.234`保持精度；无内部ID/版本/UUID列。创建时间、ETD/ETA与真实wire经独立Intl上海时区转换逐项相等，并反证UTC截取值；ETD/ETA中国00:30明确跨UTC自然日。orderDate现合同是RFC3339，CSV业务日期为对应本地日；真实纯日期transactionDate保持11月17日。

生产同域实际访问`http://127.0.0.1:8010/orders/sea-import`，200 HTML与本轮`server/internal/webassets/dist/index.html`精确相同、没有Vite客户端；实际模块JS为200及JavaScript MIME，未知`/assets/hgj-acceptance-intentionally-missing.js`为404。随后同域UI筛选、一次专用导出响应和真实205行文件逐行/关键字对应。五张匹配文件截图分别为`{se,si,ae,ai}-complete-filter.png`及`si-production-same-origin-complete-filter.png`，没有用旧构建代替本轮生产入口。

独立Python标准csv解析（独立于E2E解析器）再次核对六份文件，真实行数分别205×5及206、BOM/金额/只读裁列，SHA256及逐类财务摘要保存在`/tmp/roncin-hgj-browser-uA0yfn/hgj-independent-file-audit.json`。原始完整业务事实在`hgj-cross-module-facts.json`；身份凭据另存私有文件，不复制到本报告或Git。

## 最终清理与门禁边界

finally成功删除本次库`roncin_acc_hgj_browser_1794800581396_64a0465b`及同名角色，并核对不存在；本次追踪集合清空、子进程组和8010/9010/8002端口全部释放。随后独立只读psql再次核对本验收前缀所有库/角色残留均0，逐端口bind验证均空闲，记录`/tmp/roncin-hgj-browser-uA0yfn/hgj-independent-cleanup-proof.json`（03:45:28Z）；8001未知进程始终未操作。开发库只用管理连接创建/核验本轮隔离库，未迁移、清空或重置开发数据，未操作外站。

本代理实际执行Node语法、E2E Biome、Playwright场景发现、最终真实完整编排、独立文件/残留核验及`git diff --check`均通过。主会话告知本组新build exit0、最终check:fast重跑exit0（527文件/2855测试，既有11 SKIP单独保留，不计通过）；这些整仓命令由主会话执行，不用前置组门禁替本组。

本轮不重跑已归档航空/HBL全部PDF/长文/字体矩阵，不新增真实外站写入、员工通知、复杂拆拼改配、批量导入或另五类业务；对应批准前置证据/后续范围保持。当前真实整轮没有未解决产品缺陷；早期失败轨迹如下。

## 失败轮记录（均已完整清理）

首轮结果：exit1。SE真实创建、公共字段重读、费用、本人待办与异常完成后，夹具用`expectedVersion:0`错误期待409；当前合同正确先拒绝非法零版本为400。已修夹具为先用当前正版本成功更新，再提交刚过期的正版本验证409并重读确认未写入。没有修改产品或放宽规则。首轮本次库/角色删除及追踪进程/端口清理全部成功，未记业务PASS，准备重跑。

第二轮`/tmp/roncin-hgj-browser-EBoUED`：exit1，修正旧版本夹具后，海运普通更新未带现行完整表单要求的主单信息，被正确拒绝400。现夹具带回既有`seaMasterBill.masterNo`，不改主单，成功更新和旧版拒绝都通过同一合法输入边界；再重读核对notes和version均未被失败请求改变。该轮本次库/角色、追踪进程和端口清理成功；仍不是产品失败或业务PASS。

第三轮`/tmp/roncin-hgj-browser-woIdHW`：exit1，真实正版本409及SE业务完结已达，审核请求夹具漏费用版本/指纹被正确拒绝400。改为先读`lock-state`核对当前Order版本与可审核条件，再提交其真实费用版本、费用指纹及航空单证指纹。主会话同步指出航空不能含海运shipmentType，已将该字段仅置于海运夹具分支。均为现行合同下的夹具修正，没有产品修改。第三轮本次库/角色及追踪进程/端口完整回收。

第四轮`/tmp/roncin-hgj-browser-mMWIcH`：exit1，已达SE真实审核、应收建账/收款确认/先核销后发票登记及账单completed；结清账龄总数零被ProtoJSON省略，脚本`Number(undefined)`误得NaN。按非optional标量零值合同改为`Number(total ?? 0)`，未更改产品或报告口径；HTTP客户端同步严格要求`success === true`。该轮全部资源清理成功。

第五轮`/tmp/roncin-hgj-browser-Lt8LEf`：exit1，SE应收应付双链、登记/免票、核销、提成预览6/创建/确认/支付及利润60均已达。现金流报表夹具误把`ourAccount`当作前缀匹配，当前实际是精确相等，因此无匹配行。改为两笔共用本票完整账户串并传同一完整值筛选，同时比对报表两笔ID正是已确认的资金事实。已逐项复核当前`settlement.proto`和已跑财务helper的组token/费用集合/资金及账单版本/票据逐费用分配/提成源、支付预览token和状态enum，保持现合同。第五轮全部资源清理成功；无产品改动。

第六轮`/tmp/roncin-hgj-browser-qMUGkT`：exit1，四类真实财务全链、AE/AI三段危险品HTTP、四类各205完整API导出及单类角色读取/拒绝矩阵均已达。第二公司对照夹具创建客户时漏配置客商档案业务/客服归属，正确触发开单前置拒绝。现同主公司一样明确配置公司B内的业务/客服人员，不借订单人员载荷绕过客户档案前置。该轮全部资源清理成功；未进入浏览器，不记整轮PASS。

第七轮`/tmp/roncin-hgj-browser-5sojhH`：exit1，全部HTTP经营、财务、完整导出、单类及跨公司矩阵均已达；通知目录`canSelectOrder`非optional bool的false同样按ProtoJSON省略。断言改按该确定合同`canSelectOrder ?? false`核对，仍拒绝true；不增加权限或业务读取，发送从未调用。该轮全部资源完整回收。

第八轮`/tmp/roncin-hgj-browser-ljMAwe`：HTTP全链通过，facts已完整落盘；Playwright实际进入SE列表后，夹具未点击默认折叠的「展开筛选」，找不到未渲染的复合单号控件而超时。实际页面截图确认列表/完整导出入口和四类菜单正常；现先等筛选切换按钮就绪，若处于折叠则经正常UI展开再筛选。没有改产品。浏览器失败轮整体exit1，库/角色/进程和端口全部清理成功，保留截图/trace而不记整轮PASS。

第九轮`/tmp/roncin-hgj-browser-pWVxoI`：HTTP全链再次通过，浏览器正确展开筛选和单号下拉；Ant Select虚拟化a11y列表当前只含相邻选项，`getByRole(option)`不能定位实际可见的第四项。截图与snapshot确认可见「客户业务号」真实存在；夹具改为在当前可见`.ant-select-dropdown`内按实际文字点击，不改产品的虚拟化行为或请求。该轮整体exit1，全部资源完整回收，原始截图/trace留私有目录。

## 当前派工与恢复锚点

- 当前代理虽沿用`sea_import_browser`名称，实际负责本组最终四类SE/SI/AE/AI财务、CSV与横切验收；不是重新实施SI或重跑已归档航空PDF。航空PDF最终真实验收已通过并归档，对应提交`b3b5b407`、`65370c82`及日志`2dd58fe2`。
- Active task为`.trellis/tasks/11-15-hgj-cross-module-acceptance`。恢复时先读本报告和下列当前脚本，不从旧任务或旧SE财务夹具推断当前状态。
- 本代理独占`acceptance-hgj-{client,cross-module}.mjs`、`hgj-cross-module.e2e.ts`、私有最薄编排与本报告；不改产品、生成物或spec，不提交、不派子代理。server及所有生成物由`cross_module_server`负责，非生成web由`cross_module_web`负责，主会话负责spec、最终门禁与提交。
- 仅完成准备即可释放并发槽，后续收到主会话明确放行后再启动长轮；两实现与独立复核完成前不得将场景发现当成真实验收。
- 第二次准备派工补充：server及独立检查已经冻结完成，web独立检查正在收口；主会话会执行本组最终`pnpm run build`和门禁。只有新build完成并明确放行后才可运行当前编排，启动的`go run`应嵌入本轮新静态资源，不能先拿旧构建验收。

## 产物与执行入口

- `scripts/acceptance-hgj-client.mjs`：仅内存Cookie的真实HTTP客户端、四类身份及合法航空号码工具。
- `scripts/acceptance-hgj-cross-module.mjs`：只允许`RONCIN_HGJ_DISPOSABLE=1`启动；四类各真实经营和财务链、工作台/报表/权限矩阵、每类205条完整导出夹具；结果落仓库外私有JSON。
- `web/tests/e2e/hgj-cross-module.e2e.ts`：消费真实夹具，四类各实际下载205行CSV并解析BOM/CRLF/引号换行/公式防护/金额/数量/时区/排序，另仅SI只读身份验证授权裁列。
- 浏览器场景及额外只读context均显式使用`Asia/Shanghai`；CSV逐项与真实wire的创建时间、ETD、ETA经独立Intl时区转换的分钟值比对，并断言不同于UTC截取。ETD/ETA及业务日期夹具固定中国00:30，明确跨UTC自然日。注意现行`orderDate`契约为RFC3339时刻，CSV按本地日期显示，不能称为纯日期字段；真正纯日期如现金流`transactionDate`在HTTP链另断言原日。创建时间按服务端真实生成时刻校验8小时时差，不伪造历史创建时间。
- 私有编排`/tmp/roncin-hgj-browser-acceptance/run.mjs`：复用已通过多轮验收的随机库/角色、正式SQL、随机管理员、真实服务/Vite、finally/信号进程组回收。仅本次追踪资源，8001未知进程不动；8010/9010/8002先查空闲。通知/OSS关闭，预览回调仅本地合法URL，不发送通知。
- AE/AI样票在未锁定阶段，使用刚成功写入并重读的精确Order版本，依次真实HTTP设置UN/CLASS/截申报/PG、只更新普通文本且省略四字段验证保留、显式三个空字符串与PG0清空并重读；不发送货类、不按货类推断清空。真实成功时把各版本与三阶段结果纳入私有facts，补足F24/F25；该准备项已在最终第十轮实际通过。
- E2E在四类开发态CSV后复用同一管理员会话访问后端8010的`/orders/sea-import`：实际HTML必须等于本轮`server/internal/webassets/dist/index.html`、无Vite客户端；模块JS返回正确MIME、未知assets返回404。随后生产同域UI筛选SI，单次专用Export响应与真实CSV均205行，逐行号码/筛选关键字一致；记录独立文件和截图。没有重复前组PDF/字体矩阵，也未先使用旧build。

实际整轮命令（最终第十轮exit0）：

```bash
RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-hgj-browser-acceptance/run.mjs
```

## 验收范围映射

四类每类：真实单证关系、公共货值/保费/工厂/合约/COLLECT/不同接单货好时刻/独立备注保存重读；应收100/应付40零税→工作台本人未建账/异常→解决异常→进口2/5适用服务完成→业务完结/真实审核锁→应收应付各建账→真实收付款确认和全额核销→应收登记发票/应付适用免票→提成按不含税已实现利润60的10%=6生成/确认/支付；利润按出口ETD/进口ETA中国自然日，现金流按不同交易日，账龄核对当前未清/结清排除，主单台账号搜索覆盖四类。

四类各205匹配订单单次专用ExportOrders，API完整行数/稳定排序/当前公司类型；真实CSV与已采纳筛选排序相同、无page/pageSize。两公司+四个仅单类型只读用户验证读取/写入拒绝/工作台本人类型交集与子资源字段裁剪。

通知HTTP只验证系统管理员示例首页、canSelectOrder=false及真实orderId拒绝，不调用发送。四类正式通知深链由server无网络render/prepare专项，不能通过系统预览页越权读取真实业务。

对应父AC3/AC6/AC8、V11/V12/V13/V16/V20，公共字段补足F05–F19实际重读。前置AWB/HBL、双模式/恢复/锁/并发专项沿已提交证据，不重复全部PDF矩阵；未覆盖/失败需最终明确记录，不以脚本存在代替实证。

## 准备检查

- `node --check scripts/acceptance-hgj-client.mjs`、`node --check scripts/acceptance-hgj-cross-module.mjs`、`node --check /tmp/roncin-hgj-browser-acceptance/run.mjs`：全部通过。
- `pnpm --dir web exec biome check tests/e2e/hgj-cross-module.e2e.ts`：通过。
- `pnpm --dir web exec playwright test tests/e2e/hgj-cross-module.e2e.ts --list`：exit 0，发现1个包含四类和只读授权子步骤的场景；没有启动浏览器或经营请求。
- 首次准备运行`pnpm --dir web tsc`曾失败，范围为其他实现代理当时正在维护的`OrderListTemplate.export.test.tsx`的actionRef空值类型，以及`form-adapter-dangerous-clear.test.ts`夹具缺少`paymentTerm`；已通知主会话，未越权修改。该次验收e2e没有类型报错；后续以主会话整体验证为准，本次短准备仅重跑语法和Biome。
- 第二次短准备补充后：`node --check scripts/acceptance-hgj-cross-module.mjs`、E2E文件Biome检查和`git diff --check`通过；未启动长轮。
- 以上准备检查是放行前的历史状态；最终真实业务、文件和清理结果以本报告开头的第十轮PASS及私有证据为准。
