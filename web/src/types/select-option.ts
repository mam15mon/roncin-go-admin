/**
 * 通用下拉候选项基础类型：只承载跨领域通用字段；
 * 业务契约字段（散客、信用超额等）由领域类型组合补充。
 */
export type SelectOption = {
  label: string;
  value: string;
  code?: string;
  name?: string;
  disabled?: boolean;
};
