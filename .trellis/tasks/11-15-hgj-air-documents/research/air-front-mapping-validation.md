# 航空前端映射与保存基线验证

日期：2026-11-16（UTC）。本记录只覆盖本代理负责的航空表单映射、详情保存、外部变动检查、历史正文展示及两份 E2E 夹具迁移；模板、正文控件与主单工作台由主会话负责。未创建 Git 提交。

## 实现范围

- `web/src/pages/orders/order-kinds/form-values.ts`：新增 `airDocumentMode`，`airHouseContent` 改为生成的 `API.AirWaybillContent`。
- `web/src/pages/orders/order-kinds/air-export/form-adapter.ts`、`form-adapter.test.ts`：创建显式提交 DIRECT/HOUSE，EXISTING 只携带主单 ID/版本；详情使用真实可选 `house` 消息；更新关系身份取原基线，HAWB 携带独立预期版本。正文省略保持，`{}` 显式清空，零值/金额字符串/纯日期保留。`serializeAirWaybillContent` 只省略声明及费用汇总控件产生的空子消息，不过滤非法非空输入或原始明确零。
- `web/src/pages/orders/order-kinds/air-export/air-save-baseline.ts`、`air-save-baseline.test.ts`：只采纳同次确切成功版本；Order 增一，HAWB 修改时 House/Link 同时增一，主单身份/版本不变。DIRECT 无 HAWB 仍能保存。正文按 ProtoJSON 零值语义比较；服务端派生运费、其他费合计与总计不作为可编辑正文差异。使用 BigInt 验证超过 JS 安全整数范围的版本。
- `web/src/pages/orders/detail.tsx`：航空保存同步确认 House/Link 基线，不覆盖在途正文/原因；拒绝迟到后续版本。航空接通现有 15 秒/回焦检查及持续提醒，成功自身保存采纳快照，输入保护保持。
- `web/src/pages/orders/use-order-change-notice.ts`：变动指纹增加航空模式、活动关系、主单/分单及 Link 身份和版本。
- `web/src/pages/orders/order-kinds/air-export/AirExportDetailFeatures.tsx`：DIRECT 不展示分单历史；HOUSE 历史的正文使用同一结构化控件只读展开，不把对象作为字符串单元格渲染；时间沿用本地格式化。
- `web/src/pages/orders/detail-air-house-save.fixtures.tsx`、`detail-air-house-save.test.tsx`、`detail-air-house-save-race.test.tsx`、`detail-air-house-save-stale.test.tsx`、`detail-air-house-save-direct.test.tsx`、`detail-change-notice-air.test.tsx`：同步结构化真实夹具，验证签发后改单、失败/在途输入、A→B→A、迟到版本、直单连续保存及主单/分单/Link 单独变化提醒。
- `web/tests/e2e/air-export.e2e.ts`、`finance-bill-batch.e2e.ts`：同步新 `house.houseNo` 响应及创建输入 `documentMode/house`，不保留扁平旧合同。

## 检查与结果

1. 首轮旧模板中间态失败已随模板完成和夹具切换修复；不将中间态失败当最终结果。
2. 下列定向命令 90/90 通过，包含相邻海运变动检查与本人状态变更路径：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/order-kinds/air-export/form-adapter.test.ts \
  src/pages/orders/order-kinds/air-export/air-save-baseline.test.ts \
  src/pages/orders/detail-air-house-save.test.tsx \
  src/pages/orders/detail-air-house-save-race.test.tsx \
  src/pages/orders/detail-air-house-save-direct.test.tsx \
  src/pages/orders/detail-change-notice-air.test.tsx \
  src/pages/orders/use-order-change-notice.test.tsx \
  src/pages/orders/detail-change-notice.test.tsx \
  src/pages/orders/detail-status-own-change.test.tsx \
  --reporter=json --outputFile=/tmp/air-front-mapping-final-tests.json
```

3. 上述 race 文件耗时 21.30 秒，按仓库长尾规范拆成 `race` 与 `stale` 两文件；随后命令 4/4 通过，文件耗时分别 11.89 秒与 13.96 秒：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/detail-air-house-save-race.test.tsx \
  src/pages/orders/detail-air-house-save-stale.test.tsx \
  --reporter=json --outputFile=/tmp/air-front-mapping-split-tests.json
```

4. 最后补充组织身份必须存在的基线门禁后，单独重跑 `air-save-baseline.test.ts` 10/10 通过。
5. 16 个负责的 TS/TSX 文件执行 `pnpm --dir web exec biome check <对应相对路径>`：全部通过，无警告。
6. 使用 `pnpm --dir web exec antd info Form --format json` 核对 API；对 `AirExportDetailFeatures.tsx`、`detail.tsx`、`detail-air-house-save.fixtures.tsx`、`detail-change-notice-air.test.tsx` 分别执行 `antd lint <单文件> --format json`：均 0 问题。
7. `git diff --check` 通过。
8. `pnpm --dir web tsc` 第一轮完整通过；最后一轮只报并行新增的 `templates/components/air/AirWaybillFields.test.tsx:57` 传入 `getByRole` 的 `exact` 不属于 `ByRoleOptions`，已通知主会话处理，未越界修改。负责文件没有类型错误。最终全量门禁由主会话复核。

## 剩余验证边界

- 两份 Playwright E2E 文件已切换契约，本代理未执行真实浏览器/API 场景；后端与主会话统一验收 AE DIRECT/HOUSE 创建、财务、终止恢复等闭环。
- 本代理没有修改服务端、生成物、迁移、主单工作台、结构化正文控件，也没有提交或清理其他代理文件。
