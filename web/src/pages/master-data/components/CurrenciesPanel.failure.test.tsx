import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CurrenciesPanel from './CurrenciesPanel';

const state = vi.hoisted(() => ({ enabled: false, update: vi.fn() }));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    isSystemWorkspace: false,
    canUpdateMasterDataCurrencies: true,
  }),
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: async () => ({
    success: true,
    data: [{ id: 'usd', code: 'USD', name: '美元', enabled: state.enabled }],
  }),
  masterDataServiceSetCurrencyEnabled: state.update,
}));

beforeEach(() => {
  state.enabled = false;
  state.update.mockReset().mockImplementation(async () => {
    state.enabled = true;
    return { success: true, data: { id: 'usd' } };
  });
});

describe('币种启停的真实异步确认', () => {
  it.each([
    ['请求层消费失败', undefined],
    ['缺少实体标识', { success: true }],
    ['失败信封携带实体', { success: false, data: { id: 'usd' } }],
  ])('%s 保持确认框，重试成功才关闭并更新状态', async (_name, response) => {
    state.update.mockResolvedValueOnce(response);
    renderWithApp(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <CurrenciesPanel />
      </ConfigProvider>,
    );
    const trigger = await screen.findByRole('button', { name: /启\s*用/ });
    fireEvent.click(trigger);
    const confirm = await screen.findByRole('button', { name: /确\s*定/ });
    fireEvent.click(confirm);
    await waitFor(() => expect(state.update).toHaveBeenCalledOnce());
    await waitFor(() => expect(confirm).not.toHaveClass('ant-btn-loading'));
    expect(trigger).toHaveClass('ant-popover-open');
    expect(screen.getByText('确定要启用【美元】吗？')).toBeVisible();
    expect(screen.getByText('未启用')).toBeVisible();
    fireEvent.click(confirm);
    await waitFor(() => expect(state.update).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(trigger).not.toHaveClass('ant-popover-open'));
    expect(await screen.findByText('已启用')).toBeVisible();
  });

  it('请求拒绝后保留确认并反馈错误，允许原操作再次确认', async () => {
    state.update.mockRejectedValueOnce(new Error('币种修改失败'));
    renderWithApp(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <CurrenciesPanel />
      </ConfigProvider>,
    );
    const trigger = await screen.findByRole('button', { name: /启\s*用/ });
    fireEvent.click(trigger);
    const confirm = await screen.findByRole('button', { name: /确\s*定/ });
    fireEvent.click(confirm);
    await waitFor(() => expect(screen.getByText('币种修改失败')).toBeVisible());
    await waitFor(() => expect(confirm).not.toHaveClass('ant-btn-loading'));
    expect(trigger).toHaveClass('ant-popover-open');
    expect(state.enabled).toBe(false);
  });
});
