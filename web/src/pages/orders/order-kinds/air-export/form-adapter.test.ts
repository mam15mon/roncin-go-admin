import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import { OrderBusinessType, OrderPersonnelRole } from '@/enums.generated';
import { airExportFormAdapter, serializeAirMaster } from './form-adapter';

describe('空运表单契约', () => {
  it('已有主单必须显式携带 ID 和版本，共同资料不写入海运字段', () => {
    const request = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      paymentTerm: 1,
      airMasterMode: 'EXISTING',
      airMasterId: 'master',
      airMasterExpectedVersion: '7',
      airMaster: {
        masterNo: '12312345678',
        airlineId: 'airline',
        originAirportId: 'origin',
        destinationAirportId: 'destination',
      },
      airHouseNo: ' h-1 ',
      creatorUserId: 'creator',
      operatorUserId: 'operator',
      salesUserId: 'sales',
      customerServiceUserId: 'service',
    });
    expect(request.businessType).toBe(OrderBusinessType.BUSINESS_TYPE_AE);
    expect(request.airExport).toMatchObject({
      masterWaybillId: 'master',
      expectedMasterVersion: '7',
      houseNo: 'h-1',
    });
    expect(
      request.personnelAssignments?.some(
        (item) => item.role === OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR,
      ),
    ).toBe(false);
    expect(request.shippingLineId).toBeUndefined();
    expect(request.seaMasterBill).toBeUndefined();
    expect(request.personnelAssignments).toContainEqual({
      role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
      userId: 'operator',
    });
  });

  it('本票保存仅提交分单私有字段，不携带主单旧副本', () => {
    const request = airExportFormAdapter.buildUpdatePayload('order', '9', {
      customerId: 'customer',
      paymentTerm: 1,
      airMasterId: 'master',
      airMasterExpectedVersion: '4',
      airMaster: { flightNo: 'OLD123' },
      airHouseNo: 'H1',
      airHouseContent: '本票资料',
    });
    expect(request.airExport?.master).toBeUndefined();
    expect(request.airExport).toMatchObject({
      houseNo: 'H1',
      houseContent: '本票资料',
    });
    expect(request.vesselVoyage).toBeUndefined();
  });

  it('航空起降时间保留时刻，不截成当天零点', () => {
    const at = dayjs('2026-10-21T14:45:00+08:00');
    expect(serializeAirMaster({ etd: at as unknown as string })?.etd).toBe(
      '2026-10-21T06:45:00.000Z',
    );
  });
});
