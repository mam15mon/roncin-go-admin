# 提成 CNY 快照与归属日期设计

## 分层职责

- `server/api`：定义请求、响应和 HTTP 参数，不承载计算规则。
- `internal/service`：解析日期筛选和 DTO/DO 转换。
- `internal/biz`：定义 CNY 快照领域对象、公式、筛选校验和共享事务编排。
- `internal/data`：读取生成上下文、持久化快照、构造筛选谓词和转换 Ent。
- Ent Schema：定义最终非空结构和索引；不添加历史兼容默认值。

## 领域模型

新增 `CommissionCNYSnapshot`，包含归属日期、汇率、汇率来源、汇率日期、来源设置 ID
和原始 CNY 提成金额。`FinanceCommission` 持有这些持久化字段，并额外返回动态
`CNYAdjustmentAmount`、`CNYEffectiveCommissionAmount`。

`CommissionFilter` 增加 `CommissionDateFrom`、`CommissionDateTo`。日期校验复用
现有财务日期规则，不在 data 层重复业务校验。

## CNY 解析与计算

创建与预览共用 biz 层纯计算函数：

```text
resolved = ExchangeRate.Resolve(
  ctx, organization, WRITE_OFF, RECEIVABLE, "CNY",
  { WRITE_OFF_TIME: verification.exchange_rate_date },
)

base_currency == CNY:
  rate = 1
  source = BASE_CURRENCY
  setting_id = nil

base_currency != CNY:
  rate = round(1 / resolved.rate, 8)
  source = DERIVED
  setting_id = resolved.setting_id

cny_commission_amount = round(commission_amount * rate, 8)
cny_adjustment_amount = round(adjustment_amount * rate, 8)
cny_effective_amount = cny_commission_amount + cny_adjustment_amount
```

所有输入必须为正且日期有效；禁止 float64、补差和零值回退。

## 创建事务

```text
CommissionUsecase.Create
  ├─ 事务外：参数校验、幂等预查、编号生成
  ├─ Transactor.WithinTransaction(ctx)
  │    ├─ CommissionRepo.GetGenerationContext(txCtx)
  │    │    └─ 读取核销日期、汇率日期和本位币上下文
  │    ├─ ExchangeRateUsecase.Resolve(txCtx)
  │    │    └─ 汇率设置查询自动 ForShare
  │    ├─ biz 构造 CommissionCNYSnapshot
  │    └─ CommissionRepo.Create(txCtx, commission, snapshot, audit)
  │         └─ 现有 WithTx 自动加入外层事务
  └─ 提交后 CommissionRepo.Get(ctx) 返回完整响应
```

`CommissionRepo.Create` 调整为只负责事务内写入并返回错误，避免在共享事务提交前通过
普通连接重读。任务触及的 List/Get/GetByKey/Preview/生成上下文读取改为
`Data.client(ctx)`，防止静默逃离共享事务。

预览走普通只读上下文，读取相同生成上下文并调用相同 CNY 纯计算函数。创建不接受
预览结果，必须在事务内再次解析。

## 持久化与索引

- 六个快照字段全部 Immutable；setting ID 可空，其余非空。
- 金额使用 `numeric(28,8)`，汇率使用 `numeric(18,8)`。
- 新增 `(organization_id, commission_date)` 索引。
- 不设置业务默认值，不创建历史回填逻辑。

## API 变更

- `ListCommissionsRequest` 增加可选 `commission_date_from/to`。
- `CommissionCalculation` 增加 CNY rate、source、rate date、可选 setting ID 和
  预览 CNY 原始金额。
- `FinanceCommission` 增加归属日期、持久化 CNY 字段和两个动态 CNY 金额。
- 金额、汇率和日期均使用 string；setting ID 使用 optional string。

## 查询、审计与错误

- 数据层抽取私有 commission predicate 构造函数，供 List 与阶段 2 导出复用。
- 默认排序追加唯一 ID，保证固定数据集下分页稳定。
- 创建审计详情增加 commission date、CNY rate/source/rate date/setting ID。
- 汇率缺失、冲突、日期缺失沿用汇率领域错误，不降级。
- 任一写入或审计失败由共享事务回滚。

## 验证重点

- CNY 与非 CNY 两条汇率路径、倒数和乘法的 8 位舍入。
- 预览与创建之间设置变化、事务失败无半成品。
- 动态调整状态组合、日期筛选边界与稳定排序。
