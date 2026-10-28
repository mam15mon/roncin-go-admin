import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, ConfigProvider, Form, type FormInstance } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import {
  seaDocumentServiceGetSeaOrderDocuments,
  seaDocumentServiceUpdateSeaHouseBill,
} from '@/services/roncin/seaDocumentService';
import { SectionCards } from '../../sea-create-layout.fixtures';
import { SeaDocumentSectionComponent } from './SeaDocumentSection';
import { SeaAssociatedHouseBillsField } from './SeaTransportSection';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => true,
    canOperateOrganization: () => true,
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
vi.mock('./SeaDocumentHistoryActions', () => ({ default: () => null }));

vi.mock('./SeaBillContentFormFields', () => ({
  SeaBillContentFormFields: () => null,
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user-1', currentOrganization: { id: 'org-1' } },
    },
  }),
}));

const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
const DIRECT = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;

const getDocuments = vi.mocked(seaDocumentServiceGetSeaOrderDocuments);
const updateHouseBill = vi.mocked(seaDocumentServiceUpdateSeaHouseBill);

beforeEach(() => vi.clearAllMocks());

function mockDocuments(documentStructure: number) {
  getDocuments.mockResolvedValue({
    data: {
      orderId: 'order-1',
      documentStructure,
      linkVersion: '5',
      masterBill: {
        id: 'mbl-id',
        masterNo: 'MBL001',
        version: '7',
        content: {},
      },
      houseBill:
        documentStructure === HOUSE
          ? {
              id: 'hbl-id',
              houseNo: 'HBL-001',
              version: '3',
              currentVersionId: 'hbl-version-id',
              issuerSource:
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
              status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_DRAFT,
              content: {},
            }
          : undefined,
    },
  });
}

function TestForm({
  initialValues,
  isDetail = true,
  exposeForm,
}: {
  initialValues: Record<string, unknown>;
  isDetail?: boolean;
  exposeForm?: (form: FormInstance) => void;
}) {
  const [form] = Form.useForm();
  exposeForm?.(form);
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <Form form={form} initialValues={initialValues}>
          <SeaAssociatedHouseBillsField isDetail={isDetail} />
          <SeaDocumentSectionComponent isDetail={isDetail} />
        </Form>
      </App>
    </ConfigProvider>
  );
}

describe('详情分单号在主单旁编辑', () => {
  it('配舱区唯一输入在主单旁，修改号码后引导整页保存并保留输入', async () => {
    mockDocuments(HOUSE);
    let form!: FormInstance;
    renderWithClient(
      <SectionCards
        isDetail
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport', 'seaDocument']}
        initialValues={{
          id: 'order-1',
          version: '11',
          seaDocumentStructure: HOUSE,
        }}
      />,
    );

    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toHaveValue(
        'HBL-001',
      ),
    );
    const houseInput = screen.getByPlaceholderText('请输入分单号');
    await waitFor(() => expect(houseInput).toBeEnabled());
    expect(screen.getAllByPlaceholderText('请输入分单号')).toHaveLength(1);
    const booking = screen.getByRole('region', { name: '订舱与运输' });
    expect(within(booking).getByPlaceholderText('请输入分单号')).toBe(
      houseInput,
    );
    const masterItem = screen
      .getByPlaceholderText('请输入主单号')
      .closest('.ant-form-item')?.parentElement;
    expect(masterItem?.nextElementSibling).toContainElement(houseInput);
    expect(
      screen.getByText('修改后点击页面「保存」，与订单资料一起提交。'),
    ).toBeVisible();

    fireEvent.change(houseInput, { target: { value: 'HBL-NEW' } });
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    expect(screen.getAllByPlaceholderText('请输入分单号')).toHaveLength(1);
    expect(houseInput.closest('[role="tabpanel"]')).toBeNull();
    expect(form.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL-NEW');
    expect(form.getFieldValue(['seaHouseBill', 'expectedVersion'])).toBe('3');
    expect(screen.queryByRole('button', { name: /保存分单/ })).toBeNull();
    expect(updateHouseBill).not.toHaveBeenCalled();
    expect(getDocuments).toHaveBeenCalledTimes(1);
  });

  it('正式单证加载完成前禁用输入，摘要不能开放编辑', async () => {
    mockDocuments(HOUSE);
    const documents = await getDocuments({ orderId: 'order-1' });
    let resolveDocuments!: (value: typeof documents) => void;
    getDocuments.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveDocuments = resolve;
        }),
    );
    render(
      <TestForm
        isDetail
        initialValues={{
          id: 'order-1',
          seaDocumentSummary: {
            documentStructure: HOUSE,
            houseNo: 'SUMMARY-HBL',
          },
        }}
      />,
    );
    expect(screen.getByPlaceholderText('请输入分单号')).toBeDisabled();
    await act(async () => resolveDocuments(documents));
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toBeEnabled(),
    );
    expect(screen.getByPlaceholderText('请输入分单号')).toHaveValue('HBL-001');
  });

  it('分单空号码时整页校验仍命中上方必填字段', async () => {
    mockDocuments(HOUSE);
    let form!: FormInstance;
    render(
      <TestForm
        isDetail
        initialValues={{ id: 'order-1', seaDocumentStructure: HOUSE }}
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );
    const houseInput = screen.getByPlaceholderText('请输入分单号');
    await waitFor(() => expect(houseInput).toBeEnabled());
    fireEvent.change(houseInput, { target: { value: '' } });
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    await act(async () => {
      await expect(form.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['seaHouseBill', 'houseNo'] }),
        ]),
      });
    });
    expect(await screen.findByText('分单号不能为空')).toBeVisible();
    expect(updateHouseBill).not.toHaveBeenCalled();
  });

  it('只读详情将禁用状态传到主单旁输入', async () => {
    mockDocuments(HOUSE);
    renderWithClient(
      <SectionCards
        isDetail
        readonly
        exposeForm={() => {}}
        sectionKeys={['bookingTransport', 'seaDocument']}
        initialValues={{ id: 'order-1', seaDocumentStructure: HOUSE }}
      />,
    );
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toHaveValue(
        'HBL-001',
      ),
    );
    expect(screen.getByPlaceholderText('请输入分单号')).toBeDisabled();
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    expect(
      screen.queryByRole('button', { name: /保存分单/ }),
    ).not.toBeInTheDocument();
  });

  it('作废分单只可查看号码', async () => {
    mockDocuments(HOUSE);
    const documents = await getDocuments({ orderId: 'order-1' });
    getDocuments.mockResolvedValue({
      data: {
        ...documents.data,
        houseBill: {
          ...documents.data?.houseBill,
          status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED,
        },
      },
    });
    render(
      <TestForm
        isDetail
        initialValues={{ id: 'order-1', seaDocumentStructure: HOUSE }}
      />,
    );
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toHaveValue(
        'HBL-001',
      ),
    );
    expect(screen.getByPlaceholderText('请输入分单号')).toBeDisabled();
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    expect(
      screen.queryByRole('button', { name: /保存分单/ }),
    ).not.toBeInTheDocument();
  });

  it('DIRECT 详情不注册分单号输入', async () => {
    mockDocuments(DIRECT);
    render(
      <TestForm
        isDetail
        initialValues={{ id: 'order-1', seaDocumentStructure: DIRECT }}
      />,
    );
    await waitFor(() =>
      expect(screen.getByText('单证版本 v5')).toBeInTheDocument(),
    );
    expect(screen.getByText('直单，无HBL')).toBeVisible();
    expect(
      screen.queryByPlaceholderText('请输入分单号'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('tab', { name: /分单 \(HBL\)/ }),
    ).not.toBeInTheDocument();
  });
});
