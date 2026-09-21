import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, Form } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerAssignmentRole, PartnerRoleType } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import PartnerQuickAddSelect, {
  type PartnerSelectOption,
} from './PartnerQuickAddSelect';

const pushMock = vi.fn();
const accessOverrides: Record<string, unknown> = {};
const organizationId = 'org-1';

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

/** 依次为客户快建弹窗中的业务/操作/客服三个必选人员下拉选择同一个成员。 */
async function fillCommissionStaff() {
  const comboboxes = within(screen.getByRole('dialog')).getAllByRole(
    'combobox',
  );
  expect(comboboxes).toHaveLength(3);
  await pickSelectOption(comboboxes[0], '张三 · 公司');
  await pickSelectOption(comboboxes[1], '张三 · 公司');
  await pickSelectOption(comboboxes[2], '张三 · 公司');
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

describe('PartnerQuickAddSelect 创建失败恢复', () => {
  beforeEach(() => {
    vi.mocked(partnerServiceCreatePartner).mockReset();
    searchPartners.mockClear();
  });

  it('已通知的请求失败保留客户名称与三个岗位，重试成功才回填并关闭弹窗', async () => {
    vi.mocked(partnerServiceCreatePartner)
      .mockResolvedValueOnce(undefined as never)
      .mockResolvedValueOnce({
        data: { id: 'partner-retry', legalName: '重试客户' },
      });
    const onPartnerChange = vi.fn();
    render(<TestHost onPartnerChange={onPartnerChange} />);
    await openDropdown();
    fireEvent.click(screen.getByText('新增 委托单位'));
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '重试客户' },
    });
    await fillCommissionStaff();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledTimes(1),
    );
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /取\s*消/ }),
      ).not.toBeDisabled(),
    );
    expect(screen.getByLabelText('公司抬头')).toHaveValue('重试客户');
    expect(onPartnerChange).not.toHaveBeenCalled();
    expect(screen.getByTestId('customer-id')).toBeEmptyDOMElement();
    expect(document.querySelector('.ant-message-notice')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() =>
      expect(screen.getByTestId('customer-id')).toHaveTextContent(
        'partner-retry',
      ),
    );
    expect(partnerServiceCreatePartner).toHaveBeenCalledTimes(2);
    expect(
      vi.mocked(partnerServiceCreatePartner).mock.calls[1][0].assignments,
    ).toEqual([
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
        userId: 'user-a',
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
        userId: 'user-a',
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
        userId: 'user-a',
      },
    ]);

    expect(vi.mocked(partnerServiceCreatePartner).mock.calls[1][0]).toEqual(
      vi.mocked(partnerServiceCreatePartner).mock.calls[0][0],
    );
    expect(onPartnerChange).toHaveBeenCalledTimes(1);
    await waitFor(() =>
      expect(screen.getByRole('dialog')).toHaveClass('ant-zoom-leave'),
    );
    fireEvent.animationEnd(screen.getByRole('dialog'));
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    await openDropdown();
    fireEvent.click(screen.getByText('新增 委托单位'));
    expect(await screen.findByLabelText('公司抬头')).toHaveValue('');
  });
});
