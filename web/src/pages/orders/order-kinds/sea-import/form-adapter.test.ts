import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  OrderBusinessType,
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
  TradeDirection,
} from '@/enums.generated';
import type { CreateOrderFormValues } from '../form-values';
import { seaImportDefinition } from './definition';

const adapter = seaImportDefinition.form;

describe('海运进口共享字段映射', () => {
  it('使用SI/IMPORT真实身份，直单没有HBL，主单航程保持结构化输入', () => {
    const values: CreateOrderFormValues = {
      customerId: 'customer',
      paymentTerm: 2,
      seaDocumentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
      seaMasterBillMasterNo: 'IMPORT-MBL',
      shippingLineId: 'carrier',
      seaHouseBill: { houseNo: '隐藏草稿', issuerSource: 1 },
      vesselName: 'IMPORT SHIP',
      voyageNo: 'V12',
      etd: dayjs('2026-11-01'),
      eta: dayjs('2026-11-20'),
    };
    const payload = adapter.buildCreatePayload(values);
    expect(payload).toMatchObject({
      businessType: OrderBusinessType.BUSINESS_TYPE_SI,
      tradeDirection: TradeDirection.TRADE_DIRECTION_IMPORT,
      seaMasterBill: { masterNo: 'IMPORT-MBL' },
      seaDocument: {
        documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
      },
      vesselVoyage: 'IMPORT SHIP / V12',
    });
    expect(payload.seaDocument?.houseBill).toBeUndefined();
    expect(payload.shippingDocuments).toBeUndefined();
  });

  it('HOUSE保留外部签发主体/正文/印字，关联已有主单不覆盖共享正文', () => {
    const payload = adapter.buildCreatePayload({
      customerId: 'customer',
      paymentTerm: 2,
      seaDocumentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
      seaMasterBillMasterNo: 'IMPORT-MBL',
      seaMasterBillCandidateId: 'master',
      seaMasterBillExpectedCandidateVersion: '7',
      seaMasterBillCandidateTeId: 'voyage',
      seaMasterBillExpectedCandidateTeVersion: '3',
      seaMasterBillContent: { shipperText: '不得覆盖共享正文' },
      seaHouseBill: {
        houseNo: 'IMPORT-HBL',
        issuerSource:
          SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER,
        issuerPartnerId: 'foreign-issuer',
        content: { shipperText: '原发货人' },
        printDetails: { issueDate: '2026-11-01', originalCount: 0 },
      },
    });
    expect(payload.seaMasterBill).toMatchObject({
      candidateId: 'master',
      expectedCandidateVersion: '7',
      candidateTeId: 'voyage',
      expectedCandidateTeVersion: '3',
    });
    expect(payload.seaDocument?.masterBillContent).toBeUndefined();
    expect(payload.seaDocument?.houseBill).toMatchObject({
      issuerPartnerId: 'foreign-issuer',
      content: { shipperText: '原发货人' },
      printDetails: { issueDate: '2026-11-01', originalCount: 0 },
    });
  });

  it('新建不擅自以当前公司签发进口分单，接单/货好和金额零值往返独立', () => {
    const defaults = adapter.buildCreateDefaults({ cargoCategoryOptions: [] });
    expect(defaults.seaHouseBill?.issuerSource).toBeUndefined();
    const values = adapter.buildDetailInitialValues(
      {
        id: 'order',
        cargoValue: '0',
        cargoCurrency: 'USD',
        insurancePremium: '32.1500',
        insuranceCurrency: 'CNY',
        receivedAt: '2026-11-01T23:15:00+08:00',
        cargoReadyAt: '2026-11-02T00:30:00+08:00',
      },
      [],
      [],
    );
    const update = adapter.buildUpdatePayload('order', '4', {
      customerId: 'customer',
      paymentTerm: 2,
      ...values,
    });
    expect(update).toMatchObject({
      expectedVersion: '4',
      cargoValue: '0',
      cargoCurrency: 'USD',
      insurancePremium: '32.1500',
      insuranceCurrency: 'CNY',
      receivedAt: '2026-11-01T15:15:00.000Z',
      cargoReadyAt: '2026-11-01T16:30:00.000Z',
    });
  });

  it('清空经营字段保留JSON中的显式清除意图，不触碰独立进口服务', () => {
    const update = JSON.parse(
      JSON.stringify(
        adapter.buildUpdatePayload('order', '4', {
          customerId: 'customer',
          paymentTerm: 2,
        }),
      ),
    );
    expect(update).toMatchObject({
      tradeTerm: 0,
      customerReferenceNo: '',
      contractNo: '',
      bookingAgentId: '',
      foreignAgentId: '',
      shippingAgentId: '',
      cargoValue: '',
      cargoCurrency: '',
      insurancePremium: '',
      insuranceCurrency: '',
      factoryName: '',
      receivedAt: '',
      cargoReadyAt: '',
      bookingNotes: '',
      allocationNotes: '',
      operationNotes: '',
    });
    expect(update).not.toHaveProperty('importHandling');
    expect(update).not.toHaveProperty('businessType');
  });
});
