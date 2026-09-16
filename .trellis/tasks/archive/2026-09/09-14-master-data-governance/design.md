# design.md — 主数据多组织治理与存储重构技术设计

对应 `prd.md`。所有变更未上线一次性到位，不留兼容分支。

## 1. 存储形态总表

| 表 | 型 | organization_id | 索引变更 | 备注 |
|---|---|---|---|---|
| `currencies` | A | 已无 | 不变 | A 型范本 |
| `administrative_regions` | A | 已无 | 不变 | A 型范本 |
| `master_data_items` | A | **删除列** | `(organization_id, kind, code)` 唯一 → `UNIQUE(kind, code)`；其余 org 组合索引去 org | kind 枚举 `service_type` → `charge_category` |
| `airlines` | A | **删除列** | `(organization_id, iata_code)` 唯一 → `UNIQUE(iata_code)`；org 组合索引去 org | 让位谓词与本地行删除 |
| `shipping_lines` | A | **删除列** | `(organization_id, scac_code)` 唯一 → `UNIQUE(scac_code)` | 同上 |
| `shipping_line_container_prefixes` | A | **删除列** | 随主表 | |
| `billing_units` | A | **删除列** | `(organization_id, code)` 唯一 → `UNIQUE(code)` | |
| `ports` | B | **改可空** | `UNIQUE(un_locode) WHERE organization_id IS NULL` + `UNIQUE(organization_id, un_locode) WHERE organization_id IS NOT NULL` | 让位语义保留，谓词重构 |
| `airports` | B | **改可空** | 同上（键 `iata_code`） | |
| `exchange_rate_settings` | B | **改可空** | `UNIQUE(from_currency, to_currency, effective_from) WHERE organization_id IS NULL` + `(organization_id, from_currency, to_currency, effective_from) WHERE IS NOT NULL` | 扩展 `ar_rate`, `ap_rate`，按自然周生效；支持中行牌价一键抓取 |
| `fee_settings` | B | **改可空** | `UNIQUE(fee_code) WHERE organization_id IS NULL` + `(organization_id, fee_code) WHERE IS NOT NULL`；`service_type_id` 改必填并更名 | 引用校验放宽 |
| `taxable_services` | C | 保持必填 | 不变 | 放开归属组织自维护 |
| `enterprise_resources` | C | 保持必填 | 不变 | 收发通抬头、地址簿、标签、品名、唛头纯组织私有隔离 |
| Partner 系、号码、标签、业务单据 | C | 保持 | 不变 | |

FK 安全：`orders.shipping_line_id` 等 edge 指向本表主键，单表改列不动 FK；`order_fees.fee_setting_id` 为可空弱引用，无需变更。

## 2. B 型统一查询构造器（data 层）

新增 `internal/data/baseline_query.go`，提供两个泛型谓词工厂（每表一组，表名/列名来自代码内固定清单）：

- **点查形态** `BaselineOrLocal(codeCol, code, orgID)`：`organization_id = orgID OR organization_id IS NULL`，排序 `CASE WHEN organization_id = orgID THEN 0 ELSE 1 END`，取首行。
- **列表形态** `BaselineShadowedByLocal(table, codeCol, orgID)`：`organization_id = orgID OR (organization_id IS NULL AND NOT EXISTS (SELECT 1 FROM table AS local WHERE local.organization_id = orgID AND UPPER(local.code) = UPPER(outer.code)))`——被覆盖的基线行整行不可见，去重下推保证分页计数一致（承接 `writeIndustryLocalCodePrecedence` 形态）。

`writeIndustryLocalCodePrecedence` 与 `port/airport/airline/shippingLineOrganizationFilter` 演进为上述工厂的调用方；airline/shippingLine 分支删除。主数据读路径全部移除 `resolveHeadquartersOrganizationID` 调用；该函数仅保留给组织身份判定（见 §3），收敛到单一文件。

## 3. 拦截器设计（biz 层）

`internal/biz/organization.go` 新增：

- `RequireGlobalMasterDataWrite(ctx, principal)`：校验 principal 当前组织沿树解析到根且根 `kind == headquarters`（复用现有树解析，仅写路径执行、可缓存于 principal），并持有对应权限码。
- `RequireBaselineWrite(ctx, principal)`：同上，用于 B 型 NULL 行写路径。

挂接点：各 A 型实体的 biz Create/Update（masterdata、industry_reference 的 airline/shippingline 部分、fee_catalog 的 billing_unit 部分）；B 型 NULL 行写入前判定（当前组织为总部时才可能写 NULL 行，组织非总部一律落 org 行）。业务错误统一 403 中文提示（「XX 基础资料只能由总部维护」风格沿用）。

## 4. 汇率实战化重构与中国银行自动抓取（R3）

- **周汇率区间与存储扩展**：
  - 字段扩展：`ar_rate`（现汇卖出价/应收汇率，`numeric(18,8)`）、`ap_rate`（现汇买入价/应付汇率，`numeric(18,8)`）、`rate`（中行折算价/基准汇率）。
  - 有效期按自然周管理：`effective_from`（周一 00:00:00）至 `effective_to`（周日 23:59:59）。
  - **跨周预设与幂等 Upsert**：
    - 同步服务支持目标周期参数（`TARGET_CURRENT_WEEK` / `TARGET_NEXT_WEEK`），支持周五/周末提前发布下周汇率；
    - 同周二次抓取执行幂等 Upsert，覆盖更新该周记录，禁止抛出唯一约束冲突。
  - `to_currency` 物化为核算组织的本币（如深圳分公司 CNY、香港分公司 HKD）。
- **官方/市场外汇牌价一键自动抓取体系（覆盖 CNY 与非 CNY 本币）**：
  - **数据源与防盗链策略**：
    - **本币为 CNY（国内实体）**：
      - **主源（高效 JSON）**：调用新浪中行专线外汇接口（`https://vip.stock.finance.sina.com.cn/forex/api/openapi.php/ForexService.getBankForex`），服务端自动携带 `Referer: https://finance.sina.com.cn` 突破防盗链；提取 `bank: "boc"` 的 `xh_buy_price`（建议 `ap_rate`）与 `xh_sell_price`（建议 `ar_rate`）。JSON 纯净且数值已按 1 外币折合本币单价归一化。
      - **官方兜底源（HTML）**：直连中国银行官方公开外汇牌价页面（`https://www.boc.cn/sourcedb/whpj/`），HTML 解析 `<table id="priceTable">` 兜底，按报价基准除以 100 换算。
    - **本币为非 CNY（境外实体，如 HKD/SGD/USD）**：
      - **国际外汇直盘抓取（首选）**：根据当前组织本币（如 HKD）与常用外币请求外汇直盘行情（`fx_s{from}{to}`，如 `fx_susdhkd`, `fx_seurhkd`, `fx_susdsgd`，带 Referer）；直接提取实时 Ask（卖出价）作为建议 `ar_rate`，Bid（买入价）作为建议 `ap_rate`。
      - **中行交叉盘算法（备选/集团联动）**：中行外币牌价与本币（如 HKD）对 CNY 的比价进行交叉折算：$\text{Rate}_{\text{外币}\to\text{本币}} = \frac{\text{Rate}_{\text{外币}\to\text{CNY}}}{\text{Rate}_{\text{本币}\to\text{CNY}}}$。
  - **后端提供 `FetchExchangeRates` 服务**：自动按当前组织本位币与目标周路由至对应数据源，自动计算自然周起止时间区间，返回结构化预览数据。
  - **前端交互（防呆闭环与财务终审）**：`/finance/exchange-rates` 页面提供同步按钮，内置【本周】/【预设下周】切换；弹窗展示币种、应收汇率、应付汇率与生效周期；支持财务现场基于商业加点、报价取整或协议汇率进行微调，点击「确认发布」后批量入库生效，落实财务终审责任，绝不静默落库破坏业务账。
- **费用结算折算与三级缓冲容灾机制**：
  - `ResolveRate`：按费用类型命中对应点差——应收费用折算使用 `ar_rate`，应付费用折算使用 `ap_rate`；按发生日期匹配生效周区间。
  - **解析顺序（绝不卡死单据）**：
    1. 本组织当周汇率行命中（快照源 `WEEKLY`）；
    2. **一级缓冲（回溯继承上周）**：若当周未配，自动顺延沿用最近一个有效自然周的汇率（快照源 `INHERITED_LAST_WEEK`，前端展示黄色轻量 Tag「暂沿用上周汇率」，单据正常流转保存，不中断订舱与报关）；
    3. **二级兜底**：未命中时退到 NULL 基线行直连（`SYSTEM`，兜底）→ NULL 基线行 pivot 交叉套算（`DERIVED`，兜底）；
    4. **三级放行**：仍缺失时允许单据现场手工覆盖汇率（快照源 `MANUAL`）。
  - **漏配督办通知**：复用既有通知投递设施（`notification_delivery`）+ 定时任务，周一 10:00（Asia/Shanghai）检测分公司当周汇率仍未同步时，向财务角色推送轻量待办提醒；同周期内已同步则不发送。
  - `ResolveBaseRate`（总部基线折算，提成 CNY 口径）**退役**：跨组织提成与往来按原币记账（见下条）；组织内本币管理口径用本组织 `ResolveRate` 自行解析。调用方（`finance_commission` 等）一次性切换。
  - **严格快照隔离**：费用录入时将汇率快照落库（`order_fees` / `finance_bill_lines`）；后续汇率更新仅对新单据/未出账费用生效，绝不穿透篡改历史已锁定费用快照。快照字段扩展：按费用方向落 ar/ap 值，`exchange_rate_source` 枚举扩展为 `WEEKLY` / `INHERITED_LAST_WEEK` / `MANUAL` / `BOC_SYNC`；`exchange_rate_setting_id` 指向命中行。
- **跨组织协同结算务实化**：
  - 分公司间往来结算账单直接按**单据原币（如 USD）**进行跨组织对账核销，杜绝跨组织二次折算引入的人为人为汇差死账。
- **归属与重叠校验**：
  - 各分公司组织面向自身核算本币维护；总部亦可维护 NULL 基线行（作为公共兜底）；周区间重叠校验按同组织同货币对判定。

## 5. 费用科目两级（R4）

- `fee_settings.service_type_id` 更名 `charge_category_id` 并去 Nillable（必填，指向 A 型 `master_data_items` kind=`charge_category`）。
- 总部 NULL 行公共科目由种子/总部维护；分公司 org 行本地明细必挂 charge_category。
- 选择器（订单费用等）：`BaselineShadowedByLocal` 列表形态，同码本地优先；快照字段照旧冗余名称/币种/税率。
- `validateFeeSettingReferences`：计费单位与 charge_category 查 A 型全局表；税务名称查「本组织 org 行」；全部要求 enabled。

## 6. `service_type` → `charge_category` 就地改名清单

- Ent：`master_data_item.go` kind 枚举值、`fee_setting.go` edge/field（`service_type_id` → `charge_category_id`）及反向 edge 名。
- 迁移：`UPDATE master_data_items SET kind='charge_category' WHERE kind='service_type'` + 列更名，随本任务迁移一次性执行。
- Go：`MasterDataKindServiceType` → `MasterDataKindChargeCategory`；仓储/服务层标识符同步。
- proto/API：字段与注释同步更名；前端 `typings.d.ts` 由生成器更新，页面文案统一「费用大类」。
- 种子值（BOOKING/TRUCKING/报关等 19 项）不变，仅归属新 kind。

## 7. 契约与前端（R5）

- proto：`OrganizationKind` 枚举 `UNSPECIFIED/HEADQUARTERS/COMPANY/DEPARTMENT/TEAM`，映射既有 `organization.kind`；`auth/me` 组织体、登录组织选择等响应补字段；`make -C server api` + `pnpm run generate:web-client`。
- 前端：`initialState.currentUser.currentOrganization.kind` 驱动；`useAccess` 组合出 `isHeadquartersOrganization` 与各页签 `canMaintain*` 判定，页面不写第二套规则。
- 页面改造：`/master-data` 七页签 + `/finance/fee-settings` 三面板 + `/finance/exchange-rates`；A 型页签非总部只读横幅 + 隐藏写按钮；B 型页签隐藏基线行编辑、保留新增本地行；权限码无新增（沿用既有 canCreate/Update* 码 × kind 判定）。

## 8. 迁移设计（阶段一执行）

顺序（单次迁移内分步）：

1. master_data_items：`service_type` → `charge_category` 数据更名；`organization_id` 置 NULL → 删列；唯一索引重建。
2. airlines / shipping_lines（含前缀表）/ billing_units：Fail-fast 冲突检测（同码比对总部行与分公司行，一致→删除分公司行；冲突/无匹配→产出报告并中止）；用户已授权实施方按报告处置开发数据；随后删列、索引重建。
3. ports / airports：总部行 org 置 NULL，重建 partial unique index。
4. fee_settings：总部行 org 置 NULL，重建 partial unique index；先补 `charge_category_id` 必填（无值的行由总部按名称归类或标记待处理）。
5. exchange_rate_settings：总部行 org 置 NULL（兜底基线）；扩展 `ar_rate`/`ap_rate` 并按自然周窗口重建存量行（开发库存量任意区间行清理重建，已授权）；同步迁移 `order_fees`/`finance_bill_lines` 快照字段（来源枚举扩展，`DERIVED`/`BASE_CURRENCY` 历史值清理）。
6. 种子脚本 `server/seeds/` 与 `sync:all` 改写为 A 型直写（去组织解析）。

迁移全程记录冲突报告至 `.trellis/tasks/09-14-master-data-governance/`，不做静默合并。

## 9. 测试设计

- 单元/集成（Go）：
  - B 型两种形态谓词的行为测试（覆盖/未覆盖/分页计数）；
  - 汇率周区间与点差匹配测试（应收命中 `ar_rate`，应付命中 `ap_rate`，费用快照冻结）；
  - 容灾链测试（当周缺失→继承上周 `INHERITED_LAST_WEEK`、基线兜底、同周二次同步幂等 Upsert、跨周预设生效）；
  - 牌价抓取与区间计算服务测试（新浪主源/中行兜底/非 CNY 直盘与交叉盘换算，/100 与已归一化两种口径）；
  - proto 契约测试：解除 `exchange_rate.proto` 中 `receivable_rate`/`payable_rate` 的 reserved（71–73 行）后正式定义 `ar_rate`/`ap_rate`；
  - 拦截器（分支上下文持权限写 A 型/NULL 行 → 403；总部 → 通过）；
  - A 型全局唯一冲突映射业务错误；费用引用校验放宽。
- 前端：
  - `kind` 判定的访问组合与页签收敛（Vitest 定向）；
  - `/finance/exchange-rates` 中行同步抽屉交互与数据回填测试；
  - 既有 master-data 面板测试随交互调整更新。
- 回归：港口让位、Partner 可见性、字典 403 既有测试迁移后全绿。
