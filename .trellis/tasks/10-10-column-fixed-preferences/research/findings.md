# 代码与任务研究

公共types/preference只有order/hidden；Panel按业务fixed分区且仅区内排序；Hook解析key/dataIndex、排除结构列，再应用显隐/顺序。本期按用户明确方向增加表格级首尾冻结开关，首尾要在可见列排序后实际应用，普通列拖动不再受原fixed分区限制。

直接ColumnSettingsEntry产品使用者只有订单费用，其他入口经共享Hook；五模板也经此Hook。订单费用转存、默认相等、财务裁剪和排序应用均需同步两个开关，否则会漏存。

users/userColumns.tsx操作列无稳定key；MasterData/Setting/SubEntity的action和OrderList的option被structuralKeys排除。通过明确元数据识别尾部操作列，保持必显与位置约束；序号、方向、选择/展开不等同操作列。费用option已必显；金额计算、账单权限、批量动作不属于本次变更。

公共Hook缺省scope是anonymous/default，多数页面/模板省略scope。现有AppProvider.tsx提供认证真相，useInitialState无Provider时抛错，导出context供测试注入。订单费用fees.tsx已传真实scope。补公共隔离需测试当前组织切换且无旧偏好短暂闪现。

前置Responsive布局按实际空间取消fixed且不修改原列，已有真实验收，可以消费。财务调宽先处理原列再应用列设置，需验证冻结开关与onHeaderCell保留。

OrderListTemplate.tsx:75/106与:394/404/425存在多列原业务固定；本期默认从原fixed推导首尾开关，普通列统一按当前首尾单列冻结，不保存任意列固定映射。cashflows.tsx:241/247/401同样需覆盖默认和取消冻结路径。

已通过本地官方CLI读取Table与Select API；实施查询实际使用的Table/Switch API。无需升级CLI/依赖。相关历史09-24-unified-table-headers和09-25-column-settings-popover当时排除普通列改固定，本次用户需求覆盖该范围限制，其余即时交互继续遵循。相关文档已在上下文，session-insight明确此时无需重复mem检索。
