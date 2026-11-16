import { act, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  order,
  renderPanel,
  services,
  setupTests,
} from './ImportHandlingPanel.fixtures';

setupTests();
async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1);
  });
}

describe('进口服务前台检查', () => {
  it.each([
    OrderBusinessType.BUSINESS_TYPE_SI,
    OrderBusinessType.BUSINESS_TYPE_AI,
  ])(
    '%s每15秒探测，隐藏停止，回焦立即检查，非当前实体不继续请求',
    async (businessType) => {
      vi.useFakeTimers();
      const view = renderPanel({ order: { ...order, businessType } });
      await advance(1);
      expect(services.get).toHaveBeenCalledTimes(1);
      await advance(14_990);
      expect(services.get).toHaveBeenCalledTimes(1);
      await advance(10);
      expect(services.get).toHaveBeenCalledTimes(2);
      vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
      act(() => document.dispatchEvent(new Event('visibilitychange')));
      await advance(45_000);
      expect(services.get).toHaveBeenCalledTimes(2);
      vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
      act(() => document.dispatchEvent(new Event('visibilitychange')));
      await advance(0);
      expect(services.get).toHaveBeenCalledTimes(3);
      view.rerender(view.ui('7', { ...order, id: 'other-order' }));
      await advance(30_000);
      expect(services.get).toHaveBeenCalledTimes(3);
      expect(
        screen.getByRole('button', { name: '维护承接与完成情况' }),
      ).toBeDisabled();
    },
  );
});
