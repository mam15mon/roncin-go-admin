import { act, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { RemoteSearchSelect } from './RemoteSearchSelect';

type DeferredOption = { label: string; value: string };

/** 收敛已排队的 Promise 微任务，使挂载/联想响应的状态更新落入 act。 */
const flushAsync = () =>
  act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });

describe('RemoteSearchSelect', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('挂载时以空关键字加载首屏候选并渲染选项', async () => {
    const request = vi.fn().mockResolvedValue([
      { label: '宁波中远海运 (COSCO)', value: 'p-1' },
      { label: '上海美森轮船 (MATSON)', value: 'p-2' },
    ]);
    render(<RemoteSearchSelect request={request} />);
    await flushAsync();

    expect(request).toHaveBeenCalledTimes(1);
    expect(request).toHaveBeenCalledWith(undefined);

    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(screen.getByText('宁波中远海运 (COSCO)')).toBeInTheDocument();
    expect(screen.getByText('上海美森轮船 (MATSON)')).toBeInTheDocument();
  });

  it('输入关键字按防抖触发远程联想', async () => {
    vi.useFakeTimers();
    const request = vi.fn().mockResolvedValue([]);
    render(<RemoteSearchSelect request={request} />);
    await flushAsync();
    expect(request).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '美森' },
    });
    // 300ms 防抖窗口内不触发
    act(() => {
      vi.advanceTimersByTime(150);
    });
    expect(request).toHaveBeenCalledTimes(1);

    // 越过防抖窗口后触发联想并收敛响应
    act(() => {
      vi.advanceTimersByTime(300);
    });
    await flushAsync();
    expect(request).toHaveBeenCalledTimes(2);
    expect(request).toHaveBeenLastCalledWith('美森');
  });

  it('竞态防护：晚到的旧响应不覆盖新关键字的结果', async () => {
    vi.useFakeTimers();
    const resolvers: Array<(v: DeferredOption[]) => void> = [];
    const request = vi.fn().mockImplementation(
      () =>
        new Promise<DeferredOption[]>((resolve) => {
          resolvers.push(resolve);
        }),
    );
    render(<RemoteSearchSelect request={request} debounceMs={0} />);
    await flushAsync();
    expect(request).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '美森' },
    });
    act(() => {
      vi.advanceTimersByTime(0);
    });
    await flushAsync();
    expect(request).toHaveBeenCalledTimes(2);
    expect(request).toHaveBeenLastCalledWith('美森');

    fireEvent.mouseDown(screen.getByRole('combobox'));
    // 第 2 次联想请求先返回
    resolvers[1]([{ label: '新候选', value: 'new' }]);
    await flushAsync();
    expect(screen.getByText('新候选')).toBeInTheDocument();

    // 旧的首屏响应此时才返回，不允许把联想结果冲掉
    resolvers[0]([{ label: '旧候选', value: 'old' }]);
    await flushAsync();
    expect(screen.getByText('新候选')).toBeInTheDocument();
    expect(screen.queryByText('旧候选')).not.toBeInTheDocument();
  });

  it('请求失败时降级为空候选并记录错误，不阻塞后续联想', async () => {
    vi.useFakeTimers();
    const errorSpy = vi
      .spyOn(console, 'error')
      .mockImplementation(() => undefined);
    const request = vi
      .fn()
      .mockRejectedValueOnce(new Error('网络异常'))
      .mockResolvedValue([{ label: '恢复候选', value: 'ok' }]);
    render(<RemoteSearchSelect request={request} debounceMs={0} />);

    // 首屏加载失败：降级空候选并记录错误
    await flushAsync();
    expect(errorSpy).toHaveBeenCalledWith(
      'RemoteSearchSelect 候选项加载失败',
      expect.any(Error),
    );

    fireEvent.change(screen.getByRole('combobox'), { target: { value: '恢' } });
    act(() => {
      vi.advanceTimersByTime(0);
    });
    await flushAsync();
    expect(request).toHaveBeenCalledTimes(2);

    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(screen.getByText('恢复候选')).toBeInTheDocument();
    errorSpy.mockRestore();
  });
});
