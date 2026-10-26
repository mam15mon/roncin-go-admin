# 表格列设置与逐列冻结

### 使用范围与入口

所有表格（ProTable、EditableProTable、普通 Table、抽屉/弹窗/嵌套表及导入预览）
通过 `@/components/ui/column-settings` 的 `useColumnSettings` 接入；关闭 ProTable
原生 `options.setting`，同一表格只保留一个齿轮入口。订单费用保留专有偏好存储，
复用 `ColumnSettingsEntry` 和公共冻结函数，应收/应付共享；财务台账高级偏好保持
原契约。

### 公共签名与配置

```ts
interface ColumnFreezeValue { left: string[]; right: string[] }
interface ColumnSettingsPreference {
  order: string[];
  hidden: string[];
  freeze?: Partial<ColumnFreezeValue>;
}
interface ColumnSettingsValue extends ColumnSettingsPreference {
  freeze?: ColumnFreezeValue;
}
// 页面：useColumnSettings({ tableKey, columns, scope?, ... }) -> { columns, entry }
// 专有消费者：applyColumnFreeze(columns, freeze, { structuralKeys? })
// 面板和表格共用区域顺序：orderColumnKeys(keys, freeze)
```

冻结按稳定字段 key 保存，左右互斥；区内顺序由 order 决定。业务列最终按
「左侧冻结 → 不冻结 → 右侧冻结」排列，面板使用同一规则并显示连续编号。
`deriveColumnFreezeDefaults` 保留业务原声明的全部默认 fixed 字段；
`resolveColumnFreeze` 校验左右字段列表并过滤未知/重复 key，左右重复时左侧优先。
未设置时使用业务默认；显式空数组表示该侧不冻结任何字段。

旧 first/last 开关不转换为字段选择，不添加兼容层。已有合法显隐/顺序继续经
原解析规则应用，不删除浏览器存储。

### 交互、顺序与身份契约

- 齿轮打开充分利用视口的大弹窗，显隐、排序、冻结和恢复默认即时生效并持久化；
  关闭保留已生效配置。没有保存确认步骤；`advanced` 由「更多设置」打开独立高级
  弹窗，高级内容自行管理状态。关闭、禁用及切换高级设置时同步使原面板会话失效，
  拒绝关闭动画残留的旧回调；不能只依赖卸载或 disabled 校验。
- 列设置、独立高级弹窗和内部下拉统一由入口的局部 `ConfigProvider` 指定挂载点：
  普通模式挂到 `document.body`，浏览器全屏模式挂到 `document.fullscreenElement`。
  不继承 ProTable 为全屏设置的表格内部挂载点，否则弹窗与遮罩被表格层叠上下文
  限制，侧栏/汇总条会盖住弹窗；只提高弹窗 z-index 无效。须同时验证普通和全屏
  模式的弹窗、冻结下拉、高级设置，不能只检查主弹窗的挂载位置。
- 字段按视口宽高和数量多列铺排，从左到右、从上到下编号；常用桌面尺寸下费用
  实际字段及 40 字段集合一次可见，不恢复旧 420px 内部滚动列表。超窄或极端数量
  时保证可访问性，不能裁掉字段或过度缩小文字来掩盖溢出。
- 展示层突出字段名称与显隐勾选，序号和冻结选择采用弱边框样式，hover/focus 仍可
  识别可操作状态；不能隐藏这些控件来简化界面。空冻结区收成单行拖入提示，保留
  可投放目标；长说明放到可聚焦的「排序说明」提示，底部明确即时生效。
  搜索期间只展示有匹配字段的分区，无匹配时使用统一空状态；清空后恢复三个分区
  和完整顺序，不能清掉显隐或冻结配置。
- 普通字段支持左右任意多列冻结、跨行跨区拖动和精确目标序号移动。投放位置明确；
  跨区域移动改变该字段冻结侧，取消拖动不保存悬停状态。控件点击不误触拖动。
- 搜索只筛选面板条目，保留完整顺序号，搜索期间禁用排序，不改变冻结结果。
  至少保留一列业务字段；必显字段不可隐藏，隐藏再显示保留其冻结选择。
- 结构列保留必要锚点，不能自由重排或隐藏；操作列可选右侧冻结或不冻结。
  稳定 key 取 `key ?? dataIndex`；结构身份来自标准 key action/option、
  `valueType: 'option'` 或显式 structuralKeys，序号识别 index/indexBorder；
  选择和展开使用官方 sentinel。禁止按中文标题猜身份。ReactNode 列名经
  titleOverrides 提供纯文本名称，不读取 render 结果。
- 本地 key 为 `roncin:column-settings:v1:<表格标识>:<用户>:<组织>`，tableKey 不含
  实体 ID。缺省 scope 消费真实认证上下文，显式 scope 优先；账号/组织变化当次
  渲染重新读取对应偏好，不等待 effect 清理旧身份数据。费用继续使用专有存储键。
- 最终列交给 Responsive 层计算列宽和窄容器临时解冻；恢复宽度后重新冻结，临时
  解冻不回写偏好，不隐藏业务字段；保留列宽、onHeaderCell、编辑和请求回调。
- Responsive Pro/Editable 缺省采用受控空 columnsState.value，防止 Pro 首次 fixed
  缓存覆盖新列定义；显式 columnsState 优先。交给 Pro 时将 left/right 转为官方
  start/end，保留既定顺序和结构锚点；禁止通过表格 key 重挂修复 fixed。

### 统一列宽交互

- 调宽归属 `ResponsiveTable` / `ResponsiveProTable` / `ResponsiveEditableProTable`，
  业务模板只传最终显示列，不再维护第二套 `colWidths` 或表头拖拽组件。
- 普通业务叶子列（含冻结列、末尾操作列）默认支持表头边界拖动；选择/展开哨兵、
  序号列和分组头无手柄，分组叶子列递归处理。标识取 `key ?? dataIndex`，复合
  dataIndex 采用确定编码；缺少稳定标识的列不按位置或中文标题猜身份。
- 宽度保存于表格实例，重排/隐藏再显示仍对应原字段，刷新后恢复业务初值；
  不写入 `ColumnSettingsPreference`，不新增跨刷新持久化。
- 拖动以表头实际宽度开始（短表铺满、百分比列均不能直接用声明值当像素），
  操作开始时测量同表叶子和辅助列快照，避免少列表格重新按声明宽度分配而跳动；
  提交范围50–600px，过程中只更新参考线/像素提示，松手一次提交。双击按
  `max(initialWidth * 1.6, 240)` 展开并约束至600px，再次双击还原；这是展开/还原，
  不测量单元格内容来计算最佳宽度。未指定 `tableLayout` 的表格仅在用户调宽后
  采用官方fixed布局，调用方显式布局优先；初始布局保持原样。短表向左收窄后仍按
  `min-width:100%` 铺满，少量剩余空间由浏览器分配，实际列宽可能略高于提交像素。
- 调宽先应用再计算响应式横向宽度和临时解冻；保留官方 `onHeaderCell` 属性、
  事件、样式及 `components` 自定义内容，固定表头的 sticky 定位不能被 relative 覆盖。
  手柄阻止排序/行事件；卸载、窗口失焦及Escape取消清理参考线、帧任务、监听和body样式。
  不重挂表格、不触发业务请求、不清空选择/分页/搜索/编辑输入。
  Antd6固定定位由CSS类提供，不能以 `style.position ?? relative` 注入第二套定位；
  手柄留在本列内部，避免相邻冻结序号遮挡。
- 验证公共三类组件、原费用模板、稳定标识重排/隐藏、回调/组件组合、固定列和编辑
  节点保留；真实浏览器验证实际调宽与窄容器末列可达，不以col样式代替几何证据。

### 校验与错误行为

| 条件 | 行为 |
| --- | --- |
| freeze 缺失、旧形状或某侧非法 | 使用相应业务默认；保留合法显隐与顺序 |
| 左右列表重复 key | 去重，跨侧重复以左侧优先 |
| 未知 key、权限已裁剪字段 | 过滤，不渲染越权字段 |
| 必显列被写入 hidden | 保持可见 |
| 已隐藏字段仍被选择冻结 | 当前不渲染，恢复显示后按保存选择冻结 |
| localStorage 写入失败 | 当前视图保留效果，提示未持久化，连续失败不刷屏 |
| 编辑中禁用设置 | 入口禁用并解释原因，已打开的面板关闭且不能继续修改 |
| 窄容器空间不足 | 临时解除 fixed，保存的字段冻结配置不变 |

### 场景与验证

正常：费用名称、结算单位同时左冻结，操作右冻结，横向滚动仍正确定位；字段
跨区域拖动后按新区域排序，面板编号与表格一致。默认：恢复业务声明的全部
默认冻结字段、顺序与显隐。错误：禁止只更新公共 UI 而遗漏费用专有存取和列应用，
禁止把组织 A 的配置短暂展示到组织 B，禁止只凭 CSS 无滚动条断言全部可见。

实施后覆盖多列冻结、空数组、非法值、未知/重复 key、结构锚点、隐藏重显、排序、
拖动取消、精确移位、键盘、作用域切换、存储失败及恢复默认。真实浏览器检查
1366×768、1440×900、1920×1080 下全部字段的实际矩形，40 字段无内部滚动；
覆盖 390px 窄屏及窄→宽恢复，三类公共表格的多列固定、请求和编辑状态保留。
费用额外覆盖应收/应付共享、权限裁剪、编辑禁用、合并行和调宽。

```ts
// 正确：字段冻结绑定身份，面板和表格消费同一个最终区域顺序。
const freeze = { left: ['feeSettingId', 'settlementPartyId'], right: ['option'] };
const effective = applyColumnFreeze(visibleOrderedColumns, freeze, {
  structuralKeys: ['option'],
});
// 错误：按当前数组下标或旧 first/last 开关猜测用户选择的冻结字段。
```
