import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  passkeyServiceDeletePasskey,
  passkeyServiceGetPasskeyConfig,
  passkeyServiceListPasskeys,
} from '@/services/roncin/passkeyService';
import { PasskeyManagerModal } from './PasskeyManagerModal';

vi.mock('@/services/roncin/passkeyService');
vi.mock('@simplewebauthn/browser', () => ({
  browserSupportsWebAuthn: () => true,
}));
const list = vi.mocked(passkeyServiceListPasskeys);
const remove = vi.mocked(passkeyServiceDeletePasskey);
const secureDescriptor = Object.getOwnPropertyDescriptor(
  window,
  'isSecureContext',
);

beforeEach(() => {
  vi.resetAllMocks();
  Object.defineProperty(window, 'isSecureContext', {
    configurable: true,
    value: true,
  });
  vi.mocked(passkeyServiceGetPasskeyConfig).mockResolvedValue({
    data: {
      enabled: true,
      rpId: 'localhost',
      rpName: 'Roncin',
      origins: [window.location.origin],
    },
  });
  list.mockResolvedValue({
    success: true,
    data: {
      items: [
        { id: 'key-1', name: '办公电脑', createdAt: '2026-11-01T00:00:00Z' },
      ],
    },
  });
  remove.mockResolvedValue({ success: true });
});
afterEach(() => {
  cleanup();
  if (secureDescriptor)
    Object.defineProperty(window, 'isSecureContext', secureDescriptor);
  else Reflect.deleteProperty(window, 'isSecureContext');
});
function mount() {
  return renderWithClient(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <PasskeyManagerModal open userId="user-1" onClose={vi.fn()} />
      </App>
    </ConfigProvider>,
  );
}

describe('本人通行密钥管理', () => {
  it('显示本人名称和格式化时间，不暴露凭证载荷', async () => {
    mount();
    expect(await screen.findByText('办公电脑')).toBeVisible();
    expect(screen.getByText(/添加时间：2026-11-01/)).toBeVisible();
    expect(screen.queryByText('2026-11-01T00:00:00Z')).not.toBeInTheDocument();
    expect(list).toHaveBeenCalledWith({ skipErrorHandler: true });
  });

  it('明确确认后删除并刷新列表', async () => {
    mount();
    await screen.findByText('办公电脑');
    list.mockResolvedValue({ success: true, data: { items: [] } });
    fireEvent.click(screen.getByRole('button', { name: '删除办公电脑' }));
    const dialogs = await screen.findAllByRole('dialog');
    const confirmation = dialogs.find(
      (dialog) => within(dialog).queryAllByText('删除通行密钥').length > 0,
    );
    if (!confirmation) throw new Error('缺少删除确认');
    expect(remove).not.toHaveBeenCalled();
    await act(async () => {
      fireEvent.click(
        within(confirmation).getByRole('button', { name: /删\s*除/ }),
      );
    });
    await waitFor(() =>
      expect(remove).toHaveBeenCalledWith(
        { id: 'key-1' },
        { skipErrorHandler: true },
      ),
    );
    await waitFor(() =>
      expect(screen.queryByText('办公电脑')).not.toBeInTheDocument(),
    );
    expect(await screen.findByText('尚未添加通行密钥')).toBeVisible();
  });

  it('缺少成功写报文时保留列表，不能把删除当成成功', async () => {
    remove.mockResolvedValue({ success: false });
    mount();
    await screen.findByText('办公电脑');
    fireEvent.click(screen.getByRole('button', { name: '删除办公电脑' }));
    const dialogs = await screen.findAllByRole('dialog');
    const confirmation = dialogs.find(
      (dialog) => within(dialog).queryAllByText('删除通行密钥').length > 0,
    );
    if (!confirmation) throw new Error('缺少删除确认');
    await act(async () => {
      fireEvent.click(
        within(confirmation).getByRole('button', { name: /删\s*除/ }),
      );
    });
    await waitFor(() => expect(remove).toHaveBeenCalledOnce());
    expect(screen.getByText('办公电脑')).toBeVisible();
    expect(list).toHaveBeenCalledOnce();
  });
});
