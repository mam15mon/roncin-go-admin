import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  fetchExchangeRates: vi.fn(),
  syncExchangeRates: vi.fn(),
}));

vi.mock('@/services/roncin/exchangeRateService', () => ({
  exchangeRateServiceFetchExchangeRates: serviceMocks.fetchExchangeRates,
  exchangeRateServiceSyncExchangeRates: serviceMocks.syncExchangeRates,
}));

import ExchangeRateSyncModal from './ExchangeRateSyncModal';

const previewResponse = {
  success: true,
  data: {
    target: 1,
    baseCurrency: 'CNY',
    effectiveFrom: '2026-09-14T00:00:00+08:00',
    effectiveTo: '2026-09-20T23:59:59+08:00',
    source: '新浪财经中行专线',
    fallbackUsed: false,
    rows: [
      {
        fromCurrency: 'USD',
        arRate: '6.75230000',
        apRate: '6.71170000',
        rate: '6.73200000',
        conversionPath: '中国银行现汇买卖价（新浪中行专线，已归一化）',
      },
      {
        fromCurrency: 'EUR',
        arRate: '7.91200000',
        apRate: '7.84600000',
        rate: '7.87900000',
        conversionPath: '中国银行现汇买卖价（新浪中行专线，已归一化）',
      },
    ],
  },
};

function renderModal(baseCurrency = 'CNY') {
  const onClose = vi.fn();
  const onSuccess = vi.fn();
  render(
    <App>
      <ExchangeRateSyncModal
        open
        baseCurrency={baseCurrency}
        onClose={onClose}
        onSuccess={onSuccess}
      />
    </App>,
  );
  return { onClose, onSuccess };
}

describe('ExchangeRateSyncModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    serviceMocks.syncExchangeRates.mockResolvedValue({
      success: true,
      syncedCount: 2,
    });
  });

  it('抓取牌价后按建议值回填预览并明示数据来源', async () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
    // 来源与换算路径在预览中明示（两个币种行均携带换算路径说明）。
    expect(
      await screen.findByText(/数据来源：新浪财经中行专线/),
    ).toBeInTheDocument();
    expect(
      screen.getAllByText(/中国银行现汇买卖价（新浪中行专线，已归一化）+/)
        .length,
    ).toBeGreaterThan(0);
    // 回填：应收=现汇卖出价，应付=现汇买入价。
    expect(screen.getByDisplayValue('6.7523')).toBeInTheDocument();
    expect(screen.getByDisplayValue('6.7117')).toBeInTheDocument();
    expect(screen.getByDisplayValue('7.912')).toBeInTheDocument();
    expect(serviceMocks.fetchExchangeRates).toHaveBeenCalledWith(
      expect.objectContaining({ target: 1 }),
    );
  });

  it('财务微调后确认发布按调整值批量入库', async () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
    await screen.findByDisplayValue('6.7523');
    // 微调应收汇率（商业加点）。
    const arInput = screen.getByDisplayValue('6.7523');
    fireEvent.change(arInput, { target: { value: '6.8' } });
    fireEvent.click(screen.getByRole('button', { name: '确认发布' }));
    await waitFor(() => {
      expect(serviceMocks.syncExchangeRates).toHaveBeenCalledTimes(1);
    });
    expect(serviceMocks.syncExchangeRates).toHaveBeenCalledWith({
      target: 1,
      rows: expect.arrayContaining([
        expect.objectContaining({ fromCurrency: 'USD', arRate: '6.8' }),
        expect.objectContaining({ fromCurrency: 'EUR', arRate: '7.912' }),
      ]),
    });
  });

  it('抓取失败显式引导手工录入而非静默兜底', async () => {
    serviceMocks.fetchExchangeRates.mockRejectedValue(
      new Error('牌价源不可达'),
    );
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
    expect(
      await screen.findByText(
        /请检查网络后重试，或使用「新建汇率」手工录入本周汇率/,
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认发布' })).toBeDisabled();
    expect(serviceMocks.syncExchangeRates).not.toHaveBeenCalled();
  });

  it('预设下周切换按目标周 2 抓取', async () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal();
    fireEvent.click(
      screen.getByRole('radio', { name: /预设下周（下周一至下周日）/ }),
    );
    await waitFor(() => {
      expect(serviceMocks.fetchExchangeRates).toHaveBeenCalledWith(
        expect.objectContaining({ target: 2 }),
      );
    });
  });

  it('非 CNY 本币组织展示一键同步文案', () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal('HKD');
    expect(screen.getByText(/一键同步周汇率（本币 HKD）/)).toBeInTheDocument();
  });
});
