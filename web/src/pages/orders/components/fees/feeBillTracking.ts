/**
 * 订单费用页的关联账单投影视图。
 *
 * 由订单费用页在「具备 system.finance.fee.read 且可操作该订单组织」时通过
 * 订单维度财务接口构建（一次查询，应收/应付两表共用），按费用 ID 叠加为
 * 只读展示；无财务读取权限时不构建、不请求，不影响订单费用自身功能。
 */
export interface FeeBillTrackingView {
  state: 'loading' | 'error' | 'ready';
  byFeeId: Record<string, { billNo?: string; financialProgress?: number }>;
  onRetry: () => void;
}
