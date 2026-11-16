# 实施计划：空运排版对齐海运

## 技术要点

- 纯前端布局重组，只移动 JSX 位置，不改字段 name、规则与 adapter。
- 拆分 `AirCommercialFields.tsx`：按卡片归属导出细粒度片段
  （如 `trade/payment/contract/agents/shipmentMode/cargoValue/insurance/factory/
  times/description`），由 `air-template.tsx` 与 `AirMasterTransportSection`
  按海运同序组合；参考 `getSeaBaseInfoFields` 的「字段片段对象」写法。
- 新增空运「备注」小节组件，四个 `SeaCollapsibleNotesField`，`FormRow cols={4}`。
- 航空单证模式字段改为 `Form.Item label="航空单证模式"` 的格内呈现，详情用
  与海运 `SeaDocumentModeValue` 同规格的 32px 高只读标签。
- 订舱代理/国外代理从业务卡移入「主单与订舱」小节；该小节在新建与详情、
  NEW/EXISTING 主单模式下都需渲染这些本票字段。

## 步骤

1. 重组 `AirCommercialFields.tsx` 为字段片段，修正备注行 span 缺陷（备注迁出）。
2. 调整 `AirMasterFields.tsx` 的 `AirMasterTransportSection`：单证模式入格、
   订舱号/代理同行、时刻与申报合并、追加备注小节、操作备注改由备注小节承载。
3. 调整 `air-template.tsx` 业务卡与货物卡顺序（R1、R5）。
4. 更新测试：`templates/air-create-layout.test.tsx`、`air-transport-summary`、
   `air-document-mode`、`air-preselected-master` 等受影响用例按新归属断言。
5. 同步规范：`.trellis/spec/web/frontend/air-order-commercial-fields.md`
   （字段归属与「操作备注」入口描述）、`air-order-workspace.md`。

## 验证

```bash
pnpm --dir web exec vitest run src/pages/orders/templates src/pages/orders/order-kinds/air-export src/pages/orders/order-kinds/air-import
pnpm --dir web exec biome check <修改文件>
pnpm --dir web tsc
git diff --check
```

浏览器：`pnpm dev` 后打开 AE/AI 新建与详情，1440px、1200px 截图对照海运出口。

## 回滚点

每步一个提交；仅前端布局，回滚即 revert 对应提交。
