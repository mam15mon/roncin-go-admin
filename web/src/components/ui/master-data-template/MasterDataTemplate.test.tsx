import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MasterDataTemplate } from './MasterDataTemplate';
import type { BaseMasterDataItem } from './types';

// Mock clipboard
Object.defineProperty(navigator, 'clipboard', {
  value: {
    writeText: vi.fn(),
  },
  configurable: true,
});

interface TestItem extends BaseMasterDataItem {
  countryCode?: string;
}

const mockItems: TestItem[] = [
  {
    id: '1',
    code: 'CNSHG',
    name: '上海港',
    nameEn: 'SHANGHAI',
    enabled: true,
    updatedAt: '2026-09-01 12:00:00',
    countryCode: 'CN',
  },
  {
    id: '2',
    code: 'USLAX',
    name: '洛杉矶港',
    nameEn: 'LOS ANGELES',
    enabled: false,
    updatedAt: '2026-09-02 12:00:00',
    countryCode: 'US',
  },
];

describe('MasterDataTemplate (紧凑一体化 ProTable 模板与单行6卡片)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正确渲染单行6个指标统计卡片 (span=4 / md=4 lg=4 xl=4)', () => {
    render(
      <MasterDataTemplate<TestItem>
        title="海运港口管理"
        subtitle="维护全球港口五字码"
        codeLabel="UN/LOCODE"
        items={mockItems}
        total={mockItems.length}
        activeTotal={1}
        disabledTotal={1}
        extraStats={[
          { label: '海港枢纽', value: 1, color: '#1677ff' },
          { label: '铁路枢纽', value: 0, color: '#fa8c16' },
          { label: '多式联运港', value: 1, color: '#722ed1' },
        ]}
        formFields={[]}
      />,
    );

    // 基础3卡片 + 扩展3卡片 = 6张卡片
    expect(screen.getByText('全部海运港口')).toBeInTheDocument();
    expect(screen.getByText('启用中')).toBeInTheDocument();
    expect(screen.getByText('已停用')).toBeInTheDocument();
    expect(screen.getByText('海港枢纽')).toBeInTheDocument();
    expect(screen.getByText('铁路枢纽')).toBeInTheDocument();
    expect(screen.getByText('多式联运港')).toBeInTheDocument();

    // 验证表格内容
    expect(screen.getByText('CNSHG')).toBeInTheDocument();
    expect(screen.getByText('上海港')).toBeInTheDocument();
    expect(screen.getByText('USLAX')).toBeInTheDocument();
    expect(screen.getByText('洛杉矶港')).toBeInTheDocument();
  });

  it('支持在客户端模式下进行关键字过滤', () => {
    render(
      <MasterDataTemplate<TestItem>
        title="海运港口管理"
        items={mockItems}
        formFields={[]}
        searchPlaceholder="搜索代码或名称"
      />,
    );

    const input = screen.getByPlaceholderText('搜索代码或名称');
    fireEvent.change(input, { target: { value: 'CNSHG' } });

    expect(screen.getByText('CNSHG')).toBeInTheDocument();
    expect(screen.queryByText('USLAX')).not.toBeInTheDocument();
  });

  it('点击新增按钮能正确唤起模态弹窗', () => {
    const handleCreate = vi.fn();
    render(
      <MasterDataTemplate<TestItem>
        title="海运港口管理"
        items={mockItems}
        onCreate={handleCreate}
        formFields={[
          {
            name: 'code',
            label: '港口代码',
            required: true,
          },
        ]}
      />,
    );

    const addBtn = screen.getByRole('button', { name: /新增海运港口/ });
    fireEvent.click(addBtn);

    expect(screen.getByText('港口代码')).toBeInTheDocument();
  });

  it('配置 notice 时在顶部渲染组织治理提示横幅', () => {
    render(
      <MasterDataTemplate<TestItem>
        title="国际货币与币种管理"
        items={mockItems}
        formFields={[]}
        notice="由总部统一维护与共享"
      />,
    );

    expect(screen.getByText('由总部统一维护与共享')).toBeInTheDocument();
  });

  it('canEditRecord 返回 false 的行不渲染编辑与停用按钮', () => {
    // 模拟 B 型页签非总部视角：基线行（organizationId 为空）禁用编辑。
    render(
      <MasterDataTemplate<TestItem>
        title="海运港口管理"
        items={[
          { ...mockItems[0], organizationId: undefined },
          { ...mockItems[1], organizationId: 'org-1' },
        ]}
        onUpdate={vi.fn()}
        onToggleActive={vi.fn()}
        formFields={[]}
        canEditRecord={(record) => Boolean(record.organizationId)}
      />,
    );

    const baselineRow = screen.getByText('CNSHG').closest('tr');
    const localRow = screen.getByText('USLAX').closest('tr');
    expect(baselineRow).not.toBeNull();
    expect(localRow).not.toBeNull();

    expect(within(baselineRow as HTMLElement).queryByText('编辑')).toBeNull();
    expect(within(baselineRow as HTMLElement).queryByText('停用')).toBeNull();
    expect(
      within(localRow as HTMLElement).getByText('编辑'),
    ).toBeInTheDocument();
    // USLAX 处于停用状态，行级开关按钮显示「启用」。
    expect(
      within(localRow as HTMLElement).getByText('启用'),
    ).toBeInTheDocument();
  });

  it('未配置 canEditRecord 时所有行保持原有编辑入口', () => {
    render(
      <MasterDataTemplate<TestItem>
        title="海运港口管理"
        items={mockItems}
        onUpdate={vi.fn()}
        onToggleActive={vi.fn()}
        formFields={[]}
      />,
    );

    expect(screen.getAllByText('编辑')).toHaveLength(mockItems.length);
  });
});
