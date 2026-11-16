import { describe, expect, it } from 'vitest';
import { OrderBusinessType, SeaHouseBillIssuerSource } from '@/enums.generated';
import {
  type HblBundle,
  hblBundleFingerprint,
  hblOutputBlockedReason,
  hblUnavailableReason,
  isExternalHblArchive,
} from './model';

const bundle: HblBundle = {
  order: { id: 'import', businessType: OrderBusinessType.BUSINESS_TYPE_SI },
  documents: {
    houseBill: {
      id: 'house',
      printTemplate: 'EXTERNAL_ARCHIVE',
      issuerSource:
        SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER,
      issuerPartnerId: 'external',
      issuerPartnerName: '真实外部签发人',
    },
  },
};
describe('外部HBL归档主体边界', () => {
  it('仅SI真实外部主体允许归档，缺名明确阻止输出而不限制资料维护', () => {
    expect(isExternalHblArchive(bundle)).toBe(true);
    expect(hblOutputBlockedReason(bundle)).toBeUndefined();
    const missing = {
      ...bundle,
      documents: {
        houseBill: {
          ...bundle.documents.houseBill,
          issuerPartnerName: undefined,
          issuerOrganizationNameEn: '不得借用公司名称',
        },
      },
    };
    expect(hblUnavailableReason(missing)).toBeUndefined();
    expect(hblOutputBlockedReason(missing)).toContain('缺少真实签发方资料');
    expect(
      isExternalHblArchive({
        ...bundle,
        order: {
          ...bundle.order,
          businessType: OrderBusinessType.BUSINESS_TYPE_SE,
        },
      }),
    ).toBe(false);
  });
  it('外部签发主体身份与名称变化进入轮询指纹', () => {
    expect(hblBundleFingerprint(bundle)).not.toBe(
      hblBundleFingerprint({
        ...bundle,
        documents: {
          houseBill: {
            ...bundle.documents.houseBill,
            issuerPartnerName: '外部新名称',
          },
        },
      }),
    );
  });
});
