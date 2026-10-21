# 执行计划：空运出口危险品三元组

轻量任务，单组实现 + 一组测试，1-2 个提交。

## Step 1：实现

- [ ] `web/src/pages/orders/order-kinds/air-export/form-adapter.ts`：
      `CommonValues` 与 `commonFields` 补 `unNumber` / `hazardClass` /
      `declarationCutoffAt`（对齐 sea-export/form-adapter.ts:198-206 的
      序列化范式；detail 回填经 `commonFields(order)` 自动生效，时间字段
      用 dayjs 转 dayjs 对象回填表单）。
- [ ] `web/src/pages/orders/templates/air-template.tsx`：「本票货物」卡在
      `ProFormCheckbox.Group cargoCategoryIds` 与 `SeaCargoDescriptionFields`
      之间插入 `<SeaDangerousGoodsFields options={props.cargoCategoryOptions}
      disabled={props.readonly} />`（从 `./components/sea/SeaBasicInfoSection`
      导入，与既有 Sea* 导入并列）。

## Step 2：测试

- [ ] `order-kinds/air-export/form-adapter.test.ts`：补 payload 断言
      （填写 DG 三字段后 create/update 携带、未填不携带）。
- [ ] `templates/air-create-layout.test.tsx`：补「本票货物」卡存在
      `sea-dangerous-goods-motion` 区块断言（收起态）。

## Step 3：定向验证与提交

- [ ] `pnpm --dir web tsc`
- [ ] `pnpm --dir web exec biome check` 改动文件
- [ ] `pnpm --dir web exec vitest run src/pages/orders/order-kinds/air-export/form-adapter.test.ts src/pages/orders/templates/air-create-layout.test.tsx src/pages/orders/templates/components/sea/SeaDangerousGoodsFields.test.tsx src/pages/orders/new.test.tsx`
- [ ] `git diff --stat` 确认海运路径零变化
- [ ] Conventional Commits（feat: 前缀，中文说明），实现与测试可合并一个提交。

## 明确不做

- proto/后端/生成物、海运代码、PG/闪点/海洋污染物。
