/** 单条费用表单值：费用补录弹窗与汇率/金额预览 Hook 共用。 */
export type FeeFormValues = {
  direction?: number;
  feeSettingId: string;
  settlementPartyId: string;
  currency: string;
  unitPrice: string;
  quantity: string;
  quantitySource?: API.FeeQuantitySelection;
  billingUnitId: string;
  exchangeRateOverride?: string;
  note?: string;
};
