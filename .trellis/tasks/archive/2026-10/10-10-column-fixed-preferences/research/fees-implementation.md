# 订单费用首尾冻结实施证据

## 产品改动

- `feeColumnPreference.ts`：应收/应付共享偏好增加 `freeze`，保持既有用户/组织存储键。默认仅冻结末尾操作列；读写、解析、默认相等判断包含两个开关，合法 `false` 不被覆盖。已有显隐和排序继续保留，操作列必显且置末，重复排序键去重，财务列仍按权限裁剪。
- `orderFeeOptionalColumns.tsx`：显隐排序后调用公共 `applyColumnFreeze`；缺省冻结从输入业务列推导。操作列不受普通列排序或隐藏影响；保留列宽、表头回调与原列对象，首尾重合左侧优先由公共能力处理。
- `OrderFeeTableTabs.tsx`：两个入口共享即时配置和当前首尾中文提示，操作列不进入显隐/排序列表；编辑禁用、财务权限、请求、费用计算和批量操作保持原规则。偏好状态绑定用户/组织/财务可用性，在身份变化的当前渲染直接使用对应配置；保存失败沿用当前页面生效并明确提示的行为。

## 定向验证

命令：`pnpm --dir web exec vitest run src/pages/orders/components/fees/OrderFeeTableTabs.freeze.test.tsx src/pages/orders/components/fees/feeColumnPreference.test.ts src/pages/orders/components/fees/orderFeeColumnFreeze.test.tsx --reporter=json --outputFile=/tmp/column-freeze-fees-tests.json`。

23 个用例通过：16 个偏好单元用例、4 个列应用单元用例、3 个真实组件集成用例。组件套件 10.03 秒，未超过 20 秒约束；JSON 报告在本地临时目录，不提交临时产物。

- 覆盖首/尾分别开启和关闭、显隐与排序后目标变化、操作必显末尾、首尾重合、列宽与 `onHeaderCell` 保留、输入不变。
- 覆盖应收设置→刷新→应付读取同偏好；隐藏费用代码后两张表列头消失，当前首列提示变为费用名称。
- 覆盖同一组件实例组织切换后立即替换列顺序和开关；恢复默认删除当前组织偏好，不改另一组织配置。
- 覆盖写入失败后当前开关继续生效，页面明确提示未持久保存。
- 相邻原费用显隐/金额/编辑禁用及账单投影测试共 16 例已通过；修改文件 Biome 和 `git diff --check` 通过。

几何固定、窄宽切换、真实横向滚动及全量前端门禁由主代理完成；组件测试不作为浏览器几何证据。服务端、生成物和业务数据均未修改。
