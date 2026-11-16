import { act, render, screen } from '@testing-library/react';
import {
  createMemoryRouter,
  Outlet,
  type RouteObject,
  RouterProvider,
} from 'react-router';
import { expect, it, vi } from 'vitest';
import { BOOT_LOADING_ID } from '@/app/bootLoading';
import { buildRouterConfig } from './adaptRoutes';

it.each(['/user/login', '/orders/sea-export/new'])(
  '首次直达 %s 时保持唯一启动画面，页面提交时一并移除',
  async (pathname) => {
    const warning = vi.spyOn(console, 'warn');
    let complete!: () => void;
    const pending = new Promise<void>((resolve) => {
      complete = resolve;
    });
    function deferModules(routes: RouteObject[]): RouteObject[] {
      return routes.map(
        (route) =>
          ({
            ...route,
            ...(route.lazy
              ? {
                  lazy: async () => {
                    await pending;
                    return {
                      Component: route.children
                        ? Outlet
                        : () => <p>页面已就绪</p>,
                    };
                  },
                }
              : {}),
            ...(route.children
              ? { children: deferModules(route.children) }
              : {}),
          }) as RouteObject,
      );
    }
    const router = createMemoryRouter(deferModules(buildRouterConfig()), {
      initialEntries: [pathname],
    });
    // 模拟 index.html 中位于 #root 之外的启动画面
    const bootLoading = document.createElement('div');
    bootLoading.id = BOOT_LOADING_ID;
    document.body.append(bootLoading);
    try {
      const { container } = render(<RouterProvider router={router} />);
      // 懒加载期间不渲染第二个占位，启动画面持续保留
      expect(container).toBeEmptyDOMElement();
      expect(bootLoading).toBeInTheDocument();
      await act(async () => {
        complete();
      });
      expect(await screen.findByText('页面已就绪')).toBeVisible();
      expect(bootLoading).not.toBeInTheDocument();
      expect(warning.mock.calls.flat().join('\n')).not.toContain(
        'No `HydrateFallback`',
      );
    } finally {
      bootLoading.remove();
      router.dispose();
      warning.mockRestore();
    }
  },
);
