# 设计：外币财务连续验收

## 变更边界

主要交付物是可重复执行的验收脚本、根目录验收命令和本任务的真实运行报告。预计不
修改生产 API、领域规则、仓储或 Schema。现有应收验收脚本只做两项直接相关调整：
显式创建本轮客户，以及补充 CNY 提成快照断言。

若执行中发现生产响应与既定业务口径不符，先保留失败证据并停止，不把业务修复夹带
进本任务。

## 证据边界

```text
临时库 A（CNY 基线）
  ├─ migrate + bootstrap-admin
  ├─ 真实 PostgreSQL 集成测试（显式连接，禁止 SKIP 冒充 PASS）
  ├─ Go HTTP 服务 :8010
  ├─ Web 测试服务 :8001 → :8010
  └─ acceptance:finance（应收 + 应付 + Playwright）

临时库 B（USD 外币链）
  ├─ migrate + bootstrap-admin
  ├─ 业务数据写入前经 Admin API 设置 base_currency=USD
  ├─ Go HTTP 服务 :8010（与 A 串行，禁止共享进程/数据库）
  └─ acceptance:finance:foreign-currency（纯真实 HTTP API）
```

两个环境串行执行。编排器启动前确认 `8010/8001` 没有监听进程；端口被占用时直接
失败，不终止未知进程或另选端口。每次启动均显式设置数据库、API 和 Web 地址；脚本
缺少必需环境变量时立即失败，不读取开发环境默认值。清理由编排器使用经过固定前缀
校验的明确数据库名和角色名执行，Codex 主会话在结束后独立查询确认。

## 外币场景设计

组织本位币为 USD，业务币为 EUR，所有业务日期使用同一验收日。时间标准和汇率均由
API 创建：

| 环节 | rate_type | 时间标准 | 方向 | 系统汇率 |
| --- | --- | --- | --- | ---: |
| 订单费用 | BASE_CURRENCY | ORDER_CREATED_AT | EUR→USD RECEIVABLE | 1.10000000 |
| 账单 | BILL | BILL_DATE | EUR→USD RECEIVABLE | 1.20000000 |
| 开票 | INVOICE | INVOICE_DATE | EUR→USD RECEIVABLE | 1.22000000 |
| 收款 | SETTLEMENT | TRANSACTION_DATE | EUR→USD RECEIVABLE | 1.25000000 |
| 核销 | WRITE_OFF | WRITE_OFF_TIME | EUR→USD RECEIVABLE | 1.30000000 |
| 提成 CNY | WRITE_OFF | WRITE_OFF_TIME | CNY→USD RECEIVABLE | 0.14000000 |

每条配置同时填写合法的应收、应付汇率，但本任务只断言应收方向。`effectiveFrom` 统一
使用 Asia/Shanghai 当日零点、带时区且精确到秒的 RFC 3339 字符串，不传日期缩写或
纳秒。每个业务响应不仅检查
数值，还必须检查 `exchangeRateSource=SYSTEM`、验收日和创建时保存的 setting ID，
从而排除手工汇率或其他配置碰巧得到同一金额的假阳性。

## 金额与快照口径

外币链使用一笔 100 EUR 的应收费用和账单，开票 100 EUR，收款及核销 40 EUR：

```text
费用本币金额       = 100 × 1.10 = 110.00000000 USD
账单本币金额       = 100 × 1.20 = 120.00000000 USD
开票本币金额       = 100 × 1.22 = 122.00000000 USD
资金流水本币金额   =  40 × 1.25 =  50.00000000 USD
核销本币金额       =  40 × 1.30 =  52.00000000 USD
账单本币分摊       = 120 × 40%  =  48.00000000 USD
资金流水本币分摊   =  50 × 100% =  50.00000000 USD
应收汇兑损益       = 52 - 50    =   2.00000000 USD
提成已实现收入     = 120 × 40%  =  48.00000000 USD
10% 提成           = 48 × 10%   =   4.80000000 USD
CNY 派生汇率       = round(1 / 0.14, 8) = 7.14285714
CNY 提成           = round(4.8 × 7.14285714, 8) = 34.28571427
```

提成规则使用 `REALIZED_PROFIT` 且该订单不创建应付费用，因此已实现利润等于已实现
收入。脚本应同时断言预览与创建详情，且详情中的 `cnyExchangeRateSettingId` 必须指向
CNY→USD 的 WRITE_OFF 配置，不得错误复用 EUR→USD 核销配置。

## 脚本结构

- 新增 `scripts/run-acceptance-finance-disposable.mjs`：要求调用方显式提供 PostgreSQL
  管理连接，先确认固定验收端口 `8010/8001` 空闲，再串行创建临时库 A/B、迁移、
  初始化、启停服务、执行验收并
  在 `finally` 清理。数据库名和角色名必须带固定任务前缀并在删除前再次校验。
- 调整 `scripts/acceptance-finance-bill-batch.mjs`：`--apply` 时创建名称带唯一时间戳
  的客户，不再选择数据库中的首个客户；预检仍检查权限、编号规则和必要主数据。
- 新增 `scripts/acceptance-finance-foreign-currency.mjs`：沿用现有脚本的 HTTP 调用与
  断言风格，顺序创建本场景所需数据，不抽取与本任务无关的公共框架。
- 在 `package.json` 增加单独命令 `acceptance:finance:foreign-currency`；不把外币链
  直接并入现有 `acceptance:finance`，避免已有开发验收命令隐式修改组织本位币。
- 在 `package.json` 增加一次性环境总入口；该入口缺少 PostgreSQL 管理连接或
  bootstrap 凭据时必须直接失败，禁止读取 `.env.local` 或连接默认开发库。
- 外币脚本要求调用方提供全新的空环境，并在启动时验证当前组织尚无业务数据且本位币
  仍为初始化值；验证失败直接终止，不提供回退或自动清库。

## Agy 审查后的拓扑取舍

Agy 独立审查确认：新建子组织后直接切换会因为该组织没有默认角色而得到 403。它建议
用数据库夹具预置 USD 子组织角色。本任务不采用该方案：临时库 B 本身就是单独的空
环境，bootstrap 后使用已有管理员权限通过 Admin API 更新当前总公司本位币即可，随后
才创建第一笔业务数据。这样保留纯 HTTP 业务链，也避免直接写角色、权限和组织表。

Agy 同时建议把提成支付、调整和反核销并入本链。它们不会增强“系统汇率至 CNY 快照”
的核心证据，且 `MarkPaid` 已被审计识别为独立缺口，因此本任务不扩张这些状态流转。

## 安全与失败处理

- 只使用本任务创建并记录的临时数据库、角色和进程；不终止未知进程，不删除未通过
  名称与连接归属校验的数据库或角色。
- 任一 HTTP 非 2xx、字段缺失、金额不一致、setting ID 不一致或清理失败都使验收
  失败。脚本不捕获后继续，也不把系统汇率失败改成手工汇率。
- 输出只包含必要业务编号、非敏感 ID、断言摘要和 PASS 状态；认证信息只保留内存。
