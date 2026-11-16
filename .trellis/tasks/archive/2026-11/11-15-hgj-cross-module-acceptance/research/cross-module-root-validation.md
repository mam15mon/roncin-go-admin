# 四类订单横切、完整CSV与父最终验收

## 最终结论

本组实现、两侧独立全scope复核、正式迁移后的真实PG专项、最终全栈门禁、
生产构建及一次性库真实四类业务/财务/CSV/权限整轮均通过。无未修确认缺陷。
产品及生成物冻结后完成最终真实验收，未以准备、mock、SKIP或前组门禁替代结果。

## 实现与边界

- `ExportOrders`独立无分页，显式四类身份，复用列表filter/sort/currentCompany/
  type read与授权裁剪，完整结果及海空/箱型投影同只读RR快照。普通分页仍200。
- 箱型code/name为真实只读投影，前端不再另请求会变化字典；官方onLoad采纳后
  才导出。筛选/排序/公司/用户/kind及箱号、附件读取权变化使旧下载失效。
- 四类注册、工作台和通知共同消费features/orders/navigation公开入口；本人待办
  按真实协作及类型read/classified read交集，system通知页仍示例首页和拒绝真实订单。
- 中文长度按既有字符合同贯通Order、航空notes及海运船名航次当前/不可变版本。
  列类型/尺寸/索引/CHECK不变，无空迁移、新权限或DB列。
- 航空取消危险品更新显式空串/PG0，创建仍省略；省略保持与显式清空分别实测。
  List/Export/财务详情严格success=true，省略/false夹旧data不能采纳。
- 客户业务/客服按历史明确规则只读带入且新建必填，当前订单随换客重挂，创建时
  提成快照不重拍。更正F04/旧spec“仅默认”措辞，没有改变已有人员取数业务。

## 分层与独立验证

- `cross-module-server-validation.md`：真实205条完整导出、四类建票、同RR
  外连接改客户/箱型名称、普通200/201、跨公司/写txCtx拒绝；四类台账主单、
  权威经营日、本人工作台近期/费用/异常交集均0SKIP。中文与DG持久化通过；
  最后箱型RR和SE/SI中文船名不可变快照整组5.751秒/0SKIP通过。
- 正规Ent/API/OpenAPI/web生成1139文件重复SHA变化0，迁移schema相对HEAD
  不变。独立server本人Go定向/vet/buf/gofmt/diff通过，无产品自修；明确沿用
  实现已通过PG/生成，不无因重跑，见`cross-module-server-review.md`。
- Web独立checker修箱型快照投影、子资源撤权旧下载、严格成功封套；最终
  6文件81项PASS、tsc、29文件Biome、577产品AST、antd零问题，详见web-review。
  收紧后旧成功mock缺true的首轮失败已如实修正，不放宽产品。

## 最终门禁与构建

- `pnpm run build`实际exit0：Web27.12秒+Go生产入口，日志
  `/tmp/hgj-cross-build.log`。新共享导航进入四类注册/通知生产依赖，导出API
  与客户端进入实际生产入口，是本次最终build和同域验证的具体风险依据。
- 首轮`check:fast`因旧通知测试把已开放AI路径当未开放负例而失败，取消未完成
  web门禁；仅将该负例改为未开放LAND、通知定向通过后，重新完整执行。
- 最终`pnpm run check:fast`实际exit0，总246.2秒（WEB246.2/SERVER117.8），
  527文件2855项通过；既有1suite/11tests跳过不计通过。日志
  `/tmp/hgj-cross-check-fast-final.log`。既有测试环境TimeoutNaN、localhost3000
  拒连/重置等stderr保留，未关闭规则或忽略失败。
- 最终全量缓存显示新页面测试单文件22.69秒，按AGENTS长尾约束将原6例等价
  拆为基础/权限两suite各3例，共享`.fixtures.tsx`被生产扫描排除。定向
  两文件6项PASS（5.40/13.14秒）、tsc/Biome/diff通过；只调整测试调度，
  产品、生成和断言不变，因此未无因重复build/全量门禁/PG。

## 真实整轮与主会话独立检查

最终命令`RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-hgj-browser-acceptance/run.mjs`
实际exit0，Playwright `1 passed (1.3m)`、0SKIP，最终证据
`/tmp/roncin-hgj-browser-uA0yfn`。细节及前九轮夹具/定位失败和完整清理见
`cross-module-browser-finance-validation.md`，未将失败轮记成功。

- 四类各真实应收100/应付40，业务与适用进口服务→完结/审核→建账→真实收付
  确认/核销→应收登记/应付免票→利润60的10%提成6确认/支付，8账单completed。
  经营日11月16日与资金交易日11月17日分别验证，账龄核销前1、后0；台账号可读。
- AE/AI危险字段真实HTTP设→省略保持→显式空清除，版本1→2→3→4精确重读。
  四个单类型只读用户、两公司、system示例通知/拒真实订单/零发送全部通过。
- 四类各205行/54列CSV，生产同域SI另205行/54列；SI只读未筛选206行/52列
  正确包含205夹具+1业务票，箱号/附件列缺席。生产SPA与新dist逐字相同、
  JS MIME及未知assets404，并同次API逐行对照实际下载。
- Root独立读取最终facts及Python标准csv解析六文件，核对BOM/CRLF、完整数量、
  四类升序全序列与facts、金额15362.50、毛重12.345/体积1.234、公式防护、
  中文引号多行、授权列。生产文件按当前默认倒序，同次API一致由实际E2E证明；
  只读文件未筛选，不能误用205/升序的测试假设。最终独立脚本exit0，记录
  `root-csv-audit.json`；曾将统一假设误套生产/只读文件的自检已按真实场景纠正。
- Root实际view_image生产SI与AI完整筛选截图，列表205统计、真实海空字段、
  完整导出入口和对应采纳筛选可见。原始凭据文件单独私有保存，未读取输出或提交。
- 最终harness清理及独立psql/bind复核：本前缀数据库0、角色0，8010/9010/8002
  均空闲；未知8001未操作，开发业务未清。证明见最终目录的
  `hgj-independent-cleanup-proof.json`与`hgj-independent-file-audit.json`。

## 父验收与限制

F01–F45逐项结果见`field-delivery-audit.md`，AC1/AC5/V19研究范围复核见
`final-research-audit.md`，父V1–V20及AC1–AC8最终对应见父交付记录。前五组
正式迁移/双模式/并发/实际PDF沿各归档实证，本组没有无因重跑全部物理输出。

外站后台源码/写规则/原PDF仍未取得；其他五类只调研、无样本部分明确未知。
物理打印机、真实员工通知、航司/海关/税务通道、批量导入、真正多航段和复杂
空运拆拼改配不在批准本期范围，没有以登记或本地打印入口冒充外部办理。
