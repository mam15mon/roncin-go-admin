import { act, renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { useAsyncGuard, useLatestAsync } from './useLatestAsync';

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

describe('useLatestAsync', () => {
  it('只应用最后一次查询结果，并将旧结果标记为 stale', async () => {
    const first = deferred<string>();
    const second = deferred<string>();
    const applied: string[] = [];
    const { result } = renderHook(() => useLatestAsync());

    let firstRun!: Promise<unknown>;
    let secondRun!: Promise<unknown>;
    await act(async () => {
      firstRun = result.current.run(
        () => first.promise,
        (value) => {
          applied.push(value);
        },
      );
      secondRun = result.current.run(
        () => second.promise,
        (value) => {
          applied.push(value);
        },
      );
    });

    await act(async () => {
      second.resolve('最新结果');
    });
    await act(async () => {
      first.resolve('旧结果');
    });

    await expect(secondRun).resolves.toEqual({ current: true });
    await expect(firstRun).resolves.toEqual({
      current: false,
      reason: 'stale',
    });
    expect(applied).toEqual(['最新结果']);
  });

  it('卸载后即使不可取消的请求完成也不会 apply', async () => {
    const request = deferred<string>();
    const apply = () => {
      throw new Error('卸载后不应执行 apply');
    };
    const { result, unmount } = renderHook(() => useAsyncGuard());
    const pending = result.current.run(() => request.promise, apply);

    unmount();
    await act(async () => {
      request.resolve('迟到结果');
    });

    await expect(pending).resolves.toEqual({
      current: false,
      reason: 'unmounted',
    });
  });

  it('显式取消会 abort 当前请求并阻止 apply', async () => {
    const request = deferred<string>();
    const applied: string[] = [];
    const { result } = renderHook(() => useLatestAsync());
    let receivedSignal: AbortSignal | undefined;

    const pending = result.current.run(
      ({ signal }) => {
        receivedSignal = signal;
        return request.promise;
      },
      (value) => applied.push(value),
    );
    act(() => result.current.cancel());
    await act(async () => {
      request.resolve('已取消结果');
    });

    expect(receivedSignal?.aborted).toBe(true);
    await expect(pending).resolves.toEqual({
      current: false,
      reason: 'aborted',
    });
    expect(applied).toEqual([]);
  });

  it('普通 guard 的 invalidate 不让未完成 mutation 回填', async () => {
    const request = deferred<string>();
    const applied: string[] = [];
    const { result } = renderHook(() => useAsyncGuard());
    const pending = result.current.run(
      () => request.promise,
      (value) => {
        applied.push(value);
      },
    );

    act(() => result.current.invalidate());
    await act(async () => {
      request.resolve('旧 mutation 结果');
    });

    await expect(pending).resolves.toEqual({
      current: false,
      reason: 'aborted',
    });
    expect(applied).toEqual([]);
  });

  it('当前请求的业务错误按原对象抛出', async () => {
    const error = new Error('权限不足');
    const { result } = renderHook(() => useAsyncGuard());

    await expect(
      result.current.run(
        async () => {
          throw error;
        },
        () => undefined,
      ),
    ).rejects.toBe(error);
  });

  it('已被后续查询取代的请求即使报错也只返回 stale', async () => {
    const first = deferred<string>();
    const second = deferred<string>();
    const { result } = renderHook(() => useLatestAsync());

    const firstRun = result.current.run(
      () => first.promise,
      () => undefined,
    );
    const secondRun = result.current.run(
      () => second.promise,
      () => undefined,
    );

    await act(async () => {
      second.resolve('最新结果');
    });
    await act(async () => {
      first.reject(new Error('旧请求失败'));
    });

    await expect(secondRun).resolves.toEqual({ current: true });
    await expect(firstRun).resolves.toEqual({
      current: false,
      reason: 'stale',
    });
  });
});
