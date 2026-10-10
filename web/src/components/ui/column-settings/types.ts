/**
 * 全站统一列设置的公共类型。
 *
 * 列设置以「字段元数据 + 用户配置」表达，保存顺序、显隐与首尾冻结，
 * 不含任何业务数据。
 */

/** 单列设置元数据：稳定 key、纯文本名称与约束标记。 */
export interface ColumnSettingsField {
  /** 稳定列标识，与列定义的 key/dataIndex 对应，不随展示顺序变化。 */
  key: string;
  /** 设置界面展示的中文名称，必须是纯文本，不使用 ReactNode。 */
  title: string;
  /** 必显列：用户不可隐藏（录入必备、合并单元格对齐依赖等）。 */
  lockVisible?: boolean;
  /** 业务默认固定位置，仅用于推导首尾冻结默认值，不限制排序。 */
  fixed?: 'left' | 'right';
}

/** 表格级首尾冻结配置；首尾由实际可见列顺序决定。 */
export interface ColumnFreezeValue {
  first: boolean;
  last: boolean;
}

/** 实际冻结目标的纯文本名称，包含不进入设置的末尾操作列。 */
export interface ColumnFreezeTargetTitles {
  first?: string;
  last?: string;
}

/** 存储读取结果；每侧冻结独立校验，未设置的一侧由业务默认补齐。 */
export interface ColumnSettingsPreference {
  order: string[];
  hidden: string[];
  freeze?: Partial<ColumnFreezeValue>;
}

/** 当前生效配置：首尾开关已解析，提交时保存两个布尔值。 */
export interface ColumnSettingsValue extends ColumnSettingsPreference {
  freeze?: ColumnFreezeValue;
}

/** 列偏好在同一浏览器内的隔离范围：用户 + 当前组织。 */
export interface ColumnSettingsScope {
  userId?: string;
  organizationId?: string;
}
