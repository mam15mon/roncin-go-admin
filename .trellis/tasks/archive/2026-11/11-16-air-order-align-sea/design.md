# 设计：空运排版对齐海运

## 边界

- 只改 `web/src/pages/orders/templates/air-template.tsx` 与
  `templates/components/air/{AirCommercialFields,AirMasterFields}.tsx` 的 JSX 组合；
  字段 name、rules、只读继承、adapter 与载荷不变。
- AE/AI 共用构建器，一次对齐两者；新建与详情仍是同一构建器，`isDetail` 只影响字段能力。

## 结构

- `AirCommercialFields.tsx` 改为返回字段片段对象 `getAirCommercialFields(props)`
  （参照 `getSeaBaseInfoFields`），由模板按海运同序摆放；备注抽为
  `AirNotesFields`（四个 `SeaCollapsibleNotesField`，`FormRow cols={4}`）。
- `AirMasterTransportSection` 首行：单证模式（`Form.Item` 格内，详情 32px 只读标签）、
  订舱号、订舱代理、国外代理；身份/关联逻辑原样保留；时刻与申报合并为单个 6 列网格；
  末尾为「备注」小节，替代原独立操作备注。
- 主单备注仍属主单字段（仅新建主单时出现），不并入本票备注。

## 取舍

- 不抽「海空共用字段片段」：海运片段带船公司、船代等领域依赖，强行共用会产生分支；
  两边保持同序同规格即可，后续出现真实复用再提取。

## 回滚

纯前端布局，按提交 revert。
