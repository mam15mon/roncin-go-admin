# 表格列设置与首尾冻结

### 使用范围与入口

所有表格（ProTable、EditableProTable、普通Table、抽屉/弹窗/嵌套表、导入预览）
通过 `@/components/ui/column-settings` 的 `useColumnSettings` 接入；
ProTable 原生 `options.setting` 关闭，同一表格只保留一个齿轮入口。
无工具栏的表格把入口放上方右侧。订单费用保留专有偏好存储，复用
`ColumnSettingsEntry` 和公共冻结函数；财务台账服务端高级偏好保持原契约。

### 公共签名与配置

```ts
interface ColumnFreezeValue { first: boolean; last: boolean }
interface ColumnSettingsPreference {
  order: string[];
  hidden: string[];
  freeze?: Partial<ColumnFreezeValue>; // 读取时每侧独立校验
}
interface ColumnSettingsValue extends ColumnSettingsPreference {
  freeze?: ColumnFreezeValue; // 生效前补齐两个开关
}
// 页面：useColumnSettings({ tableKey, columns, scope?, ... }) -> { columns, entry }
// 专有偏好消费者：applyColumnFreeze(columns, freeze, { firstExcludedKeys? })
// 与应用规则一致的目标：getColumnFreezeTargets(columns, { firstExcludedKeys? })
```

`deriveColumnFreezeDefaults` 从原业务fixed推导默认两个开关；配置没有freeze时保留
显隐/顺序并使用业务默认。`resolveColumnFreeze` 保留合法false，只对缺失/非法值
使用默认。原普通数据多列fixed统一为当前首尾规则；不保存每列固定映射。
`ColumnSettingsEntry.freezeTargets` 提供当前首尾的纯文本名称，包含末尾操作列。

### 交互、顺序与身份契约

- 齿轮锚定Popover，显隐、排序、冻结及恢复默认都即时生效和持久化；关闭保留
  已生效配置。`advanced` 经「更多设置」打开二级弹窗，高级内容自行维护状态。
- 「冻结首列」「冻结尾列」两个独立开关，目标按权限裁剪、显隐和排序后的实际
  可见顺序计算；首列跳过选择、展开、序号、操作等结构列，尾列可以为操作。
  只有一个目标且首尾重合时左侧优先，保存的两个开关不改写。
- 普通列统一排序，不受原fixed分区约束；结构列保留必要锚点和必显约束。
  搜索仅筛选设置列表，搜索期间禁用排序，不改变冻结目标；至少保留一列业务字段。
- 稳定key取 `key ?? dataIndex`；操作结构身份使用已有标准key `action`/`option`、
  `valueType: 'option'` 或显式 `structuralKeys`，序号识别 `index`/`indexBorder`；
  选择/展开使用官方sentinel。禁止通过中文标题猜身份。ReactNode标题经
  `titleOverrides` 提供设置纯文本名称，不能使用render结果。
- 本地key为 `roncin:column-settings:v1:<表格标识>:<用户>:<组织>`；tableKey不含
  实体ID。同一浏览器缺省scope消费实际认证上下文，显式scope优先；账号/组织变化
  当次渲染就重新读取对应偏好，不等待effect清除旧配置。订单费用继续原存储键，
  应收/应付共享。
- 最终列交给Responsive层计算宽度与窄容器临时解冻；恢复宽度重新冻结，临时解冻
  不回写偏好，不自动隐藏列。保留列宽、onHeaderCell、编辑和请求回调。
- Responsive Pro/Editable缺省使用官方受控空 `columnsState.value`，避免Pro内部首次
  fixed缓存覆盖最新columns；显式columnsState优先透传。输出给Pro时将left/right转换为
  Antd6官方start/end，绕过Pro按left/right再排序，保留序号和哨兵锚点；布局计算仍先用
  原列。禁止通过表格key重挂修fixed，以免丢分页/编辑状态。

### 校验与错误行为

| 条件 | 行为 |
| --- | --- |
| 存储freeze缺失或值非法 | 使用业务默认首尾开关，保留合法显隐/顺序 |
| 未知/重复key、权限已裁剪字段 | 过滤，不渲染越权字段 |
| 必显列被写入hidden | 保持可见 |
| 没有合法首尾目标 | 不设置对应fixed；恢复列后重新计算 |
| localStorage写入失败 | 本次视图保持效果，明确提示未持久化，连续失败不刷屏 |
| 编辑中禁用设置 | 入口禁用并解释原因 |
| 窄容器空间不足 | 临时解除fixed，保存的开关不变 |

### 场景与验证

正常：人员表首列为用户、末列为操作，两个开关分别生效；隐藏用户后首列跟随
邮箱；调整顺序后重新计算。默认：未保存freeze时从业务fixed推导，恢复默认重置
三个维度。错误：不能把隐藏列或勾选框当首列，不能只在浮层显示开关而漏掉费用
专有存取与列应用，不能将组织A配置短暂展示在组织B。

实现后定向覆盖偏好默认/false/错误值、结构列、排序/显隐后的目标、首尾重合、
即时交互和键盘、scope切换及存储失败。真实浏览器验证横向滚动时首尾位置、
390/962/1440/1920、窄弹层和窄→宽恢复；费用额外覆盖应收/应付共享、编辑禁用、
权限裁剪、合并行和调宽。

```ts
// 错误：原始数组的首列可能是序号，隐藏和排序之后也已不是实际首列。
const wrong = columns.map((col, i) => ({ ...col, fixed: i === 0 ? 'left' : false }));
// 正确：先应用当前权限、显隐、顺序与结构约束，再交给公共规则。
const effective = applyColumnFreeze(visibleOrderedColumns, resolveColumnFreeze(value.freeze, defaults), {
  firstExcludedKeys: structuralKeys,
});
```

