import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import {
  CommissionCreateModal,
  resetCommissionModalMocks,
  serviceMocks,
  verificationCandidate,
} from './CommissionCreateModal.fixtures';

beforeEach(resetCommissionModalMocks);

it('关闭生成弹窗即取消候选在途请求，重开后不采纳旧响应', async () => {
  let oldSignal: AbortSignal | undefined;
  let resolveOld: ((response: unknown) => void) | undefined;
  serviceMocks.listCommissionVerificationCandidates
    .mockImplementationOnce((_params, options) => {
      oldSignal = options.signal;
      expect(options.skipErrorHandler).toBe(true);
      return new Promise((resolve) => {
        resolveOld = resolve;
      });
    })
    .mockResolvedValueOnce({ data: [verificationCandidate] });
  const content = (open: boolean) => (
    <App>
      <CommissionCreateModal
        open={open}
        onOpenChange={vi.fn()}
        onSuccess={vi.fn()}
      />
    </App>
  );
  const view = renderWithClient(content(true));
  await waitFor(() => expect(oldSignal).toBeDefined());
  view.rerender(content(false));
  expect(oldSignal?.aborted).toBe(true);
  await act(async () => {
    resolveOld?.({
      data: [{ ...verificationCandidate, verificationNo: '旧请求候选' }],
    });
  });
  expect(screen.queryByText(/旧请求候选/)).not.toBeInTheDocument();
  view.rerender(content(true));
  await screen.findByText('VR-A｜A单位｜200.00 CNY');
  expect(
    serviceMocks.listCommissionVerificationCandidates,
  ).toHaveBeenCalledTimes(2);
});
