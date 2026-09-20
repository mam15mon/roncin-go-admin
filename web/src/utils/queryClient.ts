import {
  MutationCache,
  QueryCache,
  QueryClient,
} from '@tanstack/react-query';
import { showErrorMessage } from '@/utils/appFeedback';

/**
 * 数据层唯一 QueryClient：全项目服务端状态统一经 React Query 管理
 * （规范见 .trellis/spec/web/frontend/state-management.md）。
 *
 * 默认策略与既有手写数据层行为等价：不自动重试、不因窗口聚焦突发重查；
 * 加载/竞态/卸载保护交由库处理，禁止在调用点再写竞态令牌。
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: false,
    },
  },
  queryCache: new QueryCache({
    onError: (error, query) => {
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
  }),
  mutationCache: new MutationCache({
    onError: (error, variables, _context, mutation) => {
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
  }),
});
