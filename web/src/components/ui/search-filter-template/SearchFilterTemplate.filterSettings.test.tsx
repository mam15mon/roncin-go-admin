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
import type { SearchFilterFieldItem } from './types';

const range = [dayjs('2026-10-01'), dayjs('2026-10-12')];
const items: SearchFilterFieldItem[] = [
  { name: 'keyword', label: '综合搜索', placeholder: '关键字' },
  {
    name: 'direction',
    label: '费用属性',
    type: 'select',
    placeholder: '属性',
    options: [{ label: '应收', value: 'RECEIVABLE' }],
  },
  {
    name: 'partyNo',
    label: '结算单位',
    placeholder: '结算单位编号',
  },
  {
    name: 'createdAtRange',
    label: '录入日期',
    type: 'date-range',
    placeholder: ['开始日期', '结束日期'],
  },
];
const allNames = items.map((item) => item.name);

function setup(
  visibleNames: string[],
  extra: Partial<React.ComponentProps<typeof SearchFilterTemplate>> = {},
) {
  const onSearch = vi.fn();
  const onCommit = vi.fn();
  const view = (names: string[]) => (
    <SearchFilterTemplate
      layout="grid"
      collapsible={false}
      items={items}
      visibleNames={names}
      initialValues={{
        partyNo: 'P001',
        createdAtRange: range,
        direction: 'RECEIVABLE',
      }}
      onSearch={onSearch}
      onCommit={onCommit}
      {...extra}
    />
  );
  const utils = render(view(visibleNames));
  return {
    ...utils,
    onSearch,
    onCommit,
    setVisible: (names: string[]) => utils.rerender(view(names)),
  };
}

describe('SearchFilterTemplate 筛选设置：移除字段清值并刷新', () => {
  afterEach(cleanup);

  it('移除已提交的结算单位与日期范围：一次刷新、完整清除起止值并保留其他条件', async () => {
    const { onSearch, onCommit, setVisible } = setup(allNames);
    expect(screen.getByDisplayValue('2026-10-01')).toBeInTheDocument();
    setVisible(['keyword', 'direction']);
    await waitFor(() => expect(onSearch).toHaveBeenCalledTimes(1));
    const next = onSearch.mock.calls[0][0];
    expect(next).toEqual({ direction: 'RECEIVABLE' });
    expect(onCommit).toHaveBeenCalledWith(next, expect.any(Object));
    expect(screen.queryByText('结算单位')).not.toBeInTheDocument();
    // 重新加入后不恢复旧值，起止日期均为空。
    setVisible(allNames);
    expect(await screen.findByPlaceholderText('结算单位编号')).toHaveValue('');
    expect(screen.queryByDisplayValue('2026-10-01')).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue('2026-10-12')).not.toBeInTheDocument();
    expect(onSearch).toHaveBeenCalledTimes(1);
  });

  it('移除没有已提交值的字段只清除未提交输入，不触发刷新', async () => {
    const { onSearch, setVisible } = setup(allNames);
    fireEvent.change(screen.getByPlaceholderText('关键字'), {
      target: { value: '未提交' },
    });
    setVisible(['direction', 'partyNo', 'createdAtRange']);
    setVisible(allNames);
    expect(await screen.findByPlaceholderText('关键字')).toHaveValue('');
    expect(onSearch).not.toHaveBeenCalled();
  });

  it('移除字段只用已提交条件刷新，不连带提交其他未提交输入', async () => {
    const { onSearch, setVisible } = setup(allNames);
    fireEvent.change(screen.getByPlaceholderText('关键字'), {
      target: { value: '未提交关键字' },
    });
    setVisible(['keyword', 'direction', 'createdAtRange']);
    await waitFor(() => expect(onSearch).toHaveBeenCalledTimes(1));
    expect(onSearch.mock.calls[0][0]).toEqual({
      direction: 'RECEIVABLE',
      createdAtRange: range,
    });
    // 未提交输入仍保留在表单中，由用户自行决定是否提交。
    expect(screen.getByPlaceholderText('关键字')).toHaveValue('未提交关键字');
  });

  it('普通折叠与展开只改变展示：折叠字段的已提交条件保留，展开后值仍可见且提交不丢失', async () => {
    const { onSearch, onCommit } = setup(allNames, {
      collapsible: true,
      defaultCollapsed: true,
      defaultVisibleCount: 2,
    });
    expect(
      screen.queryByPlaceholderText('结算单位编号'),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));
    await waitFor(() => expect(onCommit).toHaveBeenCalled());
    expect(onSearch.mock.calls[0][0]).toMatchObject({
      partyNo: 'P001',
      createdAtRange: range,
    });
    fireEvent.click(screen.getByText(/展开/));
    expect(await screen.findByPlaceholderText('结算单位编号')).toHaveValue(
      'P001',
    );
  });

  it('移除后重置再显示字段，不恢复挂载时旧筛选值', async () => {
    const { setVisible } = setup(allNames);
    setVisible(['keyword', 'direction']);
    fireEvent.click(screen.getByRole('button', { name: /重置/ }));
    setVisible(allNames);
    expect(await screen.findByPlaceholderText('结算单位编号')).toHaveValue('');
    expect(screen.queryByDisplayValue('2026-10-01')).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue('2026-10-12')).not.toBeInTheDocument();
  });

  it('未传 visibleNames 时行为不变：全部 items 渲染且不触发清值', () => {
    const onSearch = vi.fn();
    render(
      <SearchFilterTemplate
        layout="grid"
        collapsible={false}
        items={items}
        onSearch={onSearch}
      />,
    );
    expect(screen.getByPlaceholderText('结算单位编号')).toBeInTheDocument();
    expect(onSearch).not.toHaveBeenCalled();
  });
});
