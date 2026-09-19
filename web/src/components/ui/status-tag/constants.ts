/**
 * 全局业务 Tag 色彩与语义规范
 *
 * 1. 收支方向 (Direction):
 *    - 应收 (RECEIVABLE): 'green' (资产流入)
 *    - 应付 (PAYABLE): 'volcano' (成本负债流出)
 *
 * 2. 核心单据生命周期 (Status):
 *    - 草稿 (DRAFT): 'default' (低调灰色，表示未生效)
 *    - 已确认 / 有效 (CONFIRMED / ACTIVE): 'blue' (品牌蓝，表示已生效/执行中)
 *    - 已完成 / 已结清 (SETTLED / ISSUED / PAID): 'green' (成功绿，表示财务已结清)
 *    - 已取消 / 已作废 (CANCELLED / VOID / REJECTED): 'red' (警告红，表示已被废止)
 *    - 已反转 / 已红冲 (REVERSED / RED_FLUSHED): 'volcano' (撤销反转)
 *    - 锁定 / 预警 (LOCKED / WARNING): 'orange' (预警态)
 */

export const DIRECTION_TAG_COLOR = {
  RECEIVABLE: 'green',
  PAYABLE: 'volcano',
} as const;

export const DIRECTION_TEXT: Record<string, string> = {
  RECEIVABLE: '应收',
  PAYABLE: '应付',
};

export const COMMON_STATUS_COLOR = {
  DRAFT: 'default',
  CONFIRMED: 'blue',
  SETTLED: 'green',
  CANCELLED: 'red',
  REVERSED: 'volcano',
  LOCKED: 'orange',
} as const;
