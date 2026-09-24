/**
 * 全站统一列设置的公共类型。
 *
 * 列设置以「字段元数据 + 用户配置」表达，配置只保存完整顺序与隐藏集合，
 * 不含任何业务数据；固定（fixed）能力仅当业务列本身声明时透出。
 */

/** 单列设置元数据：稳定 key、纯文本名称与约束标记。 */
export interface ColumnSettingsField {
  /** 稳定列标识，与列定义的 key/dataIndex 对应，不随展示顺序变化。 */
  key: string;
  /** 设置界面展示的中文名称，必须是纯文本，不使用 ReactNode。 */
  title: string;
  /** 必显列：用户不可隐藏（录入必备、合并单元格对齐依赖等）。 */
  lockVisible?: boolean;
  /** 固定列区域；排序仅允许在同侧区域内进行。 */
  fixed?: 'left' | 'right';
}

/** 用户提交的列配置：完整顺序（含隐藏列）+ 隐藏 key 集合。 */
export interface ColumnSettingsValue {
  order: string[];
  hidden: string[];
}

/** 列偏好在同一浏览器内的隔离范围：用户 + 当前组织。 */
export interface ColumnSettingsScope {
  userId?: string;
  organizationId?: string;
}
