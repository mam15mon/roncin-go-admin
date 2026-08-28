import { describe, expect, it } from 'vitest';
import {
  MASTER_DATA_KINDS,
  ORDER_KIND_CONFIGS,
  PARTNER_ROLES,
  businessTypeOptions,
  isMasterDataKind,
  parseOrderKind,
  requireSeaServiceTypeOptions,
  seaServiceTypes,
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeDirectionOptions,
} from './common';

describe('orders common and config', () => {
  it('正确解析业务类型路径到配置', () => {
    expect(parseOrderKind('sea-export')).toEqual(ORDER_KIND_CONFIGS['sea-export']);
    expect(parseOrderKind('/orders/sea-export')?.businessType).toBe(1);
    expect(parseOrderKind('/orders/sea-export/new')?.category).toBe('sea');

    expect(parseOrderKind('sea-import')).toBeUndefined();
    expect(parseOrderKind('air-export')).toBeUndefined();
    expect(parseOrderKind('air-import')).toBeUndefined();

    expect(parseOrderKind('unknown-kind')).toBeUndefined();
    expect(parseOrderKind('')).toBeUndefined();
  });

  it('验证业务类型与贸易方向配置', () => {
    expect(ORDER_KIND_CONFIGS['sea-export'].tradeDirection).toBe(1);

    expect(businessTypeOptions).toEqual([
      { label: '海运出口', value: 1, color: 'blue' },
    ]);
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
    expect(seaServiceTypes[0]).toEqual({ code: 'BOOKING', name: '订舱' });
    expect(seaServiceTypes).toHaveLength(16);
  });

  it('按业务编码解析海运服务类型', () => {
    const options = seaServiceTypes.map(({ code, name }, index) => ({
      code,
      label: code === 'BOOKING' ? '自定义订舱名称' : name,
      value: `id-${index}`,
    }));

    expect(requireSeaServiceTypeOptions(options)[0]).toEqual({
      code: 'BOOKING',
      label: '自定义订舱名称',
      value: 'id-0',
    });
    expect(() => requireSeaServiceTypeOptions(options.slice(1))).toThrow(
      '缺少海运服务类型主数据：订舱（BOOKING）',
    );
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
