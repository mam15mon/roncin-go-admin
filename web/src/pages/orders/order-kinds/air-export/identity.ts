import { OrderBusinessType } from '@/enums.generated';

/** 航空候选接口使用领域字符串身份，订单接口使用生成枚举。 */
export function airBusinessTypeCode(
  businessType?: number,
): 'AE' | 'AI' | undefined {
  if (businessType === OrderBusinessType.BUSINESS_TYPE_AE) return 'AE';
  if (businessType === OrderBusinessType.BUSINESS_TYPE_AI) return 'AI';
  return undefined;
}

/** 成功读取必须属于当前公司与真实类型，缺失身份同样拒绝。 */
export function requireAirMasterIdentity(
  master: API.AirMasterWaybill | undefined,
  masterId: string | undefined,
  organizationId: string | undefined,
  businessType: number | undefined,
): API.AirMasterWaybill {
  const code = airBusinessTypeCode(businessType);
  if (
    !code ||
    !masterId ||
    !organizationId ||
    master?.id !== masterId ||
    master.organizationId !== organizationId ||
    master.businessType !== code ||
    !master.version ||
    !/^\d+$/.test(master.version) ||
    BigInt(master.version) < 1n
  )
    throw new Error(
      '主单与当前公司或业务类型不一致，请重新选择或从对应业务列表进入。',
    );
  return master;
}
