import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { SearchFilterTemplate } from './SearchFilterTemplate';

describe('恢复查询的搜索表单', () => {
  it('恢复远程候选名称及chips，未提交编辑不覆盖已提交，重置清除恢复initialValues', async () => {
    const onCommit = vi.fn();
    const onSearch = vi.fn();
    const onReset = vi.fn();
    render(
      <SearchFilterTemplate
        initialValues={{ keyword: '已提交', partnerId: 'outside-first-page' }}
        initialLabels={{ partnerId: '恢复客户' }}
        onCommit={onCommit}
        onSearch={onSearch}
        onReset={onReset}
        defaultCollapsed={false}
        items={[
          { name: 'keyword', label: '关键字' },
          {
            name: 'partnerId',
            label: '客户',
            type: 'searchable-select',
            request: async () => [],
          },
        ]}
      />,
    );
    expect(screen.getByText('客户: 恢复客户')).toBeInTheDocument();
    expect(screen.getByLabelText('关键字')).toHaveValue('已提交');
    await screen.findByText('恢复客户');
    fireEvent.change(screen.getByLabelText('关键字'), {
      target: { value: '未提交' },
    });
    expect(screen.getByText('关键字: 已提交')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /重\s*置/ }));
    expect(screen.getByLabelText('关键字')).toHaveValue('');
    expect(onCommit).toHaveBeenLastCalledWith({}, {});
    expect(onReset).toHaveBeenCalledTimes(1);
    expect(screen.queryByText('客户: 恢复客户')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));
    await waitFor(() =>
      expect(onSearch).toHaveBeenCalledWith({
        keyword: undefined,
        partnerId: undefined,
      }),
    );
  });
});
