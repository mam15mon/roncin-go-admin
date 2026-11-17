export const SEA_SHIPMENT_TYPE = {
  FCL: 1,
  LCL: 2,
  BREAK_BULK: 3,
} as const;

export type SeaOrderPolicyInput = {
  shipmentType?: number;
};

export type SeaOrderFormPolicy = {
  showContainerPlan: boolean;
  showRevenueTon: boolean;
};

/**
 * 海运表单策略只描述界面和草稿提示，不替代服务端领域校验。
 */
export function resolveSeaOrderFormPolicy(
  input: SeaOrderPolicyInput,
): SeaOrderFormPolicy {
  const shipmentType = input.shipmentType ?? SEA_SHIPMENT_TYPE.FCL;

  return {
    showContainerPlan: shipmentType !== SEA_SHIPMENT_TYPE.BREAK_BULK,
    showRevenueTon: shipmentType === SEA_SHIPMENT_TYPE.BREAK_BULK,
  };
}
