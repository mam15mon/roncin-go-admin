# 信用额度本币一致性实施

## 行为与写入边界

- 恢复原产品约定：额度按所属公司本币，结算币种独立；不做查询实时换汇、不重标已有外币金额。
- `ErrPartnerCreditCurrencyMismatch` 明确提示到客商结算规则重新维护；沿用该模块已有纯 Biz reason 错误定义，无 Proto 字段变化。
- 独立结算规则 Create/Update 在原事务内读取目标公司本币并校验；内嵌档案 Create/Update/Import 共用同一校验，失败事务回滚。
- 信用聚合检查所有有效正额度的原币（不只检查最终最大值），同批查询只读取一次本币，异常配置显式失败；管理 List 原样返回允许人工修正。
- 开发种子仅新建结算规则时带入公司 BaseCurrency，不覆盖旧数据、不运行种子。

## 前端

- `creditLimit.ts` 统一两个维护入口的额度金额→minor 与币种映射，空额度不自动补零。
- 主档信用额度回填由误把 minor 当元修正为 /100；非 CNY 公司及外币结算均按当前公司本币提交。
- 已有错误币种额度显示原币原金额，清空新额度输入并要求显式重录（0 表示不设额度）；无自动金额转换。
- 同根状态错误：二级规则编辑恢复省略 isActive=false，去除字段级初始 true；主档内嵌保存保留已有规则 isActive，新建仍 true。

## 已执行验证

- 专用临时 PostgreSQL、正式迁移及独立 schema：`python3 /tmp/roncin-system-audit-pg/run-integration.py go -C server test -p 32 ./internal/data -run 'TestPartnerCredit(Currency|Aggregation)Postgres$' -count=1` 通过。覆盖 CNY/USD 本币、合法外币结算、独立新建/更新拒绝外币、内嵌档案新建/更新/导入新建/导入更新的拒绝回滚、异常小额度不能被忽略、管理原配置仍能读取、正额度缺币种明确报错；原核销余额聚合回归通过。
- `go -C server test -p 32 ./cmd/sync-dev ./internal/biz ./internal/data -run 'TestPartner(SettlementRule|CreditExceeded)' -count=1` 通过（data 本模式无匹配测试，真实库证据见上）。
- 主档保存与 helper 定向 13 项通过；原主档详情 + 保存 + helper 21 项通过。
- 二级真实结算表单覆盖本币 CNY/USD、异常旧额度显式重录和省略 isActive=false，4 项通过；与主档保存/helper 合并定向共 17 项通过。主档最后增加停用规则保存断言后单文件 9 项再次通过。
- 修改文件 Biome 与 `pnpm --dir web tsc` 通过；最终总门禁由主协调者执行。

## 影响文件

`server/internal/biz/partner_settlement_rule.go`；`server/internal/data/partner_settlement_rule.go`、`partner.go`、`finance_bill_credit.go`；`server/cmd/sync-dev/main.go`；`partner_credit_aggregation_integration_test.go`、新 `partner_credit_currency_integration_test.go`；前端 `partner-detail.tsx`、`SettlementSection.tsx`、`secondary/SettlementRulesPanel.tsx`、新 `creditLimit.ts` 及相关回归。

## 边界

未运行开发库同步或自动修复配置。已有非法币种配置必须由业务人员明确重新维护本币额度；配置错误会阻止消费它的信用查询，避免静默误判。未执行真实浏览器端手工验收。

## 独立审阅增量修复：规则查询失败保护

- 审阅发现结算规则查询失败曾被当作成功无规则，可能普通保存启用旧停用规则并清空信用额度。
- 主档与结算字段分别首次回填；失败或未请求不完成结算初始化，重试成功仅回填结算字段，保留名称等普通输入。加载/失败不挂可编辑结算字段。
- 编辑的嵌套规则保存要求查询成功、非重查中且结算字段已初始化；undefined 和显式 success:false 响应均失败。失败 Alert 提供 React Query 原生 refetch 重新加载入口。
- 新单例覆盖首次 reject→undefined→成功取得停用规则及非零额度→缓存后台失败→重试→仅改名称保存，额度和停用状态均保留，失败期间零写请求且输入/dirty 保留。
- `pnpm --dir web exec vitest run src/pages/partners/partner-detail-save.test.tsx src/pages/partners/partner-detail-enabled.test.tsx src/pages/partners/partner-detail.test.tsx`：3 文件 23 项通过。修改文件 Biome 通过，类型检查由主协调者与实施者同步复核。
