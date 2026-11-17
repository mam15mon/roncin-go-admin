import { describe, expect, it } from 'vitest';
import {
  resolveSeaOrderFormPolicy,
  SEA_SHIPMENT_TYPE,
} from './sea-order-policy';

describe('海运出口表单策略', () => {
  it('按托运类型控制箱量与计费吨', () => {
    expect(
      resolveSeaOrderFormPolicy({ shipmentType: SEA_SHIPMENT_TYPE.FCL }),
    ).toMatchObject({
      showContainerPlan: true,
      showRevenueTon: false,
    });
    expect(
      resolveSeaOrderFormPolicy({ shipmentType: SEA_SHIPMENT_TYPE.LCL }),
    ).toMatchObject({
      showContainerPlan: true,
      showRevenueTon: false,
    });
    expect(
      resolveSeaOrderFormPolicy({
        shipmentType: SEA_SHIPMENT_TYPE.BREAK_BULK,
      }),
    ).toMatchObject({
      showContainerPlan: false,
      showRevenueTon: true,
    });
  });
});
