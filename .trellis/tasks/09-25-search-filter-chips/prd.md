# SearchFilterTemplate 已选条件 chips 回显行

## Goal

在共享搜索模板 `SearchFilterTemplate` 中增加「已选条件 chips 回显行」：列表搜索提交后，把生效中的筛选条件回显为可单独删除的标签，并提供一键清空，使全站 grid/bar 模式列表页的筛选状态可见、可快速撤销。

## 背景

- 全站列表搜索统一走 `SearchFilterTemplate`（grid 多字段表单 / bar 快捷单行 / custom 插槽三种模式），均为「点查询才生效」的提交式表单。
- 提交后生效条件无任何回显：费用台账 33 个筛选字段（28 个折叠隐藏）提交后，用户无法确认当前查询口径；收起折叠区域后尤其如此。
- chips 是大厂复杂表格的主流「状态可见性」解法（飞书 / Airtable / Linear 模式），且做进共享模板后全站消费方零改动受益。

## 消费方现状（本次受益面）

| 消费方 | 模式 | 筛选字段 |
|---|---|---|
| `finance/fees`（FeeLedgerSearchFilter） | grid | 33 |
| `finance/bills` | grid | 9 |
| `finance/commissions`（主列表 + 两个 Panel） | grid | 4 / 3 / 2 |
| `partners` | bar | keyword + 2 快捷项 |
| `admin/users`、`admin/roles` | bar | keyword + 快捷项 |

`layout="custom"` 当前无消费方。

## Requirements

1. grid / bar 两种模式下，搜索提交后回显已生效条件 chips；`layout="custom"` 不渲染（无消费方，留待需要时扩展）。
2. chip 文案为 `字段标签: 展示值`，按字段类型取展示值：
   - `input` / `digit`：原始值；
   - `select` / `searchable-select`：选项 label（静态 `options` 直接解析；`request` 远程字段在选择时捕获 label，未捕获到时回退显示原值）；
   - `date` / `date-range`：本地时区人可读日期（复用 `utils/format` 口径，date-range 为 `开始 ~ 结束`）。
3. 空值不产生 chip：`undefined`、`null`、空字符串。
4. 点击 chip 关闭按钮 = 撤销该条件：同步清空表单对应字段 + 以剔除该字段后的完整条件触发 `onSearch`，列表立即按新口径刷新（消费方现有 `onSearch → set 状态整体替换 + reload` 链路自动生效，页面零改动）。
5. chips 行尾提供「清除全部」：等价现有「重置」（清空表单 + 触发 `onReset`）。
6. chips 为模板内部提交态的镜像，不引入新的全局状态，不改变「点查询才生效」语义；不影响折叠/展开与现有按钮行为。
7. 属于 topBar 的外部筛选（所属公司、标签筛选）不在本次范围，维持现状。

## Out of Scope（延后任务）

- 费用台账 28 个折叠字段迁入筛选浮层、订单列表工具栏条件计数（任务 2）。
- 保存视图 / 查询方案与筛选状态 URL 同步（任务 3）。
- 订单列表 `OrderListSearchFilter` 为独立表单实现，不基于本模板，不在本次接入。

## Acceptance Criteria

- [x] grid 模式提交后，非空字段按 items 配置回显为可关闭 Tag；空串/undefined/null 不产生 chip。
- [x] select/searchable-select 的 chip 显示选项 label；request 远程字段显示所选 label。
- [x] date-range chip 显示 `YYYY-MM-DD ~ YYYY-MM-DD` 本地格式。
- [x] 删除单个 chip 后：对应表单字段被清空，`onSearch` 收到不含该字段的完整条件对象，消费方状态整体替换后列表刷新口径正确。
- [x] 「清除全部」清空所有 chips 与表单并触发 `onReset`。
- [x] bar 模式（keyword + quickFilters）同样具备回显与删除能力。
- [x] 现有折叠/展开、重置、远程搜索下拉行为无回归；现有全部测试通过。
- [x] 新增/扩展组件测试覆盖上述场景。
