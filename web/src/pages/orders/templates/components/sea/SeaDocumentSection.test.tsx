import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { App, Form } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import {
  seaDocumentServiceExecuteChangeSeaDocumentMode,
  seaDocumentServiceGetSeaOrderDocuments,
  seaDocumentServicePreviewChangeSeaDocumentMode,
} from '@/services/roncin/seaDocumentService';
import {
  DEFAULT_BILL_FORM,
  DEFAULT_FREIGHT_TERMS,
  DEFAULT_RELEASE_TYPE,
  DEFAULT_TRANSPORT_TERMS,
  SEA_BILL_FORM_OPTIONS,
  SEA_FREIGHT_TERM_OPTIONS,
  SEA_RELEASE_TYPE_OPTIONS,
  SEA_TRANSPORT_TERM_OPTIONS,
  SeaDocumentSectionComponent,
} from './SeaDocumentSection';

vi.mock('@umijs/max', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@umijs/max')>()),
  useAccess: () => ({ canOrder: () => true }),
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
const getPartner = vi.mocked(partnerServiceGetPartner);

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
    previewMode.mockResolvedValue({
      data: {
        executable: true,
        differences: [],
        impacts: [],
      },
    });
    executeMode.mockResolvedValue({ success: true });
  });

  it('创建时显式选择模式，HOUSE 只录入一张 HBL，DIRECT 清除 HBL', async () => {
    let form: FormInstance | undefined;
    render(
      <TestForm
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );

    expect(screen.getByText('请先选择单证模式')).toBeInTheDocument();
    expect(
      screen.queryByPlaceholderText('请输入分单号'),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('未确定')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('radio', { name: 'HOUSE（签发 HBL）' }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toBeInTheDocument(),
    );
    expect(screen.queryByText(/添加分单/)).not.toBeInTheDocument();
    expect(screen.queryByText('删除分单')).not.toBeInTheDocument();
    expect(screen.queryByText('箱货分配')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: 'HBL-NEW' },
    });
    fireEvent.click(
      screen.getByRole('radio', { name: 'DIRECT（直接交付 MBL）' }),
    );
    await waitFor(() =>
      expect(
        screen.queryByPlaceholderText('请输入分单号'),
      ).not.toBeInTheDocument(),
    );
    expect(form?.getFieldValue('seaHouseBill')).toBeUndefined();
    expect(form?.getFieldValue('seaDocumentStructure')).toBe(
      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
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
      name: /切换为 DIRECT/,
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
      await screen.findByRole('button', { name: /切换为 HOUSE/ }),
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

  it('支持通知人与第二通知人切换，并支持从订单国外代理带入抬头与地址', async () => {
    let capturedForm: FormInstance | undefined;
    getPartner.mockResolvedValue({
      data: {
        id: 'agent-1',
        legalName: 'Global Shipping Agent Ltd',
        profile: {
          nameEn: 'GLOBAL SHIPPING AGENT LTD',
          addressEn: '123 Ocean Blvd, Hamburg, Germany',
        },
        contacts: [
          { name: 'John Doe', phone: '+49 40 123456', email: 'john@agent.com' },
        ],
      },
    });

    render(
      <TestForm
        initialValues={{
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          foreignAgentId: 'agent-1',
          seaMasterBillContent: {
            notifyPartyText: 'SAME AS CONSIGNEE',
          },
        }}
        exposeForm={(form) => {
          capturedForm = form;
        }}
      />,
    );

    // 默认在通知人标签
    expect(
      screen.getByPlaceholderText(
        '请输入通知人名称与详细地址 (例如：SAME AS CONSIGNEE)',
      ),
    ).toBeVisible();

    // 点击切换到第二通知人
    fireEvent.click(screen.getByText('第二通知人 (Second Notify Party)'));
    const secondNotifyInput = screen.getByPlaceholderText(
      '请输入第二通知人名称与详细地址 (选填，多数提单无需填写)',
    );
    expect(secondNotifyInput).toBeVisible();

    // 录入第二通知人并检查已填写标记
    fireEvent.change(secondNotifyInput, {
      target: { value: 'ALSO NOTIFY CO., LTD' },
    });
    await waitFor(() => {
      expect(screen.getByText('已填写')).toBeInTheDocument();
    });

    // 点击从订单国外代理带入
    const importBtn = screen.getByRole('button', {
      name: /从订单国外代理带入/,
    });
    fireEvent.click(importBtn);

    await waitFor(() => {
      expect(getPartner).toHaveBeenCalledWith({ id: 'agent-1' });
      const foreignAgentVal = capturedForm?.getFieldValue([
        'seaMasterBillContent',
        'foreignAgentText',
      ]);
      expect(foreignAgentVal).toContain('GLOBAL SHIPPING AGENT LTD');
      expect(foreignAgentVal).toContain('123 Ocean Blvd, Hamburg, Germany');
      expect(foreignAgentVal).toContain(
        'TEL/CONTACT: John Doe +49 40 123456 john@agent.com',
      );
    });
  });

  it('提单运输条款与运费条款默认值及下拉选项符合海运标准', async () => {
    let capturedForm: FormInstance | undefined;
    render(
      <TestForm
        initialValues={{
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
        }}
        exposeForm={(form) => {
          capturedForm = form;
        }}
      />,
    );

    // 运输条款默认值为 CY - CY，运费条款默认值为 FREIGHT PREPAID
    await waitFor(() => {
      expect(
        capturedForm?.getFieldValue(['seaMasterBillContent', 'transportTerms']),
      ).toBe('CY - CY');
      expect(
        capturedForm?.getFieldValue(['seaMasterBillContent', 'freightTerms']),
      ).toBe('FREIGHT PREPAID');
    });

    // 运输条款选项包含核心海运条款
    expect(DEFAULT_TRANSPORT_TERMS).toBe('CY - CY');
    expect(
      SEA_TRANSPORT_TERM_OPTIONS.some((opt) => opt.value === 'CY - CY'),
    ).toBe(true);
    expect(
      SEA_TRANSPORT_TERM_OPTIONS.some((opt) => opt.value === 'CFS - CFS'),
    ).toBe(true);
    expect(
      SEA_TRANSPORT_TERM_OPTIONS.some((opt) => opt.value === 'DOOR - DOOR'),
    ).toBe(true);
    expect(
      SEA_TRANSPORT_TERM_OPTIONS.some((opt) => opt.value === 'CY - FO'),
    ).toBe(true);
    expect(
      SEA_TRANSPORT_TERM_OPTIONS.some((opt) => opt.value === 'CFS / DDU'),
    ).toBe(true);

    // 运费条款选项与默认值
    expect(DEFAULT_FREIGHT_TERMS).toBe('FREIGHT PREPAID');
    expect(SEA_FREIGHT_TERM_OPTIONS.map((opt) => opt.value)).toEqual([
      'FREIGHT PREPAID',
      'FREIGHT COLLECT',
      'FREIGHT PAYABLE AT DESTINATION',
      'PAYABLE AT XXX',
      '预付',
      '到付',
    ]);
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

  it('支持一键从委托客户带入发货人，并支持 TO ORDER、SAME AS CONSIGNEE、N/M 等快捷点选', async () => {
    getPartner.mockResolvedValueOnce({
      data: {
        id: 'cust-1',
        legalName: '上海某某外贸进出口有限公司',
        profile: {
          nameEn: 'SHANGHAI TRADING CO., LTD',
          addressEn: '100 EAST NANJING ROAD, SHANGHAI, CHINA',
        },
        contacts: [
          {
            name: 'Alice',
            phone: '+86 21 66668888',
            email: 'alice@shanghaitrading.com',
          },
        ],
      },
    });

    let capturedForm: FormInstance | undefined;
    render(
      <TestForm
        initialValues={{
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          customerId: 'cust-1',
        }}
        exposeForm={(form) => {
          capturedForm = form;
        }}
      />,
    );

    // 1. 点击从委托客户带入发货人
    const importShipperBtn = screen.getByRole('button', {
      name: /从委托客户带入/,
    });
    fireEvent.click(importShipperBtn);

    await waitFor(() => {
      expect(getPartner).toHaveBeenCalledWith({ id: 'cust-1' });
      const shipper = capturedForm?.getFieldValue([
        'seaMasterBillContent',
        'shipperText',
      ]);
      expect(shipper).toContain('SHANGHAI TRADING CO., LTD');
      expect(shipper).toContain('100 EAST NANJING ROAD, SHANGHAI, CHINA');
      expect(shipper).toContain(
        'TEL/CONTACT: Alice +86 21 66668888 alice@shanghaitrading.com',
      );
    });

    // 2. 点击快捷标签 + TO ORDER
    const toOrderTag = screen.getByText('+ TO ORDER');
    fireEvent.click(toOrderTag);
    expect(
      capturedForm?.getFieldValue(['seaMasterBillContent', 'consigneeText']),
    ).toBe('TO ORDER');

    // 3. 点击快捷标签 + SAME AS CONSIGNEE
    const sameAsConsigneeTag = screen.getByText('+ SAME AS CONSIGNEE');
    fireEvent.click(sameAsConsigneeTag);
    expect(
      capturedForm?.getFieldValue(['seaMasterBillContent', 'notifyPartyText']),
    ).toBe('SAME AS CONSIGNEE');

    // 4. 点击快捷标签 + N/M
    const nmTag = screen.getByText('+ N/M');
    fireEvent.click(nmTag);
    expect(
      capturedForm?.getFieldValue(['seaMasterBillContent', 'marksText']),
    ).toBe('N/M');

    // 5. 提单形式与放单方式默认值
    expect(DEFAULT_BILL_FORM).toBe('ORIGINAL');
    expect(DEFAULT_RELEASE_TYPE).toBe('电放');
    expect(SEA_BILL_FORM_OPTIONS.map((o) => o.value)).toContain('ORIGINAL');
    expect(SEA_RELEASE_TYPE_OPTIONS.map((o) => o.value)).toContain('电放');
  });
});
