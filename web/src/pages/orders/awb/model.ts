import { OrderBusinessType } from '@/enums.generated';
import {
  acceptsAirSaveBaseline,
  sameAirWaybillContent,
} from '../order-kinds/air-export/air-save-baseline';
import { serializeAirWaybillContent } from '../order-kinds/air-export/form-adapter';
import { requireAirMasterIdentity } from '../order-kinds/air-export/identity';
import { AWB_PDF_TEMPLATE_REVISION, type AwbPdfModel } from './pdf/types';

export type AwbTarget = 'MAWB' | 'HAWB';
export type AwbIdentity = {
  organizationId: string;
  businessType: number;
  masterId?: string;
  orderId?: string;
  target: AwbTarget;
};
export type AwbBundle = {
  master: API.AirMasterWaybill;
  order?: API.Order;
  blockedReasons: string[];
};
export type AwbFormValues = { content: API.AirWaybillContent; reason?: string };

function requireVersion(value?: string) {
  if (!value || !/^[1-9]\d*$/.test(value))
    throw new Error('航空运单缺少有效保存版本。');
}

export function requireAwbBundle(
  bundle: AwbBundle,
  identity: AwbIdentity,
): AwbBundle {
  requireAirMasterIdentity(
    bundle.master,
    identity.masterId ?? bundle.order?.airExport?.master?.id,
    identity.organizationId,
    identity.businessType,
  );
  if (
    !bundle.master.content?.masterNo ||
    !bundle.master.contentHash ||
    !bundle.master.content.waybillContent
  )
    throw new Error('主单正文或保存标识缺失。');
  if (!['DIRECT', 'HOUSE'].includes(bundle.master.documentMode ?? ''))
    throw new Error('航空主单模式缺失或不合法。');
  if (identity.orderId) {
    const order = bundle.order;
    if (
      order?.id !== identity.orderId ||
      order.organizationId !== identity.organizationId ||
      order.businessType !== identity.businessType
    )
      throw new Error('订单与当前公司或业务类型不一致。');
    requireVersion(order.version);
    const air = order.airExport;
    if (
      !air?.linkId ||
      !['DIRECT', 'HOUSE'].includes(air.documentMode ?? '') ||
      air.master?.id !== bundle.master.id ||
      air.documentMode !== bundle.master.documentMode ||
      air.master?.documentMode !== bundle.master.documentMode
    )
      throw new Error('航空单证关联不完整。');
    requireAirMasterIdentity(
      air.master,
      bundle.master.id,
      identity.organizationId,
      identity.businessType,
    );
    requireVersion(air.linkVersion);
    if (air.documentMode === 'DIRECT' && air.house)
      throw new Error('直单不能包含 HAWB。');
    if (air.documentMode === 'HOUSE') {
      if (
        !air.house?.id ||
        !air.house.houseNo ||
        !air.house.content ||
        !air.house.contentHash
      )
        throw new Error('分单正文或保存标识缺失。');
      requireVersion(air.house.version);
    }
    if (identity.target === 'HAWB' && air.documentMode !== 'HOUSE')
      throw new Error('直单没有可制作的 HAWB。');
  } else if (identity.target === 'HAWB') throw new Error('HAWB 必须绑定本票。');
  return bundle;
}

export function awbContent(
  bundle: AwbBundle,
  target: AwbTarget,
): API.AirWaybillContent {
  return structuredClone(
    (target === 'MAWB'
      ? bundle.master.content?.waybillContent
      : bundle.order?.airExport?.house?.content) ?? {},
  );
}

/** 同时绑定私有单证、共享主单与真实公司抬头，后台探测仅比较此标识。 */
export function awbFingerprint(bundle: AwbBundle): string {
  return JSON.stringify([
    AWB_PDF_TEMPLATE_REVISION,
    bundle.master.id,
    bundle.master.organizationId,
    bundle.master.businessType,
    bundle.master.version,
    bundle.master.contentHash,
    bundle.master.organizationNameEn,
    bundle.master.content,
    bundle.blockedReasons,
    bundle.order?.id,
    bundle.order?.version,
    bundle.order?.lockedAt,
    bundle.order?.terminationStatus,
    bundle.order?.airExport,
  ]);
}

export function awbOutputBlockedReason(
  bundle: AwbBundle,
  target: AwbTarget,
): string | undefined {
  if (target === 'HAWB' && bundle.master.businessType === 'AE')
    return bundle.master.organizationNameEn?.trim()
      ? undefined
      : '经营公司尚未维护英文名称，请在公司档案中补齐后输出。';
  return awbContent(bundle, target).issuedBy?.trim()
    ? undefined
    : '请先维护真实印发人并保存，再输出归档文件。';
}

export function buildAwbPdfModel(
  bundle: AwbBundle,
  target: AwbTarget,
): AwbPdfModel {
  requireAwbBundle(bundle, {
    organizationId: bundle.master.organizationId ?? '',
    masterId: bundle.master.id,
    orderId: target === 'HAWB' ? bundle.order?.id : undefined,
    businessType:
      bundle.master.businessType === 'AE'
        ? OrderBusinessType.BUSINESS_TYPE_AE
        : bundle.master.businessType === 'AI'
          ? OrderBusinessType.BUSINESS_TYPE_AI
          : OrderBusinessType.BUSINESS_TYPE_UNSPECIFIED,
    target,
  });
  const reason = awbOutputBlockedReason(bundle, target);
  if (reason) throw new Error(reason);
  const house = bundle.order?.airExport?.house;
  const self = target === 'HAWB' && bundle.master.businessType === 'AE';
  return {
    target,
    businessType: bundle.master.businessType as 'AE' | 'AI',
    documentId: (target === 'MAWB' ? bundle.master.id : house?.id) as string,
    version: (target === 'MAWB'
      ? bundle.master.version
      : house?.version) as string,
    contentHash: (target === 'MAWB'
      ? bundle.master.contentHash
      : house?.contentHash) as string,
    masterNo: bundle.master.content?.masterNo as string,
    houseNo: target === 'HAWB' ? house?.houseNo : undefined,
    organizationId: bundle.master.organizationId as string,
    issuerDisplayName: (self
      ? bundle.master.organizationNameEn
      : awbContent(bundle, target).issuedBy) as string,
    outputKind: self ? 'SELF_ISSUED' : 'EXTERNAL_ARCHIVE',
    content: awbContent(bundle, target),
  };
}

/** 只提交明确的本票分单正文，不复制经营字段或主单运输输入。 */
export function awbHouseSavePayload(
  bundle: AwbBundle,
  values: AwbFormValues,
): API.UpdateOrderRequest {
  const air = bundle.order?.airExport;
  if (air?.documentMode !== 'HOUSE' || !air.house?.id)
    throw new Error('当前没有真实 HAWB。');
  return {
    id: bundle.order?.id as string,
    expectedVersion: bundle.order?.version as string,
    airExport: {
      documentMode: 'HOUSE',
      masterWaybillId: bundle.master.id,
      expectedMasterVersion: bundle.master.version,
      expectedLinkVersion: air.linkVersion,
      house: {
        houseNo: air.house.houseNo,
        status: air.house.status,
        expectedVersion: air.house.version,
        content: serializeAirWaybillContent(values.content) ?? {},
      },
      changeReason: values.reason,
    },
  };
}

export function acceptsAwbHouseSave(
  bundle: AwbBundle,
  payload: API.UpdateOrderRequest,
  saved: API.Order,
): boolean {
  return (
    acceptsAirSaveBaseline(
      bundle.order?.id ?? '',
      bundle.master.organizationId,
      bundle.order?.businessType ?? 0,
      bundle.order?.airExport,
      payload,
      saved,
    ) &&
    saved.airExport?.master?.organizationNameEn ===
      bundle.master.organizationNameEn
  );
}

export function acceptsAwbMasterSave(
  bundle: AwbBundle,
  values: AwbFormValues,
  saved?: API.AirMasterWaybill,
): boolean {
  try {
    requireAirMasterIdentity(
      saved,
      bundle.master.id,
      bundle.master.organizationId,
      bundle.order?.businessType ??
        (bundle.master.businessType === 'AE'
          ? OrderBusinessType.BUSINESS_TYPE_AE
          : OrderBusinessType.BUSINESS_TYPE_AI),
    );
  } catch {
    return false;
  }
  return Boolean(
    saved?.version &&
      BigInt(saved.version) === BigInt(bundle.master.version ?? '0') + 1n &&
      saved?.organizationNameEn === bundle.master.organizationNameEn &&
      saved?.content?.masterNo === bundle.master.content?.masterNo &&
      sameAirWaybillContent(values.content, saved?.content?.waybillContent),
  );
}
