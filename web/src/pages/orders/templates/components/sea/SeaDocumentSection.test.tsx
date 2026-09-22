import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { App, Form } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
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

const getDocuments = vi.mocked(seaDocumentServiceGetSeaOrderDocuments);
const previewMode = vi.mocked(seaDocumentServicePreviewChangeSeaDocumentMode);
const executeMode = vi.mocked(seaDocumentServiceExecuteChangeSeaDocumentMode);

function TestForm({
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
    <App>
      <Form form={form} initialValues={initialValues}>
        {includeModeField ? <SeaCreateDocumentModeField /> : null}
        <SeaDocumentSectionComponent isDetail={isDetail} disabled={disabled} />
      </Form>
    </App>
  );
}

function fillExternalConfirmation() {
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

function mockDocuments(structure: number) {
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

  it('新建默认 HOUSE 分单制，切 DIRECT 隐藏 HBL 但保留草稿，切回恢复', async () => {
    let form: FormInstance | undefined;
    render(
      <TestForm
        includeModeField
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );

    // 全员分单制：新建默认 HOUSE，出现 HBL 分单页签且无「请先选择模式」提示。
    await waitFor(() => {
      expect(
        screen.getByRole('tab', { name: /分单 \(HBL\)/ }),
      ).toBeInTheDocument();
      expect(form?.getFieldValue('seaDocumentStructure')).toBe(
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
      );
    });
    expect(screen.queryByText('请先选择提单模式')).not.toBeInTheDocument();

    // HBL 页签强制渲染，无需点击即可展示唯一分单录入。
    expect(screen.getByPlaceholderText('请输入分单号')).toBeInTheDocument();
    expect(screen.queryByText(/添加分单/)).not.toBeInTheDocument();
    expect(screen.queryByText('删除分单')).not.toBeInTheDocument();
    expect(screen.queryByText('箱货分配')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: 'HBL-NEW' },
    });
    fireEvent.click(
      screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
    );
    await waitFor(() =>
      expect(
        screen.queryByPlaceholderText('请输入分单号'),
      ).not.toBeInTheDocument(),
    );
    // 隐藏草稿保留在 Form store，仅页签与提交口径排除 HBL。
    expect(form?.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL-NEW');
    expect(form?.getFieldValue('seaDocumentStructure')).toBe(
      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
    );

    // 切回 HOUSE 恢复分单录入与草稿。
    fireEvent.click(screen.getByRole('radio', { name: '有货代分单（HOUSE）' }));
    fireEvent.click(await screen.findByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toHaveValue(
        'HBL-NEW',
      ),
    );
  });

  it('HOUSE→DIRECT 预览后携带外部确认和当前 HBL 版本执行', async () => {
    mockDocuments(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE);
    render(
      <TestForm
        isDetail
        disabled
        initialValues={{ id: 'order-1', version: '11' }}
      />,
    );

    const switchButton = await screen.findByRole('button', {
      name: /切换为仅船公司主单（DIRECT）/,
    });
    expect(switchButton).toBeEnabled();
    fireEvent.click(switchButton);
    fireEvent.change(
      screen.getByPlaceholderText('说明客户请求及本次 HOUSE/DIRECT 切换原因'),
      {
        target: { value: '客户要求改为直单' },
      },
    );
    fillExternalConfirmation();
    fireEvent.click(screen.getByRole('button', { name: '预览切换影响' }));

    await waitFor(() => {
      expect(previewMode).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        expect.objectContaining({
          orderId: 'order-1',
          targetMode: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          reason: '客户要求改为直单',
          newHouseBill: undefined,
        }),
      );
    });
    fireEvent.click(await screen.findByRole('button', { name: '确认执行' }));
    await waitFor(() => {
      expect(executeMode).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        expect.objectContaining({
          expectedOrderVersion: '11',
          expectedLinkVersion: '5',
          expectedHouseBillVersion: '3',
          expectedCurrentVersionId: 'hbl-version-id',
          targetMode: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          confirmation: expect.objectContaining({
            confirmedByParty: '测试船代',
            confirmationNote: '船代邮件确认可以变更',
          }),
        }),
      );
    });
  });

  it('DIRECT→HOUSE 提交唯一 HBL，确认信息变化后要求重新预览', async () => {
    mockDocuments(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT);
    render(
      <TestForm isDetail initialValues={{ id: 'order-1', version: '11' }} />,
    );

    fireEvent.click(
      await screen.findByRole('button', { name: /切换为有货代分单（HOUSE）/ }),
    );
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: 'HBL-NEW' },
    });
    fireEvent.click(screen.getByRole('radio', { name: '本公司' }));
    fireEvent.change(
      screen.getByPlaceholderText('说明客户请求及本次 HOUSE/DIRECT 切换原因'),
      {
        target: { value: '客户要求签发分单' },
      },
    );
    fillExternalConfirmation();
    fireEvent.click(screen.getByRole('button', { name: '预览切换影响' }));

    await screen.findByRole('button', { name: '确认执行' });
    expect(previewMode).toHaveBeenLastCalledWith(
      { orderId: 'order-1' },
      expect.objectContaining({
        targetMode: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
        newHouseBill: expect.objectContaining({
          houseNo: 'HBL-NEW',
          issuerSource:
            SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
        }),
      }),
    );

    fireEvent.change(
      screen.getByPlaceholderText('记录确认渠道、联系人和确认结论'),
      {
        target: { value: '船代再次邮件确认可以变更' },
      },
    );
    expect(
      screen.queryByRole('button', { name: '确认执行' }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '预览切换影响' })).toBeEnabled();
  }, 60000);

  it('外公司单证详情可查看，但不能切换提单模式', async () => {
    workspaceAccess.canOperate = false;
    mockDocuments(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE);
    render(
      <TestForm
        isDetail
        disabled
        initialValues={{ id: 'order-1', version: '11' }}
      />,
    );
    await waitFor(() => expect(getDocuments).toHaveBeenCalled());
    await screen.findByText('MBL001');
    expect(
      screen.queryByRole('button', { name: /切换为仅船公司主单（DIRECT）/ }),
    ).not.toBeInTheDocument();
    expect(previewMode).not.toHaveBeenCalled();
    expect(executeMode).not.toHaveBeenCalled();
  });
});
