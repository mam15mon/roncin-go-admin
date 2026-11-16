import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Page, response, state } from './list-export-page.fixtures';

describe('完整订单CSV的基础页面与读取资格', () => {
  it.each([false, true])(
    '成功采纳后一次请求完整CSV，子资源权限%s准确决定授权列',
    async (resources) => {
      state.canReadContainers = resources;
      state.canReadAttachments = resources;
      state.export.mockResolvedValue(response());
      renderWithClient(<Page />);
      await screen.findByText('AI-CURRENT');
      const button = screen.getByRole('button', { name: /导出完整筛选结果/ });
      await waitFor(() => expect(button).toBeEnabled());
      fireEvent.click(button);
      await waitFor(() => expect(state.download).toHaveBeenCalledOnce());
      expect(state.export).toHaveBeenCalledOnce();
      expect(state.download.mock.calls[0][0]).toContain('AI-CURRENT');
      expect(state.download.mock.calls[0][0].includes('附件数')).toBe(
        resources,
      );
      expect(state.download.mock.calls[0][0].includes('实际箱号')).toBe(
        resources,
      );
    },
  );

  it('公司身份未就绪或无本类读取权限，不请求列表也不展示导出', async () => {
    state.company = undefined;
    const view = renderWithClient(<Page />);
    await act(async () => {});
    expect(state.list).not.toHaveBeenCalled();
    expect(
      screen.queryByRole('button', { name: /导出完整筛选结果/ }),
    ).not.toBeInTheDocument();
    view.unmount();
    state.company = 'company-a';
    state.canRead = false;
    renderWithClient(<Page />);
    await act(async () => {});
    expect(state.list).not.toHaveBeenCalled();
    expect(state.export).not.toHaveBeenCalled();
  });
});
