# 技术设计：共享主单批次自动关联与分单批次排重

## 目标口径（对齐 prd.md R0–R3）

- 新建默认 HOUSE、分单号必填；直单 = 显式选择 = 独占主单。
- auto-link 命中全 HOUSE 批次 → 自动确认入批；命中含直单成员的批次 → 明确阻断。
- 分单号唯一性收敛为 `(master_bill_id, normalized_house_no)` 批次内唯一（跨签发主体、
  含作废行）；撤销组织+签发主体的两个全局部分唯一索引。

## 决策

### D1 排重索引换轨（唯一 schema 变更）

`server/internal/data/ent/schema/sea_house_bill.go` Indexes：

- 删除 `idx_sea_house_bills_self_org_unique`
  （`organization_id, issuer_organization_id, normalized_house_no` WHERE self）与
  `idx_sea_house_bills_partner_unique`（WHERE customer/other partner）；
- 新增 `index.Fields("master_bill_id", "normalized_house_no").Unique()`
  StorageKey `idx_sea_house_bills_batch_no_unique`，**不加状态过滤**（作废行天然
  覆盖，"一号一案"）；若 `master_bill_id` 列可空则补
  `Annotations(entsql.IndexWhere("master_bill_id IS NOT NULL"))`（实施时核实列定义）；
- 生成正式迁移；Postgres 建唯一索引对存量重复天然失败即终止（fail-fast），迁移
  文件内在建索引前加 DO 块预检，给出可读的中文重复明细后 RAISE EXCEPTION。
- 错误映射：新索引冲突经 `ent.IsConstraintError` 映射新业务错误
  `ErrSeaHouseBillBatchNoDuplicate`（409，通用批次重复文案）；事务内预查负责给出
  含冲突号的友好提示，索引只作并发兜底。

### D2 批次规则的服务端校验点（三处，均在既有事务/锁序内）

新增 biz 错误（`internal/biz`，中文提示一次写定，service 不二次翻译）：

```go
ErrSeaOrderBatchRequiresHouse  // 400 该主单已存在共享批次，本单必须签发分单
ErrSeaMasterBillBatchDirectBlocked // 409 该主单已被直单订单占用，如需拼单请先将其转为分单
ErrSeaDocumentBatchMemberExitBlocked // 409 共享主单批次存在其他成员订单，不能转为直单
ErrSeaHouseBillBatchNoDuplicate    // 409 分单号在该主单批次内已存在
```

1. **创建订单**（`order_usecase.CreateOrder` 确认候选分支）：候选命中且该 MBL 存在
   其他活动成员票时——若任一成员 `document_structure = DIRECT` →
   `ErrSeaMasterBillBatchDirectBlocked`；否则本单 `sea_document` 必须
   `HOUSE` 且 `house_bill.house_no` 非空 → 否则 `ErrSeaOrderBatchRequiresHouse`。
   未命中候选：维持现状（HOUSE 缺分单号是否已拒需实施时核实，缺失则按本规则补齐）。
2. **模式切换**（`sea_document_change`）：`DIRECT→HOUSE` 创建 HBL 前做批次排重预查
   （该 `master_bill_id` 下同 `normalized_house_no` 的既有行，含作废）→ 命中
   `ErrSeaHouseBillBatchNoDuplicate`（提示冲突号）；`HOUSE→DIRECT` 时若同 MBL 存在
   其他活动成员票 → `ErrSeaDocumentBatchMemberExitBlocked`，仅剩自己时放行（现有
   作废流程不变）。
3. **活动成员口径**：成员票 = 该 MBL 上 `status = 'ACTIVE'` 的 Link 且订单未删除；
   与候选查询 `member_count` 同源（实施时抽取共用查询，避免两套口径）。

直单创建（显式 DIRECT）不校验批次占用——候选命中时若批次全为直单成员？不存在：
直单成员存在即阻断（含全直单批次），语义为"直单票独占，后来者一律阻断"。

### D3 候选查询扩展（Q3 决策：扩展候选，不加新接口）

proto `SeaMasterBillMemberSummary` 增字段、`SeaMasterBillCandidate` 增批次号清单：

```protobuf
message SeaMasterBillMemberSummary {
  string order_id = 1;
  string order_no = 2;
  optional string customer_reference_no = 3;
  SeaDocumentStructure document_structure = 4;  // 新增：成员单证结构（直单成员判定）
}
message SeaMasterBillCandidate {
  ...
  repeated string batch_normalized_house_nos = 10;  // 新增：批次内全部规范化分单号（含作废）
}
```

- data 层：成员查询补 link 的 document_structure（与成员摘要同查询）；新增一条
  `SELECT normalized_house_no FROM sea_house_bills WHERE master_bill_id = ?` 聚合
  （不过滤状态）。
- 前端失焦即时提示用 `batch_normalized_house_nos` 比对（输入值按 NFC + 去首尾空白
  + 大写归一后比对，仅体验层）；服务端事务内预查为权威。
- 生成链：`make -C server api` → `pnpm run generate:web-client`，生成物同提交。

### D4 前端新建表单（R0 + R1 + R2 体验层）

`web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx` 与
`sea-template.tsx`：

1. `SeaCreateDocumentModeField` 的 `seaDocumentStructure` 加
   `initialValue = SEA_DOCUMENT_STRUCTURE_HOUSE`，HBL 分单内容区块默认渲染；
   切 DIRECT 仍走既有 `changeCreateMode` 清理联动；分单号输入加必填规则；
2. 录入「船公司/主单号」变更时防抖调用 `orderServiceMatchSeaMasterBillCandidate`：
   - 未命中：静默（或轻提示）；
   - 命中：保存「已自动关联共享主单批次（N 票）」横幅；候选确认参数（candidate
     id + version）写入表单提交载荷（现有 `SeaMasterBillInput` 确认字段）；任一
     成员 `document_structure = DIRECT` → 横幅转红色阻断文案（提交前拦截，服务端
     仍兜底）；
   - 分单号失焦：与 `batch_normalized_house_nos` 比对，重复即行内错误提示；
3. 详情页模式切换弹窗：服务端新错误原样展示（无前端新逻辑）。

### 影响面与回滚

- schema 迁移为索引换轨：回滚 = revert 恢复旧索引定义并生成反向迁移；新索引上线后
  若业务产生跨批次重号数据，反向迁移可能失败——回滚前需人工核对（记录于 implement
  风险）。
- 候选响应为纯增字段（proto 兼容），无破坏性。
- 直单创建路径行为变化：显式直单仍可建（独占语义），无阻断。

## 测试设计

- 集成（`RONCIN_INTEGRATION_DATABASE_SOURCE`，隔离 Schema；**-v 确认无 SKIP**）：
  - T1 命中全 HOUSE 批次创建成功入批（A2）；候选并发变化 409；
  - T2 命中含直单成员批次 → `ErrSeaMasterBillBatchDirectBlocked`（A3）；
  - T3 批次成员 HOUSE→DIRECT：还有其他成员 → 阻断；仅剩自己 → 放行（A3）；
  - T4 批次排重：跨主体同号（含与作废行同号）→ 友好错误；跨批次同号成功；
    唯一索引并发兜底（A4）；
  - T5 迁移：植入同批次跨主体重号存量 → 迁移失败终止；干净库迁移成功（A4）；
  - T6 现有共享主单/单证用例回归（A1）。
- biz 单测：三个新错误触发条件、活动成员口径边界。
- 前端定向 vitest：默认 HOUSE + 分单必填、切直单清理、候选命中横幅与直单阻断、
  分单号失焦排重提示；biome + tsc。
