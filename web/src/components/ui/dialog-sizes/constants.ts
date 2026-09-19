/**
 * 全局 Modal 与 Drawer 弹窗标准尺寸规范
 *
 * 杜绝在各业务页面硬编码 440, 540, 560, 580, 620, 750, 820, 850, 920, 1040, 1050, 1060 等随机数值。
 * 统一收敛为规范的 T-shirt 尺寸阶梯。
 */

export const MODAL_SIZE = {
  /** 520px: 单列简易输入、扫码、审批确认/驳回、密码重置 */
  SM: 520,
  /** 680px: 标准双列表单、业务编辑、费用录入、规则编辑 */
  MD: 680,
  /** 960px: 复杂多列表单、网格配置、批量导入配置 */
  LG: 960,
  /** 1200px: 跨表协同大型工作台 */
  XL: 1200,
} as const;

export const DRAWER_SIZE = {
  /** 600px: 简易历史记录、轻量只读下钻 */
  SM: 600,
  /** 860px: 标准业务详情下钻、履约里程碑、人员与单据详情 */
  MD: 860,
  /** 1080px: 复杂全功能详情、提成规则、多标签抽屉 */
  LG: 1080,
  /** 1200px: 双表协同大型工作台（如批量建账、海运拆单、箱货分配） */
  XL: 1200,
} as const;

export type ModalSizeKey = keyof typeof MODAL_SIZE;
export type DrawerSizeKey = keyof typeof DRAWER_SIZE;
