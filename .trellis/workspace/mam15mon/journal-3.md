# Journal - mam15mon (Part 3)

> Continuation from `journal-2.md` (archived at ~2000 lines)
> Started: 2026-10-21

---



## Session 152: 空运出口表单 UI 对齐海运出口五卡片架构
<!-- trellis-session: v=2 fp=d02ad5aa4fdd89d2 -->

**Date**: 2026-10-21
**Task**: 空运出口表单 UI 对齐海运出口五卡片架构
**Branch**: `main`

### Summary

把空运出口新建/详情表单从四卡片平铺重组为海运同款五卡片架构（业务与客户/订舱与运输/本票货物/分单信息/内部人员）：提取 TemplateFieldGroup 小节竖标组件供两 kind 共用，AirMasterFields 拆为航线/班期/申报子件，备注换 SeaCollapsibleNotesField 可折叠交互（name 支持嵌套路径）。字段与 payload 契约零变化，海运视觉零回归；新增 air-create-layout 结构测试锁定布局。trellis-check 8/8 验收通过，tsc/biome/architecture 全绿。

### Git Commits

| Hash | Message |
|------|---------|
| `632f5320` | refactor: 提取订单模板小节竖标组件 TemplateFieldGroup 供海运空运共用 |
| `234663d3` | refactor: 拆分 AirMasterFields 为航线/班期/申报子件，字段属性逐字保留 |
| `97cb74d7` | feat: 空运出口表单重组为五卡片信息架构对齐海运出口 |
| `201ba2ad` | test: 空运模板布局测试冲刷挂载请求并锁定小节与备注顺序 |
| `82a34fcc` | docs: 归档空运出口表单对齐海运五卡片任务与模板架构导航 |

### Status

[OK] **Completed**


## Session 153: 空运出口危险品三元组对齐海运交互
<!-- trellis-session: v=2 fp=077ed52509b45e21 -->

**Date**: 2026-10-21
**Task**: 空运出口危险品三元组对齐海运交互
**Branch**: `main`

### Summary

空运出口「本票货物」卡接入 SeaDangerousGoodsFields：勾选危险品动效展开 UN NO./CLASS NO./截申报时间，取消清值、UN 四位校验仅选中激活；air form-adapter 按海运范式补三字段序列化与 dayjs 回填。契约与服务端零改动（Order 共享字段，写路径 kind 无关）。trellis-check 6/6 验收通过，展开态断言锁 options 接线；海运路径零变化。

### Git Commits

| Hash | Message |
|------|---------|
| `ef6502c3` | feat: 空运出口补危险品 UN NO. / CLASS NO. / 截申报时间三元组 |
| `a0950af0` | test: 空运危险品布局用例补勾选危险品后展开断言锁定 options 接线 |

### Status

[OK] **Completed**
