# 在配舱信息中增加只读关联分单号展示

## Goal

在海运出口订单模板（包括新增与详情）的“配舱信息”区块中增加只读的“关联分单号”展示。业务人员无需切换到“提单信息”即可快速确认当前订单的分单配置，同时严格保持分单编辑权唯一归属提单信息。

## Requirements

1. **展示位置与只读防护**：
   - 在 `SeaTransportSection.tsx`（配舱信息）第 1 行紧随「MBL 主单号」与「实际签发主体」后增加「关联分单号」展示项（栅格 `.col-5`）。
   - 仅作为纯数据呈现视图，禁止包含可编辑输入框或就地修改入口，分单号的录入与修改唯一保留在“提单信息”区块。

2. **状态与展示分支**：
   - **直单模式**（`seaDocumentStructure === DIRECT`）：展示标签 `<Tag color="success">直单，无HBL</Tag>`。
   - **已录入分单**（`houseNos.length > 0`）：多张 HBL 全部以标签形式展示（如 `<Tag color="processing">{no}</Tag>`），支持灵活换行。
   - **未录入分单**（非直单且暂无分单号）：展示提示文本 `<Text type="secondary">暂未录入分单号</Text>`。

3. **表单动态联动**：
   - 通过 `Form.useWatch` 监听表单的 `seaDocumentStructure` 与 `seaHouseBills`（及 `seaDocumentSummary`），当下方的“提单信息”添加、修改、移除分单或切换直单时，配舱信息区域即时联动更新。

4. **单测覆盖**：
   - 在 `sea-template.test.tsx` 中补齐针对“关联分单号”展示在直单、分单标签、未录入状态下的渲染测试。

## Acceptance Criteria

- [x] 配舱信息第 1 行包含「关联分单号」只读展示项。
- [x] 直单模式下展示“直单，无HBL”。
- [x] 存在分单时以标签组形式正确展示各个分单号。
- [x] 尚未录入分单时展示“暂未录入分单号”。
- [x] 不可在此处编辑分单号，不产生非预期表单输入字段。
- [x] `pnpm --dir web test` 与 `pnpm --dir web tsc` 通过。

