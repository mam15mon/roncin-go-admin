/** 全站列偏好只包含字段标识和展示配置，不包含业务数据。 */
export interface ColumnSettingsField {
  key: string;
  title: string;
  lockVisible?: boolean;
  /** 业务声明的默认冻结位置。 */
  fixed?: 'left' | 'right';
}

/** 按字段身份保存冻结；左右两侧互斥，区内顺序由 order 决定。 */
export interface ColumnFreezeValue {
  left: string[];
  right: string[];
}

export interface ColumnSettingsPreference {
  order: string[];
  hidden: string[];
  freeze?: Partial<ColumnFreezeValue>;
}

export interface ColumnSettingsValue extends ColumnSettingsPreference {
  freeze?: ColumnFreezeValue;
}

export interface ColumnSettingsScope {
  userId?: string;
  organizationId?: string;
}
