import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import { OrderBusinessType, TradeDirection } from '@/enums.generated';
import type { CreateOrderFormValues } from '../form-values';
import { airImportDefinition } from './definition';

const form = airImportDefinition.form;
const base: CreateOrderFormValues = {
  customerId: 'customer',
  paymentTerm: 2,
  airMasterMode: 'NEW',
  airDocumentMode: 'HOUSE',
  airHouseNo: ' AI-HOUSE ',
  airMaster: {
    masterNo: '78112345675',
    eta: '2026-11-17T00:30:00+08:00',
    waybillContent: { remarks: '进口外部主单\n原始印字' },
  },
  airHouseContent: {
    carriageValue: { kind: 'AMOUNT', amount: '0' },
    remarks: '分单资料\n保留换行',
  },
};
describe('AI复用航空建票与编辑映射', () => {
  it.each(['DIRECT', 'HOUSE'] as const)(
    '%s按真实AI+IMPORT提交且直单没有虚构HAWB',
    (airDocumentMode) => {
      const payload = JSON.parse(
        JSON.stringify(form.buildCreatePayload({ ...base, airDocumentMode })),
      );
      expect(payload).toMatchObject({
        businessType: OrderBusinessType.BUSINESS_TYPE_AI,
        tradeDirection: TradeDirection.TRADE_DIRECTION_IMPORT,
        airExport: {
          documentMode: airDocumentMode,
          master: {
            eta: '2026-11-16T16:30:00.000Z',
            waybillContent: { remarks: '进口外部主单\n原始印字' },
          },
        },
      });
      if (airDocumentMode === 'DIRECT')
        expect(payload.airExport).not.toHaveProperty('house');
      else
        expect(payload.airExport.house).toMatchObject({
          houseNo: 'AI-HOUSE',
          content: base.airHouseContent,
        });
    },
  );
  it('明确关联已有主单时只提交ID与预期版本，不提交输入中的主单副本', () => {
    const payload = form.buildCreatePayload({
      ...base,
      airMasterMode: 'EXISTING',
      airMasterId: 'master-ai',
      airMasterExpectedVersion: '9',
    });
    expect(payload.airExport).toMatchObject({
      masterWaybillId: 'master-ai',
      expectedMasterVersion: '9',
    });
    expect(payload.airExport?.master).toBeUndefined();
  });
  it('省略默认值和optional未设仍可回显；经营字段显式清空不写共享主单或转换模式', () => {
    const order: API.Order = {
      id: 'order-ai',
      organizationId: 'company',
      businessType: OrderBusinessType.BUSINESS_TYPE_AI,
      version: '4',
      customerId: 'customer',
      receivedAt: '2026-11-16T16:30:00Z',
      cargoValue: '0',
      cargoCurrency: 'CNY',
      airExport: {
        documentMode: 'DIRECT',
        master: {
          id: 'master-ai',
          organizationId: 'company',
          businessType: 'AI',
          version: '9',
          content: base.airMaster,
        },
        linkId: 'link-ai',
        linkVersion: '2',
      },
    };
    const values = form.buildDetailInitialValues(order, [], []);
    expect(dayjs.isDayjs(values.receivedAt)).toBe(true);
    expect(values.airHouseContent).toBeUndefined();
    expect(values.cargoValue).toBe('0');
    const payload = JSON.parse(
      JSON.stringify(
        form.buildUpdatePayload('order-ai', '4', {
          ...values,
          customerId: 'customer',
          paymentTerm: 2,
          cargoValue: undefined,
          cargoCurrency: undefined,
          receivedAt: undefined,
          factoryName: undefined,
          airDocumentMode: 'HOUSE',
        }),
      ),
    );
    expect(payload).toMatchObject({
      cargoValue: '',
      cargoCurrency: '',
      receivedAt: '',
      factoryName: '',
      airExport: {
        documentMode: 'DIRECT',
        masterWaybillId: 'master-ai',
        expectedMasterVersion: '9',
        expectedLinkVersion: '2',
      },
    });
    expect(payload.airExport).not.toHaveProperty('house');
    expect(payload.airExport).not.toHaveProperty('master');
  });
});
