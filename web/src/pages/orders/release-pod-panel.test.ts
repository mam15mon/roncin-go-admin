import { describe, expect, it } from 'vitest';
import { SeaDocumentType } from '@/enums.generated';
import {
  buildLegacyReleasePodDocumentOptions,
  buildSeaReleasePodDocumentOptions,
  getReleasePodDocumentValue,
  getReleasePodTransition,
} from './release-pod-panel';

describe('getReleasePodTransition', () => {
  it('只允许待签收和已签收状态向前流转', () => {
    expect(
      getReleasePodTransition({
        status: 1,
        allowedTargetStatuses: [2],
      }),
    ).toEqual({
      currentText: '待签收',
      nextText: '已签收',
      toStatus: 2,
    });
    expect(
      getReleasePodTransition({
        status: 2,
        allowedTargetStatuses: [3],
      }),
    ).toEqual({
      currentText: '已签收',
      nextText: '已回单',
      toStatus: 3,
    });
    expect(
      getReleasePodTransition({ status: 1, allowedTargetStatuses: [] }),
    ).toBeUndefined();
    expect(
      getReleasePodTransition({ status: 3, allowedTargetStatuses: [] }),
    ).toBeUndefined();
    expect(getReleasePodTransition()).toBeUndefined();
  });
});

describe('放货记录单证引用映射', () => {
  it('把真实 MBL/HBL 映射成显式类型与 ID', () => {
    const options = buildSeaReleasePodDocumentOptions({
      masterBill: { id: 'mbl-1', masterNo: 'MBL001' },
      houseBills: [{ id: 'hbl-1', houseNo: 'HBL001' }],
    });

    expect(options).toEqual([
      {
        label: 'MBL: MBL001',
        value: 'mbl:mbl-1',
        seaDocumentType: SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
        seaDocumentId: 'mbl-1',
      },
      {
        label: 'HBL: HBL001',
        value: 'hbl:hbl-1',
        seaDocumentType: SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL,
        seaDocumentId: 'hbl-1',
      },
    ]);
  });

  it('非 SE 旧分单只映射 shippingDocumentId', () => {
    expect(
      buildLegacyReleasePodDocumentOptions([
        { id: 'legacy-1', houseNo: 'OLD001' },
      ]),
    ).toEqual([
      {
        label: '分单: OLD001',
        value: 'legacy:legacy-1',
        shippingDocumentId: 'legacy-1',
      },
    ]);
  });

  it('按 API 显式字段回显关联类型，不猜测 UUID', () => {
    expect(
      getReleasePodDocumentValue({ shippingDocumentId: 'legacy-1' }),
    ).toBe('legacy:legacy-1');
    expect(
      getReleasePodDocumentValue({
        seaDocumentType: SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
        seaDocumentId: 'mbl-1',
      }),
    ).toBe('mbl:mbl-1');
    expect(
      getReleasePodDocumentValue({
        seaDocumentType: SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL,
        seaDocumentId: 'hbl-1',
      }),
    ).toBe('hbl:hbl-1');
    expect(
      getReleasePodDocumentValue({
        seaDocumentType: SeaDocumentType.SEA_DOCUMENT_TYPE_UNSPECIFIED,
        seaDocumentId: 'unknown-1',
      }),
    ).toBeUndefined();
  });
});
