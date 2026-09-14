# implement.md — 主数据多组织治理与存储重构执行清单

前置：`prd.md`、`design.md` 已评审。每阶段独立提交（Conventional Commits），阶段内先实现后定向验证。

## 阶段一：存储地基 + 契约

1. Ent schema 改造：master_data_items / airlines / shipping_lines / shipping_line_container_prefixes / billing_units 去 `organization_id`；kind 枚举 `service_type` → `charge_category`；ports / airports / exchange_rate_settings / fee_settings `organization_id` 可空；fee_settings `charge_category_id` 必填更名；partial unique index 逐表落位（design.md §1）。
2. 生成迁移并按 design.md §8 顺序处理数据（含航司/船司 Fail-fast 冲突报告，报告归档任务目录；exchange_rate_settings 周汇率/ar-ap 结构重建与 `order_fees`/`finance_bill_lines` 快照字段迁移同在此步，见 §8 第 4–5 步）。
3. `internal/data/baseline_query.go`：B 型两种形态谓词工厂；改造 port/airport/exchange_rate/fee_setting 仓储调用；删除 airline/shippingline 让位分支与 `writeIndustryLocalCodePrecedence` 旧路径。
4. 主数据读路径移除 `resolveHeadquartersOrganizationID`；`RequireGlobalMasterDataWrite` / `RequireBaselineWrite` 拦截器挂接 A 型与 B 型 NULL 行写路径。
5. biz/data/service 全量标识符更名（design.md §6 清单）；种子与 `sync:all` 改 A 型直写。
6. proto `OrganizationKind` 枚举 + `auth/me` 暴露；`make -C server api`、`pnpm run generate:web-client`、`pnpm run generate:permission-keys`（如有权限码变动）。
7. 验证：`go -C server test ./...`（迁移相关集成测试）+ `go -C server vet`；新增测试见 design.md §9。

## 阶段二：汇率实战化重构（周汇率 + 买卖点差 + 中国银行一键同步）

1. Schema & Data 扩展：`exchange_rate_settings` 扩展 `ar_rate`、`ap_rate` 与周区间字段；重构 `ResolveRate` 支持根据费用收支类型（AR/AP）匹配对应点差率；四级解析链容灾（当周 org 行 `WEEKLY` → 回溯最近历史周 `INHERITED_LAST_WEEK` 黄色 Tag → NULL 基线兜底 `SYSTEM`/`DERIVED` → 现场手工 `MANUAL`，不阻断单据保存）；同周幂等 Upsert 与跨周预设参数；`ResolveBaseRate` 退役并将 `finance_commission` 等调用方切换为原币记账/本组织解析；费用录入打快照，跨组织协同往来原币对账。
2. 中国银行（BOC）牌价自动抓取服务（`FetchExchangeRates` 按组织本币路由）：CNY 本币走新浪中行专线 JSON 主源（带 Referer、已归一化）+ 中行官方牌价页 HTML 兜底（/100）；非 CNY 本币走国际直盘（Ask/Bid）首选 + 中行交叉盘换算备选；来源在预览中明示；自动计算当周/预设下周生效区间，提供结构化预览数据；漏配督办通知（周一 10:00 定时检测 + `notification_delivery` 财务待办）。
3. 前端 `/finance/exchange-rates`：应收/应付双列列表与录入，「从中国银行同步周汇率」/「一键同步周汇率」按钮（按本币动态显示）及微调确认弹窗（本周/预设下周切换、来源明示、财务终审），开放分公司本币汇率维护与一键同步入口；抓取失败显式引导手工录入。
4. 验证：汇率点差与周区间解析定向测试 + 容灾链/幂等 Upsert/退役回归测试 + 牌价抓取解析单元测试（含 /100 与已归一化两种口径）+ 前端 vitest 定向 + `pnpm --dir web tsc`。

## 阶段三：费用科目两级

1. FeeSetting org 行创建/更新（分公司）、必挂 charge_category；引用校验放宽（A 型全局 + 本组织税务名称）。
2. 订单费用选择器接 `BaselineShadowedByLocal`（同码本地优先）；费用快照字段核对。
3. `/finance/fee-settings` 三面板按 C/A/B 各自口径放开与收敛。
4. 验证：费用选择器穿透测试、引用校验测试、前端定向。

## 阶段四：前端收敛收尾

1. `useAccess` 组织身份组合判定；`/master-data` 七页签横幅与按钮收敛（A 型只读、B 型保留本地新增、基线行禁编辑）。
2. 文案统一「费用大类」；总部视角回归。
3. 终验：design.md §9 全部测试 + `go -C server vet` + 前端 tsc；按风险决定是否 `pnpm run check:web`。验收标准逐条对照 `prd.md` §6。

## 完成动作

- 更新 `AGENTS.md` 或 `.trellis/spec` 沉淀「存储三型 + 维护权」约定（trellis-update-spec）。
- 阶段提交信息建议：`refactor(masterdata): 主数据存储三型改造（A/B/C 型与部分唯一索引）`、`feat(finance): 汇率周汇率双轨点差与组织自治（含容灾链）`、`feat(finance): 外汇牌价一键同步（新浪/中行/国际直盘多源）`、`feat(finance): 费用科目总部大类+本地明细两级结构`、`feat(web): 主数据多组织视角收敛`。
