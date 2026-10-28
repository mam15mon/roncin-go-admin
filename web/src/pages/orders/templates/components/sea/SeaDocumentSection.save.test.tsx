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

async function save(kind: 'master' | 'house') {
  fireEvent.click(
    screen.getByRole('tab', {
      name: kind === 'master' ? /主单 \(MBL\)/ : /分单 \(HBL\)/,
    }),
  );
  await act(async () => {
    fireEvent.click(
      screen.getByRole('button', {
        name: kind === 'master' ? /保存主单内容/ : /保存分单/,
      }),
    );
  });
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
  vi.mocked(seaDocumentServiceUpdateSeaMasterBillContent).mockImplementation(
    async (_params, body) => ({
      data: { ...masterBill, version: '8', content: body.content },
    }),
  );
  vi.mocked(seaDocumentServiceUpdateSeaHouseBill).mockImplementation(
    async (_params, body) => ({
      data: { ...houseBill, ...body.houseBill, version: '5' },
    }),
  );
});

describe('海运单证独立保存', () => {
  it.each(['master', 'house'] as const)(
    '先保存 %s 时保留另一份草稿和原版本',
    async (first) => {
      const getForm = setup();
      await screen.findByRole('button', { name: /保存主单内容/ });
      act(() => {
        getForm().setFieldValue(
          ['seaMasterBillContent', 'shipperText'],
          '主单未保存修改',
        );
        getForm().setFieldValue(
          ['seaHouseBill', 'content', 'shipperText'],
          '分单未保存修改',
        );
      });
      await save(first);
      expect(
        getForm().getFieldValue(['seaMasterBillContent', 'shipperText']),
      ).toBe('主单未保存修改');
      expect(
        getForm().getFieldValue(['seaHouseBill', 'content', 'shipperText']),
      ).toBe('分单未保存修改');
      expect(getForm().getFieldValue(['seaHouseBill', 'expectedVersion'])).toBe(
        first === 'master' ? '4' : '5',
      );
      await save(first === 'master' ? 'house' : 'master');
      expect(seaDocumentServiceUpdateSeaHouseBill).toHaveBeenCalledWith(
        { orderId: 'order-1', id: 'hbl-1' },
        expect.objectContaining({
          expectedVersion: '4',
          expectedLinkVersion: '2',
          houseBill: expect.objectContaining({
            content: expect.objectContaining({ shipperText: '分单未保存修改' }),
          }),
        }),
      );
      expect(seaDocumentServiceUpdateSeaMasterBillContent).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        expect.objectContaining({
          expectedMblVersion: '7',
          content: expect.objectContaining({ shipperText: '主单未保存修改' }),
        }),
      );
      expect(seaDocumentServiceGetSeaOrderDocuments).toHaveBeenCalledTimes(1);
      // 再次保存使用各自刚返回的新版本。
      await save('master');
      await save('house');
      expect(
        vi.mocked(seaDocumentServiceUpdateSeaMasterBillContent).mock
          .lastCall?.[1].expectedMblVersion,
      ).toBe('8');
      expect(
        vi.mocked(seaDocumentServiceUpdateSeaHouseBill).mock.lastCall?.[1]
          .expectedVersion,
      ).toBe('5');
    },
  );

  it('保存失败保留输入与版本，允许继续重试', async () => {
    // 请求层消费错误后会返回 undefined。
    vi.mocked(
      seaDocumentServiceUpdateSeaMasterBillContent,
    ).mockResolvedValueOnce(undefined as never);
    const getForm = setup();
    await screen.findByRole('button', { name: /保存主单内容/ });
    act(() =>
      getForm().setFieldValue(
        ['seaMasterBillContent', 'shipperText'],
        '待重试主单',
      ),
    );
    await save('master');
    expect(
      getForm().getFieldValue(['seaMasterBillContent', 'shipperText']),
    ).toBe('待重试主单');
    await save('master');
    expect(
      vi.mocked(seaDocumentServiceUpdateSeaMasterBillContent).mock
        .lastCall?.[1],
    ).toEqual(
      expect.objectContaining({
        expectedMblVersion: '7',
        content: expect.objectContaining({ shipperText: '待重试主单' }),
      }),
    );
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
