import { act, render, screen } from '@testing-library/react';
import {
  createMemoryRouter,
  Outlet,
  type RouteObject,
  RouterProvider,
} from 'react-router';
import { expect, it, vi } from 'vitest';
import { buildRouterConfig } from './adaptRoutes';

it.each(['/user/login', '/orders/sea-export/new'])(
  '首次直达 %s 时展示路由加载占位，模块就绪后展示页面',
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
    try {
      render(<RouterProvider router={router} />);
      expect(
        screen.getByRole('status', { name: '正在加载页面' }),
      ).toBeVisible();
      expect(screen.queryByText('页面已就绪')).not.toBeInTheDocument();
      await act(async () => {
        complete();
      });
      expect(await screen.findByText('页面已就绪')).toBeVisible();
      expect(
        screen.queryByRole('status', { name: '正在加载页面' }),
      ).not.toBeInTheDocument();
      expect(warning.mock.calls.flat().join('\n')).not.toContain(
        'No `HydrateFallback`',
      );
    } finally {
      router.dispose();
      warning.mockRestore();
    }
  },
);
