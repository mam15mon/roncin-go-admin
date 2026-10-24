import { describe, expect, it } from 'vitest';
import {
  resolveSeaOrderFormPolicy,
  SEA_SHIPMENT_MODE,
  SEA_SHIPMENT_TYPE,
} from './sea-order-policy';

describe('海运出口表单策略', () => {
  it('按集运与跨境模式返回模式标签与合规提示', () => {
    expect(
      resolveSeaOrderFormPolicy({
        shipmentMode: SEA_SHIPMENT_MODE.TRADITIONAL_FORWARDING,
      }),
    ).toMatchObject({ modeLabel: '集运', emphasizeCrossBorderCompliance: false });
    expect(
      resolveSeaOrderFormPolicy({ shipmentMode: SEA_SHIPMENT_MODE.CROSS_BORDER }),
    ).toMatchObject({ modeLabel: '跨境', emphasizeCrossBorderCompliance: true });
  });

  it('按托运类型控制箱量、自拼入口与计费吨', () => {
    expect(
      resolveSeaOrderFormPolicy({ shipmentType: SEA_SHIPMENT_TYPE.FCL }),
    ).toMatchObject({
      showContainerPlan: true,
      showConsolidationReference: false,
      showRevenueTon: false,
    });
    expect(
      resolveSeaOrderFormPolicy({ shipmentType: SEA_SHIPMENT_TYPE.LCL }),
    ).toMatchObject({
      showContainerPlan: true,
      showConsolidationReference: true,
      showRevenueTon: false,
    });
    expect(
      resolveSeaOrderFormPolicy({
        shipmentType: SEA_SHIPMENT_TYPE.BREAK_BULK,
      }),
    ).toMatchObject({
      showContainerPlan: false,
      showConsolidationReference: false,
      showRevenueTon: true,
    });
  });

});
