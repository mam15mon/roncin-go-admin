import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// 组织身份与权限判定统一走 @/app/access 的 useAccess（access.ts 契约），
// 测试通过 hoisted 可变对象切换系统管理 / 非系统管理视角。
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
    // 共享港口
    id: 'port-1',
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
    // 另一个共享港口
    id: 'port-2',
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

describe('公共主数据按工作台控制维护入口', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListItems.mockResolvedValue({ data: countryItems, total: 2 });
    mockListPorts.mockResolvedValue({ data: portItems, total: 2 });
  });

  afterEach(() => {
    cleanup();
  });

  it('A 型页签：非系统管理组织显示系统管理维护横幅并隐藏全部写按钮', async () => {
    accessRef.current = {
      isSystemWorkspace: false,
      canCreateMasterDataItems: true,
      canUpdateMasterDataItems: true,
    };
    renderPanel(<CountriesPanel />);

    await waitFor(() => expect(screen.getByText('CN')).toBeInTheDocument());
    expect(screen.getByText('由系统管理员统一维护与共享')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /新增国家与地区/ })).toBeNull();
    expect(screen.queryByText('编辑')).toBeNull();
  });

  it('A 型页签：系统管理组织且持有写权限时入口正常且无横幅', async () => {
    accessRef.current = {
      isSystemWorkspace: true,
      canCreateMasterDataItems: true,
      canUpdateMasterDataItems: true,
    };
    renderPanel(<CountriesPanel />);

    await waitFor(() => expect(screen.getByText('CN')).toBeInTheDocument());
    expect(screen.queryByText('由系统管理员统一维护与共享')).toBeNull();
    expect(
      screen.getByRole('button', { name: /新增国家与地区/ }),
    ).toBeInTheDocument();
    expect(screen.getAllByText('编辑')).toHaveLength(countryItems.length);
  });

  it('A 型页签：系统管理组织但无写权限码时同样隐藏写按钮', async () => {
    accessRef.current = {
      isSystemWorkspace: true,
      canCreateMasterDataItems: false,
      canUpdateMasterDataItems: false,
    };
    renderPanel(<CountriesPanel />);

    await waitFor(() => expect(screen.getByText('CN')).toBeInTheDocument());
    expect(screen.queryByRole('button', { name: /新增国家与地区/ })).toBeNull();
    expect(screen.queryByText('编辑')).toBeNull();
  });

  it('公司只读共享港口，不显示新增或编辑入口', async () => {
    accessRef.current = {
      isSystemWorkspace: false,
      canCreateMasterDataPorts: true,
      canUpdateMasterDataPorts: true,
    };
    renderPanel(<PortsPanel />);
    await waitFor(() => expect(screen.getByText('CNSHG')).toBeInTheDocument());
    expect(screen.getByText('由系统管理员统一维护与共享')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /新增海运港口/ })).toBeNull();
    expect(screen.queryByText('编辑')).toBeNull();
    expect(screen.queryByText('归属')).toBeNull();
  });

  it('系统管理有权限时可维护全部公共港口', async () => {
    accessRef.current = {
      isSystemWorkspace: true,
      canCreateMasterDataPorts: true,
      canUpdateMasterDataPorts: true,
    };
    renderPanel(<PortsPanel />);

    await waitFor(() => expect(screen.getByText('CNSHG')).toBeInTheDocument());
    expect(screen.queryByText('由系统管理员统一维护与共享')).toBeNull();
    expect(screen.getAllByText('编辑')).toHaveLength(portItems.length);
  });
});
