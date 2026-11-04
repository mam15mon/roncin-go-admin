import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { seaDocumentServiceGetSeaOrderDocuments } from '@/services/roncin/seaDocumentService';
import type { HblBundle } from './model';
import { fetchHblBundle, useHblChangeNotice } from './use-hbl-data';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: vi.fn(),
}));
vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceGetSeaOrderDocuments: vi.fn(),
}));
const getOrder = vi.mocked(orderServiceGetOrder);
const getDocuments = vi.mocked(seaDocumentServiceGetSeaOrderDocuments);

function bundle(id = 'order-1'): HblBundle {
  return {
    order: {
      id,
      version: '1',
      seaMasterBill: {
        masterBillId: 'mbl-1',
        transportExecutionId: 'te-1',
        transportExecutionVersion: '2',
      },
    },
    documents: {
      orderId: id,
      linkVersion: '3',
      houseBill: { id: 'house-1', version: '4' },
    },
  };
}

function mount(baseline = bundle()) {
  const client = createTestQueryClient();
  return renderHook(
    ({ snapshot, paused }) => useHblChangeNotice(snapshot, 'user:org', paused),
    {
      initialProps: { snapshot: baseline, paused: false },
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
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
  vi.clearAllMocks();
  vi.useFakeTimers();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  getOrder.mockResolvedValue({ success: true, data: bundle().order });
  getDocuments.mockResolvedValue({ success: true, data: bundle().documents });
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('HBL 前台变动检查', () => {
  it('初始快照不重复读取，15秒后发现HBL版本变化且不推进基线，失败保留提醒', async () => {
    const initial = bundle();
    const { result } = mount(initial);
    await advance(14_998);
    expect(getDocuments).not.toHaveBeenCalled();
    getDocuments.mockResolvedValue({
      success: true,
      data: {
        ...initial.documents,
        houseBill: { id: 'house-1', version: '5' },
      },
    });
    await advance(1);
    expect(getDocuments).toHaveBeenCalledOnce();
    expect(result.current.changed).toBe(true);
    expect(initial.documents.houseBill?.version).toBe('4');
    getDocuments.mockRejectedValueOnce(new Error('暂时失败'));
    await advance(15_000);
    expect(result.current.changed).toBe(true);
  });

  it('隐藏、失焦和保存中暂停，返回前台立即检查，卸载停止请求', async () => {
    const initial = bundle();
    const { rerender, unmount } = mount(initial);
    const hidden = vi.spyOn(document, 'visibilityState', 'get');
    hidden.mockReturnValue('hidden');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await advance(30_000);
    expect(getOrder).not.toHaveBeenCalled();
    hidden.mockReturnValue('visible');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await advance(0);
    expect(getOrder).toHaveBeenCalledOnce();
    act(() => window.dispatchEvent(new Event('blur')));
    await advance(30_000);
    expect(getOrder).toHaveBeenCalledOnce();
    act(() => window.dispatchEvent(new Event('focus')));
    await advance(0);
    expect(getOrder).toHaveBeenCalledTimes(2);
    rerender({ snapshot: initial, paused: true });
    await advance(30_000);
    expect(getOrder).toHaveBeenCalledTimes(2);
    unmount();
    await advance(30_000);
    expect(getOrder).toHaveBeenCalledTimes(2);
  });

  it('共享航程单独更新也提醒，本人保存接纳新快照后不误报', async () => {
    const next = {
      ...bundle(),
      order: {
        ...bundle().order,
        seaMasterBill: {
          ...bundle().order.seaMasterBill,
          transportExecutionVersion: '6',
        },
      },
    };
    getOrder.mockResolvedValue({ success: true, data: next.order });
    const { result, rerender } = mount();
    await advance(15_000);
    expect(result.current.changed).toBe(true);
    rerender({ snapshot: next, paused: false });
    await advance(15_000);
    expect(result.current.changed).toBe(false);
  });

  it('生成前复核失败可重试，发现关联版本变化拒绝旧基线输出', async () => {
    const { result } = mount();
    getDocuments.mockRejectedValueOnce(new Error('网络错误'));
    await expect(result.current.verifyCurrent()).rejects.toThrow(
      'HBL 资料检查失败',
    );
    getDocuments.mockResolvedValue({
      success: true,
      data: { ...bundle().documents, linkVersion: '8' },
    });
    await act(async () =>
      expect(await result.current.verifyCurrent()).toBe(false),
    );
    await advance(0);
    expect(result.current.changed).toBe(true);
  });

  it('跨订单响应和失败响应不能当成当前制作资料', async () => {
    getOrder.mockResolvedValueOnce({
      success: true,
      data: bundle('other').order,
    });
    await expect(fetchHblBundle('order-1')).rejects.toThrow('读取失败');
    getDocuments.mockResolvedValueOnce({ success: false });
    await expect(fetchHblBundle('order-1')).rejects.toThrow('读取失败');
  });

  it('公司英文名更新不改变HBL版本时，15秒检查仍发现变化且保留基线', async () => {
    const initial = bundle();
    initial.documents.houseBill = {
      ...initial.documents.houseBill,
      issuerOrganizationId: 'issuer-1',
      issuerOrganizationNameEn: 'Original Confirmed Company Co., Ltd',
      printTemplate: 'RONCIN_SELF',
    };
    const { result } = mount(initial);
    getDocuments.mockResolvedValue({
      success: true,
      data: {
        ...initial.documents,
        houseBill: {
          ...initial.documents.houseBill,
          issuerOrganizationNameEn: 'Updated Confirmed Company Co., Ltd',
        },
      },
    });
    await advance(15_000);
    expect(result.current.changed).toBe(true);
    expect(initial.documents.houseBill.issuerOrganizationNameEn).toBe(
      'Original Confirmed Company Co., Ltd',
    );
    expect(initial.documents.houseBill.version).toBe('4');
  });
});
