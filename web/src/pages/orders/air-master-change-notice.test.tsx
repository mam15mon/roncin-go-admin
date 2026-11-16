import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { useAirMasterChangeNotice } from './air-master-change-notice';

const getMaster = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: getMaster,
}));

const baseline: API.GetAirMasterWaybillResponse = {
  data: {
    id: 'master',
    organizationId: 'org',
    businessType: 'AE',
    version: '1',
  },
};

function setup() {
  const queryClient = createTestQueryClient();
  return renderHook(
    (props: {
      enabled: boolean;
      session: number;
      baseline: API.GetAirMasterWaybillResponse;
    }) =>
      useAirMasterChangeNotice({
        ...props,
        organizationId: 'org',
        businessType: OrderBusinessType.BUSINESS_TYPE_AE,
      }),
    {
      initialProps: { enabled: true, session: 1, baseline },
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    },
  );
}

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1);
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  getMaster.mockReset().mockResolvedValue(baseline);
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('航空主单前台变动检查', () => {
  it('15秒前台检查，隐藏暂停，恢复立即检查，失去当前页资格后不再请求', async () => {
    const view = setup();
    await advance(14_998);
    expect(getMaster).not.toHaveBeenCalled();
    await advance(1);
    expect(getMaster).toHaveBeenCalledTimes(1);
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await advance(45_000);
    expect(getMaster).toHaveBeenCalledTimes(1);
    getMaster.mockResolvedValue({
      data: {
        id: 'master',
        organizationId: 'org',
        businessType: 'AE',
        version: '2',
      },
    });
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await advance(0);
    expect(getMaster).toHaveBeenCalledTimes(2);
    expect(view.result.current).toBe(true);
    view.rerender({ enabled: false, session: 1, baseline });
    act(() => window.dispatchEvent(new Event('blur')));
    act(() => window.dispatchEvent(new Event('focus')));
    await advance(30_000);
    expect(getMaster).toHaveBeenCalledTimes(2);
  });

  it('检查失败仍保留提醒，仅新编辑基线成功采纳才清除', async () => {
    const view = setup();
    getMaster.mockResolvedValue({
      data: {
        id: 'master',
        organizationId: 'org',
        businessType: 'AE',
        version: '2',
      },
    });
    await advance(15_000);
    expect(view.result.current).toBe(true);
    getMaster.mockResolvedValue(undefined);
    await advance(15_000);
    expect(view.result.current).toBe(true);
    view.rerender({
      enabled: true,
      session: 2,
      baseline: {
        data: {
          id: 'master',
          organizationId: 'org',
          businessType: 'AE',
          version: '2',
        },
      },
    });
    expect(view.result.current).toBe(false);
  });
  it('本人写入在途时暂停探测，写后 invalidate 与迟到探测读到本人版本都不误报', async () => {
    const queryClient = createTestQueryClient();
    const v2: API.GetAirMasterWaybillResponse = {
      data: { ...baseline.data, version: '2' },
    };
    let finishProbe!: (value: API.GetAirMasterWaybillResponse) => void;
    const view = renderHook(
      (props: { ownWritePending: boolean }) =>
        useAirMasterChangeNotice({
          baseline,
          session: 1,
          organizationId: 'org',
          businessType: OrderBusinessType.BUSINESS_TYPE_AE,
          enabled: true,
          ...props,
        }),
      {
        initialProps: { ownWritePending: false },
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={queryClient}>
            {children}
          </QueryClientProvider>
        ),
      },
    );
    getMaster.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishProbe = resolve;
        }),
    );
    await advance(15_000);
    expect(getMaster).toHaveBeenCalledTimes(1);
    view.rerender({ ownWritePending: true });
    await act(async () => finishProbe(v2));
    await advance(0);
    expect(view.result.current).toBe(false);
    getMaster.mockResolvedValue(v2);
    await act(async () => {
      await queryClient.invalidateQueries({ queryKey: ['air-orders'] });
    });
    await advance(30_000);
    expect(getMaster).toHaveBeenCalledTimes(1);
    expect(view.result.current).toBe(false);
  });

  it('AI公司A→B→A往返的旧探测迟到不能影响新会话，也不读取AE探测缓存', async () => {
    const queryClient = createTestQueryClient();
    const snapshot = (
      organizationId = 'org-a',
      version = '1',
    ): API.GetAirMasterWaybillResponse => ({
      data: { id: 'master', organizationId, businessType: 'AI', version },
    });
    let finishOld!: (value: API.GetAirMasterWaybillResponse) => void;
    getMaster.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishOld = resolve;
        }),
    );
    const view = renderHook(
      (props: {
        organizationId: string;
        baseline: API.GetAirMasterWaybillResponse;
      }) =>
        useAirMasterChangeNotice({
          ...props,
          session: 1,
          businessType: OrderBusinessType.BUSINESS_TYPE_AI,
          enabled: true,
        }),
      {
        initialProps: { organizationId: 'org-a', baseline: snapshot() },
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={queryClient}>
            {children}
          </QueryClientProvider>
        ),
      },
    );
    await advance(15_000);
    expect(getMaster).toHaveBeenCalledTimes(1);
    view.rerender({ organizationId: 'org-b', baseline: snapshot('org-b') });
    view.rerender({ organizationId: 'org-a', baseline: snapshot() });
    await act(async () => finishOld(snapshot('org-a', '2')));
    await advance(0);
    expect(view.result.current).toBe(false);
    getMaster.mockResolvedValue(snapshot('org-a', '3'));
    await advance(15_000);
    expect(view.result.current).toBe(true);
    getMaster.mockResolvedValue({
      data: {
        id: 'master',
        organizationId: 'org-a',
        businessType: 'AE',
        version: '4',
      },
    });
    await advance(15_000);
    expect(view.result.current).toBe(true);
    expect(
      queryClient
        .getQueryCache()
        .findAll({ queryKey: ['air-orders', 'master-change-notice'] })
        .every(
          (query) =>
            query.queryKey[2] &&
            typeof query.queryKey[2] === 'object' &&
            'businessType' in query.queryKey[2] &&
            query.queryKey[2].businessType ===
              OrderBusinessType.BUSINESS_TYPE_AI,
        ),
    ).toBe(true);
  });
});
