# 执行计划：空运出口表单 UI 对齐海运出口信息架构

按步骤提交，每步是一个可独立验证的回滚点。禁用 TDD：先实现，后补验证。

## Step 1：提取共享小节组件（海运零变化）

- [ ] 新建 `web/src/pages/orders/templates/components/FieldGroup.tsx`，
      导出 `TemplateFieldGroup`（从 `sea-template.tsx` 的 `SeaFieldGroup`
      原样搬移，含样式常量）。
- [ ] `sea-template.tsx` 删除本地 `SeaFieldGroup`，改用
      `TemplateFieldGroup`；不重排、不改字段。

验证：`pnpm --dir web exec vitest run src/pages/orders/templates/sea-create-layout.test.tsx src/pages/orders/templates/sea-template.test.tsx`

## Step 2：拆分 AirMasterFields 子件（工作台零变化）

- [ ] `AirMasterFields.tsx` 拆出 `AirMasterRouteFields` /
      `AirMasterScheduleFields` / `AirMasterDeclarationFields`，字段 props
      逐字保留；`AirMasterFields` 组合原 5 组 FormRow 输出不变。

验证：`pnpm --dir web exec vitest run src/pages/orders/air-master.permissions.test.tsx`

## Step 3：重组 air-template 五卡片

- [ ] `getAirTemplateSections` 重写为五节（见 design.md §2）：卡1 移除
      订舱号；卡2 三小节 + 两个折叠备注；卡3 去掉操作备注；卡4 分单信息
      （本票分运单）；卡5 人员。
- [ ] 主单关联方式 / 共享主单选择 / 摘要 Alert / 同号提示 / hidden 字段 /
      masterId 预选全部归入卡2「主单与订舱」小节，逻辑不改。
- [ ] 备注换用 `SeaCollapsibleNotesField`。

## Step 4：补结构测试（镜像 sea-create-layout）

- [ ] 新建 `web/src/pages/orders/templates/air-create-layout.test.tsx`：
      断言五卡片标题与顺序、卡2 三个小节竖标标题存在、折叠备注 testid
      存在（`sea-notes-airMaster-notes` 语义按 name 断言）、订舱号位于
      卡2、HAWB 位于卡4。
- [ ] 断言「同输入同 payload」不回归：复用/引用既有
      `order-kinds/air-export/form-adapter.test.ts` 结果即可，不重复实现。

## Step 5：定向验证与提交

- [ ] `pnpm --dir web tsc`
- [ ] `pnpm --dir web exec biome check` 改动文件
- [ ] 定向测试：
      `pnpm --dir web exec vitest run src/pages/orders/templates/air-create-layout.test.tsx src/pages/orders/air-master.permissions.test.tsx src/pages/orders/order-kinds/air-export/form-adapter.test.ts src/pages/orders/new.test.tsx src/pages/orders/detail-draft-lifecycle.test.tsx`
      （后两个共享页面渲染两 kind，锁定海运零回归与空运新结构共存。）
- [ ] 如共享页面测试因分节变化需要调整断言，只改断言不改实现逻辑。
- [ ] 按步骤分组提交（Conventional Commits，`feat:` 前缀，中文说明）。

## 回滚点

- Step 1、2 各自独立成提交，异常单独 revert。
- Step 3+4 一个提交（实现与锁定测试同组）。

## 明确不做

- 不动 `form-adapter.ts`、`server/`、生成物、列表页、工作台页面结构。
- 不渲染空运未暴露的契约字段（bookingNotes/allocationNotes/tradeTerm 等）。
