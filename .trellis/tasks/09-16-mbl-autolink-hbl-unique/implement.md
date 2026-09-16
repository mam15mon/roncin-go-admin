# 执行计划：共享主单批次自动关联与分单批次排重

## 前置确认

- [ ] `git status` 干净；基线在 PRD 落定提交（9cdee32c）之后。
- [ ] 核实 `sea_house_bills.master_bill_id` 列可空性（决定 D1 索引是否需要
  `WHERE master_bill_id IS NOT NULL`）。
- [ ] 探查存量数据：
  - 同 `master_bill_id` 下跨签发主体重号（新索引直接决定迁移成败）：

    ```sql
    SELECT master_bill_id, normalized_house_no, count(*)
    FROM sea_house_bills GROUP BY 1, 2 HAVING count(*) > 1;
    ```

  - 开发库当前 HBL 行数与模式分布（预期极小或为空）。
  结果为空 → 迁移自然通过；有冲突行 → 列表报用户确认后再动索引。

## 步骤

1. [ ] Ent Schema 索引换轨（design D1）：删两个全局部分唯一索引、新增批次唯一
   索引；生成迁移并加建索引前 DO 块预检（中文重复明细）；`biz` 新增
   `ErrSeaOrderBatchRequiresHouse` / `ErrSeaMasterBillBatchDirectBlocked` /
   `ErrSeaDocumentBatchMemberExitBlocked` / `ErrSeaHouseBillBatchNoDuplicate`。
2. [ ] 服务端校验（design D2）：创建订单候选确认分支（直单阻断 / 强制 HOUSE +
   分单号必填）、模式切换双向校验（DIRECT→HOUSE 批次排重预查、HOUSE→DIRECT 成员
   阻断）；抽取「活动成员票」共用查询；候选命中未确认时现有 409 行为不变。
3. [ ] 候选查询扩展（design D3）：proto 成员摘要加 `document_structure`、候选加
   `batch_normalized_house_nos`；`make -C server api` +
   `pnpm run generate:web-client`；data 层成员结构与批次号聚合查询。
4. [ ] 服务端集成测试（design 测试设计 T1–T6）+ biz 单测；`go vet`。

```bash
go -C server vet ./...
go -C server test ./internal/data/ -run 'TestSea' -count=1
RONCIN_INTEGRATION_DATABASE_SOURCE='postgresql://roncin@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable' \
  go -C server test ./internal/data/ -run 'TestSeaMasterBill|TestSeaHouseBill|TestSeaDocumentModeChange' -count=1 -v
```

   （注意：`.env.local` 无 `RONCIN_INTEGRATION_DATABASE_SOURCE`，必须显式传 DSN
   并用 `-v` 确认无 SKIP。）

5. [ ] 前端（design D4）：模式默认 HOUSE + 分单号必填、候选自动查询与横幅
   （含直单阻断文案）、确认参数自动携带、分单号失焦排重提示；更新
   `sea-create-layout.test.tsx`「创建时显式选择模式」相关断言为新口径。
6. [ ] 前端定向验证：

```bash
pnpm --dir web exec vitest run src/pages/orders/templates/ src/pages/orders/order-kinds/sea-export/
pnpm --dir web exec biome check <改动文件>
pnpm --dir web tsc
```

7. [ ] spec 固化：更新 `sea-export-document-contract.md`——唯一键清单（撤销两个
   全局部分唯一索引、新增批次唯一索引）、单证结构补充「批次成员转直单限制」、
   错误矩阵增补四个新错误；`migration` 注意事项（存量重号 fail-fast）。
8. [ ] `cmd/migrate` 端到端演练：集成库临时 schema 从零跑迁移（T5），再植入重号
   存量验证失败路径；跑两遍验证幂等。

## 收尾核验（全任务完成时执行一次）

```bash
go -C server test ./internal/data/
pnpm --dir web tsc
pnpm --dir web exec vitest run src/pages/orders/
git diff --check
```

## 提交

- 建议 2–3 笔：
  1. `feat(orders): 共享主单批次自动关联与直单禁拼（服务端）`（步骤 1–4 + 迁移）；
  2. `feat(orders): 新建默认分单制与批次排重交互（前端）`（步骤 5–6）；
  3. `docs(spec): 同步共享主单批次唯一性契约`（步骤 7）。
  或按 review 需要合并；生成物随对应源文件提交。

## 回滚点

- 步骤 1 索引换轨为唯一不可逆风险点：新索引生效后若产生跨批次重号数据，回滚迁移
  会因旧全局唯一索引对存量失败；上线前在集成库演练反向迁移，出现该情形时的人工
  处置（去重或保留新索引）先与用户确认。
- 其余步骤均为纯代码，`git checkout --` / revert 可回退。

## 风险与备注

- 候选确认参数自动携带后，`SEA_MASTER_BILL_CONFIRMATION_REQUIRED` 409 仍保留作
  并发兜底（候选在保存前被第三方变更），前端需保留该错误的刷新重试文案。
- `changeCreateMode` 现有联动（委托件重尺带入 MBL/HBL 内容）在默认 HOUSE 下首次
  不触发 onChange——核实初始渲染时 `seaHouseBill.content` 默认值仍被正确初始化。
- 本任务无新增事务需求；模式切换/创建沿用既有事务与锁序，禁止手写事务。
