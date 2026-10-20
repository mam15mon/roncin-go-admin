import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import ExchangeRateImportModal from './ExchangeRateImportModal';

const mocks = vi.hoisted(() => ({ preview: vi.fn(), confirm: vi.fn() }));
vi.mock('@/services/roncin/exchangeRateService', () => ({
  exchangeRateServicePreviewExchangeRateImport: mocks.preview,
  exchangeRateServiceConfirmExchangeRateImport: mocks.confirm,
}));
vi.mock('./exchangeRateTemplate', () => ({
  downloadExchangeRateImportTemplate: vi.fn(),
}));
beforeEach(() => {
  window.history.replaceState({}, '', '/finance/exchange-rates');
  vi.clearAllMocks();
  mocks.preview.mockResolvedValue({
    previewToken: 'token',
    data: {
      id: 'batch',
      fileName: 'rates.xlsx',
      totalCount: 1,
      validCount: 1,
      canConfirm: true,
      rows: [],
    },
  });
});

it('实际载入Excel预检payload才标脏，确认失败保留，成功清理', async () => {
  mocks.confirm
    .mockResolvedValueOnce(undefined)
    .mockResolvedValueOnce({ success: true, data: { id: 'batch' } });
  const onClose = vi.fn();
  render(
    <App>
      <ExchangeRateImportModal open onClose={onClose} onSuccess={vi.fn()} />
    </App>,
  );
  expect(getAnyDirtyTabKey()).toBeUndefined();
  const input = document.querySelector('input[type="file"]');
  if (!input) throw new Error('缺少文件上传入口');
  await act(async () => {
    fireEvent.change(input, {
      target: { files: [new File(['rates'], 'rates.xlsx')] },
    });
  });
  await screen.findByText('预检通过，数据格式完全合规');
  expect(getAnyDirtyTabKey()).toBe('/finance/exchange-rates');
  fireEvent.click(screen.getByRole('button', { name: /确认导入/ }));
  await waitFor(() => expect(mocks.confirm).toHaveBeenCalledOnce());
  await waitFor(() =>
    expect(screen.getByRole('button', { name: /确认导入/ })).not.toHaveClass(
      'ant-btn-loading',
    ),
  );
  expect(onClose).not.toHaveBeenCalled();
  expect(getAnyDirtyTabKey()).toBe('/finance/exchange-rates');
  fireEvent.click(screen.getByRole('button', { name: /确认导入/ }));
  await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  expect(getAnyDirtyTabKey()).toBeUndefined();
});

it('无效文件预检失败不误提示，取消清理', async () => {
  mocks.preview.mockRejectedValueOnce(new Error('预检失败'));
  const onClose = vi.fn();
  const view = render(
    <App>
      <ExchangeRateImportModal open onClose={onClose} onSuccess={vi.fn()} />
    </App>,
  );
  const input = document.querySelector('input[type="file"]');
  if (!input) throw new Error('缺少文件上传入口');
  await act(async () => {
    fireEvent.change(input, {
      target: { files: [new File(['rates'], 'rates.xlsx')] },
    });
  });
  await waitFor(() => expect(mocks.preview).toHaveBeenCalledOnce());
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }));
  expect(onClose).toHaveBeenCalledOnce();
  expect(getAnyDirtyTabKey()).toBeUndefined();
  view.unmount();
});
