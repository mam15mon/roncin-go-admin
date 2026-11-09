import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const workspace = vi.hoisted(() => ({
  organizationId: 'org-a' as string | undefined,
  userId: 'user-1' as string | undefined,
  canOperate: true,
}));
vi.mock('@/app/AppProvider', () => ({
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
const serviceMocks = vi.hoisted(() => ({
  createCommissionRule: vi.fn(),
  updateCommissionRule: vi.fn(),
  copyCommissionRule: vi.fn(),
  assignCommissionRuleEmployees: vi.fn(),
  removeCommissionRuleEmployees: vi.fn(),
  listCommissionRules: vi.fn(),
  listCommissionEmployees: vi.fn(),
}));

const drawerState = vi.hoisted(() => ({
  forms: {} as Record<string, Record<string, any>>,
  table: undefined as Record<string, any> | undefined,
  sourceOrganizationId: 'org-a' as string | undefined,
}));

vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: Record<string, any>) => {
    const title = props.title ?? '';
    if (props.formRef) {
      props.formRef.current = {
        getFieldValue: (field: string) =>
          field === 'organizationId' ? 'org-a' : undefined,
      };
    }
    drawerState.forms[title] = props;
    return <div>{props.children}</div>;
  },
  ProFormDateRangePicker: () => null,
  ProFormDatePicker: () => null,
  ProFormDigit: () => null,
  ProFormSelect: () => null,
  ProFormSearchableSelect: () => null,
  ProFormSwitch: () => null,
  ProFormText: () => null,
  ProFormTextArea: () => null,
  ProTable: (props: Record<string, any>) => {
    drawerState.table = props;
    const sample = {
      id: 'rule-source',
      name: '旧方案',
      organizationId: drawerState.sourceOrganizationId,
      version: 3,
      assignments: [{ employeeId: 'employee-1', employeeName: '张三' }],
    };
    return (
      <div>
        {/* 真实 ProTable 会按列 key 为每个单元格容器补 key；替身直接铺开 option
            列渲染结果时同样需要按列提供稳定 key，否则触发 React 缺 key 警告。 */}
        {(props.columns ?? [])
          .filter((column: any) => column.valueType === 'option')
          .map((column: any) => (
            <div key={column.key ?? column.dataIndex ?? 'option'}>
              {column.render ? column.render(undefined, sample) : null}
            </div>
          ))}
      </div>
    );
  },
}));

vi.mock('@/components/ui', async () => ({
  GuardedModalForm: (await import('@/components/ui/GuardedModalForm'))
    .GuardedModalForm,
  ResponsiveProTable: (await import('@ant-design/pro-components')).ProTable,
  ResponsiveTable: (await import('antd')).Table,
  MODAL_SIZE: { SM: 520, MD: 680, LG: 960, XL: 1200 },
  DRAWER_SIZE: { SM: 600, MD: 860, LG: 1080, XL: 1200 },
  ProFormSearchableSelect: () => null,
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceAssignCommissionRuleEmployees:
    serviceMocks.assignCommissionRuleEmployees,
  settlementServiceCopyCommissionRule: serviceMocks.copyCommissionRule,
  settlementServiceCreateCommissionRule: serviceMocks.createCommissionRule,
  settlementServiceListCommissionEmployees:
    serviceMocks.listCommissionEmployees,
  settlementServiceListCommissionRules: serviceMocks.listCommissionRules,
  settlementServiceRemoveCommissionRuleEmployees:
    serviceMocks.removeCommissionRuleEmployees,
  settlementServiceUpdateCommissionRule: serviceMocks.updateCommissionRule,
}));

import CommissionRulesDrawer from './CommissionRulesDrawer';

describe('提成方案抽屉', () => {
  beforeEach(() => {
    workspace.organizationId = 'org-a';
    workspace.userId = 'user-1';
    workspace.canOperate = true;
    drawerState.forms = {};
    drawerState.table = undefined;
    drawerState.sourceOrganizationId = 'org-a';
    for (const mock of Object.values(serviceMocks)) {
      mock.mockReset();
    }
    serviceMocks.listCommissionEmployees.mockResolvedValue({
      data: [
        { id: 'employee-1', displayName: '张三' },
        { id: 'employee-2', displayName: '李四' },
      ],
      total: 2,
      page: 1,
      pageSize: 200,
    });
  });

  it('新建启用方案必须携带初始员工列表', async () => {
    render(
      <App>
        <CommissionRulesDrawer open onClose={vi.fn()} canManage />
      </App>,
    );
    await screen.findByText('适用员工');

    const createForm = drawerState.forms['新建提成方案'];
    expect(createForm).toBeTruthy();

    // 启用方案缺少员工时不发起请求；提交走表单异步回调，包进 act 收敛消息与状态更新。
    await act(async () => {
      await createForm.onFinish({
        name: '销售方案',
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: 10,
        enabled: true,
        employeeIds: [],
      });
    });
    expect(serviceMocks.createCommissionRule).not.toHaveBeenCalled();

    await act(async () => {
      await createForm.onFinish({
        name: '销售方案',
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: 10,
        enabled: true,
        employeeIds: ['employee-1', 'employee-2'],
      });
    });
    await waitFor(() =>
      expect(serviceMocks.createCommissionRule).toHaveBeenCalledWith(
        expect.objectContaining({
          organizationId: 'org-a',
          rule: expect.objectContaining({
            name: '销售方案',
            personnelRole: 'SALES',
            employeeIds: ['employee-1', 'employee-2'],
          }),
        }),
      ),
    );
  });

  it('复制为新方案提交新起始日与员工列表', async () => {
    render(
      <App>
        <CommissionRulesDrawer open onClose={vi.fn()} canManage />
      </App>,
    );
    await screen.findByText('适用员工');

    fireEvent.click(screen.getByText('复制为新方案'));
    await waitFor(() => expect(drawerState.forms['复制为新方案']).toBeTruthy());
    const copyForm = drawerState.forms['复制为新方案'];

    // 提交走表单异步回调，包进 act 收敛消息与状态更新。
    await act(async () => {
      await copyForm.onFinish({
        name: '旧方案-新方案',
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: 12,
        effectiveFrom: dayjs('2026-09-20'),
        employeeIds: ['employee-1'],
      });
    });
    await waitFor(() =>
      expect(serviceMocks.copyCommissionRule).toHaveBeenCalledWith(
        { id: 'rule-source' },
        expect.objectContaining({
          id: 'rule-source',
          name: '旧方案-新方案',
          personnelRole: 'SALES',
          effectiveFrom: '2026-09-20',
          employeeIds: ['employee-1'],
        }),
      ),
    );
  });

  it('复制表单缺少生效起始日或员工时不提交', async () => {
    render(
      <App>
        <CommissionRulesDrawer open onClose={vi.fn()} canManage />
      </App>,
    );
    await screen.findByText('适用员工');

    fireEvent.click(screen.getByText('复制为新方案'));
    await waitFor(() => expect(drawerState.forms['复制为新方案']).toBeTruthy());
    const copyForm = drawerState.forms['复制为新方案'];
    // 缺起始日/缺员工均不提交；提交走表单异步回调，包进 act 收敛消息与状态更新。
    await act(async () => {
      await copyForm.onFinish({
        name: '旧方案-新方案',
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: 12,
        employeeIds: ['employee-1'],
      });
    });
    await act(async () => {
      await copyForm.onFinish({
        name: '旧方案-新方案',
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: 12,
        effectiveFrom: dayjs('2026-09-20'),
        employeeIds: [],
      });
    });
    expect(serviceMocks.copyCommissionRule).not.toHaveBeenCalled();
  });

  it('方案列表与复制员工候选固定当前公司，旧表单公司不改变新建目标', async () => {
    serviceMocks.listCommissionRules.mockResolvedValue({ data: [], total: 0 });
    serviceMocks.createCommissionRule.mockResolvedValue({
      data: { id: 'new-rule' },
    });
    render(
      <App>
        <CommissionRulesDrawer open onClose={vi.fn()} canManage />
      </App>,
    );
    await act(async () => {
      await drawerState.table?.request({
        current: 2,
        pageSize: 40,
        organizationId: 'org-b',
      });
    });
    expect(serviceMocks.listCommissionRules).toHaveBeenCalledWith(
      expect.objectContaining({
        organizationId: 'org-a',
        page: 2,
        pageSize: 40,
      }),
    );
    fireEvent.click(screen.getByText('复制为新方案'));
    await waitFor(() =>
      expect(serviceMocks.listCommissionEmployees).toHaveBeenCalledWith({
        page: 1,
        pageSize: 200,
        organizationId: 'org-a',
      }),
    );
    await act(async () => {
      expect(
        await drawerState.forms.新建提成方案.onFinish({
          organizationId: 'org-b',
          name: '当前公司方案',
          personnelRole: 'SALES',
          calculationBasis: 'REALIZED_PROFIT',
          ratePercent: 10,
          enabled: false,
        }),
      ).toBe(true);
    });
    expect(serviceMocks.createCommissionRule).toHaveBeenCalledWith(
      expect.objectContaining({ organizationId: 'org-a' }),
    );
  });

  it.each([undefined, 'org-b'])(
    '来源方案公司为%s时拒绝编辑和复制，不加载外公司员工',
    async (sourceOrganizationId) => {
      drawerState.sourceOrganizationId = sourceOrganizationId;
      render(
        <App>
          <CommissionRulesDrawer open onClose={vi.fn()} canManage />
        </App>,
      );
      fireEvent.click(screen.getByText('复制为新方案'));
      fireEvent.click(screen.getByText('编辑'));
      expect(
        await screen.findAllByText('只能操作当前公司工作区的提成方案'),
      ).toHaveLength(2);
      expect(drawerState.forms.复制为新方案.open).toBe(false);
      expect(drawerState.forms.新建提成方案.open).toBe(false);
      expect(serviceMocks.listCommissionEmployees).not.toHaveBeenCalled();
      expect(serviceMocks.copyCommissionRule).not.toHaveBeenCalled();
      expect(serviceMocks.updateCommissionRule).not.toHaveBeenCalled();
    },
  );

  it.each(['missing-user', 'missing-company', 'non-company'])(
    '%s 不查询方案或允许提交',
    async (scenario) => {
      if (scenario === 'missing-user') workspace.userId = undefined;
      if (scenario === 'missing-company') workspace.organizationId = undefined;
      if (scenario === 'non-company') workspace.canOperate = false;
      render(
        <App>
          <CommissionRulesDrawer open onClose={vi.fn()} canManage />
        </App>,
      );
      await act(async () => {
        expect(
          await drawerState.table?.request({ current: 1, pageSize: 20 }),
        ).toMatchObject({ success: false });
        expect(
          await drawerState.forms.新建提成方案.onFinish({
            name: '无效提交',
            personnelRole: 'SALES',
            calculationBasis: 'REALIZED_PROFIT',
            ratePercent: 10,
            enabled: false,
          }),
        ).toBe(false);
      });
      expect(serviceMocks.listCommissionRules).not.toHaveBeenCalled();
      expect(serviceMocks.listCommissionEmployees).not.toHaveBeenCalled();
      expect(serviceMocks.createCommissionRule).not.toHaveBeenCalled();
    },
  );
});
