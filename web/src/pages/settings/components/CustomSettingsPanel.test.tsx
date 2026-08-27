import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { CustomSettingsPanel } from './CustomSettingsPanel';

// Mock umi access hook
vi.mock('@umijs/max', () => ({
  useAccess: () => ({
    canReadExchangeRates: true,
    canUpdateExchangeRates: true,
  }),
}));

// Mock API service
const mockGetExchangeRateCustomSetting = vi.fn();
const mockUpdateExchangeRateCustomSetting = vi.fn();

vi.mock('@/services/roncin/exchangeRateService', () => ({
  exchangeRateServiceGetExchangeRateCustomSetting: () => mockGetExchangeRateCustomSetting(),
  exchangeRateServiceUpdateExchangeRateCustomSetting: (body: any) =>
    mockUpdateExchangeRateCustomSetting(body),
}));

describe('CustomSettingsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正确加载并渲染自定义汇率设置项与开关', async () => {
    mockGetExchangeRateCustomSetting.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        inheritBaseCurrencyRate: false,
        version: '0',
      },
    });

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    expect(screen.getByText('财务汇率设置')).toBeInTheDocument();
    expect(screen.getByText('专用汇率未配置时继承折本币汇率')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('默认关闭')).toBeInTheDocument();
    });
  });

  it('点击切换开关时提交 expectedVersion 并成功更新状态', async () => {
    mockGetExchangeRateCustomSetting.mockResolvedValueOnce({
      success: true,
      data: {
        organizationId: 'org-headquarter',
        inheritBaseCurrencyRate: false,
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
      expect(screen.getByText('默认关闭')).toBeInTheDocument();
    });

    const switchBtn = screen.getByRole('switch');
    expect(switchBtn).toBeInTheDocument();
    expect(switchBtn).not.toBeChecked();

    fireEvent.click(switchBtn);

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

  it('更新失败或版本冲突时重新拉取最新设置', async () => {
    mockGetExchangeRateCustomSetting
      .mockResolvedValueOnce({
        success: true,
        data: {
          organizationId: 'org-headquarter',
          inheritBaseCurrencyRate: false,
          version: '0',
        },
      })
      .mockResolvedValueOnce({
        success: true,
        data: {
          organizationId: 'org-headquarter',
          inheritBaseCurrencyRate: true,
          version: '2',
        },
      });

    mockUpdateExchangeRateCustomSetting.mockRejectedValueOnce(
      new Error('汇率自定义设置已被更新，请刷新后重试'),
    );

    render(
      <App>
        <CustomSettingsPanel />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText('默认关闭')).toBeInTheDocument();
    });

    const switchBtn = screen.getByRole('switch');
    fireEvent.click(switchBtn);

    await waitFor(() => {
      expect(mockUpdateExchangeRateCustomSetting).toHaveBeenCalled();
    });

    // 发生错误后应自动重新拉取最新版本
    await waitFor(() => {
      expect(mockGetExchangeRateCustomSetting).toHaveBeenCalledTimes(2);
      expect(screen.getByText('已开启继承')).toBeInTheDocument();
    });
  });
});
