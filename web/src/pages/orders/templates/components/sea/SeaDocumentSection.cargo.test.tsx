import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
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

  it('委托只读摘要同步源数据，MBL 与 HBL 实际件重尺独立编辑和带入', async () => {
    let capturedForm!: FormInstance;
    render(
      <TestForm
        initialValues={{
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
          totalPackages: 0,
          totalVolumeCbm: '10.500',
          seaMasterBillContent: { packageCount: 8, grossWeightKg: 12.125 },
          seaHouseBill: { content: { packageCount: 3, grossWeightKg: 4.5 } },
        }}
        exposeForm={(form) => {
          capturedForm = form;
        }}
      />,
    );

    let panel = within(screen.getByRole('tabpanel'));
    expect(panel.getByLabelText('委托件数及单位')).toHaveTextContent(/^0$/);
    expect(panel.getByLabelText('委托毛重（KGS）')).toHaveTextContent(/^-$/);
    expect(panel.getByLabelText('委托体积（CBM）')).toHaveTextContent(/^10.5$/);
    fireEvent.change(panel.getByLabelText('实际毛重（KGS）'), {
      target: { value: '15.125' },
    });
    await waitFor(() => {
      expect(
        capturedForm.getFieldValue(['seaMasterBillContent', 'grossWeightKg']),
      ).toBe(15.125);
    });

    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    panel = within(screen.getByRole('tabpanel'));
    expect(panel.getByLabelText('实际毛重（KGS）')).toHaveValue('4.500');
    act(() => {
      capturedForm.setFieldsValue({
        totalPackages: 20,
        totalPackageUnit: 'CTNS',
        totalGrossWeightKg: 33.25,
        totalVolumeCbm: 6.5,
      });
    });
    await waitFor(() => {
      expect(panel.getByLabelText('委托件数及单位')).toHaveTextContent(
        /^20 CTNS$/,
      );
      expect(panel.getByLabelText('委托毛重（KGS）')).toHaveTextContent(
        /^33.25$/,
      );
      expect(panel.getByLabelText('委托体积（CBM）')).toHaveTextContent(
        /^6.5$/,
      );
    });
    expect(
      capturedForm.getFieldValue(['seaHouseBill', 'content', 'grossWeightKg']),
    ).toBe(4.5);
    fireEvent.click(panel.getByRole('button', { name: /带入委托件重尺/ }));
    await waitFor(() => {
      expect(
        capturedForm.getFieldValue(['seaHouseBill', 'content']),
      ).toMatchObject({
        packageCount: 20,
        packageUnit: 'CTNS',
        grossWeightKg: 33.25,
        volumeCbm: 6.5,
      });
    });
    expect(
      capturedForm.getFieldValue(['seaMasterBillContent', 'packageCount']),
    ).toBe(8);
    expect(
      capturedForm.getFieldValue(['seaMasterBillContent', 'grossWeightKg']),
    ).toBe(15.125);
    fireEvent.click(screen.getByRole('tab', { name: /主单 \(MBL\)/ }));
    panel = within(screen.getByRole('tabpanel'));
    expect(panel.getByLabelText('委托件数及单位')).toHaveTextContent(
      /^20 CTNS$/,
    );
    expect(panel.getByLabelText('实际毛重（KGS）')).toHaveValue('15.125');
  });
});
