import { useCallback, useEffect, useMemo, useRef } from 'react';

export type AsyncRunContext = {
  signal: AbortSignal;
};

export type GuardedResult =
  | { current: true }
  | { current: false; reason: 'stale' | 'unmounted' | 'aborted' };

type AsyncRequest<T> = (context: AsyncRunContext) => Promise<T>;
type AsyncApply<T, TResult> = (value: T) => TResult;
type AsyncApplyMustBeSync<TResult> = [TResult] extends [never]
  ? []
  : TResult extends PromiseLike<unknown>
    ? [message: never]
    : [];

type GuardedAsync = {
  run<T, TResult>(
    request: AsyncRequest<T>,
    apply: AsyncApply<T, TResult>,
    ...asyncApplyError: AsyncApplyMustBeSync<TResult>
  ): Promise<GuardedResult>;
  invalidate(): void;
};

/**
 * 为查询和写入共用请求生命周期：失效或卸载后不再执行 UI 副作用。
 *
 * latest 模式额外保证同一 Hook 中只有最后一次请求可以应用结果；普通模式
 * 保留并发 mutation 的语义，仅在显式失效或卸载后拦截回填。
 */
function useGuardedAsync(latest: boolean): GuardedAsync {
  const mountedRef = useRef(false);
  const generationRef = useRef(0);
  const latestSequenceRef = useRef(0);
  const controllersRef = useRef(new Set<AbortController>());

  const abortAll = useCallback(() => {
    for (const controller of controllersRef.current) {
      controller.abort();
    }
    controllersRef.current.clear();
  }, []);

  const invalidate = useCallback(() => {
    generationRef.current += 1;
    abortAll();
  }, [abortAll]);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      invalidate();
    };
  }, [invalidate]);

  const run = useCallback(
    async <T, TResult>(
      request: AsyncRequest<T>,
      apply: AsyncApply<T, TResult>,
      ..._asyncApplyError: AsyncApplyMustBeSync<TResult>
    ): Promise<GuardedResult> => {
      if (latest) {
        abortAll();
      }

      const generation = generationRef.current;
      const sequence = latest ? ++latestSequenceRef.current : 0;
      const controller = new AbortController();
      controllersRef.current.add(controller);

      const getInactiveResult = (): GuardedResult | undefined => {
        if (!mountedRef.current) {
          return { current: false, reason: 'unmounted' };
        }
        if (generation !== generationRef.current) {
          return { current: false, reason: 'aborted' };
        }
        if (latest && sequence !== latestSequenceRef.current) {
          return { current: false, reason: 'stale' };
        }
        if (controller.signal.aborted) {
          return { current: false, reason: 'aborted' };
        }
        return undefined;
      };

      try {
        const value = await request({ signal: controller.signal });
        const inactive = getInactiveResult();
        if (inactive) return inactive;

        // apply 只允许同步副作用；后续异步操作必须再次进入 guarded run。
        apply(value);
        return { current: true };
      } catch (error) {
        const inactive = getInactiveResult();
        if (inactive) return inactive;
        // 当前请求的业务错误必须原样交给调用方处理。
        throw error;
      } finally {
        controllersRef.current.delete(controller);
      }
    },
    [abortAll, latest],
  );

  return useMemo(() => ({ run, invalidate }), [invalidate, run]);
}

/** 搜索、联想等 latest-wins 查询。 */
export function useLatestAsync(): GuardedAsync & { cancel(): void } {
  const guarded = useGuardedAsync(true);
  return useMemo(() => ({ ...guarded, cancel: guarded.invalidate }), [guarded]);
}

/** 保存、创建等 mutation：仅在失效或卸载后禁止后续 UI 副作用。 */
export function useAsyncGuard(): GuardedAsync {
  return useGuardedAsync(false);
}
