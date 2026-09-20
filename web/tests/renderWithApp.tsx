import type { QueryClient } from '@tanstack/react-query';
import { QueryClientProvider } from '@tanstack/react-query';
import type { RenderOptions, RenderResult } from '@testing-library/react';
import { render } from '@testing-library/react';
import { App as AntdApp, ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import type { ReactElement, ReactNode } from 'react';
import { useMemo, useState } from 'react';
import { HelmetProvider } from 'react-helmet-async';
import type { MemoryRouterProps } from 'react-router';
import { MemoryRouter } from 'react-router';
import type { InitialState } from '@/app/AppProvider';
import { appInitialStateContext } from '@/app/AppProvider';
import { themeConfig } from '../config/theme';
import { createTestQueryClient } from './queryClientTestUtils';

export interface RenderWithAppOptions extends Omit<RenderOptions, 'wrapper'> {
  user?: API.CurrentUser;
  routerProps?: MemoryRouterProps;
}

/**
 * 全应用上下文测试包装：MemoryRouter + HelmetProvider + ConfigProvider +
 * antd App + React Query + 初始状态（可注入 currentUser）。
 * 替代历史上对 @umijs/max 的 vi.mock 样板；仅注入用户态仍不满足用例时，
 * 再按文件 vi.mock('@/app/access') 等新模块源。
 */
export function renderWithApp(
  ui: ReactElement,
  options: RenderWithAppOptions = {},
): RenderResult & { queryClient: QueryClient } {
  const { user, routerProps, ...renderOptions } = options;
  const queryClient = createTestQueryClient();

  function TestApp({ children }: { children: ReactNode }) {
    const [initialState, setInitialState] = useState<InitialState | undefined>(
      () => (user ? { currentUser: user } : undefined),
    );
    const model = useMemo(
      () => ({
        initialState,
        setInitialState,
        refresh: () => {},
        loading: false,
      }),
      [initialState],
    );
    return (
      <HelmetProvider>
        <ConfigProvider locale={zhCN} theme={themeConfig}>
          <AntdApp>
            <QueryClientProvider client={queryClient}>
              <appInitialStateContext.Provider value={model}>
                <MemoryRouter {...routerProps}>{children}</MemoryRouter>
              </appInitialStateContext.Provider>
            </QueryClientProvider>
          </AntdApp>
        </ConfigProvider>
      </HelmetProvider>
    );
  }

  const result = render(ui, { wrapper: TestApp, ...renderOptions });
  return { ...result, queryClient };
}
