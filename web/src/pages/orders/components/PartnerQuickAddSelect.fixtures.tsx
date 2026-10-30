import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App, ConfigProvider, Form } from 'antd';
import { afterEach, beforeEach, expect, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import type * as PartnerService from '@/services/roncin/partnerService';

import PartnerQuickAddSelect, {
  type PartnerSelectOption,
} from './PartnerQuickAddSelect';

const partnerServiceCreatePartner = vi.hoisted(() =>
  vi.fn<typeof PartnerService.partnerServiceCreatePartner>(),
);

const pushMock = vi.fn();
const testState = vi.hoisted(() => ({
  accessOverrides: {} as Record<string, unknown>,
  organizationId: 'org-1',
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({ canCreatePartners: true, ...testState.accessOverrides }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: testState.organizationId, name: '测试组织' },
      },
    },
  }),
}));

vi.mock('@/router/history', () => ({
  history: {
    get location() {
      return window.location;
    },
    push: (...args: unknown[]) => pushMock(...args),
  },
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner,
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

/** 依次为客户快建弹窗中的业务/操作/客服三个人员下拉选择不同成员。 */
async function fillCommissionStaff() {
  const comboboxes = ['业务人员', '操作人员', '客服人员'].map((label) =>
    screen.getByLabelText(label),
  );
  expect(
    screen
      .getByLabelText('公司抬头')
      .closest('[role="dialog"]')
      ?.querySelectorAll('[role="combobox"]'),
  ).toHaveLength(3);
  await pickSelectOption(comboboxes[0], '张三 · 公司');
  await pickSelectOption(comboboxes[1], '李四 · 公司');
  await pickSelectOption(comboboxes[2], '王五 · 公司');
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
    <ConfigProvider theme={{ token: { motion: false } }}>
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
    </ConfigProvider>
  );
}

async function openDropdown() {
  fireEvent.mouseDown(screen.getByRole('combobox'));
  await screen.findByText('既有单位');
}

function getSaveButton() {
  const dialog = screen.getByLabelText('公司抬头').closest('[role="dialog"]');
  if (!dialog) throw new Error('公司抬头字段不在快建弹窗中');
  const button = Array.from(dialog.querySelectorAll('button')).find(
    (element) => element.textContent?.replace(/\s/g, '') === '保存',
  );
  if (!button) throw new Error('快建弹窗缺少保存按钮');
  return button;
}

async function savePartner() {
  const button = await waitFor(() => {
    const current = getSaveButton();
    expect(current).toBeVisible();
    return current;
  });
  expect(button).toBeEnabled();
  fireEvent.click(button);
}

beforeEach(() => {
  testState.accessOverrides = {};
  testState.organizationId = 'org-1';
  pushMock.mockReset();
  vi.mocked(partnerServiceCreatePartner).mockReset();
  searchPartners
    .mockReset()
    .mockResolvedValue([{ label: '既有单位', value: 'partner-0', code: 'P0' }]);
});

afterEach(() => {
  vi.clearAllMocks();
});

function SupplierHost() {
  const [form] = Form.useForm();
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
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
    </ConfigProvider>
  );
}

export {
  fillCommissionStaff,
  getSaveButton,
  openDropdown,
  partnerServiceCreatePartner,
  pickSelectOption,
  pushMock,
  SupplierHost,
  savePartner,
  searchPartners,
  TestHost,
  testState,
};
