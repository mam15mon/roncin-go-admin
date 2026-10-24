export const SEA_SHIPMENT_MODE = {
  TRADITIONAL_FORWARDING: 1,
  CROSS_BORDER: 2,
} as const;

export const SEA_SHIPMENT_TYPE = {
  FCL: 1,
  LCL: 2,
  BREAK_BULK: 3,
} as const;

export type SeaOrderPolicyInput = {
  shipmentMode?: number;
  shipmentType?: number;
};

export type SeaOrderFormPolicy = {
  modeLabel: string;
  showConsolidationReference: boolean;
  showContainerPlan: boolean;
  showRevenueTon: boolean;
  emphasizeCrossBorderCompliance: boolean;
};

/**
 * 海运表单策略只描述界面和草稿提示，不替代服务端领域校验。
 */
export function resolveSeaOrderFormPolicy(
  input: SeaOrderPolicyInput,
): SeaOrderFormPolicy {
  const crossBorder = input.shipmentMode === SEA_SHIPMENT_MODE.CROSS_BORDER;
  const shipmentType = input.shipmentType ?? SEA_SHIPMENT_TYPE.FCL;

  return {
    modeLabel: crossBorder ? '跨境' : '集运',
    showConsolidationReference: shipmentType === SEA_SHIPMENT_TYPE.LCL,
    showContainerPlan: shipmentType !== SEA_SHIPMENT_TYPE.BREAK_BULK,
    showRevenueTon: shipmentType === SEA_SHIPMENT_TYPE.BREAK_BULK,
    emphasizeCrossBorderCompliance: crossBorder,
  };
}
