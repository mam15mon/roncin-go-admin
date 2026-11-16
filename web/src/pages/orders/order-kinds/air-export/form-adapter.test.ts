import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  DangerousPackingGroup,
  OrderBusinessType,
  OrderPersonnelRole,
} from '@/enums.generated';
import {
  airExportFormAdapter,
  serializeAirMaster,
  serializeAirWaybillContent,
} from './form-adapter';

describe('空运表单契约', () => {
  it('直单不发送残留分单资料，已有订单不能通过表单换模式或主单', () => {
    const created = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      paymentTerm: 1,
      airDocumentMode: 'DIRECT',
      airMasterMode: 'NEW',
      airHouseNo: '残留分单',
      airHouseContent: { remarks: '残留正文' },
    });
    expect(JSON.parse(JSON.stringify(created.airExport))).toEqual({
      documentMode: 'DIRECT',
    });
    const updated = airExportFormAdapter.buildUpdatePayload('order', '7', {
      customerId: 'customer',
      paymentTerm: 1,
      airDocumentMode: 'HOUSE',
      airMasterId: 'other-master',
      airMasterExpectedVersion: '99',
      airHouseNo: '残留分单',
      airExport: {
        documentMode: 'DIRECT',
        master: { id: 'original-master', version: '3' },
        linkVersion: '2',
      },
    });
    expect(JSON.parse(JSON.stringify(updated.airExport))).toEqual({
      documentMode: 'DIRECT',
      masterWaybillId: 'original-master',
      expectedMasterVersion: '3',
      expectedLinkVersion: '2',
    });
  });

  it.each([
    undefined,
    {},
    { remarks: '', asArranged: false },
    { remarks: '正文', asArranged: true },
  ])('正文省略、显式清空与零值按 wire 语义往返：%j', (content) => {
    const values = airExportFormAdapter.buildDetailInitialValues(
      {
        customerId: 'customer',
        airExport: {
          documentMode: 'HOUSE',
          linkVersion: '3',
          master: {
            id: 'master',
            version: '5',
            content: { waybillContent: { remarks: '主单不可被整票覆盖' } },
          },
          house: {
            id: 'house',
            houseNo: 'H1',
            status: 'DRAFT',
            version: '2',
            content,
          },
        },
      },
      [],
      [],
    );
    expect(values.airHouseContent).toEqual(content);
    const payload = airExportFormAdapter.buildUpdatePayload('order', '4', {
      ...values,
      customerId: 'customer',
      paymentTerm: 1,
    });
    const wire = JSON.parse(JSON.stringify(payload.airExport));
    expect(wire.house.content).toEqual(content);
    expect(wire.house.expectedVersion).toBe('2');
    expect(wire.master).toBeUndefined();
  });

  it('声明与费用空注册消息省略，显式零、无效非空输入及正文清空保留', () => {
    expect(serializeAirWaybillContent(undefined)).toBeUndefined();
    expect(serializeAirWaybillContent({})).toEqual({});
    expect(
      JSON.parse(
        JSON.stringify(
          serializeAirWaybillContent({
            carriageValue: { kind: undefined, amount: '' },
            customsValue: { kind: 'AMOUNT', amount: '0' },
            insuranceValue: { amount: '非法输入' },
            prepaid: { valuationCharge: undefined, tax: '' },
            collect: { tax: '0' },
            remarks: '',
            asArranged: false,
          }),
        ),
      ),
    ).toEqual({
      customsValue: { kind: 'AMOUNT', amount: '0' },
      insuranceValue: { amount: '非法输入' },
      collect: { tax: '0' },
      remarks: '',
      asArranged: false,
    });
    expect(
      serializeAirMaster({
        waybillContent: { carriageValue: {}, remarks: '主单' },
      })?.waybillContent,
    ).toEqual({ remarks: '主单' });
  });
  it('已有主单必须显式携带 ID 和版本，共同资料不写入海运字段', () => {
    const request = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      airDocumentMode: 'HOUSE',
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
      documentMode: 'HOUSE',
      house: { houseNo: 'h-1' },
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
      airDocumentMode: 'HOUSE',
      paymentTerm: 1,
      airMasterId: 'master',
      airMasterExpectedVersion: '4',
      airMaster: { flightNo: 'OLD123' },
      airHouseNo: 'H1',
      airHouseContent: { remarks: '本票资料' },
      airExport: {
        documentMode: 'HOUSE',
        master: { id: 'master', version: '4' },
        house: { version: '2' },
        linkVersion: '3',
      },
    });
    expect(request.airExport?.master).toBeUndefined();
    expect(request.airExport).toMatchObject({
      documentMode: 'HOUSE',
      expectedLinkVersion: '3',
      house: {
        houseNo: 'H1',
        content: { remarks: '本票资料' },
        expectedVersion: '2',
      },
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
      airDocumentMode: 'HOUSE',
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
      airDocumentMode: 'HOUSE',
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
      airDocumentMode: 'HOUSE',
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
      airDocumentMode: 'HOUSE',
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
      airDocumentMode: 'HOUSE',
      paymentTerm: 1,
      dangerousPackingGroup: DangerousPackingGroup.DANGEROUS_PACKING_GROUP_I,
    });
    expect(updated.dangerousPackingGroup).toBe(
      DangerousPackingGroup.DANGEROUS_PACKING_GROUP_I,
    );

    const empty = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      airDocumentMode: 'HOUSE',
      paymentTerm: 1,
    });
    expect(empty.dangerousPackingGroup).toBeUndefined();

    const unspecified = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      airDocumentMode: 'HOUSE',
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

describe('空运本票经营字段往返', () => {
  it('新建保留原始金额精度、币种及本票日期时刻，不写进主单', () => {
    const payload = airExportFormAdapter.buildCreatePayload({
      customerId: 'customer',
      airDocumentMode: 'HOUSE',
      paymentTerm: 2,
      shipmentMode: 2,
      tradeTerm: 1,
      bookingAgentId: 'supplier',
      foreignAgentId: 'agent',
      contractNo: ' contract ',
      cargoValue: '0',
      cargoCurrency: 'USD',
      insurancePremium: '123.4567',
      insuranceCurrency: 'CNY',
      factoryName: ' 工厂 ',
      receivedAt: dayjs('2026-11-16T10:20:00+08:00'),
      cargoReadyAt: dayjs('2026-11-17T01:02:03+08:00'),
      bookingNotes: ' 订舱 ',
      allocationNotes: ' 配舱 ',
      operationNotes: ' 操作 ',
      notes: ' 普通 ',
    });
    expect(payload).toMatchObject({
      paymentTerm: 2,
      shipmentMode: 2,
      tradeTerm: 1,
      bookingAgentId: 'supplier',
      foreignAgentId: 'agent',
      contractNo: 'contract',
      cargoValue: '0',
      cargoCurrency: 'USD',
      insurancePremium: '123.4567',
      insuranceCurrency: 'CNY',
      factoryName: '工厂',
      receivedAt: '2026-11-16T02:20:00.000Z',
      cargoReadyAt: '2026-11-16T17:02:03.000Z',
      bookingNotes: '订舱',
      allocationNotes: '配舱',
      operationNotes: '操作',
      notes: '普通',
    });
    expect(payload.airExport?.master).toBeUndefined();
  });

  it('详情将时间回填DatePicker对象，省略optional字段不制造值，显式零金额不丢失', () => {
    const values = airExportFormAdapter.buildDetailInitialValues(
      {
        customerId: 'customer',
        paymentTerm: 2,
        shipmentMode: 2,
        cargoValue: '0',
        cargoCurrency: 'USD',
        receivedAt: '2026-11-16T02:20:00Z',
        cargoReadyAt: '2026-11-16T17:02:03Z',
      },
      [],
      [],
    );
    expect(dayjs.isDayjs(values.receivedAt)).toBe(true);
    expect(dayjs.isDayjs(values.cargoReadyAt)).toBe(true);
    expect(values.cargoValue).toBe('0');
    expect(values.insurancePremium).toBeUndefined();
    const payload = airExportFormAdapter.buildUpdatePayload('order', '4', {
      ...values,
      customerId: 'customer',
      airDocumentMode: 'HOUSE',
      paymentTerm: 2,
    });
    expect(payload.receivedAt).toBe('2026-11-16T02:20:00.000Z');
    expect(payload.cargoValue).toBe('0');
    expect(payload.shipmentMode).toBe(2);
  });

  it('更新清空经营字段时发送明确空值，JSON序列化后仍保留清除意图', () => {
    const payload = airExportFormAdapter.buildUpdatePayload('order', '4', {
      customerId: 'customer',
      airDocumentMode: 'HOUSE',
      paymentTerm: 1,
      contractNo: ' ',
      factoryName: '',
      bookingNotes: '',
      allocationNotes: '',
      notes: '',
    });
    expect(JSON.parse(JSON.stringify(payload))).toMatchObject({
      id: 'order',
      expectedVersion: '4',
      shipmentMode: 0,
      tradeTerm: 0,
      bookingAgentId: '',
      foreignAgentId: '',
      cargoValue: '',
      cargoCurrency: '',
      insurancePremium: '',
      insuranceCurrency: '',
      receivedAt: '',
      cargoReadyAt: '',
      factoryName: '',
      contractNo: '',
      bookingNotes: '',
      allocationNotes: '',
      notes: '',
      operationNotes: '',
    });
  });
});
