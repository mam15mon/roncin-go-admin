import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// 组织身份与权限判定统一走 @/app/access 的 useAccess（access.ts 契约），
// 测试通过 hoisted 可变对象切换总部 / 非总部视角。
const accessRef = vi.hoisted(() => ({
  current: {} as Record<string, boolean>,
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessRef.current,
}));

const mockListItems = vi.hoisted(() => vi.fn());
const mockListPorts = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListItems: mockListItems,
  masterDataServiceCreateItem: vi.fn(),
  masterDataServiceUpdateItem: vi.fn(),
  masterDataServiceListPorts: mockListPorts,
  masterDataServiceCreatePort: vi.fn(),
  masterDataServiceUpdatePort: vi.fn(),
}));

import CountriesPanel from './CountriesPanel';
import PortsPanel from './PortsPanel';

const countryItems = [
  {
    id: 'country-1',
    code: 'CN',
    name: '中国',
    nameEn: 'China',
    enabled: true,
    source: 'seed',
    sortOrder: 100,
    attributes: { continent: '亚洲', currencyCode: 'CNY' },
  },
  {
    id: 'country-2',
    code: 'US',
    name: '美国',
    nameEn: 'United States',
    enabled: true,
    source: 'seed',
    sortOrder: 100,
    attributes: { continent: '北美洲', currencyCode: 'USD' },
  },
];

const portItems = [
  {
    // 集团基线行：organizationId 为空
    id: 'port-1',
    organizationId: undefined,
    unLocode: 'CNSHG',
    nameZh: '上海港',
    nameEn: 'SHANGHAI',
    countryCode: 'CN',
    transportModes: ['SEA'],
    enabled: true,
    source: 'seed',
    sortOrder: 100,
  },
  {
    // 本组织本地补充行
    id: 'port-2',
    organizationId: 'org-branch',
    unLocode: 'CNZJG',
    nameZh: '张家港港',
    nameEn: 'ZHANGJIAGANG',
    countryCode: 'CN',
    transportModes: ['SEA'],
    enabled: true,
    source: 'manual',
    sortOrder: 100,
  },
];

function renderPanel(ui: React.ReactElement) {
  // 面板数据层已迁移 React Query：每用例独立 QueryClient，防止缓存串味。
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <App>{ui}</App>
    </QueryClientProvider>,
  );
}

describe('主数据页签组织身份收敛（A 型只读 / B 型基线+本地）', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListItems.mockResolvedValue({ data: countryItems, total: 2 });
    mockListPorts.mockResolvedValue({ data: portItems, total: 2 });
  });

  afterEach(() => {
    cleanup();
  });

  it('A 型页签：非总部组织显示总部维护横幅并隐藏全部写按钮', async () => {
    accessRef.current = {
      isHeadquartersOrganization: false,
      canCreateMasterDataItems: true,
      canUpdateMasterDataItems: true,
    };
    renderPanel(<CountriesPanel />);

    await waitFor(() => expect(screen.getByText('CN')).toBeInTheDocument());
    expect(screen.getByText('由总部统一维护与共享')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /新增国家与地区/ })).toBeNull();
    expect(screen.queryByText('编辑')).toBeNull();
  });

  it('A 型页签：总部组织且持有写权限时入口正常且无横幅', async () => {
    accessRef.current = {
      isHeadquartersOrganization: true,
      canCreateMasterDataItems: true,
      canUpdateMasterDataItems: true,
    };
    renderPanel(<CountriesPanel />);

    await waitFor(() => expect(screen.getByText('CN')).toBeInTheDocument());
    expect(screen.queryByText('由总部统一维护与共享')).toBeNull();
    expect(
      screen.getByRole('button', { name: /新增国家与地区/ }),
    ).toBeInTheDocument();
    expect(screen.getAllByText('编辑')).toHaveLength(countryItems.length);
  });

  it('A 型页签：总部组织但无写权限码时同样隐藏写按钮', async () => {
    accessRef.current = {
      isHeadquartersOrganization: true,
      canCreateMasterDataItems: false,
      canUpdateMasterDataItems: false,
    };
    renderPanel(<CountriesPanel />);

    await waitFor(() => expect(screen.getByText('CN')).toBeInTheDocument());
    expect(screen.queryByRole('button', { name: /新增国家与地区/ })).toBeNull();
    expect(screen.queryByText('编辑')).toBeNull();
  });

  it('B 型页签：非总部组织保留本地新增，基线行禁编辑且归属可见', async () => {
    accessRef.current = {
      isHeadquartersOrganization: false,
      canCreateMasterDataPorts: true,
      canUpdateMasterDataPorts: true,
    };
    renderPanel(<PortsPanel />);

    await waitFor(() => expect(screen.getByText('CNSHG')).toBeInTheDocument());
    expect(
      screen.getByText('总部共享基线 + 本地补充行仅本组织可见'),
    ).toBeInTheDocument();
    // 本地新增入口保留
    expect(
      screen.getByRole('button', { name: /新增海运港口/ }),
    ).toBeInTheDocument();

    // 基线行（organizationId 为空）不渲染编辑/停用
    const baselineRow = screen.getByText('CNSHG').closest('tr');
    expect(baselineRow).not.toBeNull();
    expect(within(baselineRow as HTMLElement).queryByText('编辑')).toBeNull();
    expect(within(baselineRow as HTMLElement).queryByText('停用')).toBeNull();

    // 本组织行可编辑
    const localRow = screen.getByText('CNZJG').closest('tr');
    expect(localRow).not.toBeNull();
    expect(
      within(localRow as HTMLElement).getByText('编辑'),
    ).toBeInTheDocument();

    // 归属列区分基线与本地行（antd 固定列会重复渲染节点，只断言存在性）
    expect(screen.getAllByText('集团基线行').length).toBeGreaterThan(0);
    expect(screen.getAllByText('本组织行').length).toBeGreaterThan(0);
  });

  it('B 型页签：总部组织基线行与本地行均可编辑', async () => {
    accessRef.current = {
      isHeadquartersOrganization: true,
      canCreateMasterDataPorts: true,
      canUpdateMasterDataPorts: true,
    };
    renderPanel(<PortsPanel />);

    await waitFor(() => expect(screen.getByText('CNSHG')).toBeInTheDocument());
    expect(
      screen.queryByText('总部共享基线 + 本地补充行仅本组织可见'),
    ).toBeNull();
    expect(screen.getAllByText('编辑')).toHaveLength(portItems.length);
  });
});
