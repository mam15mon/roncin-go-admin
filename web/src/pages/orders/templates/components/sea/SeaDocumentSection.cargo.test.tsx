import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { App, Form } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  seaDocumentServiceExecuteChangeSeaDocumentMode,
  seaDocumentServicePreviewChangeSeaDocumentMode,
} from '@/services/roncin/seaDocumentService';
import { SeaDocumentSectionComponent } from './SeaDocumentSection';

const workspaceAccess = vi.hoisted(() => ({ canOperate: true }));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => workspaceAccess.canOperate,
    canOrder: () => true,
  }),
}));

vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceExecuteChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceGetSeaOrderDocuments: vi.fn(),
  seaDocumentServicePreviewChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceUpdateSeaHouseBill: vi.fn(),
  seaDocumentServiceUpdateSeaMasterBillContent: vi.fn(),
}));

vi.mock('@/services/roncin/orderReleasePodService', () => ({
  orderReleasePodServiceListReleasePods: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderAttachmentService', () => ({
  orderAttachmentServiceListAttachments: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

vi.mock('./SeaDocumentHistoryActions', () => ({
  default: () => <span data-testid="document-history-actions" />,
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceGetPartner: vi.fn(),
}));

vi.mock('./SeaExternalConfirmationFields', async () => {
  const { Form: AntForm, Input } = await import('antd');
  return {
    default: () => (
      <>
        <AntForm.Item name="confirmedByParty" rules={[{ required: true }]}>
          <Input placeholder="例如：XX 船代操作部" />
        </AntForm.Item>
        <AntForm.Item
          name="confirmedAt"
          initialValue="2026-09-07T12:00:00.000Z"
          rules={[{ required: true }]}
        >
          <Input />
        </AntForm.Item>
        <AntForm.Item name="confirmationNote" rules={[{ required: true }]}>
          <Input placeholder="记录确认渠道、联系人和确认结论" />
        </AntForm.Item>
      </>
    ),
    buildSeaExternalConfirmation: (values: {
      confirmedByParty?: string;
      confirmedAt?: string;
      confirmationNote?: string;
    }) => ({
      confirmedByParty: values.confirmedByParty?.trim() ?? '',
      confirmedAt: values.confirmedAt ?? '',
      confirmationNote: values.confirmationNote?.trim() ?? '',
    }),
  };
});

const previewMode = vi.mocked(seaDocumentServicePreviewChangeSeaDocumentMode);
const executeMode = vi.mocked(seaDocumentServiceExecuteChangeSeaDocumentMode);

function TestForm({
  initialValues,
  isDetail = false,
  disabled = false,
  exposeForm,
}: {
  initialValues?: Record<string, unknown>;
  isDetail?: boolean;
  disabled?: boolean;
  exposeForm?: (form: FormInstance) => void;
}) {
  const [form] = Form.useForm();
  exposeForm?.(form);
  return (
    <App>
      <Form form={form} initialValues={initialValues}>
        <SeaDocumentSectionComponent isDetail={isDetail} disabled={disabled} />
      </Form>
    </App>
  );
}

describe('SeaDocumentSectionComponent', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    workspaceAccess.canOperate = true;
    previewMode.mockResolvedValue({
      data: {
        executable: true,
        differences: [],
        impacts: [],
      },
    });
    executeMode.mockResolvedValue({ success: true });
  });

  it('支持一键从订单货物信息带入品名、件数、单位及毛重体积', async () => {
    let capturedForm: FormInstance | undefined;
    render(
      <TestForm
        initialValues={{
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          goodsDescription: 'AUTO PARTS / 汽车配件',
          totalPackages: 800,
          totalPackageUnit: 'CTNS',
          totalGrossWeightKg: 15200.5,
          totalVolumeCbm: 45.8,
        }}
        exposeForm={(form) => {
          capturedForm = form;
        }}
      />,
    );

    const importCargoBtn = screen.getByRole('button', {
      name: /从订单货物信息带入/,
    });
    expect(importCargoBtn).toBeInTheDocument();
    fireEvent.click(importCargoBtn);

    await waitFor(() => {
      const content = capturedForm?.getFieldValue('seaMasterBillContent');
      expect(content?.goodsDescriptionText).toBe('AUTO PARTS / 汽车配件');
      expect(content?.packageCount).toBe(800);
      expect(content?.packageUnit).toBe('CTNS');
      expect(content?.grossWeightKg).toBe(15200.5);
      expect(content?.volumeCbm).toBe(45.8);
    });
  });

  it('支持英文品名免责条款快捷勾选与一键带入委托件重尺', async () => {
    let capturedForm: FormInstance | undefined;
    render(
      <TestForm
        initialValues={{
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          totalPackages: 500,
          totalPackageUnit: 'PLTS',
          totalGrossWeightKg: 12000.5,
          totalVolumeCbm: 32.4,
          seaMasterBillContent: {
            goodsDescriptionText: 'CAR PARTS',
          },
        }}
        exposeForm={(form) => {
          capturedForm = form;
        }}
      />,
    );

    // 1. 勾选免责条款 SHIPPER LOAD,COUNT AND SEAL
    const clauseCheckbox = screen.getByLabelText('免责条款');
    expect(clauseCheckbox).toBeInTheDocument();
    expect(clauseCheckbox).not.toBeChecked();

    fireEvent.click(clauseCheckbox);
    await waitFor(() => {
      expect(clauseCheckbox).toBeChecked();
    });
    expect(
      capturedForm?.getFieldValue([
        'seaMasterBillContent',
        'goodsDescriptionText',
      ]),
    ).toBe('CAR PARTS\nSHIPPER LOAD,COUNT AND SEAL');

    // 2. 取消勾选免责条款
    fireEvent.click(clauseCheckbox);
    await waitFor(() => {
      expect(clauseCheckbox).not.toBeChecked();
    });
    expect(
      capturedForm?.getFieldValue([
        'seaMasterBillContent',
        'goodsDescriptionText',
      ]),
    ).toBe('CAR PARTS');

    // 3. 点击「带入委托件重尺 ↓」按钮
    const importMeasurementsBtn = screen.getByRole('button', {
      name: /带入委托件重尺/,
    });
    fireEvent.click(importMeasurementsBtn);

    await waitFor(() => {
      const content = capturedForm?.getFieldValue('seaMasterBillContent');
      expect(content?.packageCount).toBe(500);
      expect(content?.packageUnit).toBe('PLTS');
      expect(content?.grossWeightKg).toBe(12000.5);
      expect(content?.volumeCbm).toBe(32.4);
    });
  });
});
