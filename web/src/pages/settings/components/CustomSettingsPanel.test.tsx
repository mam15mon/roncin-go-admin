import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  BILLED_FEE_EDITABLE_FIELD,
  CustomSettingsPanel,
} from './CustomSettingsPanel';

// Mock umi access hook
vi.mock('@umijs/max', () => ({
  useAccess: () => ({
    canReadExchangeRates: true,
    canUpdateExchangeRates: true,
    canReadFinanceBills: true,
    canUpdateFinanceBills: true,
  }),
}));

// Mock API services
const mockGetExchangeRateCustomSetting = vi.fn();
const mockUpdateExchangeRateCustomSetting = vi.fn();
const mockGetBilledFeeEditPolicy = vi.fn();
const mockUpdateBilledFeeEditPolicy = vi.fn();

vi.mock('@/services/roncin/exchangeRateService', () => ({
  exchangeRateServiceGetExchangeRateCustomSetting: () => mockGetExchangeRateCustomSetting(),
  exchangeRateServiceUpdateExchangeRateCustomSetting: (body: any) =>
    mockUpdateExchangeRateCustomSetting(body),
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetBilledFeeEditPolicy: () => mockGetBilledFeeEditPolicy(),
  settlementServiceUpdateBilledFeeEditPolicy: (body: any) =>
    mockUpdateBilledFeeEditPolicy(body),
}));

describe('CustomSettingsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('并行加载并渲染财务汇率设置与账单费用修改策略', async () => {
    mockGetExchangeRateCustomSetting.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        inheritBaseCurrencyRate: false,
        version: '0',
      },
    });

    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
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

    // 检查财务汇率设置项
    expect(screen.getByText('财务汇率设置')).toBeInTheDocument();
    expect(screen.getByText('专用汇率未配置时继承折本币汇率')).toBeInTheDocument();

    // 检查账单费用修改策略项
    expect(screen.getByText('账单费用修改策略')).toBeInTheDocument();
    expect(screen.getByText('账单创建后允许修改费用')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('默认关闭')).toBeInTheDocument();
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
    mockGetExchangeRateCustomSetting.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        inheritBaseCurrencyRate: false,
        version: '0',
      },
    });

    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
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
      const defaultClosed = screen.getAllByText('默认关闭');
      expect(defaultClosed.length).toBe(2);
    });

    const switches = screen.getAllByRole('switch');
    expect(switches.length).toBe(2);
    const feePolicySwitch = switches[1];

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
    mockGetExchangeRateCustomSetting.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        inheritBaseCurrencyRate: false,
        version: '0',
      },
    });

    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
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
      expect(calledArg.editableFields).toContain(BILLED_FEE_EDITABLE_FIELD.QUANTITY);
      expect(calledArg.editableFields).toContain(BILLED_FEE_EDITABLE_FIELD.UNIT_PRICE);
    });
  });

  it('切换汇率继承开关时提交 expectedVersion 并成功更新状态', async () => {
    mockGetExchangeRateCustomSetting.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        inheritBaseCurrencyRate: false,
        version: '0',
      },
    });

    mockGetBilledFeeEditPolicy.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        enabled: false,
        editableFields: [],
        version: '0',
      },
    });

    mockUpdateExchangeRateCustomSetting.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        inheritBaseCurrencyRate: true,
        version: '1',
        updatedAt: '2026-08-27T16:50:00Z',
        updatedBy: 'admin',
      },
    });

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    await waitFor(() => {
      const defaultClosed = screen.getAllByText('默认关闭');
      expect(defaultClosed.length).toBe(2);
    });

    const switches = screen.getAllByRole('switch');
    const rateSwitch = switches[0];
    expect(rateSwitch).not.toBeChecked();

    fireEvent.click(rateSwitch);

    await waitFor(() => {
      expect(mockUpdateExchangeRateCustomSetting).toHaveBeenCalledWith({
        inheritBaseCurrencyRate: true,
        expectedVersion: '0',
      });
    });

    await waitFor(() => {
      expect(screen.getByText('已开启继承')).toBeInTheDocument();
    });
  });
});
