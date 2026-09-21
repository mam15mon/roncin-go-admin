import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, Form } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerAssignmentRole, PartnerRoleType } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import PartnerQuickAddSelect, {
  type PartnerSelectOption,
} from './PartnerQuickAddSelect';

const pushMock = vi.fn();
let accessOverrides: Record<string, unknown> = {};
let organizationId = 'org-1';

vi.mock('@/app/access', () => ({
  useAccess: () => ({ canCreatePartners: true, ...accessOverrides }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: organizationId, name: '测试组织' },
      },
    },
  }),
}));

vi.mock('@/router/history', () => ({
  history: { push: (...args: unknown[]) => pushMock(...args) },
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
}));

const searchPartners = vi.fn<
  (keyword?: string) => Promise<PartnerSelectOption[]>
>(async () => [{ label: '既有单位', value: 'partner-0', code: 'P0' }]);

const STAFF_OPTIONS = [
  { userId: 'user-a', displayName: '张三' },
  { userId: 'user-b', displayName: '李四' },
  { userId: 'user-c', displayName: '王五' },
];

/** 打开指定下拉并点击候选项；过滤掉已关闭下拉残留的隐藏 DOM（antd 关闭后选项仍挂载）。 */
async function pickSelectOption(combobox: Element, name: string) {
  fireEvent.mouseDown(combobox);
  const option = await waitFor(() => {
    const candidates = screen.getAllByText(name).filter((element) => {
      const dropdown = element.closest('.ant-select-dropdown');
      return (
        dropdown !== null &&
        !dropdown.className.includes('-hidden') &&
        !dropdown.className.includes('ant-slide-up-leave')
      );
    });
    if (candidates.length === 0) {
      throw new Error(`选项 ${name} 尚未出现在打开的下拉中`);
    }
    return candidates[0];
  });
  fireEvent.click(option);
  // 等待下拉确认关闭（离开动画类已生效），下一次挑选才不会被残留 DOM 干扰。
  await waitFor(() =>
    expect(combobox).toHaveAttribute('aria-expanded', 'false'),
  );
}

/** 依次为客户快建弹窗中的业务/操作/客服三个必选人员下拉选择不同成员。 */
async function fillCommissionStaff() {
  const comboboxes = within(screen.getByRole('dialog')).getAllByRole(
    'combobox',
  );
  expect(comboboxes).toHaveLength(3);
  await pickSelectOption(comboboxes[0], '张三');
  await pickSelectOption(comboboxes[1], '李四');
  await pickSelectOption(comboboxes[2], '王五');
}

function TestHost({
  disabled = false,
  onPartnerChange,
}: {
  disabled?: boolean;
  onPartnerChange?: (option: PartnerSelectOption | undefined) => void;
}) {
  const [form] = Form.useForm();
  const customerId = Form.useWatch('customerId', form);
  const customerCode = Form.useWatch('customerCode', {
    form,
    preserve: true,
  });
  return (
    <App>
      <Form form={form}>
        <PartnerQuickAddSelect
          name="customerId"
          displayName="委托单位"
          role={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
          createRoute="/partners/customers/create"
          searchPartners={searchPartners}
          staffOptions={STAFF_OPTIONS}
          required
          disabled={disabled}
          onPartnerChange={(option) => {
            form.setFieldValue('customerCode', option?.code);
            onPartnerChange?.(option);
          }}
        />
        <output data-testid="customer-id">{customerId ?? ''}</output>
        <output data-testid="customer-code">{customerCode ?? ''}</output>
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
    searchPartners
      .mockReset()
      .mockResolvedValue([
        { label: '既有单位', value: 'partner-0', code: 'P0' },
      ]);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('有权限且可编辑时下拉底部显示新增入口；无权限时不显示', async () => {
    const { unmount } = render(<TestHost />);
    await openDropdown();
    const addButton = screen.getByRole('button', { name: '新增 委托单位' });
    expect(addButton).toHaveAttribute('type', 'button');
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
    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
        {
          legalName: '新测试单位',
          roles: [
            {
              type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
              enabled: true,
            },
          ],
          isCasual: true,
          assignments: [
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
              userId: 'user-a',
            },
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
              userId: 'user-b',
            },
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
              userId: 'user-c',
            },
          ],
        },
        { signal: expect.any(AbortSignal) },
      ),
    );

    // 回填后选中项显示公司抬头（而非裸 ID），下拉中出现新建单位且与远程结果去重
    await waitFor(() =>
      expect(screen.getAllByTitle('新测试单位').length).toBeGreaterThanOrEqual(
        1,
      ),
    );
    expect(screen.getByTestId('customer-id')).toHaveTextContent('partner-1');
    expect(screen.getByTestId('customer-code')).toHaveTextContent('P1');
    // 弹窗关闭动画期间人员下拉 combobox 仍挂载，主字段按标签精确定位
    fireEvent.mouseDown(screen.getByLabelText('委托单位'));
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
    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    expect(
      await screen.findByText('创建结果缺少伙伴 ID，请重试'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('缺 ID 单位');
  });

  it('当前创建请求失败时仍由 QuickCreateModal 展示原始错误', async () => {
    vi.mocked(partnerServiceCreatePartner).mockRejectedValue(
      new Error('无权创建往来单位'),
    );
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '创建被拒绝单位' },
    });
    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    expect(await screen.findByText('无权创建往来单位')).toBeInTheDocument();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('创建被拒绝单位');
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
    // 弹窗关闭的 motion 起步更新由宏任务/帧回调驱动，
    // 在 act 作用域内让出事件循环，确保用例结束前全部落地。
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
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
    await fillCommissionStaff();
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

  it('组织切换后丢弃迟到的搜索结果', async () => {
    let resolveOldSearch!: (options: PartnerSelectOption[]) => void;
    searchPartners
      .mockImplementationOnce(
        () =>
          new Promise<PartnerSelectOption[]>((resolve) => {
            resolveOldSearch = resolve;
          }),
      )
      .mockResolvedValue([{ label: '新组织单位', value: 'partner-new' }]);

    const { rerender } = render(<TestHost key="stable" />);
    await waitFor(() => expect(searchPartners).toHaveBeenCalledTimes(1));

    organizationId = 'org-2';
    rerender(<TestHost key="stable" />);
    await act(async () => {
      resolveOldSearch([{ label: '旧组织单位', value: 'partner-old' }]);
    });

    // 组织切换会按交互保护关闭下拉；重新展开后验证新组织结果已写入受控 options。
    fireEvent.mouseDown(screen.getByRole('combobox'));
    await waitFor(
      () => {
        expect(searchPartners).toHaveBeenCalledTimes(2);
        expect(screen.getByText('新组织单位')).toBeInTheDocument();
      },
      { timeout: 10000 },
    );
    expect(screen.queryByText('旧组织单位')).not.toBeInTheDocument();
  });

  it('连续搜索时只显示最后一次请求的候选项', async () => {
    let resolveFirst!: (options: PartnerSelectOption[]) => void;
    let resolveSecond!: (options: PartnerSelectOption[]) => void;
    searchPartners
      .mockImplementationOnce(
        () =>
          new Promise<PartnerSelectOption[]>((resolve) => {
            resolveFirst = resolve;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise<PartnerSelectOption[]>((resolve) => {
            resolveSecond = resolve;
          }),
      );

    render(<TestHost />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '最新关键字' },
    });
    await waitFor(() => expect(searchPartners).toHaveBeenCalledTimes(2));

    await act(async () => {
      resolveSecond([{ label: '最新关键字单位', value: 'partner-latest' }]);
    });
    expect(await screen.findByText('最新关键字单位')).toBeInTheDocument();

    await act(async () => {
      resolveFirst([{ label: '旧单位', value: 'partner-stale' }]);
    });
    expect(screen.queryByText('旧单位')).not.toBeInTheDocument();
  });

  it('组件卸载后创建成功不再回填表单或调用变更回调', async () => {
    let resolveCreate!: (value: {
      data: { id: string; legalName: string; code: string };
    }) => void;
    vi.mocked(partnerServiceCreatePartner).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve as typeof resolveCreate;
        }),
    );
    const onPartnerChange = vi.fn();
    const { unmount } = render(<TestHost onPartnerChange={onPartnerChange} />);
    await openDropdown();
    fireEvent.click(screen.getByRole('button', { name: '新增 委托单位' }));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '卸载前新建单位' },
    });
    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());

    unmount();
    await act(async () => {
      resolveCreate({
        data: {
          id: 'partner-after-unmount',
          legalName: '卸载前新建单位',
          code: 'PUNMOUNT',
        },
      });
    });

    expect(onPartnerChange).not.toHaveBeenCalled();
  });

  it('组织切换后丢弃迟到的创建结果', async () => {
    let resolveCreate!: (value: {
      data: { id: string; legalName: string; code: string };
    }) => void;
    vi.mocked(partnerServiceCreatePartner).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve as typeof resolveCreate;
        }),
    );
    const onPartnerChange = vi.fn();
    const { rerender } = render(
      <TestHost key="stable" onPartnerChange={onPartnerChange} />,
    );
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(
      await screen.findByRole('button', { name: '新增 委托单位' }),
    );
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '旧组织新建单位' },
    });
    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());

    organizationId = 'org-2';
    rerender(<TestHost key="stable" onPartnerChange={onPartnerChange} />);
    fireEvent.mouseDown(screen.getByLabelText('委托单位'));
    fireEvent.click(
      await screen.findByRole('button', { name: '新增 委托单位' }),
    );
    expect(await screen.findByLabelText('公司抬头')).toHaveValue('');
    expect(screen.getByRole('button', { name: /保\s*存/ })).not.toBeDisabled();

    await act(async () => {
      resolveCreate({
        data: {
          id: 'partner-old',
          legalName: '旧组织新建单位',
          code: 'POLD',
        },
      });
    });

    expect(screen.getByTestId('customer-id')).toHaveTextContent('');
    expect(screen.getByTestId('customer-code')).toHaveTextContent('');
    expect(onPartnerChange).not.toHaveBeenCalled();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('');
  });

  it('保存期间切换只读状态后隔离旧请求与新弹窗', async () => {
    let resolveCreate!: (value: {
      data: { id: string; legalName: string; code: string };
    }) => void;
    vi.mocked(partnerServiceCreatePartner).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve as typeof resolveCreate;
        }),
    );
    const onPartnerChange = vi.fn();
    const { rerender } = render(<TestHost onPartnerChange={onPartnerChange} />);
    await openDropdown();
    fireEvent.click(screen.getByRole('button', { name: '新增 委托单位' }));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '锁单前新建单位' },
    });
    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());

    rerender(<TestHost disabled onPartnerChange={onPartnerChange} />);
    rerender(<TestHost onPartnerChange={onPartnerChange} />);
    await openDropdown();
    fireEvent.click(screen.getByRole('button', { name: '新增 委托单位' }));
    const reopenedNameInput = await screen.findByLabelText('公司抬头');
    fireEvent.change(reopenedNameInput, {
      target: { value: '解锁后新输入' },
    });

    await act(async () => {
      resolveCreate({
        data: {
          id: 'partner-before-lock',
          legalName: '锁单前新建单位',
          code: 'PLOCK',
        },
      });
    });

    expect(screen.getByTestId('customer-id')).toHaveTextContent('');
    expect(screen.getByTestId('customer-code')).toHaveTextContent('');
    expect(onPartnerChange).not.toHaveBeenCalled();
    expect(screen.getByLabelText('公司抬头')).toHaveValue('解锁后新输入');
  });

  it('普通选择与清空同步更新 customerCode', async () => {
    render(<TestHost />);
    await openDropdown();
    fireEvent.click(screen.getByText('既有单位'));

    await waitFor(() => {
      expect(screen.getByTestId('customer-id')).toHaveTextContent('partner-0');
      expect(screen.getByTestId('customer-code')).toHaveTextContent('P0');
    });

    const selector = screen.getByRole('combobox').closest('.ant-select');
    expect(selector).not.toBeNull();
    fireEvent.mouseEnter(selector as Element);
    const clearButton = (selector as Element).querySelector(
      '.ant-select-clear',
    );
    expect(clearButton).not.toBeNull();
    fireEvent.mouseDown(clearButton as Element);
    fireEvent.click(clearButton as Element);

    await waitFor(() => {
      expect(screen.getByTestId('customer-id')).toHaveTextContent('');
      expect(screen.getByTestId('customer-code')).toHaveTextContent('');
    });
  });

  it('远程返回本地同 ID 时只保留本地名称', async () => {
    searchPartners
      .mockResolvedValueOnce([
        { label: '既有单位', value: 'partner-0', code: 'P0' },
      ])
      .mockResolvedValue([
        { label: '远程旧名称', value: 'partner-1', code: 'P1' },
      ]);
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: { id: 'partner-1', legalName: '新测试单位', code: 'P1' } as never,
    });
    render(<TestHost />);
    await openDropdown();
    fireEvent.click(screen.getByRole('button', { name: /新增 委托单位/ }));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '新测试单位' },
    });
    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() =>
      expect(screen.getByTestId('customer-id')).toHaveTextContent('partner-1'),
    );
    // 弹窗关闭动画期间人员下拉 combobox 仍挂载，主字段按标签精确定位
    fireEvent.mouseDown(screen.getByLabelText('委托单位'));
    await waitFor(() => {
      const dropdownOptions = screen
        .getAllByText('新测试单位')
        .filter((element) => element.closest('.ant-select-item-option'));
      expect(dropdownOptions).toHaveLength(1);
      expect(screen.queryByText('远程旧名称')).not.toBeInTheDocument();
    });
    expect(screen.getAllByTitle('新测试单位').length).toBeGreaterThanOrEqual(1);
  });

  it('支持取消单次合作勾选，且散客候选项渲染散客标签', async () => {
    searchPartners.mockResolvedValue([
      { label: '散客货代', value: 'partner-c', code: 'PC', isCasual: true },
      { label: '正式客户', value: 'partner-r', code: 'PR', isCasual: false },
    ]);
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {
        id: 'partner-new',
        legalName: '正式新单位',
        code: 'PN',
        isCasual: false,
      } as never,
    });
    render(<TestHost />);
    fireEvent.mouseDown(screen.getByRole('combobox'));

    // 下拉应渲染散客标签
    await screen.findByText('散客货代');
    expect(screen.getByText('散客')).toBeInTheDocument();

    // 打开快捷新增
    fireEvent.click(screen.getByRole('button', { name: /新增 委托单位/ }));
    const legalNameInput = await screen.findByLabelText('公司抬头');
    fireEvent.change(legalNameInput, { target: { value: '正式新单位' } });

    const checkbox = screen.getByLabelText('单次合作往来单位（散客）');
    expect(checkbox).toBeChecked();
    fireEvent.click(checkbox);
    expect(checkbox).not.toBeChecked();

    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
        {
          legalName: '正式新单位',
          roles: [
            {
              type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
              enabled: true,
            },
          ],
          isCasual: false,
          assignments: [
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
              userId: 'user-a',
            },
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
              userId: 'user-b',
            },
            {
              role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
              userId: 'user-c',
            },
          ],
        },
        { signal: expect.any(AbortSignal) },
      ),
    );
  });

  it('客户快建未选提成责任岗位时保存被拦截且不发起创建', async () => {
    render(<TestHost />);
    await openDropdown();

    fireEvent.click(screen.getByText('新增 委托单位'));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '缺责任人单位' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    expect(await screen.findByText('请选择业务人员')).toBeInTheDocument();
    expect(screen.getByText('请选择操作人员')).toBeInTheDocument();
    expect(screen.getByText('请选择客服人员')).toBeInTheDocument();
    expect(partnerServiceCreatePartner).not.toHaveBeenCalled();
    // 校验失败时弹窗保持打开，已录入内容不丢失
    expect(screen.getByLabelText('公司抬头')).toHaveValue('缺责任人单位');
  });

  it('非客户角色快建不要求提成责任岗位', async () => {
    function SupplierHost() {
      const [form] = Form.useForm();
      return (
        <App>
          <Form form={form}>
            <PartnerQuickAddSelect
              name="bookingAgentId"
              displayName="订舱代理"
              role={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
              createRoute="/partners/suppliers/create"
              searchPartners={searchPartners}
              staffOptions={STAFF_OPTIONS}
            />
          </Form>
        </App>
      );
    }
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: { id: 'partner-s1', legalName: '新订舱代理', code: 'PS1' } as never,
    });
    render(<SupplierHost />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('新增 订舱代理'));

    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '新订舱代理' },
    });
    expect(screen.queryByLabelText('业务人员')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith(
        {
          legalName: '新订舱代理',
          roles: [
            {
              type: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
              enabled: true,
            },
          ],
          isCasual: true,
          assignments: undefined,
        },
        { signal: expect.any(AbortSignal) },
      ),
    );
  });
});
