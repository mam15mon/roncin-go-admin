import type { ProFormInstance } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import dayjs from 'dayjs';
import type { RefObject } from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import * as services from '@/services/roncin/settlementService';
import CommissionRuleFormModal from './CommissionRuleFormModal';
import CommissionRulesDrawer from './CommissionRulesDrawer';

const workspace = vi.hoisted(() => ({
  organizationId: 'org-a' as string | undefined,
  userId: 'user-1' as string | undefined,
  canOperate: true,
}));
vi.mock('@/app/AppProvider', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/app/AppProvider')>()),
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: workspace.userId,
        currentOrganization: { id: workspace.organizationId },
      },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canReadFinanceCommissions: true,
    canConfigureFinanceCommissions: true,
    canManageFinanceCommissions: true,
    canOperateOrganization: (id?: string) =>
      workspace.canOperate && Boolean(id) && id === workspace.organizationId,
  }),
}));
const state = vi.hoisted(() => ({
  formRef: undefined as RefObject<ProFormInstance | undefined> | undefined,
  reload: vi.fn(),
}));
vi.mock('./CommissionRuleFormModal', async (importOriginal) => {
  const { default: Actual } =
    await importOriginal<typeof import('./CommissionRuleFormModal')>();
  return {
    default: (props: React.ComponentProps<typeof Actual>) => {
      state.formRef = props.formRef;
      return <Actual {...props} />;
    },
  };
});
vi.mock('@/components/ui', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/components/ui')>();
  return {
    ...actual,
    ResponsiveProTable: (props: any) => {
      props.actionRef.current = { reload: state.reload };
      return <div>{props.toolBarRender?.()}</div>;
    },
  };
});
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListCommissionRules: vi.fn(),
  settlementServiceListCommissionEmployees: vi.fn(),
  settlementServiceCreateCommissionRule: vi.fn(),
  settlementServiceUpdateCommissionRule: vi.fn(),
  settlementServiceCopyCommissionRule: vi.fn(),
  settlementServiceAssignCommissionRuleEmployees: vi.fn(),
  settlementServiceRemoveCommissionRuleEmployees: vi.fn(),
}));

beforeEach(() => {
  workspace.organizationId = 'org-a';
  workspace.userId = 'user-1';
  workspace.canOperate = true;
  vi.clearAllMocks();
  vi.mocked(
    services.settlementServiceListCommissionEmployees,
  ).mockResolvedValue({ data: [{ id: 'employee-1', displayName: '张冠楠' }] });
  vi.mocked(services.settlementServiceCreateCommissionRule).mockResolvedValue({
    data: { id: 'rule-1' },
  });
});

async function openForm() {
  renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <CommissionRulesDrawer open onClose={vi.fn()} canManage />
    </ConfigProvider>,
  );
  fireEvent.click(await screen.findByText('新建方案'));
  const title = await screen.findByText('新建提成方案');
  const dialog = title.closest('[role="dialog"]') as HTMLElement;
  await waitFor(() => expect(dialog).toBeVisible());
  await waitFor(() => expect(state.formRef?.current).toBeTruthy());
  await act(async () =>
    state.formRef?.current?.setFieldsValue({
      organizationId: 'ignored-form-company',
      name: 'CODEX测试方案1108',
      personnelRole: 'SALES',
      calculationBasis: 'REALIZED_PROFIT',
      ratePercent: 10,
      employeeIds: ['employee-1'],
      enabled: true,
    }),
  );
  return dialog;
}

it('真实 ProForm 提交有限日期并刷新成功列表', async () => {
  const dialog = await openForm();
  const start = dayjs().startOf('day');
  const end = start.add(10, 'day');
  await act(async () =>
    state.formRef?.current?.setFieldsValue({ effectiveRange: [start, end] }),
  );
  fireEvent.click(within(dialog).getByRole('button', { name: '确 定' }));
  await waitFor(() =>
    expect(
      services.settlementServiceCreateCommissionRule,
    ).toHaveBeenCalledOnce(),
  );
  expect(services.settlementServiceCreateCommissionRule).toHaveBeenCalledWith({
    organizationId: 'org-a',
    rule: expect.objectContaining({
      name: 'CODEX测试方案1108',
      personnelRole: 'SALES',
      calculationBasis: 'REALIZED_PROFIT',
      ratePercent: '10',
      employeeIds: ['employee-1'],
      effectiveFrom: start.format('YYYY-MM-DD'),
      effectiveTo: end.format('YYYY-MM-DD'),
    }),
  });
  expect(await screen.findByText('提成方案已创建')).toBeVisible();
  expect(state.reload).toHaveBeenCalledOnce();
});

it('启用方案缺日期和过去日期在真实表单拦截，停用草稿可无日期提交', async () => {
  const dialog = await openForm();
  const submit = within(dialog).getByRole('button', { name: '确 定' });
  fireEvent.click(submit);
  expect(await screen.findByText('启用方案请选择生效起始日')).toBeVisible();
  expect(services.settlementServiceCreateCommissionRule).not.toHaveBeenCalled();
  await act(async () =>
    state.formRef?.current?.setFieldsValue({
      effectiveRange: [dayjs().subtract(1, 'day'), dayjs().add(1, 'day')],
    }),
  );
  fireEvent.click(submit);
  expect(
    await screen.findByText('新启用方案的起始日必须是当天或未来日期'),
  ).toBeVisible();
  expect(services.settlementServiceCreateCommissionRule).not.toHaveBeenCalled();
  await act(async () =>
    state.formRef?.current?.setFieldsValue({
      enabled: false,
      effectiveRange: undefined,
    }),
  );
  fireEvent.click(submit);
  await waitFor(() =>
    expect(
      services.settlementServiceCreateCommissionRule,
    ).toHaveBeenCalledOnce(),
  );
});

it('请求失败保留完整日期和输入且不刷新列表', async () => {
  vi.mocked(services.settlementServiceCreateCommissionRule).mockResolvedValue(
    undefined as any,
  );
  const dialog = await openForm();
  const start = dayjs().startOf('day');
  await act(async () =>
    state.formRef?.current?.setFieldsValue({
      effectiveRange: [start, start.add(1, 'day')],
    }),
  );
  fireEvent.click(within(dialog).getByRole('button', { name: '确 定' }));
  await waitFor(() =>
    expect(
      services.settlementServiceCreateCommissionRule,
    ).toHaveBeenCalledOnce(),
  );
  expect(dialog).toBeVisible();
  expect(state.formRef?.current?.getFieldValue('name')).toBe(
    'CODEX测试方案1108',
  );
  expect(
    state.formRef?.current
      ?.getFieldValue('effectiveRange')[0]
      .format('YYYY-MM-DD'),
  ).toBe(start.format('YYYY-MM-DD'));
  expect(state.reload).not.toHaveBeenCalled();
});

it('已生效有限区间编辑保留真实 Dayjs 提交且不触发新建日期限制', async () => {
  const ref: RefObject<ProFormInstance | undefined> = { current: undefined };
  const start = dayjs().subtract(5, 'day');
  const end = dayjs().add(10, 'day');
  const onFinish = vi.fn().mockResolvedValue(true);
  renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <CommissionRuleFormModal
        open
        editingRule={{
          id: 'existing',
          name: '已有方案',
          personnelRole: 'SALES',
          calculationBasis: 'REALIZED_PROFIT',
          ratePercent: '10',
          enabled: true,
          effectiveFrom: start.format('YYYY-MM-DD'),
          effectiveTo: end.format('YYYY-MM-DD'),
        }}
        formRef={ref}
        employeeOptions={[]}
        searchEmployees={vi.fn()}
        onEmployeeOptionsChange={vi.fn()}
        onCancel={vi.fn()}
        onFinish={onFinish}
      />
    </ConfigProvider>,
  );
  const title = await screen.findByText('编辑提成方案');
  const dialog = title.closest('[role="dialog"]') as HTMLElement;
  await waitFor(() => expect(dialog).toBeVisible());
  fireEvent.click(within(dialog).getByRole('button', { name: '确 定' }));
  await waitFor(() => expect(onFinish).toHaveBeenCalledOnce());
  const values = onFinish.mock.calls[0][0];
  expect(
    values.effectiveRange.map((date: dayjs.Dayjs) => date.format('YYYY-MM-DD')),
  ).toEqual([start.format('YYYY-MM-DD'), end.format('YYYY-MM-DD')]);
});
