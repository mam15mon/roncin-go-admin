import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import dayjs from 'dayjs';
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

  it('searchable-select 配置 request 时渲染远程搜索下拉：挂载加载首屏、关键字联想服务端过滤', async () => {
    const request = vi
      .fn()
      .mockResolvedValue([{ label: '测试合作方 (TP)', value: 'p-1' }]);

    render(
      <SearchFilterTemplate
        layout="grid"
        items={[
          {
            name: 'settlementPartyId',
            label: '结算单位',
            type: 'searchable-select',
            placeholder: '输入名称/全拼搜索结算单位',
            request,
          },
        ]}
      />,
    );

    // 挂载即以空关键字加载首屏候选
    await waitFor(() =>
      expect(request).toHaveBeenCalledWith({ keyWords: undefined }),
    );

    fireEvent.mouseDown(screen.getByLabelText('结算单位'));
    expect(await screen.findByText('测试合作方 (TP)')).toBeInTheDocument();
  });

  it('grid 模式提交后回显已选条件 chips：select 从静态 options 解析 label', async () => {
    const handleSearch = vi.fn();

    render(
      <SearchFilterTemplate
        layout="grid"
        collapsible={false}
        items={[
          { name: 'keyword', label: '关键字', placeholder: '请输入关键字' },
          {
            name: 'status',
            label: '状态',
            placeholder: '请选择状态',
            type: 'select',
            options: [
              { label: '已生效', value: 'ACTIVE' },
              { label: '已作废', value: 'VOIDED' },
            ],
          },
        ]}
        onSearch={handleSearch}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText('请输入关键字'), {
      target: { value: '测试订单' },
    });
    fireEvent.mouseDown(screen.getByLabelText('状态'));
    fireEvent.click(await screen.findByText('已生效'));
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));

    await waitFor(() => expect(handleSearch).toHaveBeenCalledTimes(1));
    expect(handleSearch).toHaveBeenCalledWith(
      expect.objectContaining({ keyword: '测试订单', status: 'ACTIVE' }),
    );
    expect(screen.getByText('关键字: 测试订单')).toBeInTheDocument();
    expect(screen.getByText('状态: 已生效')).toBeInTheDocument();
  });

  it('grid 模式空字符串提交不产生 chip', async () => {
    const handleSearch = vi.fn();

    render(
      <SearchFilterTemplate
        layout="grid"
        collapsible={false}
        items={[
          { name: 'keyword', label: '关键字', placeholder: '请输入关键字' },
        ]}
        onSearch={handleSearch}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText('请输入关键字'), {
      target: { value: '   ' },
    });
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));

    await waitFor(() => expect(handleSearch).toHaveBeenCalledTimes(1));
    expect(screen.queryByText(/关键字\s*:/)).not.toBeInTheDocument();
  });

  it('grid 模式删除 chip：表单字段清空且 onSearch 参数剔除该字段并保留其余字段', async () => {
    const handleSearch = vi.fn();

    render(
      <SearchFilterTemplate
        layout="grid"
        collapsible={false}
        items={[
          { name: 'keyword', label: '关键字', placeholder: '请输入关键字' },
          {
            name: 'status',
            label: '状态',
            placeholder: '请选择状态',
            type: 'select',
            options: [
              { label: '已生效', value: 'ACTIVE' },
              { label: '已作废', value: 'VOIDED' },
            ],
          },
        ]}
        onSearch={handleSearch}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText('请输入关键字'), {
      target: { value: '测试订单' },
    });
    fireEvent.mouseDown(screen.getByLabelText('状态'));
    fireEvent.click(await screen.findByText('已生效'));
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));

    await waitFor(() => expect(handleSearch).toHaveBeenCalledTimes(1));

    // 删除「状态」chip：仅剔除该字段，关键字条件保留
    const statusTag = screen
      .getByText('状态: 已生效')
      .closest('.ant-tag') as HTMLElement;
    fireEvent.click(
      statusTag.querySelector('.ant-tag-close-icon') as HTMLElement,
    );

    await waitFor(() => expect(handleSearch).toHaveBeenCalledTimes(2));
    expect(handleSearch).toHaveBeenLastCalledWith({ keyword: '测试订单' });
    // 对应表单字段被同步清空，其余字段不受影响
    expect(screen.getByText('请选择状态')).toBeInTheDocument();
    expect(screen.queryByText('状态: 已生效')).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText('请输入关键字')).toHaveValue('测试订单');
  });

  it('grid 模式 date-range chip 显示本地日期格式，「清除全部」清空并触发 onReset', async () => {
    const handleReset = vi.fn();

    render(
      <SearchFilterTemplate
        layout="grid"
        collapsible={false}
        items={[
          {
            name: 'dateRange',
            label: '创建日期',
            type: 'date-range',
            initialValue: [dayjs('2026-01-01'), dayjs('2026-01-15')],
          },
        ]}
        onReset={handleReset}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /查询/ }));
    expect(
      await screen.findByText('创建日期: 2026-01-01 ~ 2026-01-15'),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByText('清除全部'));
    expect(handleReset).toHaveBeenCalledTimes(1);
    expect(screen.queryByText(/创建日期\s*:/)).not.toBeInTheDocument();
    expect(screen.queryByText('清除全部')).not.toBeInTheDocument();
  });

  it('bar 模式 quickFilters 提交后回显 chip 并可删除', async () => {
    const handleSearch = vi.fn();

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
      />,
    );

    fireEvent.change(screen.getByPlaceholderText('搜索代码或名称'), {
      target: { value: '张三' },
    });
    // 选择快捷筛选后自动触发提交
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('业务员'));

    expect(await screen.findByText('角色筛选: 业务员')).toBeInTheDocument();
    expect(screen.getByText('关键字: 张三')).toBeInTheDocument();

    const roleTag = screen
      .getByText('角色筛选: 业务员')
      .closest('.ant-tag') as HTMLElement;
    fireEvent.click(
      roleTag.querySelector('.ant-tag-close-icon') as HTMLElement,
    );

    expect(handleSearch).toHaveBeenLastCalledWith({ keyword: '张三' });
    expect(screen.queryByText('角色筛选: 业务员')).not.toBeInTheDocument();
  });

  it('远程搜索字段选择后提交，chip 显示所选候选项 label', async () => {
    const handleSearch = vi.fn();
    const request = vi
      .fn()
      .mockResolvedValue([{ label: '测试合作方 (TP)', value: 'p-1' }]);

    render(
      <SearchFilterTemplate
        layout="grid"
        items={[
          {
            name: 'settlementPartyId',
            label: '结算单位',
            type: 'searchable-select',
            placeholder: '输入名称搜索结算单位',
            request,
          },
        ]}
        onSearch={handleSearch}
      />,
    );

    await waitFor(() =>
      expect(request).toHaveBeenCalledWith({ keyWords: undefined }),
    );
    fireEvent.mouseDown(screen.getByLabelText('结算单位'));
    fireEvent.click(await screen.findByText('测试合作方 (TP)'));
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));

    await waitFor(() => expect(handleSearch).toHaveBeenCalledTimes(1));
    expect(handleSearch).toHaveBeenCalledWith(
      expect.objectContaining({ settlementPartyId: 'p-1' }),
    );
    expect(screen.getByText('结算单位: 测试合作方 (TP)')).toBeInTheDocument();
  });
});
