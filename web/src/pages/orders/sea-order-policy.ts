export const SEA_SHIPMENT_MODE = {
  TRADITIONAL_FORWARDING: 1,
  CROSS_BORDER: 2,
} as const;

export const SEA_SHIPMENT_TYPE = {
  FCL: 1,
  LCL: 2,
  BREAK_BULK: 3,
} as const;

export const SEA_SERVICE_CODE = {
  BOOKING: 'BOOKING',
  TRUCKING: 'TRUCKING',
  STUFFING: 'STUFFING',
  CUSTOMS_EXPORT: 'CUSTOMS_EXPORT',
  CUSTOMS_IMPORT: 'CUSTOMS_IMPORT',
  OVERSEA_SEGMENT: 'OVERSEA_SEGMENT',
  WAREHOUSING: 'WAREHOUSING',
  INSURANCE: 'INSURANCE',
} as const;

const traditionalForwardingServices = [
  SEA_SERVICE_CODE.BOOKING,
  SEA_SERVICE_CODE.TRUCKING,
  SEA_SERVICE_CODE.CUSTOMS_EXPORT,
  SEA_SERVICE_CODE.STUFFING,
] as const;

const crossBorderServices = [
  SEA_SERVICE_CODE.TRUCKING,
  SEA_SERVICE_CODE.CUSTOMS_EXPORT,
  SEA_SERVICE_CODE.CUSTOMS_IMPORT,
  SEA_SERVICE_CODE.OVERSEA_SEGMENT,
  SEA_SERVICE_CODE.WAREHOUSING,
  SEA_SERVICE_CODE.INSURANCE,
] as const;

export type SeaOrderPolicyInput = {
  shipmentMode?: number;
  shipmentType?: number;
  serviceTypeCodes?: string[];
};

export type SeaOrderFormPolicy = {
  modeLabel: string;
  recommendedServiceCodes: string[];
  showConsolidationReference: boolean;
  showContainerPlan: boolean;
  showRevenueTon: boolean;
  emphasizeCrossBorderCompliance: boolean;
  requireInsuranceValues: boolean;
};

/**
 * 海运表单策略只描述界面和草稿提示，不替代服务端领域校验。
 * 推荐服务是首次默认建议，用户手工调整后不得被模式切换静默覆盖。
 */
export function resolveSeaOrderFormPolicy(
  input: SeaOrderPolicyInput,
): SeaOrderFormPolicy {
  const crossBorder = input.shipmentMode === SEA_SHIPMENT_MODE.CROSS_BORDER;
  const shipmentType = input.shipmentType ?? SEA_SHIPMENT_TYPE.FCL;
  const serviceTypeCodes = new Set(input.serviceTypeCodes ?? []);

  return {
    modeLabel: crossBorder ? '跨境' : '集运',
    recommendedServiceCodes: [
      ...(crossBorder ? crossBorderServices : traditionalForwardingServices),
    ],
    showConsolidationReference: shipmentType === SEA_SHIPMENT_TYPE.LCL,
    showContainerPlan: shipmentType !== SEA_SHIPMENT_TYPE.BREAK_BULK,
    showRevenueTon: shipmentType === SEA_SHIPMENT_TYPE.BREAK_BULK,
    emphasizeCrossBorderCompliance: crossBorder,
    requireInsuranceValues: serviceTypeCodes.has(SEA_SERVICE_CODE.INSURANCE),
  };
}

export function recommendedServiceIDs(
  options: Array<{ code?: string; value: string | number }>,
  shipmentMode?: number,
): string[] {
  const recommendedCodes = new Set(
    resolveSeaOrderFormPolicy({ shipmentMode }).recommendedServiceCodes,
  );
  return options
    .filter((option) => option.code && recommendedCodes.has(option.code))
    .map((option) => String(option.value));
}
