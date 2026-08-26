import { describe, expect, it } from 'vitest';
import {
  recommendedServiceIDs,
  resolveSeaOrderFormPolicy,
  SEA_SERVICE_CODE,
  SEA_SHIPMENT_MODE,
  SEA_SHIPMENT_TYPE,
} from './sea-order-policy';

describe('海运出口表单策略', () => {
  it('按集运与跨境模式返回不同推荐服务', () => {
    expect(
      resolveSeaOrderFormPolicy({
        shipmentMode: SEA_SHIPMENT_MODE.TRADITIONAL_FORWARDING,
      }).recommendedServiceCodes,
    ).toEqual([
      SEA_SERVICE_CODE.BOOKING,
      SEA_SERVICE_CODE.TRUCKING,
      SEA_SERVICE_CODE.CUSTOMS_EXPORT,
      SEA_SERVICE_CODE.STUFFING,
    ]);
    expect(
      resolveSeaOrderFormPolicy({
        shipmentMode: SEA_SHIPMENT_MODE.CROSS_BORDER,
      }).recommendedServiceCodes,
    ).toEqual([
      SEA_SERVICE_CODE.TRUCKING,
      SEA_SERVICE_CODE.CUSTOMS_EXPORT,
      SEA_SERVICE_CODE.CUSTOMS_IMPORT,
      SEA_SERVICE_CODE.OVERSEA_SEGMENT,
      SEA_SERVICE_CODE.WAREHOUSING,
      SEA_SERVICE_CODE.INSURANCE,
    ]);
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

  it('只将存在于当前组织主数据中的推荐服务转换为 ID', () => {
    expect(
      recommendedServiceIDs(
        [
          { code: SEA_SERVICE_CODE.BOOKING, value: 'booking-id' },
          { code: SEA_SERVICE_CODE.TRUCKING, value: 'trucking-id' },
          { code: SEA_SERVICE_CODE.OVERSEA_SEGMENT, value: 'oversea-id' },
        ],
        SEA_SHIPMENT_MODE.TRADITIONAL_FORWARDING,
      ),
    ).toEqual(['booking-id', 'trucking-id']);
  });
});
