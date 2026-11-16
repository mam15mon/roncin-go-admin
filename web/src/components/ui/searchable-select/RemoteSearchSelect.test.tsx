import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import React, { type ReactElement, type ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { RemoteSearchSelect } from './RemoteSearchSelect';

type Option = { label: string; value: string };
const key = ['select', 'candidates'] as const;
function mount(ui: ReactElement) {
  const client = createTestQueryClient();
  return {
    ...render(ui, {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    }),
    client,
  };
}
async function advance(ms = 0) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
  // 防抖更新提交后，Query 的观察者通知在下一轮任务队列中采纳。
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe('RemoteSearchSelect 的资源查询', () => {
  it('挂载以空关键词加载首批候选', async () => {
    const request = vi
      .fn()
      .mockResolvedValue([{ label: '宁波中远海运', value: 'p-1' }]);
    mount(<RemoteSearchSelect queryKey={key} request={request} />);
    await advance();
    expect(request).toHaveBeenCalledExactlyOnceWith(undefined);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(screen.getByText('宁波中远海运')).toBeInTheDocument();
  });

  it('只将防抖后的规范化关键词放入查询键', async () => {
    const request = vi.fn().mockResolvedValue([]);
    const { client } = mount(
      <RemoteSearchSelect queryKey={key} request={request} />,
    );
    await advance();
    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '  美森  ' },
    });
    await advance(150);
    expect(request).toHaveBeenCalledOnce();
    await advance(150);
    expect(request).toHaveBeenLastCalledWith('美森');
    expect(client.getQueryState([...key, { keyword: '美森' }])?.status).toBe(
      'success',
    );
    expect(
      client.getQueryState([...key, { keyword: '  美森  ' }]),
    ).toBeUndefined();
  });

  it('旧关键词迟到不会覆盖当前候选', async () => {
    const resolvers: Array<(items: Option[]) => void> = [];
    const request = vi.fn(
      () => new Promise<Option[]>((resolve) => resolvers.push(resolve)),
    );
    mount(
      <RemoteSearchSelect queryKey={key} request={request} debounceMs={0} />,
    );
    await advance();
    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '美森' },
    });
    await advance();
    expect(request).toHaveBeenCalledTimes(2);
    await act(async () => resolvers[1]([{ label: '新候选', value: 'new' }]));
    await advance();
    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(screen.getByText('新候选')).toBeInTheDocument();
    await act(async () => resolvers[0]([{ label: '旧候选', value: 'old' }]));
    await advance();
    expect(screen.getByText('新候选')).toBeInTheDocument();
    expect(screen.queryByText('旧候选')).toBeNull();
  });

  it('同一挂载切换父级后立即隔离旧候选和迟到响应', async () => {
    let resolveOld!: (items: Option[]) => void;
    const oldRequest = vi.fn(
      () =>
        new Promise<Option[]>((resolve) => {
          resolveOld = resolve;
        }),
    );
    const newRequest = vi
      .fn()
      .mockResolvedValue([{ label: 'B候选', value: 'b' }]);
    const view = mount(
      <RemoteSearchSelect
        queryKey={['select', { parent: 'a' }]}
        request={oldRequest}
      />,
    );
    await advance();
    view.rerender(
      <RemoteSearchSelect
        queryKey={['select', { parent: 'b' }]}
        request={newRequest}
      />,
    );
    await advance();
    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(screen.getByText('B候选')).toBeInTheDocument();
    await act(async () => resolveOld([{ label: 'A候选', value: 'a' }]));
    await advance();
    expect(screen.queryByText('A候选')).toBeNull();
  });

  it('回调引用变化不自行重查，新关键词使用最新回调', async () => {
    const first = vi.fn().mockResolvedValue([]);
    const second = vi.fn().mockResolvedValue([]);
    const view = mount(
      <RemoteSearchSelect queryKey={key} request={first} debounceMs={0} />,
    );
    await advance();
    view.rerender(
      <RemoteSearchSelect queryKey={key} request={second} debounceMs={0} />,
    );
    await advance();
    expect(first).toHaveBeenCalledOnce();
    expect(second).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '新词' },
    });
    await advance();
    expect(second).toHaveBeenCalledExactlyOnceWith('新词');
  });

  it('失败保留已选名称，后续搜索正常恢复且错误留在Query状态', async () => {
    const request = vi
      .fn()
      .mockRejectedValueOnce(new Error('网络异常'))
      .mockResolvedValue([{ label: '恢复候选', value: 'ok' }]);
    const { client } = mount(
      <RemoteSearchSelect
        queryKey={key}
        request={request}
        debounceMs={0}
        value="chosen"
        selectedOptions={[{ label: '已选名称', value: 'chosen' }]}
      />,
    );
    await advance();
    expect(
      client.getQueryState([...key, { keyword: '' }])?.error?.message,
    ).toBe('网络异常');
    expect(screen.getAllByText('已选名称').length).toBeGreaterThan(0);
    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '恢复' },
    });
    await advance();
    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(screen.getByText('恢复候选')).toBeInTheDocument();
    expect(screen.getAllByText('已选名称').length).toBeGreaterThan(0);
  });
});
