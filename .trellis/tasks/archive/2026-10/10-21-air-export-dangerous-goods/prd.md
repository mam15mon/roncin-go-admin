# 空运出口危险品三元组对齐海运交互

## Goal

空运出口补危险品 UN NO. / CLASS NO. / 截申报时间三字段：复用海运
`SeaDangerousGoodsFields` 联动交互（勾选出现、取消清值、校验仅选中时激活），
air form-adapter 补序列化，契约不动（三个字段已在共享 `Order` 消息上）。

## 背景与已核事实

- 竞品 HGJ 的海/空字段模板（`GET /defined/field/orderDetail`）实证危险品三字段
  连排且海空一致：UN NO.（order 30）/ CLASS NO.（40）/ 截申报时间（50），
  与我们海运实现一一对应，用户已拍板照抄自己的海运模式。
- 契约层：`Order.un_number`(46) / `hazard_class`(47) / `declaration_cutoff_at`(59)
  均在共享 `Order` 消息与 Create/UpdateOrderRequest 上，空运订单契约已支持。
- 服务端写路径 kind 无关：`order_write.go:62,66`、`order_write_draft.go:228,232`
  直接 `SetUnNumber`/`SetDeclarationCutoffAt`，无海运分支。**后端零改动。**
- 前端差距：air 模板不渲染 DG 字段（空运勾「危险品」品类只是标签），
  air form-adapter 未序列化。
- 空运品类多选已含 `DANGEROUS`（cargo_category 字典共用），联动判定机制
  与海运同源。

## Requirements

1. 空运出口新建/详情表单的「本票货物」卡，在货物品类之后、货物描述之前
   渲染危险品字段组（与海运「货物与商业」卡内顺序一致）。
2. 复用 `SeaDangerousGoodsFields`（直接从 `SeaBasicInfoSection.tsx` 导入，
   沿用空运已导入 `SeaCustomerField` 等 Sea* 共享件的既定先例；不提取、
   不改名、不复制）。组件行为保持海运现状：勾选危险品类目时动效展开、
   取消勾选自动清空三字段值、UN NO. 四位数字校验仅在选中时激活、
   `aria-hidden`/`inert` 无障碍语义不变。
3. air form-adapter：`commonFields` 补 `unNumber` / `hazardClass` /
   `declarationCutoffAt` 序列化（对齐海运范式：`trim() || undefined`，
   时间 `dayjs(...).toISOString()`），create / update / detail 回填随
   `commonFields` 一并生效。
4. 详情只读态：三字段随表单 `readonly` 禁用，展开态跟随品类勾选数据。

## 非目标（Out of Scope）

- 不加 PG（包装类别）、闪点、海洋污染物字段——PG 待 proto 方案单独过目
  后另开任务；闪点缓；海洋污染物不做（用户已拍板）。
- 不改 proto / 后端 / 生成物；不动海运任何代码。
- 不动服务类型字典（「危险品」服务类型 `DANGEROUS_SERVICE` 与品类
  `DANGEROUS` 是两个既有字典项，本任务只消费品类）。

## Acceptance Criteria

- [ ] 空运新建表单：货物品类勾选「危险品」后动效展开 UN NO. / CLASS NO. /
      截申报时间三字段；取消勾选自动清值；未选中时 UN 校验不拦截提交。
- [ ] 空运详情表单：订单带 DG 数据时三字段按数据展开并只读展示；无数据
      且品类未含危险品时收起。
- [ ] 提交载荷：勾选并填写后 create/update payload 含三个字段（值与格式
      对齐海运：日期 ISO 字符串）；未填写时不携带（undefined）。
- [ ] 海运出口相关测试全部通过、代码零变化（`git diff` 海运路径为空）。
- [ ] `air form-adapter.test`、`air-create-layout.test` 补对应用例并通过。
- [ ] `pnpm --dir web tsc`、改动文件 Biome、定向测试通过。

## Notes

- 组件级行为（展开动效、清值、校验激活）已由存量
  `SeaDangerousGoodsFields.test.tsx` 锁定，本任务只需集成层验证，不重复
  组件测试。
