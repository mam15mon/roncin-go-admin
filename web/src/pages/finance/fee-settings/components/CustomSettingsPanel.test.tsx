import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  BILLED_FEE_EDITABLE_FIELD,
  CustomSettingsPanel,
} from './CustomSettingsPanel';

// Mock API services
const mockGetBilledFeeEditPolicy = vi.fn();
const mockUpdateBilledFeeEditPolicy = vi.fn();
const mockGetCreditLimitControlPolicy = vi.fn();
const mockUpdateCreditLimitControlPolicy = vi.fn();

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetBilledFeeEditPolicy: () => mockGetBilledFeeEditPolicy(),
  settlementServiceUpdateBilledFeeEditPolicy: (body: any) =>
    mockUpdateBilledFeeEditPolicy(body),
  settlementServiceGetCreditLimitControlPolicy: () =>
    mockGetCreditLimitControlPolicy(),
  settlementServiceUpdateCreditLimitControlPolicy: (body: any) =>
    mockUpdateCreditLimitControlPolicy(body),
}));

describe('CustomSettingsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // 信用额度管控策略默认返回仅提醒模式，避免每个用例重复准备。
    mockGetCreditLimitControlPolicy.mockResolvedValue({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        allowSelectionWhenCreditExceeded: true,
        version: '1',
      },
    });
  });

  afterEach(() => {
    cleanup();
  });

  it('加载并渲染账单费用修改策略', async () => {
    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: true,
        editableFields: [
          BILLED_FEE_EDITABLE_FIELD.QUANTITY,
          BILLED_FEE_EDITABLE_FIELD.UNIT_PRICE,
        ],
        version: '1',
      },
    });

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    expect(screen.getByText('账单费用修改策略')).toBeInTheDocument();
    expect(screen.getByText('账单创建后允许修改费用')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('已开启修改')).toBeInTheDocument();
    });

    // 检查可修改字段 Checkbox
    expect(screen.getByLabelText('费用名称')).toBeInTheDocument();
    expect(screen.getByLabelText('币种')).toBeInTheDocument();
    expect(screen.getByLabelText('汇率')).toBeInTheDocument();
    expect(screen.getByLabelText('数量')).toBeInTheDocument();
    expect(screen.getByLabelText('单价')).toBeInTheDocument();
    expect(screen.getByLabelText('税率')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByLabelText('数量')).toBeChecked();
      expect(screen.getByLabelText('单价')).toBeChecked();
      expect(screen.getByLabelText('费用名称')).not.toBeChecked();
    });
  });

  it('切换账单费用修改策略总开关并提交 expectedVersion', async () => {
    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: false,
        editableFields: [],
        version: '0',
      },
    });

    mockUpdateBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: true,
        editableFields: [],
        version: '1',
        updatedAt: '2026-08-27T18:00:00Z',
        updatedBy: 'admin',
      },
    });

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText('默认关闭')).toBeInTheDocument();
    });

    const feePolicySwitch = screen.getByRole('switch', {
      name: '账单费用修改开关',
    });

    fireEvent.click(feePolicySwitch);

    await waitFor(() => {
      expect(mockUpdateBilledFeeEditPolicy).toHaveBeenCalledWith({
        enabled: true,
        editableFields: [],
        expectedVersion: '0',
      });
    });

    await waitFor(() => {
      expect(screen.getByText('已开启修改')).toBeInTheDocument();
    });
  });

  it('勾选可修改字段时正确更新 editableFields', async () => {
    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: true,
        editableFields: [BILLED_FEE_EDITABLE_FIELD.QUANTITY],
        version: '1',
      },
    });

    mockUpdateBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: true,
        editableFields: [
          BILLED_FEE_EDITABLE_FIELD.QUANTITY,
          BILLED_FEE_EDITABLE_FIELD.UNIT_PRICE,
        ],
        version: '2',
      },
    });

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByLabelText('数量')).toBeChecked();
    });

    const unitPriceCheckbox = screen.getByLabelText('单价');
    fireEvent.click(unitPriceCheckbox);

    await waitFor(() => {
      expect(mockUpdateBilledFeeEditPolicy).toHaveBeenCalled();
      const calledArg = mockUpdateBilledFeeEditPolicy.mock.calls[0][0];
      expect(calledArg.enabled).toBe(true);
      expect(calledArg.expectedVersion).toBe('1');
      expect(calledArg.editableFields).toContain(
        BILLED_FEE_EDITABLE_FIELD.QUANTITY,
      );
      expect(calledArg.editableFields).toContain(
        BILLED_FEE_EDITABLE_FIELD.UNIT_PRICE,
      );
    });
  });

  it('服务端声明当前公司不可编辑时禁用费用策略，而不依赖权限键', async () => {
    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      canUpdate: false,
      data: {
        organizationId: 'org-a',
        enabled: true,
        editableFields: [],
        version: '1',
      },
    });
    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    await waitFor(() =>
      expect(
        screen.getByRole('switch', { name: '账单费用修改开关' }),
      ).toBeDisabled(),
    );
    expect(screen.getByLabelText('费用名称')).toBeDisabled();
    expect(mockUpdateBilledFeeEditPolicy).not.toHaveBeenCalled();
  });

  it('渲染信用额度管控策略卡片并默认展示仅提醒模式', async () => {
    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: false,
        editableFields: [],
        version: '1',
      },
    });
    mockGetCreditLimitControlPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        allowSelectionWhenCreditExceeded: true,
        version: '3',
      },
    });

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    expect(screen.getByText('往来单位信用额度管控策略')).toBeInTheDocument();
    expect(
      screen.getByText('往来户超信用额度后是否可以选择'),
    ).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('仅提醒模式')).toBeInTheDocument();
    });

    const creditSwitch = screen.getByRole('switch', {
      name: '信用额度管控开关',
    });
    expect(creditSwitch).toBeEnabled();
    expect(creditSwitch).toBeChecked();
  });

  it('关闭信用额度管控开关进入直接干预模式并提交 expectedVersion', async () => {
    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: false,
        editableFields: [],
        version: '1',
      },
    });
    mockGetCreditLimitControlPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        allowSelectionWhenCreditExceeded: true,
        version: '3',
      },
    });
    mockUpdateCreditLimitControlPolicy.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        allowSelectionWhenCreditExceeded: false,
        version: '4',
        updatedAt: '2026-09-12T10:00:00Z',
        updatedBy: 'admin',
      },
    });

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    await waitFor(() => {
      expect(
        screen.getByRole('switch', { name: '信用额度管控开关' }),
      ).toBeChecked();
    });

    fireEvent.click(screen.getByRole('switch', { name: '信用额度管控开关' }));

    await waitFor(() => {
      expect(mockUpdateCreditLimitControlPolicy).toHaveBeenCalledWith({
        allowSelectionWhenCreditExceeded: false,
        expectedVersion: '3',
      });
    });

    await waitFor(() => {
      expect(screen.getByText('直接干预模式')).toBeInTheDocument();
    });
  });

  it('最近修改操作人优先展示姓名，姓名缺失时回退用户 ID', async () => {
    const fallbackUserId = '01a09ead-242d-7318-9a75-eca6447ce258';
    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: true,
        editableFields: [],
        version: '1',
        updatedAt: '2026-09-15T05:59:47Z',
        updatedBy: fallbackUserId,
        updatedByName: '张三',
      },
    });
    mockGetCreditLimitControlPolicy.mockResolvedValueOnce({
      success: true,
      canUpdate: true,
      data: {
        organizationId: 'org-headquarter',
        allowSelectionWhenCreditExceeded: true,
        version: '3',
        updatedAt: '2026-09-15T05:59:47Z',
        updatedBy: fallbackUserId,
      },
    });

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText(/（操作人：张三）/)).toBeInTheDocument();
    });
    expect(
      screen.getByText(new RegExp(`（操作人：${fallbackUserId}`)),
    ).toBeInTheDocument();
  });
});
