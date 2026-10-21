import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  DangerousPackingGroup,
  OrderBusinessType,
  OrderPersonnelRole,
} from '@/enums.generated';
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

  it('危险品三元组随共同资料序列化：填写时携带，未填写时不携带', () => {
    const filled = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      paymentTerm: 1,
      unNumber: ' 1263 ',
      hazardClass: ' 3 ',
      declarationCutoffAt: dayjs('2026-10-21T14:45:00+08:00'),
    });
    expect(filled.unNumber).toBe('1263');
    expect(filled.hazardClass).toBe('3');
    expect(filled.declarationCutoffAt).toBe('2026-10-21T06:45:00.000Z');

    const updated = airExportFormAdapter.buildUpdatePayload('order', '9', {
      customerId: 'customer',
      paymentTerm: 1,
      unNumber: '1263',
      hazardClass: '3',
      declarationCutoffAt: '2026-10-21T06:45:00.000Z',
    });
    expect(updated.unNumber).toBe('1263');
    expect(updated.hazardClass).toBe('3');
    expect(updated.declarationCutoffAt).toBe('2026-10-21T06:45:00.000Z');

    const empty = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      paymentTerm: 1,
      unNumber: '   ',
    });
    expect(empty.unNumber).toBeUndefined();
    expect(empty.hazardClass).toBeUndefined();
    expect(empty.declarationCutoffAt).toBeUndefined();
  });

  it('危险品包装类别随共同资料序列化：选择时携带，未选/UNSPECIFIED 不携带', () => {
    const filled = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      paymentTerm: 1,
      unNumber: '1263',
      hazardClass: '3',
      dangerousPackingGroup: DangerousPackingGroup.DANGEROUS_PACKING_GROUP_III,
    });
    expect(filled.dangerousPackingGroup).toBe(
      DangerousPackingGroup.DANGEROUS_PACKING_GROUP_III,
    );

    const updated = airExportFormAdapter.buildUpdatePayload('order', '9', {
      customerId: 'customer',
      paymentTerm: 1,
      dangerousPackingGroup: DangerousPackingGroup.DANGEROUS_PACKING_GROUP_I,
    });
    expect(updated.dangerousPackingGroup).toBe(
      DangerousPackingGroup.DANGEROUS_PACKING_GROUP_I,
    );

    const empty = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      paymentTerm: 1,
    });
    expect(empty.dangerousPackingGroup).toBeUndefined();

    const unspecified = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      paymentTerm: 1,
      dangerousPackingGroup:
        DangerousPackingGroup.DANGEROUS_PACKING_GROUP_UNSPECIFIED,
    });
    expect(unspecified.dangerousPackingGroup).toBeUndefined();
  });

  it('详情回填携带危险品包装类别，后端未填时不携带', () => {
    const filled = airExportFormAdapter.buildDetailInitialValues(
      {
        customerId: 'customer',
        unNumber: '1263',
        dangerousPackingGroup: DangerousPackingGroup.DANGEROUS_PACKING_GROUP_II,
      },
      [],
      [],
    );
    expect(filled.dangerousPackingGroup).toBe(
      DangerousPackingGroup.DANGEROUS_PACKING_GROUP_II,
    );

    const empty = airExportFormAdapter.buildDetailInitialValues(
      { customerId: 'customer' },
      [],
      [],
    );
    expect(empty.dangerousPackingGroup).toBeUndefined();
  });

  it('详情回填携带危险品三元组，截申报时间转为 dayjs 对象回显', () => {
    const values = airExportFormAdapter.buildDetailInitialValues(
      {
        customerId: 'customer',
        unNumber: '1263',
        hazardClass: '3',
        declarationCutoffAt: '2026-10-21T06:45:00.000Z',
      },
      [],
      [],
    );
    expect(values.unNumber).toBe('1263');
    expect(values.hazardClass).toBe('3');
    const cutoff = values.declarationCutoffAt;
    expect(dayjs.isDayjs(cutoff)).toBe(true);
    if (dayjs.isDayjs(cutoff)) {
      expect(cutoff.toISOString()).toBe('2026-10-21T06:45:00.000Z');
    }
  });
});
