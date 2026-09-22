import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SearchFilterTemplate } from './SearchFilterTemplate';

describe('SearchFilterTemplate', () => {
  afterEach(() => {
    cleanup();
  });

  it('渲染快捷单行搜索栏模式 (layout="bar")', () => {
    const handleSearch = vi.fn();
    const handleReset = vi.fn();

    render(
      <SearchFilterTemplate
        layout="bar"
        keywordPlaceholder="搜索代码或名称"
        quickFilters={[
          {
            name: 'role',
            placeholder: '角色筛选',
            options: [{ label: '业务员', value: 'SALES' }],
          },
        ]}
        onSearch={handleSearch}
        onReset={handleReset}
      />,
    );

    expect(screen.getByPlaceholderText('搜索代码或名称')).toBeInTheDocument();
    expect(screen.getByText('查询')).toBeInTheDocument();
    expect(screen.getByText('重置')).toBeInTheDocument();

    fireEvent.click(screen.getByText('重置'));
    expect(handleReset).toHaveBeenCalledTimes(1);
  });

  it('渲染多字段配置网格表单模式 (layout="grid") 并支持折叠展开', () => {
    const handleSearch = vi.fn();

    render(
      <SearchFilterTemplate
        layout="grid"
        collapsible
        defaultCollapsed
        defaultVisibleCount={2}
        items={[
          { name: 'keyword', label: '关键字', placeholder: '请输入关键字' },
          {
            name: 'status',
            label: '状态',
            placeholder: '请选择状态',
            type: 'select',
          },
          { name: 'creator', label: '创建人', placeholder: '请输入创建人' },
        ]}
        onSearch={handleSearch}
      />,
    );

    // 默认折叠，只展示前 2 个字段
    expect(screen.getByText('关键字')).toBeInTheDocument();
    expect(screen.getByText('状态')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('请输入关键字').closest('form'),
    ).toHaveClass('roncin-search-filter-grid-horizontal');
    expect(screen.queryByText('创建人')).not.toBeInTheDocument();
    expect(screen.getByText(/展开/)).toBeInTheDocument();

    // 点击展开
    fireEvent.click(screen.getByText(/展开/));
    expect(screen.getByText('创建人')).toBeInTheDocument();
    expect(screen.getByText(/收起/)).toBeInTheDocument();
  });

  it('渲染自定义插槽模式 (layout="custom")', () => {
    render(
      <SearchFilterTemplate layout="custom">
        <div data-testid="custom-content">自定义筛选区域</div>
      </SearchFilterTemplate>,
    );

    expect(screen.getByTestId('custom-content')).toBeInTheDocument();
  });

  it('多字段网格表单模式下剩余栅格不足或含 extraRight 且剩余不足时，操作区自动换行独立成行 (span=24)', () => {
    render(
      <SearchFilterTemplate
        layout="grid"
        collapsible={false}
        items={[
          { name: 'field1', label: '字段1', span: 6 },
          { name: 'field2', label: '字段2', span: 4 },
          { name: 'field3', label: '字段3', span: 8 },
          { name: 'field4', label: '字段4', span: 4 },
        ]}
        extraRight={<button type="button">导出数据</button>}
      />,
    );

    // 4 个字段合计 22 栅格，剩余仅 2 栅格且有 extraRight，操作区应自动换行至 span=24
    expect(screen.getByText('字段1')).toBeInTheDocument();
    expect(screen.getByText('字段4')).toBeInTheDocument();
    expect(screen.getByText('导出数据')).toBeInTheDocument();
    expect(screen.getByText('查询')).toBeInTheDocument();
    expect(screen.getByText('重置')).toBeInTheDocument();

    const exportBtn = screen.getByText('导出数据');
    const actionCol = exportBtn.closest('.ant-col-24');
    expect(actionCol).toBeInTheDocument();
  });
});
