import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query';
import { showErrorMessage } from '@/utils/appFeedback';

/**
 * 公开登录入口与各认证工作区共用客户端策略，各实例的缓存独立。
 * 业务组件通过 Provider 的 useQueryClient 取得当前工作区实例。
 *
 * 默认策略与既有手写数据层行为等价：不自动重试、不因窗口聚焦突发重查；
 * 加载/竞态/卸载保护交由库处理，禁止在调用点再写竞态令牌。
 */
export function createQueryClient(): QueryClient {
  const queryCache = new QueryCache({
    onError: (error, query) => {
      // 请求层 errorHandler 已统一弹 notification；这里的 message 仅在
      // 查询未声明静默时补充展示（历史上有 catch 文案的组件）。声明
      // meta: { silent: true } 的查询只保留请求层通知，行为对齐旧空 catch。
      if (
        queryCache.get(query.queryHash) !== query ||
        query.meta?.silent === true
      ) {
        return;
      }
      const message = query.meta?.errorMessage;
      if (typeof message === 'string') {
        showErrorMessage(message);
        return;
      }
      const detail = (error as { message?: string })?.message;
      if (detail) {
        showErrorMessage(detail);
      }
    },
  });
  const mutationCache = new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      // 工作区 clear 后，尚未结束的写请求也不得继续展示旧身份错误。
      if (
        !mutationCache.getAll().some((cached) => cached === mutation) ||
        mutation.meta?.silent === true
      ) {
        return;
      }
      const message = mutation.meta?.errorMessage;
      if (typeof message === 'string') {
        showErrorMessage(message);
        return;
      }
      const detail = (error as { message?: string })?.message;
      if (detail) {
        showErrorMessage(detail);
      }
    },
  });
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        refetchOnWindowFocus: false,
        refetchOnReconnect: false,
      },
      mutations: { retry: false },
    },
    queryCache,
    mutationCache,
  });
}

/** 只供公开登录预热与登录路由使用；业务工作区另建实例。 */
export const queryClient = createQueryClient();
