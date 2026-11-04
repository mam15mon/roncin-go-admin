import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { buildHouseBillInput } from '../templates/components/sea/SeaCreateDocumentModeField';
import {
  buildHblFormValues,
  formatHblContainers,
  type HblBundle,
  hblFormPayload,
  hblUnavailableReason,
  hblValuesFingerprint,
} from './model';

const bundle: HblBundle = {
  order: {
    id: 'order-1',
    bookingNo: 'BOOK-001',
    vesselVoyage: 'SHIP V1',
    originLocationName: 'SHANGHAI',
    destinationLocationName: 'SINGAPORE',
  },
  documents: {
    orderId: 'order-1',
    linkVersion: '4',
    houseBill: {
      id: 'hbl-1',
      houseNo: 'HBL-001',
      version: '3',
      issuerSource: 1,
      issuerOrganizationId: 'issuer-company-1',
      issuerOrganizationNameEn: 'CONFIRMED ISSUER COMPANY LIMITED',
      printTemplate: 'RONCIN_SELF',
      content: { shipperText: '发货人', grossWeightKg: 33.125 },
    },
  },
};

describe('HBL 制作资料模型', () => {
  it('首次带入运输印字，不编造签发日、装船日、联系方式或正本份数', () => {
    const values = buildHblFormValues(bundle);
    expect(values.printDetails).toMatchObject({
      bookingNoText: 'BOOK-001',
      vesselVoyageText: 'SHIP V1',
      portOfLoadingText: 'SHANGHAI',
    });
    expect(values.printDetails.issueDate).toBeUndefined();
    expect(values.printDetails.shippedOnBoardDate).toBeUndefined();
    expect(values.printDetails.originalCount).toBeUndefined();
    expect(values.printDetails.headerContactText).toBeUndefined();
    expect(values.content.grossWeightKg).toBe(33.125);
  });
  it('已保存印字优先，包括有意清空；纯日期保持当天，零份数无损', () => {
    const updated = {
      ...bundle,
      documents: {
        ...bundle.documents,
        houseBill: {
          ...bundle.documents.houseBill,
          printDetails: {
            portOfLoadingText: 'CUSTOM PORT',
            issueDate: '2026-11-04',
            originalCount: 0,
          },
        },
      },
    };
    const values = buildHblFormValues(updated);
    expect(values.printDetails.bookingNoText).toBeUndefined();
    expect(hblFormPayload(values).printDetails).toEqual({
      portOfLoadingText: 'CUSTOM PORT',
      issueDate: '2026-11-04',
      originalCount: 0,
      shippedOnBoardDate: undefined,
    });
    values.printDetails.shippedOnBoardDate = dayjs('2026-10-31');
    expect(hblFormPayload(values).printDetails.shippedOnBoardDate).toBe(
      '2026-10-31',
    );
  });
  it('草稿的旧版本属于编辑基线，不能借刷新后的版本提交', () => {
    const draft = buildHblFormValues(bundle);
    const latest = buildHblFormValues({
      ...bundle,
      documents: {
        ...bundle.documents,
        linkVersion: '5',
        houseBill: { ...bundle.documents.houseBill, version: '4' },
      },
    });
    expect(draft.source).toEqual({
      houseId: 'hbl-1',
      houseVersion: '3',
      linkVersion: '4',
    });
    expect(hblValuesFingerprint(draft)).not.toBe(hblValuesFingerprint(latest));
  });
  it('改单构造保留打印补充资料', () => {
    const details = { issuePlaceText: 'BEIJING', originalCount: 3 };
    expect(
      buildHouseBillInput({
        houseNo: 'HBL-1',
        issuerSource: 1,
        printDetails: details,
      })?.printDetails,
    ).toEqual(details);
  });
  it('各公司自签真实 HBL 可制作，审核未参与模板判断', () => {
    expect(hblUnavailableReason(bundle)).toBeUndefined();
    expect(
      hblUnavailableReason({
        ...bundle,
        documents: { ...bundle.documents, houseBill: undefined },
      }),
    ).toContain('没有可制作');
    expect(
      hblUnavailableReason({
        ...bundle,
        documents: {
          ...bundle.documents,
          houseBill: { ...bundle.documents.houseBill, issuerSource: 2 },
        },
      }),
    ).toContain('暂无适用模板');
    expect(
      hblUnavailableReason({
        ...bundle,
        documents: {
          ...bundle.documents,
          houseBill: { ...bundle.documents.houseBill, printTemplate: '' },
        },
      }),
    ).toContain('暂无适用模板');
    expect(resolveTabKey('/orders/sea-export/order-1/hbl')).toBe(
      '/orders/sea-export',
    );
  });
  it('实际箱只带入本票共享箱分配，不混入箱计划或别票共享箱', () => {
    expect(
      formatHblContainers(
        'order-1',
        [{ containerNo: 'OWN001', containerSpecId: '20gp', sealNo: 'SEAL1' }],
        [
          {
            containerNo: 'SHARED001',
            containerSpecName: '40HQ',
            allocations: [{ orderId: 'order-1' }, { orderId: 'other' }],
          },
          {
            containerNo: 'OTHER001',
            containerSpecName: '40HQ',
            allocations: [{ orderId: 'other' }],
          },
        ],
        [{ id: '20gp', code: '20GP' }],
      ),
    ).toBe('OWN001 / SEAL SEAL1 / 20GP\nSHARED001 / 40HQ');
    expect(() =>
      formatHblContainers(
        'order-1',
        [{ containerNo: 'OWN001', containerSpecId: 'missing' }],
        [],
        [],
      ),
    ).toThrow('箱型资料不完整');
  });
});
