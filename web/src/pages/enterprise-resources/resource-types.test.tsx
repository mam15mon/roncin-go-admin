import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { useEffect } from 'react';
import { useNavigate } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import EnterpriseResourcesPage from './index';

const state = vi.hoisted(() => ({
  list: vi.fn(),
  replace: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
}));
vi.mock('@/router/history', () => ({
  history: {
    replace: state.replace,
    location: { pathname: '/enterprise-resources/config' },
  },
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateEnterpriseResources: true,
    canUpdateEnterpriseResources: true,
    canDeleteEnterpriseResources: true,
  }),
}));
vi.mock('./useRegionOptions', () => ({
  useRegionOptions: () => ({
    provinceOptions: [],
    cityOptions: [],
    districtOptions: [],
  }),
}));
vi.mock(
  '@/services/roncin/enterpriseResourceService',
  async (importOriginal) => ({
    ...(await importOriginal<
      typeof import('@/services/roncin/enterpriseResourceService')
    >()),
    enterpriseResourceServiceGetEnterpriseResourceCapabilities: vi
      .fn()
      .mockResolvedValue({}),
    enterpriseResourceServiceListEnterpriseResources: state.list,
    enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions: vi
      .fn()
      .mockResolvedValue({ data: [] }),
    enterpriseResourceServiceUpdateEnterpriseResource: state.update,
    enterpriseResourceServiceDeleteEnterpriseResource: state.remove,
  }),
);

const resources: Record<number, API.EnterpriseResource> = {
  1: {
    id: 'address-1',
    resourceType: 1,
    shortName: 'CODEX地址验收1109',
    enabled: true,
    address: { countryCode: 'CN', addressDetail: '上海测试路1号' },
  },
  2: {
    id: 'remark-1',
    resourceType: 2,
    shortName: '本类备注',
    enabled: true,
    remark: { remarkType: 4, content: '备注正文' },
  },
  6: {
    id: 'consignee-1',
    resourceType: 6,
    shortName: '本类收货人',
    enabled: true,
  },
};
const result = (type: number) => ({
  success: true,
  data: [resources[type]],
  total: 1,
});
function RouterBridge() {
  const navigate = useNavigate();
  useEffect(() => {
    state.replace.mockImplementation((path: string) =>
      navigate(path, { replace: true }),
    );
  }, [navigate]);
  return null;
}
function setup(tab = 'addresses') {
  return renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <RouterBridge />
      <EnterpriseResourcesPage />
    </ConfigProvider>,
    {
      routerProps: {
        initialEntries: [`/enterprise-resources/config?tab=${tab}`],
      },
    },
  );
}
beforeEach(() => {
  vi.clearAllMocks();
  state.list.mockImplementation(
    async ({ resourceType }: { resourceType: number }) => result(resourceType),
  );
});

describe('企业资源类型隔离', () => {
  it('切换备注、收货人后自动查询本类，编辑目标和表头同属当前类型', async () => {
    setup();
    await screen.findByText('CODEX地址验收1109');
    for (const [tab, name, type] of [
      ['备注管理', '本类备注', 2],
      ['收货人管理', '本类收货人', 6],
    ] as const) {
      fireEvent.click(screen.getByRole('tab', { name: tab }));
      expect(screen.queryByText('CODEX地址验收1109')).not.toBeInTheDocument();
      await screen.findByText(name);
      expect(state.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ resourceType: type, page: 1 }),
      );
      const row = screen.getByText(name).closest('tr');
      if (!row) throw new Error('缺少本类资源行');
      fireEvent.click(within(row).getByRole('button', { name: /编辑/ }));
      const dialog = await screen.findByRole('dialog');
      expect(within(dialog).getByText(`编辑${tab}`)).toBeInTheDocument();
      expect(within(dialog).getByLabelText('简称/名称')).toHaveValue(name);
      fireEvent.click(within(dialog).getByRole('button', { name: /取.*消/ }));
      await waitFor(() =>
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
      );
    }
    expect(state.update).not.toHaveBeenCalled();
    expect(state.remove).not.toHaveBeenCalled();
  });

  it('旧类型请求迟到不会把地址重新填回备注列表', async () => {
    let finishAddress: (value: ReturnType<typeof result>) => void = () => {};
    state.list.mockImplementation(
      ({ resourceType }: { resourceType: number }) =>
        resourceType === 1
          ? new Promise<ReturnType<typeof result>>((resolve) => {
              finishAddress = resolve;
            })
          : Promise.resolve(result(resourceType)),
    );
    setup();
    await waitFor(() => expect(state.list).toHaveBeenCalledOnce());
    fireEvent.click(screen.getByRole('tab', { name: '备注管理' }));
    await screen.findByText('本类备注');
    await act(async () => finishAddress(result(1)));
    expect(screen.queryByText('CODEX地址验收1109')).not.toBeInTheDocument();
    expect(screen.getByText('本类备注')).toBeInTheDocument();
  });

  it('切换类型关闭旧编辑会话；新类型请求失败也不能操作旧记录', async () => {
    setup();
    await screen.findByText('CODEX地址验收1109');
    fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    state.list.mockResolvedValueOnce({ success: false, data: [], total: 0 });
    await act(async () =>
      state.replace('/enterprise-resources/config?tab=remarks'),
    );
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await waitFor(() => expect(state.list).toHaveBeenCalledTimes(2));
    expect(screen.queryByText('CODEX地址验收1109')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /编辑/ }),
    ).not.toBeInTheDocument();
    expect(state.update).not.toHaveBeenCalled();
    expect(state.remove).not.toHaveBeenCalled();
  });
  it('移除标签页签后旧链接按未知页签规则展示地址，绝不查询退役类型', async () => {
    setup('tags');
    await screen.findByText('CODEX地址验收1109');
    expect(
      screen.queryByRole('tab', { name: '标签管理' }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('标签组')).not.toBeInTheDocument();
    expect(state.list).toHaveBeenCalled();
    for (const [params] of state.list.mock.calls) {
      expect(params.resourceType).toBe(1);
    }
  });
});
