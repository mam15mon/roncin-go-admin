import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import ExchangeRateImportModal from './ExchangeRateImportModal';

const api = vi.hoisted(() => ({ preview: vi.fn(), confirm: vi.fn() }));
vi.mock('@/services/roncin/exchangeRateService', () => ({
  exchangeRateServicePreviewExchangeRateImport: api.preview,
  exchangeRateServiceConfirmExchangeRateImport: api.confirm,
}));
vi.mock('./exchangeRateTemplate', () => ({
  downloadExchangeRateImportTemplate: vi.fn(),
}));

beforeEach(() => {
  vi.clearAllMocks();
  api.preview.mockResolvedValue({
    success: true,
    previewToken: 'preview-a',
    data: { id: 'batch-a', canConfirm: true, validCount: 1 },
  });
});

it('确认在途冻结所有关闭入口，失败后仍可用原键重试', async () => {
  let resolveConfirmation: ((response: unknown) => void) | undefined;
  api.confirm
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveConfirmation = resolve;
        }),
    )
    .mockResolvedValueOnce({ success: true, data: { id: 'batch-a' } });
  const onClose = vi.fn();
  const onSuccess = vi.fn();
  renderWithClient(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <ExchangeRateImportModal open onClose={onClose} onSuccess={onSuccess} />
      </App>
    </ConfigProvider>,
  );
  const input = document.querySelector('input[type="file"]');
  if (!input) throw new Error('缺少上传入口');
  fireEvent.change(input, {
    target: { files: [new File(['rows'], 'rates.xlsx')] },
  });
  await screen.findByText('预检通过，数据格式完全合规');
  fireEvent.click(screen.getByRole('button', { name: /确认导入/ }));
  await waitFor(() => expect(resolveConfirmation).toBeDefined());
  expect(screen.getByRole('button', { name: /取\s*消/ })).toBeDisabled();
  expect(screen.getByRole('button', { name: /重新上传/ })).toBeDisabled();
  expect(document.querySelector('.ant-modal-close')).toBeNull();
  const wrap = document.querySelector('.ant-modal-wrap');
  if (!wrap) throw new Error('缺少弹窗容器');
  fireEvent.keyDown(wrap, { key: 'Escape', keyCode: 27 });
  fireEvent.click(wrap);
  expect(onClose).not.toHaveBeenCalled();
  await act(async () => {
    resolveConfirmation?.(undefined);
  });
  await waitFor(() =>
    expect(screen.getByRole('button', { name: /确认导入/ })).not.toHaveClass(
      'ant-btn-loading',
    ),
  );
  expect(onSuccess).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: /确认导入/ }));
  await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  expect(onSuccess).toHaveBeenCalledOnce();
  expect(api.confirm.mock.calls[0][0]).toEqual(api.confirm.mock.calls[1][0]);
});
