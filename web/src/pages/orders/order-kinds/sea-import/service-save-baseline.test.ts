import { describe, expect, it } from 'vitest';
import { OrderBusinessType, SeaDocumentStructure } from '@/enums.generated';
import type { OrderDetailFormValues } from '../form-values';
import { acceptsImportServiceSaveBaseline } from './service-save-baseline';

const accepts = (
  ...args: Parameters<typeof acceptsImportServiceSaveBaseline> extends [
    ...infer P,
    number,
  ]
    ? P
    : never
) =>
  acceptsImportServiceSaveBaseline(...args, OrderBusinessType.BUSINESS_TYPE_SI);

const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
const values: OrderDetailFormValues = {
  customerId: 'customer',
  paymentTerm: 2,
  id: 'order',
  version: '4',
  customerReferenceNo: '未保存输入',
  seaMasterBill: {
    masterBillId: 'master',
    version: '7',
    transportExecutionId: 'voyage',
    transportExecutionVersion: '2',
  },
  seaMasterBillExpectedCandidateVersion: '7',
  seaMasterBillContentVersion: '7',
  seaDocumentStructure: HOUSE,
  seaDocumentLinkVersion: '3',
  seaHouseBill: {
    id: 'house',
    houseNo: 'H1',
    issuerSource: 3,
    expectedVersion: '6',
  },
};
const saved: API.OrderImportHandling = { orderId: 'order', orderVersion: '5' };
const order: API.Order = {
  id: 'order',
  organizationId: 'org',
  businessType: OrderBusinessType.BUSINESS_TYPE_SI,
  version: '5',
  seaMasterBill: values.seaMasterBill as API.SeaMasterBillSummary,
  seaDocumentLinkVersion: '3',
};
const documents: API.SeaOrderDocuments = {
  orderId: 'order',
  documentStructure: HOUSE,
  linkVersion: '3',
  masterBill: { id: 'master', version: '7' },
  houseBill: { id: 'house', version: '6' },
};

describe('进口服务保存后的整票提交基线', () => {
  it('只接受本人单次保存且所有海运单证和航程版本未变，保留未保存字段', () => {
    expect(accepts(values, '4', saved, order, documents, 'org')).toBe(true);
    expect(values.customerReferenceNo).toBe('未保存输入');
    expect(values.version).toBe('4');
  });
  it.each([
    { version: '6' },
    { id: 'other' },
    { organizationId: 'other' },
    { seaMasterBill: { ...order.seaMasterBill, masterBillId: 'other' } },
    { seaMasterBill: { ...order.seaMasterBill, version: '8' } },
    {
      seaMasterBill: { ...order.seaMasterBill, transportExecutionId: 'other' },
    },
    {
      seaMasterBill: { ...order.seaMasterBill, transportExecutionVersion: '3' },
    },
    { seaDocumentLinkVersion: '4' },
  ] satisfies Partial<API.Order>[])(
    '拒绝更晚写入或运输/单证变化 %j',
    (patch) => {
      expect(
        accepts(values, '4', saved, { ...order, ...patch }, documents, 'org'),
      ).toBe(false);
    },
  );
  it.each([
    { masterBill: { id: 'master', version: '8' } },
    { houseBill: { id: 'other', version: '6' } },
    { houseBill: { id: 'house', version: '7' } },
    { linkVersion: '4' },
    { documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT },
  ] satisfies Partial<API.SeaOrderDocuments>[])(
    '拒绝最新单证和旧表单基线错配 %j',
    (patch) => {
      expect(
        accepts(values, '4', saved, order, { ...documents, ...patch }, 'org'),
      ).toBe(false);
    },
  );
  it('保存期间整票基线已前进、版本非法或直接单无真实主单时均不偷换版本', () => {
    expect(
      accepts({ ...values, version: '5' }, '4', saved, order, documents, 'org'),
    ).toBe(false);
    expect(accepts(values, 'bad', saved, order, documents, 'org')).toBe(false);
    expect(
      accepts(
        values,
        '4',
        { ...saved, orderVersion: '6' },
        order,
        documents,
        'org',
      ),
    ).toBe(false);
    expect(
      accepts(
        { ...values, seaMasterBill: undefined },
        '4',
        saved,
        order,
        documents,
        'org',
      ),
    ).toBe(false);
  });
});
