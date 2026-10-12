import { act, render, screen } from '@testing-library/react';
import { useState } from 'react';
import {
  createMemoryRouter,
  Outlet,
  RouterProvider,
  useParams,
} from 'react-router';
import { describe, expect, it } from 'vitest';
import {
  ListQueryStateProvider,
  useClearListQueries,
  useListQueryState,
} from './index';

type Filters = { keyword?: string; enabled?: boolean; status?: number };
type Query = ReturnType<typeof useListQueryState<Filters>>;
function mount(enabled = true) {
  let query: Query | undefined;
  let clear: ReturnType<typeof useClearListQueries> | undefined;
  let setIdentity: ((identity: string) => void) | undefined;
  function List({ role }: { role: string }) {
    query = useListQueryState<Filters>({
      tabKey: '/partners',
      listId: role,
      defaultPageSize: 20,
      defaultFilters: {},
    });
    return <div>{query.initial.filters.keyword ?? '默认查询'}</div>;
  }
  function Route() {
    const { role = '' } = useParams();
    return <List key={role} role={role} />;
  }
  function Workspace() {
    clear = useClearListQueries();
    return <Outlet />;
  }
  const router = createMemoryRouter(
    [
      {
        element: <Workspace />,
        children: [
          { path: '/lists/:role', element: <Route /> },
          { path: '/away', element: <div>其它页面</div> },
        ],
      },
    ],
    { initialEntries: ['/lists/customer'] },
  );
  function IdentityWorkspace() {
    const [identity, updateIdentity] = useState('user1:org1');
    setIdentity = updateIdentity;
    return (
      <ListQueryStateProvider key={identity} enabled={enabled}>
        <RouterProvider router={router} />
      </ListQueryStateProvider>
    );
  }
  const result = render(<IdentityWorkspace />);
  return {
    ...result,
    get query() {
      if (!query) throw new Error('列表未挂载');
      return query;
    },
    clear: (keys: string[]) => clear?.(keys),
    changeIdentity: (identity: string) => {
      act(() => setIdentity?.(identity));
    },
    navigate: async (path: string) => {
      await act(async () => {
        await router.navigate(path);
      });
    },
  };
}

describe('工作区内列表查询快照', () => {
  it('真实路由卸载返回恢复已提交筛选、显示名和分页，保留false与0', async () => {
    const view = mount();
    view.query.commit(
      { keyword: '已提交', enabled: false, status: 0 },
      { owner: '负责人' },
    );
    view.query.pagination.onChange(3, 40);
    await view.navigate('/away');
    expect(screen.getByText('其它页面')).toBeInTheDocument();
    await view.navigate('/lists/customer');
    expect(screen.getByText('已提交')).toBeInTheDocument();
    expect(view.query.initial).toEqual({
      filters: { keyword: '已提交', enabled: false, status: 0 },
      labels: { owner: '负责人' },
      current: 3,
      pageSize: 40,
    });
    view.query.commit({ keyword: '新查询' });
    expect(view.query.snapshotRef.current.current).toBe(1);
    expect(view.query.snapshotRef.current.pageSize).toBe(40);
  });

  it('同页签的角色列表ID独立，返回各自恢复', async () => {
    const view = mount();
    view.query.commit({ keyword: '客户条件' });
    await view.navigate('/lists/supplier');
    expect(view.query.initial.filters).toEqual({});
    view.query.commit({ keyword: '供应商条件' });
    await view.navigate('/lists/agent');
    expect(view.query.initial.filters).toEqual({});
    view.query.commit({ keyword: '代理条件' });
    await view.navigate('/lists/customer');
    expect(screen.getByText('客户条件')).toBeInTheDocument();
    await view.navigate('/lists/supplier');
    expect(screen.getByText('供应商条件')).toBeInTheDocument();
    await view.navigate('/lists/agent');
    expect(screen.getByText('代理条件')).toBeInTheDocument();
  });

  it('关闭页签后拒绝旧实例写回，重新打开可正常记录新快照', async () => {
    const view = mount();
    view.query.commit({ keyword: '待清理' });
    const oldQuery = view.query;
    view.clear(['/partners']);
    oldQuery.commit({ keyword: '迟到写回' });
    oldQuery.pagination.onChange(9, 100);
    await view.navigate('/away');
    await view.navigate('/lists/customer');
    expect(view.query.initial.filters).toEqual({});
    expect(view.query.initial.current).toBe(1);
    view.query.commit({ keyword: '重开后新条件' });
    await view.navigate('/away');
    await view.navigate('/lists/customer');
    expect(screen.getByText('重开后新条件')).toBeInTheDocument();
  });

  it('清理其它页签不影响当前查询', async () => {
    const view = mount();
    view.query.commit({ keyword: '保留条件' });
    view.clear(['/orders']);
    await view.navigate('/away');
    await view.navigate('/lists/customer');
    expect(screen.getByText('保留条件')).toBeInTheDocument();
  });

  it('身份缺失禁用共享快照，工作区重建亦不复用旧快照', async () => {
    const disabled = mount(false);
    disabled.query.commit({ keyword: '无身份输入' });
    disabled.query.pagination.onChange(4, 100);
    await disabled.navigate('/away');
    await disabled.navigate('/lists/customer');
    expect(disabled.query.initial.filters).toEqual({});
    expect(disabled.query.initial.current).toBe(1);
    disabled.unmount();
    const firstWorkspace = mount();
    firstWorkspace.query.commit({ keyword: '旧组织条件' });
    firstWorkspace.unmount();
    const nextWorkspace = mount();
    expect(nextWorkspace.query.initial.filters).toEqual({});
  });

  it('同树内切组织及切用户的workspace key重建后查询均回默认', () => {
    const view = mount();
    view.query.commit({ keyword: '原组织已提交' });
    view.changeIdentity('user1:org2');
    expect(view.query.initial.filters).toEqual({});
    view.query.commit({ keyword: '第二组织已提交' });
    view.changeIdentity('user2:org2');
    expect(view.query.initial.filters).toEqual({});
  });
});
