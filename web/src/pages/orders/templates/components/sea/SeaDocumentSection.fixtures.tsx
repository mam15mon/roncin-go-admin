import { fireEvent, screen } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { App, ConfigProvider, Form } from 'antd';
import { beforeEach, vi } from 'vitest';
import {
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import {
  seaDocumentServiceExecuteChangeSeaDocumentMode,
  seaDocumentServiceGetSeaOrderDocuments,
  seaDocumentServicePreviewChangeSeaDocumentMode,
} from '@/services/roncin/seaDocumentService';
import { SeaCreateDocumentModeField } from './SeaCreateDocumentModeField';
import { SeaDocumentSectionComponent } from './SeaDocumentSection';
import { SeaAssociatedHouseBillsField } from './SeaTransportSection';

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

export const getDocuments = vi.mocked(seaDocumentServiceGetSeaOrderDocuments);
export const previewMode = vi.mocked(
  seaDocumentServicePreviewChangeSeaDocumentMode,
);
export const executeMode = vi.mocked(
  seaDocumentServiceExecuteChangeSeaDocumentMode,
);

export function TestForm({
  initialValues,
  isDetail = false,
  disabled = false,
  includeModeField = false,
  exposeForm,
}: {
  initialValues?: Record<string, unknown>;
  isDetail?: boolean;
  disabled?: boolean;
  includeModeField?: boolean;
  exposeForm?: (form: FormInstance) => void;
}) {
  const [form] = Form.useForm();
  exposeForm?.(form);
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <Form form={form} initialValues={initialValues}>
          {includeModeField ? <SeaCreateDocumentModeField /> : null}
          {!isDetail && <SeaAssociatedHouseBillsField isDetail={false} />}
          <SeaDocumentSectionComponent
            isDetail={isDetail}
            disabled={disabled}
          />
        </Form>
      </App>
    </ConfigProvider>
  );
}

export function fillExternalConfirmation() {
  fireEvent.change(screen.getByPlaceholderText('例如：XX 船代操作部'), {
    target: { value: '测试船代' },
  });
  fireEvent.change(
    screen.getByPlaceholderText('记录确认渠道、联系人和确认结论'),
    {
      target: { value: '船代邮件确认可以变更' },
    },
  );
}

export function mockDocuments(structure: number) {
  const houseBill =
    structure === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
      ? {
          id: 'hbl-id',
          houseNo: 'HBL-001',
          issuerSource:
            SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
          status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_DRAFT,
          version: '3',
          currentVersionId: 'hbl-version-id',
          content: {},
        }
      : undefined;
  getDocuments.mockResolvedValue({
    data: {
      orderId: 'order-1',
      documentStructure: structure,
      linkVersion: '5',
      masterBill: {
        id: 'mbl-id',
        masterNo: 'MBL001',
        version: '7',
        currentVersionId: 'mbl-version-id',
        content: {},
      },
      houseBill,
    },
  });
}

export function setupDocumentTests() {
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
}

export function setWorkspaceCanOperate(value: boolean) {
  workspaceAccess.canOperate = value;
}
