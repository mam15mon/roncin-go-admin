import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import {
  keepPreviousData,
  type QueryClient,
  QueryClientProvider,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { StrictMode, useLayoutEffect, useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import { createQueryClient } from '@/utils/queryClient';
import { OrganizationWorkspace } from './OrganizationWorkspace';

const { showErrorMessage } = vi.hoisted(() => ({ showErrorMessage: vi.fn() }));
vi.mock('@/utils/appFeedback', () => ({ showErrorMessage }));
vi.mock('./TagsView', () => ({ TagsView: () => null }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

function identity(
  id = 'user-a',
  organizationId = 'company-a',
): API.CurrentUser {
  return { id, currentOrganization: { id: organizationId } };
}

function QueryView({
  load,
  inspect,
}: {
  load: (page: number) => Promise<string>;
  inspect: (client: QueryClient, data: string | undefined) => void;
}) {
  const client = useQueryClient();
  const [page, setPage] = useState(1);
  const { data } = useQuery({
    queryKey: ['workbench', 'my-commissions', { page }],
    queryFn: () => load(page),
    placeholderData: keepPreviousData,
  });
  // 记录每次渲染，覆盖新身份第一次渲染而非等待 effect 清理后的状态。
  inspect(client, data);
  return (
    <>
      <output>{data ?? '正在加载'}</output>
      <button type="button" onClick={() => setPage(2)}>
        下一页
      </button>
    </>
  );
}

function mountWorkspace(
  user: API.CurrentUser | undefined,
  load: (page: number) => Promise<string>,
  inspect = vi.fn<(client: QueryClient, data: string | undefined) => void>(),
) {
  const entryClient = createTestQueryClient();
  entryClient.setQueryData(['auth', 'dingtalk-login-config'], '公开预热');
  function tree(currentUser: API.CurrentUser | undefined, visible = true) {
    return (
      <QueryClientProvider client={entryClient}>
        <appInitialStateContext.Provider
          value={{
            initialState: { currentUser },
            setInitialState: vi.fn(),
            refresh: vi.fn(),
            loading: false,
          }}
        >
          <OrganizationWorkspace>
            {visible && <QueryView load={load} inspect={inspect} />}
          </OrganizationWorkspace>
        </appInitialStateContext.Provider>
      </QueryClientProvider>
    );
  }
  const view = render(tree(user));
  return {
    ...view,
    entryClient,
    inspect,
    change: (nextUser: API.CurrentUser | undefined, visible = true) =>
      view.rerender(tree(nextUser, visible)),
  };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('认证工作区查询生命周期', () => {
  it.each([
    ['同公司换用户', identity('user-b', 'company-a')],
    ['同用户换公司', identity('user-a', 'company-b')],
  ])('%s时首个渲染即使用全新客户端及空缓存', async (_name, nextUser) => {
    const next = deferred<string>();
    const load = vi
      .fn()
      .mockResolvedValueOnce('A 的私有提成')
      .mockReturnValueOnce(next.promise);
    const view = mountWorkspace(identity(), load);
    expect(await screen.findByText('A 的私有提成')).toBeVisible();
    const oldClient = view.inspect.mock.lastCall?.[0];
    // 同时给其他业务域放入缓存，证明边界覆盖全部 key。
    for (const key of [
      ['finance', 'applications'],
      ['orders', 'detail'],
      ['workbench', 'my-receivables'],
    ]) {
      oldClient?.setQueryData(key, '旧身份业务数据');
    }
    view.inspect.mockClear();
    view.change(nextUser);
    expect(screen.queryByText('A 的私有提成')).not.toBeInTheDocument();
    const newClient = view.inspect.mock.calls[0]?.[0];
    expect(newClient).not.toBe(oldClient);
    expect(
      view.inspect.mock.calls.every(([, data]) => data === undefined),
    ).toBe(true);
    expect(oldClient?.getQueryCache().getAll()).toEqual([]);
    expect(
      newClient?.getQueryData(['finance', 'applications']),
    ).toBeUndefined();
    expect(newClient?.getQueryData(['orders', 'detail'])).toBeUndefined();
    expect(
      newClient?.getQueryData(['workbench', 'my-receivables']),
    ).toBeUndefined();
    expect(
      view.entryClient.getQueryData(['auth', 'dingtalk-login-config']),
    ).toBe('公开预热');
    await act(async () => next.resolve('B 的私有提成'));
    expect(await screen.findByText('B 的私有提成')).toBeVisible();
  });

  it('A→B→A重返时不复活旧实例，忽略不支持AbortSignal的迟到响应', async () => {
    const firstA = deferred<string>();
    const b = deferred<string>();
    const secondA = deferred<string>();
    const load = vi
      .fn()
      .mockReturnValueOnce(firstA.promise)
      .mockReturnValueOnce(b.promise)
      .mockReturnValueOnce(secondA.promise);
    const view = mountWorkspace(identity(), load);
    await waitFor(() => expect(load).toHaveBeenCalledTimes(1));
    const firstClient = view.inspect.mock.lastCall?.[0];
    view.change(identity('user-a', 'company-b'));
    await waitFor(() => expect(load).toHaveBeenCalledTimes(2));
    const bClient = view.inspect.mock.lastCall?.[0];
    view.change(identity());
    await waitFor(() => expect(load).toHaveBeenCalledTimes(3));
    const currentClient = view.inspect.mock.lastCall?.[0];
    expect(currentClient).not.toBe(firstClient);
    expect(currentClient).not.toBe(bClient);
    await act(async () => {
      firstA.resolve('旧 A 迟到结果');
      b.resolve('旧 B 迟到结果');
    });
    expect(screen.getByText('正在加载')).toBeVisible();
    expect(firstClient?.getQueryCache().getAll()).toEqual([]);
    expect(bClient?.getQueryCache().getAll()).toEqual([]);
    expect(
      currentClient?.getQueryData(['workbench', 'my-commissions', { page: 1 }]),
    ).toBeUndefined();
    await act(async () => secondA.resolve('重新登录的 A'));
    expect(await screen.findByText('重新登录的 A')).toBeVisible();
  });

  it('匿名或组织缺失时不挂查询，退出后同身份登录也重新请求', async () => {
    const reload = deferred<string>();
    const load = vi
      .fn()
      .mockResolvedValueOnce('退出前的业务数据')
      .mockReturnValueOnce(reload.promise);
    const view = mountWorkspace(identity(), load);
    expect(await screen.findByText('退出前的业务数据')).toBeVisible();
    const oldClient = view.inspect.mock.lastCall?.[0];
    view.change(undefined);
    expect(
      screen.queryByRole('button', { name: '下一页' }),
    ).not.toBeInTheDocument();
    view.change({ id: 'user-a' });
    view.change({ currentOrganization: { id: 'company-a' } });
    expect(load).toHaveBeenCalledTimes(1);
    expect(oldClient?.getQueryCache().getAll()).toEqual([]);
    view.change(identity());
    expect(screen.getByText('正在加载')).toBeVisible();
    expect(view.inspect.mock.lastCall?.[0]).not.toBe(oldClient);
    await act(async () => reload.resolve('登录后的新数据'));
    expect(await screen.findByText('登录后的新数据')).toBeVisible();
  });

  it('同身份重渲染、页签重开保持缓存，翻页保留内容直到新页完成', async () => {
    const secondPage = deferred<string>();
    const load = vi.fn((page: number) =>
      page === 1 ? Promise.resolve('第一页') : secondPage.promise,
    );
    const view = mountWorkspace(identity(), load);
    expect(await screen.findByText('第一页')).toBeVisible();
    const client = view.inspect.mock.lastCall?.[0];
    view.change(identity());
    expect(view.inspect.mock.lastCall?.[0]).toBe(client);
    expect(load).toHaveBeenCalledTimes(1);
    view.change(identity(), false);
    view.change(identity());
    expect(screen.getByText('第一页')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '下一页' }));
    expect(screen.getByText('第一页')).toBeVisible();
    await act(async () => secondPage.resolve('第二页'));
    expect(await screen.findByText('第二页')).toBeVisible();
    expect(
      client?.getQueryData(['workbench', 'my-commissions', { page: 1 }]),
    ).toBe('第一页');
  });

  it('旧查询与mutation迟到失败不触发新工作区错误提示，新工作区仍正常提示', async () => {
    const pending = deferred<string>();
    const load = vi
      .fn()
      .mockReturnValueOnce(pending.promise)
      .mockResolvedValue('B 数据');
    const view = mountWorkspace(identity(), load);
    await waitFor(() => expect(load).toHaveBeenCalledOnce());
    const oldClient = view.inspect.mock.lastCall?.[0];
    const pendingWrite = deferred<string>();
    const mutation = oldClient
      ?.getMutationCache()
      .build(oldClient, { mutationFn: () => pendingWrite.promise });
    const oldWrite = mutation?.execute(undefined).catch(() => {});
    view.change(identity('user-b'));
    await act(async () => {
      pending.reject(new Error('旧查询失败'));
      pendingWrite.reject(new Error('旧写入失败'));
      await oldWrite;
    });
    expect(await screen.findByText('B 数据')).toBeVisible();
    expect(showErrorMessage).not.toHaveBeenCalled();
    const newClient = view.inspect.mock.lastCall?.[0];
    await act(async () => {
      await newClient
        ?.fetchQuery({
          queryKey: ['finance', 'fail'],
          queryFn: () => Promise.reject(new Error('当前错误')),
        })
        .catch(() => {});
    });
    expect(showErrorMessage).toHaveBeenCalledWith('当前错误');
  });

  it('StrictMode重挂后仍能完成业务查询，最终卸载释放缓存', async () => {
    const clientSeen = vi.fn<(client: QueryClient) => void>();
    function Probe() {
      const client = useQueryClient();
      const { data } = useQuery({
        queryKey: ['strict'],
        queryFn: () => Promise.resolve('业务查询完成'),
      });
      useLayoutEffect(() => clientSeen(client), [client]);
      return <output>{data}</output>;
    }
    const view = render(
      <StrictMode>
        <appInitialStateContext.Provider
          value={{
            initialState: { currentUser: identity() },
            setInitialState: vi.fn(),
            refresh: vi.fn(),
            loading: false,
          }}
        >
          <OrganizationWorkspace>
            <Probe />
          </OrganizationWorkspace>
        </appInitialStateContext.Provider>
      </StrictMode>,
    );
    expect(await screen.findByText('业务查询完成')).toBeVisible();
    const client = clientSeen.mock.lastCall?.[0];
    expect(client?.getDefaultOptions().queries).toMatchObject({
      retry: false,
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
    });
    view.unmount();
    expect(client?.getQueryCache().getAll()).toEqual([]);
    expect(createQueryClient()).not.toBe(client);
  });
});
