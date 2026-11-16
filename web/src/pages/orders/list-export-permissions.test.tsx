import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Page, response, state } from './list-export-page.fixtures';

describe('完整订单CSV的在途身份与授权变化', () => {
  it('导出在途切换公司后，旧响应即使成功也不下载', async () => {
    let finish!: (value: ReturnType<typeof response>) => void;
    state.export.mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const { rerender, queryClient } = renderWithClient(<Page />);
    await screen.findByText('AI-CURRENT');
    const button = screen.getByRole('button', { name: /导出完整筛选结果/ });
    await waitFor(() => expect(button).toBeEnabled());
    fireEvent.click(button);
    await waitFor(() => expect(state.export).toHaveBeenCalledOnce());
    state.company = 'company-b';
    state.list.mockResolvedValue(response('company-b'));
    rerender(
      <QueryClientProvider client={queryClient}>
        <Page />
      </QueryClientProvider>,
    );
    await act(async () => finish(response()));
    expect(state.download).not.toHaveBeenCalled();
  });

  it.each([
    ['canReadContainers', '实际箱号'],
    ['canReadAttachments', '附件数'],
  ] as const)(
    '导出在途撤回%s时旧响应不下载，新授权会话排除对应列',
    async (permission, column) => {
      state[permission] = true;
      let finish!: (value: ReturnType<typeof response>) => void;
      state.export.mockReturnValueOnce(
        new Promise((resolve) => {
          finish = resolve;
        }),
      );
      const { rerender, queryClient } = renderWithClient(<Page />);
      await screen.findByText('AI-CURRENT');
      const button = screen.getByRole('button', { name: /导出完整筛选结果/ });
      await waitFor(() => expect(button).toBeEnabled());
      fireEvent.click(button);
      await waitFor(() => expect(state.export).toHaveBeenCalledOnce());
      state[permission] = false;
      rerender(
        <QueryClientProvider client={queryClient}>
          <Page />
        </QueryClientProvider>,
      );
      await act(async () => finish(response()));
      expect(state.download).not.toHaveBeenCalled();
      state.export.mockResolvedValue(response());
      const next = screen.getByRole('button', { name: /导出完整筛选结果/ });
      await waitFor(() => expect(next).toBeEnabled());
      fireEvent.click(next);
      await waitFor(() => expect(state.download).toHaveBeenCalledOnce());
      expect(state.download.mock.calls[0][0]).not.toContain(column);
    },
  );
});
