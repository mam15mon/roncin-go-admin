# 执行计划

## 实施前

- [x] 用户确认：没有设置入口的表格也全部补齐。
- [x] 盘点所有 Table、ProTable、EditableProTable 与封装模板，完善 research/entry-inventory.md，逐项记录页面／入口、能力、存储、稳定标识、必显规则、分组／合并单元格、迁移结果；清单已附静态扫描初稿，不能将调用点数当作运行时表格总数。
- [x] 查阅当前安装的 Ant Design／ProComponents API；读取 roncin-web-stack、trellis-before-dev 及相关前端规范。
- [x] 用户评审最终方案后才运行 task.py start（用户已批准实施）。

## 实施顺序

1. 实现公共设置弹窗、字段行、草稿交互和标准／增强版；实现后补定向交互测试，禁止 TDD。
2. 迁移订单费用：保留权限、必显、作用域与编辑保护；验证搜索排序、保存／取消／恢复默认。
3. 迁移财务费用台账：适配服务端偏好与高级设置，移出公共 UI 的业务 API；验证请求失败不丢草稿、保存后回显与重置取消。
4. 迁移主数据与订单列表等 ProTable 内置入口，再按清单处理管理、客商、工作台、财务其他页面及详情弹层；保留各表已有固定列等能力。
5. 为所有原无入口表格补齐标准版，覆盖普通 Table、options=false、无工具栏、嵌套表、导入预览及编辑表；补充用户／组织／表格隔离的本地存储，验证分组、汇总、必填列和弹层焦点。
6. 清理无调用的旧配置 UI；更新 component-guidelines.md 与 capability-navigation.md，逐项关闭迁移清单。

## 验证

- 定向测试：`pnpm --dir web exec vitest run src/pages/orders/components/fees/OrderFeeTableTabs.feeColumns.test.tsx src/pages/orders/components/fees/feeColumnPreference.test.ts`，并运行新增公共设置测试、受影响财务／模板测试。
- 修改文件 Biome 检查、`pnpm --dir web tsc`、`git diff --check`；每组可验证修改提交 Git。
- 任务最终前端验收：`pnpm run check:web`。不涉及构建入口或依赖变化时不固定执行 build。
- 浏览器验证普通主数据、订单列表、订单费用应收／应付、财务费用增强设置和抽屉入口，覆盖空结果、长标题、必显／无权限列、固定列、保存失败、取消重开、本地存储失败、刷新恢复、作用域隔离、嵌套弹层焦点及两档屏幕尺寸；记录截图与检查结果。
- 视觉样式不编写镜像断言；测试重点为草稿提交边界、字段映射、权限约束、排序与保存失败路径。单个重型测试文件超 20 秒时拆分。

## 关键风险文件

- `components/ui/master-data-template/MasterDataTemplate.tsx` 与 `order-list-template/OrderListTemplate.tsx`：影响多个页面，重点检查重复入口及列状态。
- `components/ui/finance-ledger-template/TableColumnConfigModal.tsx` 与 `pages/finance/fees/index.tsx`：保存／恢复默认语义及高级设置映射。
- `pages/orders/components/fees/OrderFeeTableTabs.tsx` 与 `feeColumnPreference.ts`：应收应付同步、编辑保护及用户／组织隔离。
