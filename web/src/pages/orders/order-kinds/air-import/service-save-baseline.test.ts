import { describe, expect, it } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import type { OrderDetailFormValues } from '../form-values';
import { acceptsImportServiceSaveBaseline } from '../sea-import/service-save-baseline';

const type = OrderBusinessType.BUSINESS_TYPE_AI;
const air: API.AirOrderInfo = {
  documentMode: 'HOUSE',
  active: true,
  master: {
    id: 'master',
    organizationId: 'company',
    businessType: 'AI',
    version: '8',
  },
  linkId: 'link',
  linkVersion: '3',
  house: { id: 'house', version: '6', content: { remarks: '已保存正文' } },
};
const values: OrderDetailFormValues = {
  id: 'order',
  version: '9007199254740993',
  customerId: 'customer',
  paymentTerm: 2,
  customerReferenceNo: '未保存输入',
  airHouseContent: { remarks: '未保存分单正文' },
  airExport: air,
};
const saved: API.OrderImportHandling = {
  orderId: 'order',
  orderVersion: '9007199254740994',
};
const order: API.Order = {
  id: 'order',
  organizationId: 'company',
  businessType: type,
  version: saved.orderVersion,
  airExport: air,
};
const accepts = (target = order, input = values) =>
  acceptsImportServiceSaveBaseline(
    input,
    values.version as string,
    saved,
    target,
    undefined,
    'company',
    type,
  );

describe('AI独立服务保存的确切航空基线', () => {
  it.each(['HOUSE', 'DIRECT'] as const)(
    '%s只采纳Order恰加一，未保存主表和分单正文不改变',
    (documentMode) => {
      const relation = {
        ...air,
        documentMode,
        house: documentMode === 'DIRECT' ? undefined : air.house,
      };
      expect(
        accepts(
          { ...order, airExport: relation },
          { ...values, airExport: relation },
        ),
      ).toBe(true);
      expect(values.version).toBe('9007199254740993');
      expect(values.airHouseContent?.remarks).toBe('未保存分单正文');
    },
  );
  it.each([
    { version: '9007199254740995' },
    { businessType: OrderBusinessType.BUSINESS_TYPE_AE },
    { businessType: undefined },
    { organizationId: 'other' },
    { id: 'other' },
    { airExport: { ...air, master: { ...air.master, version: '9' } } },
    { airExport: { ...air, master: { ...air.master, id: 'other' } } },
    {
      airExport: { ...air, master: { ...air.master, organizationId: 'other' } },
    },
    { airExport: { ...air, master: { ...air.master, businessType: 'AE' } } },
    { airExport: { ...air, linkId: 'other' } },
    { airExport: { ...air, linkVersion: '4' } },
    { airExport: { ...air, house: { ...air.house, version: '7' } } },
    { airExport: { ...air, house: { ...air.house, id: 'other' } } },
    { airExport: { ...air, house: undefined } },
    { airExport: { ...air, documentMode: 'DIRECT' } },
    { airExport: { ...air, active: undefined } },
  ] satisfies Partial<API.Order>[])(
    '拒绝其他版本/类型/主单/关联/分单的变动 %j',
    (patch) => expect(accepts({ ...order, ...patch })).toBe(false),
  );
  it('整票基线已推进、服务更晚版本和直单夹带分单均拒绝', () => {
    expect(accepts(order, { ...values, version: saved.orderVersion })).toBe(
      false,
    );
    expect(
      acceptsImportServiceSaveBaseline(
        values,
        values.version as string,
        { ...saved, orderVersion: '9007199254740995' },
        order,
        undefined,
        'company',
        type,
      ),
    ).toBe(false);
    const direct = { ...air, documentMode: 'DIRECT', house: undefined };
    expect(
      accepts(
        { ...order, airExport: { ...direct, house: air.house } },
        { ...values, airExport: direct },
      ),
    ).toBe(false);
  });
});
