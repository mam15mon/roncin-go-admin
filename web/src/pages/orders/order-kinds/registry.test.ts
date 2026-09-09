import { describe, expect, it } from 'vitest';
import { OrderBusinessType, TradeDirection } from '@/enums.generated';
import { seaExportDefinition } from './sea-export/definition';
import { getOrderKindDefinition, ORDER_KIND_REGISTRY } from './registry';

describe('订单类型注册表', () => {
  it('只注册 sea-export，且元数据与生成枚举一致', () => {
    expect(Object.keys(ORDER_KIND_REGISTRY)).toEqual(['sea-export']);
    expect(ORDER_KIND_REGISTRY['sea-export']).toBe(seaExportDefinition);
    expect(seaExportDefinition).toMatchObject({
      kind: 'sea-export',
      businessType: OrderBusinessType.BUSINESS_TYPE_SE,
      tradeDirection: TradeDirection.TRADE_DIRECTION_EXPORT,
      transportMode: 'sea',
      title: '海运出口订单',
      navigationTitle: '海运出口',
    });
  });

  it('直接 kind 与合法订单路径解析到同一注册对象', () => {
    const byKind = getOrderKindDefinition('sea-export');
    expect(byKind).toBe(seaExportDefinition);
    expect(getOrderKindDefinition('/orders/sea-export')).toBe(byKind);
    expect(getOrderKindDefinition('/orders/sea-export/new')).toBe(byKind);
    expect(getOrderKindDefinition('/orders/sea-export/ord-1')).toBe(byKind);
    expect(getOrderKindDefinition('/orders/sea-export/ord-1/fees')).toBe(byKind);
  });

  it.each([
    ['空字符串', ''],
    ['未知类型', 'unknown-kind'],
    ['未注册的海运进口', 'sea-import'],
    ['未注册的空运出口', 'air-export'],
    ['未注册的空运进口', 'air-import'],
    ['未注册的陆运', 'land'],
    ['未注册的铁路', 'rail'],
    ['未注册路径', '/orders/sea-import/ord-1'],
    ['非订单路径', '/partners'],
    ['undefined', undefined],
  ])('%s 返回 undefined，fail-closed 不做默认兜底', (_name, input) => {
    expect(getOrderKindDefinition(input)).toBeUndefined();
  });

  it('不接受原型属性或数值业务枚举反查路由', () => {
    expect(getOrderKindDefinition('constructor')).toBeUndefined();
    expect(getOrderKindDefinition('toString')).toBeUndefined();
    expect(getOrderKindDefinition('1')).toBeUndefined();
    expect(getOrderKindDefinition('/orders/1')).toBeUndefined();
  });
});
