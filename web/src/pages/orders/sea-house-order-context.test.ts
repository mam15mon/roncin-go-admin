import { describe, expect, it } from 'vitest';
import { OrderBusinessType, SeaDocumentStructure } from '@/enums.generated';
import {
  buildSeaHouseOrderCreatePath,
  getSeaHouseOrderCreateSource,
  resolveSeaHouseOrderCreateRequest,
} from './sea-house-order-context';

const sourceOrder: API.Order = {
  id: 'source-A',
  organizationId: 'org-1',
  businessType: OrderBusinessType.BUSINESS_TYPE_SE,
  seaDocumentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
  shippingLineId: 'line-1',
  customerId: 'private-customer',
  goodsDescription: '本票货物',
  seaMasterBill: {
    masterBillId: 'mbl-1',
    masterNo: 'MASTER-1',
    shippingLineId: 'line-1',
    shippingLineName: '船公司',
    transportExecutionId: 'te-1',
    version: '7',
    transportExecutionVersion: '8',
  },
};

describe('连续分单来源上下文', () => {
  it('规范来源地址只保留唯一已知参数并编码订单ID', () => {
    expect(buildSeaHouseOrderCreatePath('订单/A & B')).toBe(
      '/orders/sea-export/new?sourceOrderId=%E8%AE%A2%E5%8D%95%2FA%20%26%20B',
    );
    expect(
      resolveSeaHouseOrderCreateRequest(
        'sea-export',
        '?x=1&sourceOrderId=source-A&y=2',
      ),
    ).toEqual({
      sourceOrderId: 'source-A',
      draftPathname: '/orders/sea-export/new?sourceOrderId=source-A',
    });
    expect(resolveSeaHouseOrderCreateRequest('sea-export', '?x=1')).toEqual({
      draftPathname: '/orders/sea-export/new',
    });
  });

  it.each([
    ['sea-export', '?sourceOrderId='],
    ['sea-export', '?sourceOrderId=%20'],
    ['sea-export', '?sourceOrderId=A&sourceOrderId=A'],
    ['air-export', '?sourceOrderId=A'],
  ])('阻断不合法来源参数 %s %s', (kind, search) => {
    expect(resolveSeaHouseOrderCreateRequest(kind, search).error).toBeTruthy();
  });

  it('只提取共享身份，不复制本票资料或来源版本', () => {
    expect(
      getSeaHouseOrderCreateSource(sourceOrder, 'source-A', 'org-1'),
    ).toEqual({
      orderId: 'source-A',
      masterBillId: 'mbl-1',
      masterNo: 'MASTER-1',
      shippingLineId: 'line-1',
      shippingLineName: '船公司',
      transportExecutionId: 'te-1',
    });
  });

  it.each([
    undefined,
    { ...sourceOrder, id: 'another-order' },
    { ...sourceOrder, organizationId: 'another-org' },
    { ...sourceOrder, businessType: OrderBusinessType.BUSINESS_TYPE_AE },
    {
      ...sourceOrder,
      seaDocumentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
    },
    { ...sourceOrder, seaMasterBill: undefined },
    { ...sourceOrder, shippingLineId: 'another-line' },
    ...[
      'masterBillId',
      'masterNo',
      'shippingLineId',
      'transportExecutionId',
    ].map((field) => ({
      ...sourceOrder,
      seaMasterBill: { ...sourceOrder.seaMasterBill, [field]: '' },
    })),
    {
      ...sourceOrder,
      seaMasterBill: { ...sourceOrder.seaMasterBill, status: 'VOIDED' },
    },
  ])('来源缺失、跨组织、DIRECT或活动共享身份不完整时拒绝 %#', (order) => {
    expect(() =>
      getSeaHouseOrderCreateSource(order, 'source-A', 'org-1'),
    ).toThrow();
  });
});
