import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App, Form } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import PartnerQuickAddSelect from './PartnerQuickAddSelect';

const pushMock = vi.fn();
let accessOverrides: Record<string, unknown> = {};
let organizationId = 'org-1';

vi.mock('@umijs/max', () => ({
  useAccess: () => ({ canCreatePartners: true, ...accessOverrides }),
  useModel: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: organizationId, name: '测试组织' },
      },
    },
  }),
  history: { push: (...args: unknown[]) => pushMock(...args) },
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
}));

const searchPartners = vi.fn(async () => [
  { label: '既有单位', value: 'partner-0', code: 'P0' },
]);

function TestHost({
  disabled = false,
  taxIdentifierRequired = false,
}: {
  disabled?: boolean;
  taxIdentifierRequired?: boolean;
}) {
  const [form] = Form.useForm();
  return (
    <App>
      <Form form={form}>
        <PartnerQuickAddSelect
          name="customerId"
          displayName="委托单位"
          role={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
          createRoute="/partners/customers/create"
          searchPartners={searchPartners}
          required
          taxIdentifierRequired={taxIdentifierRequired}
          disabled={disabled}
        />
      </Form>
    </App>
  );
}

async function openDropdown() {
  fireEvent.mouseDown(screen.getByRole('combobox'));
  await screen.findByText('既有单位');
}

describe('PartnerQuickAddSelect', () => {
  beforeEach(() => {
    accessOverrides = {};
    organizationId = 'org-1';
    pushMock.mockReset();
    vi.mocked(partnerServiceCreatePartner).mockReset();
    searchPartners.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('有权限且可编辑时下拉底部显示新增入口；无权限时不显示', async () => {
    const { unmount } = render(<TestHost />);
    await openDropdown();
    expect(await screen.findByText('新增 委托单位')).toBeInTheDocument();
    unmount();

    accessOverrides = { canCreatePartners: false };
    render(<TestHost />);
    await openDropdown();
    expect(screen.queryByText('新增 委托单位')).not.toBeInTheDocument();
  });

  it('disabled 时不显示新增入口', async () => {
    render(<TestHost disabled />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    await waitFor(() =>
      expect(screen.queryByText('新增 委托单位')).not.toBeInTheDocument(),
    );
  });

  it('保存仅提交公司抬头与单一角色，成功后回填字段并注入本地选项', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {
        id: 'partner-1',
        legalName: '新测试单位',
        code: 'P1',
      } as never,
    });
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    const input = await screen.findByLabelText('公司抬头');
    fireEvent.change(input, { target: { value: '  新测试单位  ' } });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith({
        legalName: '新测试单位',
        unifiedSocialCreditCode: undefined,
        roles: [
          { type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER, enabled: true },
        ],
      }),
    );

    // 回填后选中项显示公司抬头（而非裸 ID），下拉中出现新建单位且与远程结果去重
    await waitFor(() =>
      expect(screen.getAllByTitle('新测试单位').length).toBeGreaterThanOrEqual(
        1,
      ),
    );
    fireEvent.mouseDown(screen.getByRole('combobox'));
    await waitFor(() => {
      expect(screen.getAllByText('既有单位').length).toBe(1);
    });
  });

  it('响应缺少伙伴 ID 时报错、弹窗保持打开且输入保留', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {},
    } as never);
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    const input = await screen.findByLabelText('公司抬头');
    fireEvent.change(input, { target: { value: '缺 ID 单位' } });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    expect(
      await screen.findByText('创建结果缺少伙伴 ID，请重试'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('缺 ID 单位');
  });

  it('添加公司详情不触发保存，携带编码后的公司抬头跳转', async () => {
    vi.mocked(partnerServiceCreatePartner);
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    const input = await screen.findByLabelText('公司抬头');
    fireEvent.change(input, { target: { value: 'AB C公司' } });
    fireEvent.click(screen.getByRole('button', { name: '添加公司详情' }));

    expect(pushMock).toHaveBeenCalledWith(
      '/partners/customers/create?legalName=AB%20C%E5%85%AC%E5%8F%B8',
    );
    expect(partnerServiceCreatePartner).not.toHaveBeenCalled();
  });

  it('taxIdentifierRequired 时校验并提交纳税人识别号', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: { id: 'partner-9', legalName: '带税号单位' } as never,
    });
    render(<TestHost taxIdentifierRequired />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    await screen.findByLabelText('公司抬头');
    fireEvent.change(screen.getByLabelText('公司抬头'), {
      target: { value: '带税号单位' },
    });
    // 非法税号阻断提交
    fireEvent.change(screen.getByLabelText('纳税人识别号'), {
      target: { value: '123' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    expect(partnerServiceCreatePartner).not.toHaveBeenCalled();
    expect(
      await screen.findByText('请输入正确的18位统一社会信用代码'),
    ).toBeInTheDocument();

    // 合法税号后提交并大写化
    fireEvent.change(screen.getByLabelText('纳税人识别号'), {
      target: { value: '91310000ma1fl7a21q' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith({
        legalName: '带税号单位',
        unifiedSocialCreditCode: '91310000MA1FL7A21Q',
        roles: [
          { type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER, enabled: true },
        ],
      }),
    );
  });

  it('组织切换后本地新建选项被清理', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: { id: 'partner-1', legalName: '切换前单位' } as never,
    });
    const { rerender } = render(<TestHost key="stable" />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '切换前单位' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());

    organizationId = 'org-2';
    rerender(<TestHost key="stable" />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    await screen.findByText('既有单位');
    await waitFor(() =>
      expect(screen.queryByText('切换前单位')).not.toBeInTheDocument(),
    );
  });
});
