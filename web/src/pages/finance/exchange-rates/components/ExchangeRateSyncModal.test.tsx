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
    localStorage.clear();
    serviceMocks.syncExchangeRates.mockResolvedValue({
      success: true,
      syncedCount: 2,
    });
  });

  it('抓取牌价后按建议值回填预览并明示数据来源与点差', async () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
    // 来源与换算路径在预览中明示
    expect(
      await screen.findByText(/数据来源：新浪财经中行专线/),
    ).toBeInTheDocument();
    expect(
      screen.getAllByText(/中国银行现汇买卖价（新浪中行专线，已归一化）+/)
        .length,
    ).toBeGreaterThan(0);
    // 默认 4 位精度回填：应收=现汇卖出价，应付=现汇买入价
    expect(screen.getByDisplayValue('6.7523')).toBeInTheDocument();
    expect(screen.getByDisplayValue('6.7117')).toBeInTheDocument();
    expect(screen.getByDisplayValue('7.9120')).toBeInTheDocument();
    // 币种名称与代码
    expect(screen.getByText('USD')).toBeInTheDocument();
    expect(screen.getByText('美元')).toBeInTheDocument();
    // 买卖点差展示：6.7523 - 6.7117 = +0.0406
    expect(screen.getByText('+0.0406')).toBeInTheDocument();
    expect(serviceMocks.fetchExchangeRates).toHaveBeenCalledWith(
      expect.objectContaining({ target: 1 }),
    );
  });

  it('支持切换小数精度并在 2 位/3 位/原始数据之间无损换算', async () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
    await screen.findByDisplayValue('6.7523');

    // 切换到 2 位小数（对账简化）
    fireEvent.click(screen.getByRole('radio', { name: /2 位（对账简化）/ }));
    expect(screen.getByDisplayValue('6.75')).toBeInTheDocument();
    expect(screen.getByDisplayValue('6.71')).toBeInTheDocument();
    expect(screen.getByDisplayValue('7.91')).toBeInTheDocument();
    expect(screen.getByText('+0.04')).toBeInTheDocument();

    // 切换到 3 位小数
    fireEvent.click(screen.getByRole('radio', { name: /3 位/ }));
    expect(screen.getByDisplayValue('6.752')).toBeInTheDocument();
    expect(screen.getByDisplayValue('6.712')).toBeInTheDocument();
    expect(screen.getByDisplayValue('7.912')).toBeInTheDocument();

    // 切换到原始抓取数据
    fireEvent.click(screen.getByRole('radio', { name: /原始抓取/ }));
    expect(screen.getByDisplayValue('6.7523')).toBeInTheDocument();
    expect(screen.getByDisplayValue('7.912')).toBeInTheDocument();
  });

  it('支持一键套用上次微调加点并能还原原价', async () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
    await screen.findByDisplayValue('6.7117');

    // 点击「套用上次微调」（预置/记忆中的 USD apOffset +0.02、EUR +0.02）
    fireEvent.click(screen.getByRole('button', { name: /套用上次微调/ }));
    // 6.7117 + 0.02 = 6.7317
    expect(screen.getByDisplayValue('6.7317')).toBeInTheDocument();
    // 7.8460 + 0.02 = 7.8660
    expect(screen.getByDisplayValue('7.8660')).toBeInTheDocument();
    expect(screen.getAllByText(/加点 \+0.0200/).length).toBeGreaterThan(0);

    // 点击「还原原价」恢复为原始牌价
    fireEvent.click(screen.getByRole('button', { name: /还原原价/ }));
    expect(screen.getByDisplayValue('6.7117')).toBeInTheDocument();
    expect(screen.getByDisplayValue('7.8460')).toBeInTheDocument();
  });

  it('买卖倒挂时显示警示标签', async () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
    await screen.findByDisplayValue('6.7523');

    // 修改卖出价使其低于买入价：6.70 < 6.7117
    const arInput = screen.getByDisplayValue('6.7523');
    fireEvent.change(arInput, { target: { value: '6.7000' } });

    // 点差变为倒挂
    expect(screen.getByText(/倒挂 -0.0117/)).toBeInTheDocument();
  });

  it('财务微调后确认发布按调整值批量入库并自动记忆微调量', async () => {
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
        expect.objectContaining({ fromCurrency: 'EUR', arRate: '7.9120' }),
      ]),
    });
    // 验证 localStorage 记忆了微调
    expect(
      localStorage.getItem('roncin_exchange_rate_last_markups'),
    ).toBeTruthy();
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

  it('支持点击抓取牌价刷新本周牌价', async () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
    await waitFor(() => {
      expect(serviceMocks.fetchExchangeRates).toHaveBeenCalledWith(
        expect.objectContaining({ target: 1 }),
      );
    });
  });

  it('非 CNY 本币组织展示一键同步文案', () => {
    serviceMocks.fetchExchangeRates.mockResolvedValue(previewResponse);
    renderModal('HKD');
    expect(screen.getByText(/一键同步周汇率（本币 HKD）/)).toBeInTheDocument();
  });
});
