import { describe, expect, it } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  acceptsAirSaveBaseline,
  sameAirWaybillContent,
} from './air-save-baseline';

const previous: API.AirOrderInfo = {
  documentMode: 'HOUSE',
  active: true,
  master: {
    id: 'master',
    organizationId: 'org',
    businessType: 'AE',
    version: '3',
  },
  house: {
    id: 'house',
    houseNo: 'H1',
    status: 'DRAFT',
    version: '2',
    content: { remarks: '原文' },
  },
  linkId: 'link',
  linkVersion: '1',
};
const payload: API.UpdateOrderRequest = {
  id: 'order',
  expectedVersion: '9007199254740993',
  airExport: {
    documentMode: 'HOUSE',
    expectedLinkVersion: '1',
    house: {
      houseNo: 'H1',
      status: 'DRAFT',
      expectedVersion: '2',
      content: { remarks: '新正文' },
    },
  },
};
function saved(): API.Order {
  return {
    id: 'order',
    organizationId: 'org',
    businessType: OrderBusinessType.BUSINESS_TYPE_AE,
    version: '9007199254740994',
    airExport: {
      ...previous,
      linkVersion: '2',
      house: {
        ...previous.house,
        content: { remarks: '新正文' },
        version: '3',
      },
    },
  };
}
const accepts = (snapshot: API.Order, input = payload, baseline = previous) =>
  acceptsAirSaveBaseline(
    'order',
    'org',
    OrderBusinessType.BUSINESS_TYPE_AE,
    baseline,
    input,
    snapshot,
  );

describe('航空保存精确版本基线', () => {
  it('分单修改同步确认订单、分单及关联各自版本，不受JS整数精度影响', () => {
    expect(accepts(saved())).toBe(true);
  });

  it.each(['order', 'master', 'house', 'link', 'company', 'content'] as const)(
    '拒绝保存后重读到的后续 %s 变化',
    (target) => {
      const snapshot = saved();
      if (!snapshot.airExport?.house || !snapshot.airExport.master)
        throw new Error('测试缺少航空正文');
      if (target === 'order') snapshot.version = '9007199254740995';
      if (target === 'master')
        snapshot.airExport.master = {
          ...snapshot.airExport.master,
          version: '4',
        };
      if (target === 'house')
        snapshot.airExport.house = {
          ...snapshot.airExport.house,
          version: '4',
        };
      if (target === 'link') snapshot.airExport.linkVersion = '3';
      if (target === 'company') snapshot.organizationId = 'other-company';
      if (target === 'content')
        snapshot.airExport.house = {
          ...snapshot.airExport.house,
          content: { remarks: '其他人正文' },
        };
      expect(accepts(snapshot)).toBe(false);
    },
  );

  it('普通字段保存省略正文时保持正文和分单/关联版本', () => {
    expect(
      accepts(
        { ...saved(), airExport: previous },
        {
          ...payload,
          airExport: {
            house: { houseNo: 'H1', status: 'DRAFT', expectedVersion: '2' },
          },
        },
      ),
    ).toBe(true);
  });

  it('直单无分单仍可确认订单保存，响应夹带分单则拒绝', () => {
    const direct = { ...previous, documentMode: 'DIRECT', house: undefined };
    expect(
      accepts(
        { ...saved(), airExport: direct },
        { ...payload, airExport: { documentMode: 'DIRECT' } },
        direct,
      ),
    ).toBe(true);
    expect(
      accepts(
        { ...saved(), airExport: { ...direct, house: previous.house } },
        payload,
        direct,
      ),
    ).toBe(false);
  });

  it('Proto省略默认值及服务端生成合计不误判为他人修改，原始可编辑事实仍逐一核对', () => {
    expect(
      sameAirWaybillContent(
        { remarks: '', asArranged: false, rateLines: [], collect: {} },
        {},
      ),
    ).toBe(true);
    expect(
      sameAirWaybillContent(
        { prepaid: { tax: '0' }, remarks: '正文' },
        {
          remarks: '正文',
          prepaid: {
            tax: '0',
            weightCharge: '3',
            otherDueAgent: '0',
            otherDueCarrier: '0',
            total: '3',
          },
          collect: { weightCharge: '0', total: '0' },
        },
      ),
    ).toBe(true);
    expect(sameAirWaybillContent({ prepaid: { tax: '0' } }, {})).toBe(false);
    expect(sameAirWaybillContent({ asArranged: true }, {})).toBe(false);
    expect(
      sameAirWaybillContent(
        { carriageValue: { kind: 'AMOUNT', amount: '0' } },
        {},
      ),
    ).toBe(false);
  });
});
