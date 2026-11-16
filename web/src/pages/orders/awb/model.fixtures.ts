import { OrderBusinessType } from '@/enums.generated';
import type { AwbBundle, AwbIdentity } from './model';

export const aeIdentity: AwbIdentity = {
  organizationId: 'org-1',
  businessType: OrderBusinessType.BUSINESS_TYPE_AE,
  orderId: 'order-1',
  target: 'HAWB',
};
export function awbFixture(): AwbBundle {
  const master: API.AirMasterWaybill = {
    id: 'master-1',
    organizationId: 'org-1',
    organizationNameEn: 'ACTUAL COMPANY LTD',
    businessType: 'AE',
    version: '2',
    contentHash: 'master-hash',
    documentMode: 'HOUSE',
    content: {
      masterNo: '78112345675',
      airlineId: 'airline-1',
      originAirportId: 'airport-1',
      destinationAirportId: 'airport-2',
      status: 'DRAFT',
      waybillContent: { issuedBy: 'REAL AIRLINE', remarks: '主单原稿' },
    },
  };
  return {
    master,
    blockedReasons: [],
    order: {
      id: 'order-1',
      organizationId: 'org-1',
      businessType: OrderBusinessType.BUSINESS_TYPE_AE,
      version: '4',
      airExport: {
        documentMode: 'HOUSE',
        active: true,
        master: structuredClone(master),
        linkId: 'link-1',
        linkVersion: '3',
        house: {
          id: 'house-1',
          houseNo: 'H001',
          content: {
            remarks: '分单原稿',
            customsValue: { kind: 'AMOUNT', amount: '0' },
          },
          contentHash: 'house-hash',
          version: '5',
          status: 'DRAFT',
        },
      },
    },
  };
}

export function savedHouseOrder(
  bundle: AwbBundle,
  content: API.AirWaybillContent,
): API.Order {
  const saved = structuredClone(bundle.order) as API.Order;
  saved.version = (BigInt(saved.version ?? '0') + 1n).toString();
  if (!saved.airExport?.house) throw new Error('测试分单缺失');
  saved.airExport.linkVersion = (
    BigInt(saved.airExport.linkVersion ?? '0') + 1n
  ).toString();
  saved.airExport.house.version = (
    BigInt(saved.airExport.house.version ?? '0') + 1n
  ).toString();
  saved.airExport.house.content = structuredClone(content);
  saved.airExport.house.contentHash = 'saved-house-hash';
  return saved;
}
