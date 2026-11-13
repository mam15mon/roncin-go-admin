# 财务扩围初步盘点

## 深度与限制

本轮仅定位路由、文件与少量关键条件，不是全量功能验证；未运行产品测试。用户已确认六模块及四报表纳入本期，差距需逐子任务深入。

| 范围 | 已见证据 | 下一步 |
| --- | --- | --- |
| 费用、账单、发票、收付、核销、对冲、提成 | `web/config/routes.ts:212` 起已有对应财务路由；各 pages/finance 子目录及 biz/finance_* 存在 | 对照参考逐项核实入口、数据链路及验收缺口，复用已有能力 |
| 四类财务报表 | 当前 routes.ts 未发现利润分析、利润报表、实际收付款报表、账龄入口；pages、biz、api 的 report/aging/profit 文件名扫描未命中 | 初步按缺少独立页面规划；不能据此断言没有可复用统计查询 |
| 提成 | `server/internal/biz/finance_commission_rules.go:179`、`:246` 校验已实现利润/收入计算依据，比例范围为正且不大于 100 | 对照参考的利润区间、阶梯与含税口径；明确用户要新增规则还是改变计提基础，不能默改现有机制 |
| 账单与发票 | 父 PRD 已记录免确认、分次发票登记与现有契约差异 | 账单、发票、核销联合确定契约，按子任务分工实施 |

优先加载规范：finance-bill-currency、exchange-rate-single-rate、finance-commission-lock、commission-scheme-assignment、order-fee-correction、finance-ledger-snapshots、finance-current-workspace。各项需按子任务细读。
