import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import ExchangeRateSyncModal from './ExchangeRateSyncModal';

const api = vi.hoisted(() => ({ fetch: vi.fn(), publish: vi.fn() }));
vi.mock('@/services/roncin/exchangeRateService', () => ({
  exchangeRateServiceFetchExchangeRates: api.fetch,
  exchangeRateServiceSyncExchangeRates: api.publish,
}));

beforeEach(() => {
  vi.clearAllMocks();
  api.fetch.mockResolvedValue({
    success: true,
    data: {
      baseCurrency: 'CNY',
      rows: [
        { fromCurrency: 'USD', arRate: '7.2', apRate: '7.1', rate: '7.15' },
      ],
    },
  });
});

it('发布期间不允许继续编辑或关闭，失败保留人工输入并恢复操作', async () => {
  let resolvePublish: ((response: unknown) => void) | undefined;
  api.publish
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolvePublish = resolve;
        }),
    )
    .mockResolvedValueOnce({ success: true });
  const onClose = vi.fn();
  const onSuccess = vi.fn();
  renderWithClient(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <ExchangeRateSyncModal
          open
          baseCurrency="CNY"
          onClose={onClose}
          onSuccess={onSuccess}
        />
      </App>
    </ConfigProvider>,
  );
  fireEvent.click(screen.getByRole('button', { name: /抓取牌价/ }));
  const rate = await screen.findByDisplayValue('7.2000');
  fireEvent.change(rate, { target: { value: '7.3' } });
  fireEvent.click(screen.getByRole('button', { name: /确认发布/ }));
  await waitFor(() => expect(resolvePublish).toBeDefined());
  expect(rate).toBeDisabled();
  expect(screen.getByRole('button', { name: /抓取牌价/ })).toBeDisabled();
  expect(screen.getByRole('button', { name: /取\s*消/ })).toBeDisabled();
  expect(document.querySelector('.ant-modal-close')).toBeNull();
  const wrap = document.querySelector('.ant-modal-wrap');
  if (!wrap) throw new Error('缺少弹窗容器');
  fireEvent.keyDown(wrap, { key: 'Escape', keyCode: 27 });
  fireEvent.click(wrap);
  expect(onClose).not.toHaveBeenCalled();
  await act(async () => {
    resolvePublish?.(undefined);
  });
  await waitFor(() => expect(rate).not.toBeDisabled());
  expect(rate).toHaveValue('7.3');
  expect(onClose).not.toHaveBeenCalled();
  expect(onSuccess).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: /确认发布/ }));
  await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  expect(onSuccess).toHaveBeenCalledOnce();
  expect(api.publish.mock.calls[0][0]).toEqual(api.publish.mock.calls[1][0]);
});
