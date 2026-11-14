/** 筛选设置只涉及字段标识与显隐，不含业务数据；公司工作区、权限范围不是筛选字段，不得声明。 */
export interface FilterSettingsField {
  name: string;
  label: string;
  /** 页面声明的默认可见性；新增字段在没有用户明确选择时按它补入。 */
  defaultVisible: boolean;
  /** 必须可见的字段，不允许通过设置移除。 */
  required?: boolean;
}

export interface FilterSettingsScope {
  userId?: string;
  organizationId?: string;
}

/** 浏览器本地偏好：用户明确勾选保留与明确移除的字段名。 */
export interface FilterSettingsPreference {
  visible: string[];
  hidden: string[];
}
