import { ProForm } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
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
  seaDocumentServiceUpdateSeaMasterBillContent,
} from '@/services/roncin/seaDocumentService';
import { SeaDocumentSectionComponent } from './SeaDocumentSection';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => true,
    canOperateOrganization: () => true,
  }),
}));
vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceGetSeaOrderDocuments: vi.fn(),
  seaDocumentServiceUpdateSeaHouseBill: vi.fn(),
  seaDocumentServiceUpdateSeaMasterBillContent: vi.fn(),
  seaDocumentServicePreviewChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceExecuteChangeSeaDocumentMode: vi.fn(),
}));
vi.mock('@/services/roncin/orderReleasePodService', () => ({
  orderReleasePodServiceListReleasePods: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));
vi.mock('./SeaDocumentHistoryActions', () => ({ default: () => null }));
// 正文布局另有专项测试，此处保留真实 Form 绑定以验证两个文档的写入边界。
vi.mock('./SeaBillContentFormFields', () => ({
  SeaBillContentFormFields: ({
    namePathPrefix,
    disabled,
  }: {
    namePathPrefix: string[];
    disabled?: boolean;
  }) => (
    <Form.Item name={[...namePathPrefix, 'shipperText']}>
      <input disabled={disabled} />
    </Form.Item>
  ),
}));

const masterBill: API.SeaMasterBillDetail = {
  id: 'mbl-1',
  masterNo: 'MBL001',
  version: '7',
  status: 'DRAFT',
  content: { shipperText: '主单原文' },
};
const houseBill: API.SeaHouseBill = {
  id: 'hbl-1',
  houseNo: 'HBL001',
  version: '4',
  status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_DRAFT,
  issuerSource:
    SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
  content: { shipperText: '分单原文' },
};

function setup(disabled = false) {
  let form!: FormInstance;
  function Harness() {
    [form] = Form.useForm();
    return (
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <ProForm
            form={form}
            submitter={false}
            initialValues={{ id: 'order-1', version: '3' }}
          >
            <SeaDocumentSectionComponent
              isDetail
              disabled={disabled}
              organizationId="org-1"
            />
          </ProForm>
        </App>
      </ConfigProvider>
    );
  }
  render(<Harness />);
  return () => form;
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(seaDocumentServiceGetSeaOrderDocuments).mockResolvedValue({
    data: {
      documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
      linkVersion: '2',
      masterBill,
      houseBill,
    },
  });
});

describe('海运单证统一保存呈现', () => {
  it('主分单字段保留在同一表单，两个页签均不再提供局部保存', async () => {
    const getForm = setup();
    await waitFor(() => expect(screen.getByText('单证版本 v2')).toBeVisible());
    act(() => {
      getForm().setFieldValue(
        ['seaMasterBillContent', 'shipperText'],
        '主单修改',
      );
      getForm().setFieldValue(
        ['seaHouseBill', 'content', 'shipperText'],
        '分单修改',
      );
    });
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    expect(
      screen.getByText('分单号、签发主体与分单正文将随整页保存一起提交。'),
    ).toBeVisible();
    expect(screen.queryByRole('button', { name: /保存分单/ })).toBeNull();
    fireEvent.click(screen.getByRole('tab', { name: /主单 \(MBL\)/ }));
    expect(screen.queryByRole('button', { name: /保存主单内容/ })).toBeNull();
    expect(
      getForm().getFieldValue(['seaMasterBillContent', 'shipperText']),
    ).toBe('主单修改');
    expect(
      getForm().getFieldValue(['seaHouseBill', 'content', 'shipperText']),
    ).toBe('分单修改');
    expect(seaDocumentServiceUpdateSeaHouseBill).not.toHaveBeenCalled();
    expect(seaDocumentServiceUpdateSeaMasterBillContent).not.toHaveBeenCalled();
  });

  it('真实单证表单注册基线元数据，使草稿能够保存原始版本', async () => {
    const getForm = setup();
    await waitFor(() => expect(screen.getByText('单证版本 v2')).toBeVisible());
    const registered = getForm().getFieldsValue();
    expect(registered).toMatchObject({
      seaMasterBillContentVersion: '7',
      seaDocumentLinkVersion: '2',
      seaHouseBill: { id: 'hbl-1', expectedVersion: '4' },
    });
  });

  it('加载空响应显示中文错误', async () => {
    vi.mocked(seaDocumentServiceGetSeaOrderDocuments).mockResolvedValueOnce(
      undefined as never,
    );
    setup();
    expect(await screen.findByText('获取单证信息失败')).toBeVisible();
    expect(
      screen.getAllByText('海运单证加载失败，请重试').length,
    ).toBeGreaterThan(0);
    expect(screen.queryByText(/Cannot read properties/)).toBeNull();
  });

  it('锁定详情禁用模式切换且不显示保存入口', async () => {
    setup(true);
    await waitFor(() => expect(screen.getByText('单证版本 v2')).toBeVisible());
    expect(
      screen.getByRole('button', { name: /切换为仅船公司主单/ }),
    ).toBeDisabled();
    expect(screen.queryByRole('button', { name: /保存主单内容/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /保存分单/ })).toBeNull();
  });
});
