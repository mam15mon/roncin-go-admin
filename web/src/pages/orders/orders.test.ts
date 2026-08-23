import { describe, expect, it } from 'vitest';
import {
  MASTER_DATA_KINDS,
  ORDER_KIND_CONFIGS,
  PARTNER_ROLES,
  businessTypeOptions,
  isMasterDataKind,
  parseOrderKind,
  seaServiceTypeNames,
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeDirectionOptions,
} from './common';

describe('orders common and config', () => {
  it('正确解析业务类型路径到配置', () => {
    expect(parseOrderKind('sea-export')).toEqual(ORDER_KIND_CONFIGS['sea-export']);
    expect(parseOrderKind('sea-import')).toEqual(ORDER_KIND_CONFIGS['sea-import']);
    expect(parseOrderKind('air-export')).toEqual(ORDER_KIND_CONFIGS['air-export']);
    expect(parseOrderKind('air-import')).toEqual(ORDER_KIND_CONFIGS['air-import']);

    expect(parseOrderKind('/orders/sea-export')?.businessType).toBe(1);
    expect(parseOrderKind('/orders/sea-import')?.businessType).toBe(2);
    expect(parseOrderKind('/orders/air-export')?.businessType).toBe(3);
    expect(parseOrderKind('/orders/air-import')?.businessType).toBe(4);

    expect(parseOrderKind('/orders/sea-export/new')?.category).toBe('sea');
    expect(parseOrderKind('/orders/air-export/new')?.category).toBe('air');

    expect(parseOrderKind('unknown-kind')).toBeUndefined();
    expect(parseOrderKind('')).toBeUndefined();
  });

  it('验证业务类型与贸易方向配置', () => {
    expect(ORDER_KIND_CONFIGS['sea-export'].tradeDirection).toBe(1);
    expect(ORDER_KIND_CONFIGS['sea-import'].tradeDirection).toBe(2);
    expect(ORDER_KIND_CONFIGS['air-export'].tradeDirection).toBe(1);
    expect(ORDER_KIND_CONFIGS['air-import'].tradeDirection).toBe(2);

    expect(businessTypeOptions).toHaveLength(6);
    expect(tradeDirectionOptions).toHaveLength(2);
    expect(PARTNER_ROLES.BOOKING_AGENT).toBe(PARTNER_ROLES.SUPPLIER);
    expect(PARTNER_ROLES.FOREIGN_AGENT).toBe(3);
    expect(shipmentModeOptions).toEqual([
      { label: '集运', value: 1 },
      { label: '跨境', value: 2 },
    ]);
    expect(shipmentTypeOptions).toEqual([
      { label: '整箱', value: 1 },
      { label: '拼箱', value: 2 },
      { label: '散杂', value: 3 },
    ]);
    expect(seaServiceTypeNames).toEqual([
      '订舱',
      '拖车',
      '内装',
      '报关',
      '清关',
      '海外段',
      '保险',
      '租箱',
      '熏蒸',
      '买单',
      '办证',
      '制单',
      '危险品',
      '超重',
      '仓储',
      '买箱',
    ]);
  });

  it('按 REST 枚举名称识别订单主数据类型', () => {
    expect(
      isMasterDataKind(
        'MASTER_DATA_KIND_SERVICE_TYPE',
        MASTER_DATA_KINDS.SERVICE_TYPE,
      ),
    ).toBe(true);
    expect(
      isMasterDataKind(
        'MASTER_DATA_KIND_CARGO_CATEGORY',
        MASTER_DATA_KINDS.SERVICE_TYPE,
      ),
    ).toBe(false);
  });
});
