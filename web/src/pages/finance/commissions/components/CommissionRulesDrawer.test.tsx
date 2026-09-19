import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  createCommissionRule: vi.fn(),
  updateCommissionRule: vi.fn(),
  copyCommissionRule: vi.fn(),
  assignCommissionRuleEmployees: vi.fn(),
  removeCommissionRuleEmployees: vi.fn(),
  listCommissionRules: vi.fn(),
  listCommissionEmployees: vi.fn(),
  listFinanceOrganizationOptions: vi.fn(),
}));

const drawerState = vi.hoisted(() => ({
  forms: {} as Record<string, Record<string, any>>,
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
    const sample = {
      id: 'rule-source',
      name: '旧方案',
      organizationId: 'org-a',
      version: 3,
      assignments: [{ employeeId: 'employee-1', employeeName: '张三' }],
    };
    return (
      <div>
        {(props.columns ?? [])
          .filter((column: any) => column.valueType === 'option')
          .flatMap((column: any) =>
            column.render ? column.render(undefined, sample) : [],
          )}
      </div>
    );
  },
}));

vi.mock('@/components/ui', () => ({
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
  settlementServiceListFinanceOrganizationOptions:
    serviceMocks.listFinanceOrganizationOptions,
  settlementServiceRemoveCommissionRuleEmployees:
    serviceMocks.removeCommissionRuleEmployees,
  settlementServiceUpdateCommissionRule: serviceMocks.updateCommissionRule,
}));

import CommissionRulesDrawer from './CommissionRulesDrawer';

describe('提成方案抽屉', () => {
  beforeEach(() => {
    drawerState.forms = {};
    for (const mock of Object.values(serviceMocks)) {
      mock.mockReset();
    }
    serviceMocks.listFinanceOrganizationOptions.mockResolvedValue({
      data: [],
    });
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

    // 启用方案缺少员工时不发起请求。
    await createForm.onFinish({
      name: '销售方案',
      personnelRole: 'SALES',
      calculationBasis: 'REALIZED_PROFIT',
      ratePercent: 10,
      enabled: true,
      employeeIds: [],
    });
    expect(serviceMocks.createCommissionRule).not.toHaveBeenCalled();

    await createForm.onFinish({
      name: '销售方案',
      personnelRole: 'SALES',
      calculationBasis: 'REALIZED_PROFIT',
      ratePercent: 10,
      enabled: true,
      employeeIds: ['employee-1', 'employee-2'],
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

    await copyForm.onFinish({
      name: '旧方案-新方案',
      personnelRole: 'SALES',
      calculationBasis: 'REALIZED_PROFIT',
      ratePercent: 12,
      effectiveFrom: dayjs('2026-09-20'),
      employeeIds: ['employee-1'],
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
    await copyForm.onFinish({
      name: '旧方案-新方案',
      personnelRole: 'SALES',
      calculationBasis: 'REALIZED_PROFIT',
      ratePercent: 12,
      employeeIds: ['employee-1'],
    });
    await copyForm.onFinish({
      name: '旧方案-新方案',
      personnelRole: 'SALES',
      calculationBasis: 'REALIZED_PROFIT',
      ratePercent: 12,
      effectiveFrom: dayjs('2026-09-20'),
      employeeIds: [],
    });
    expect(serviceMocks.copyCommissionRule).not.toHaveBeenCalled();
  });
});
