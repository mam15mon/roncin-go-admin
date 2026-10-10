import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  type SubEntityDrawerRef,
  SubEntityDrawerTemplate,
} from './SubEntityDrawerTemplate';

describe('SubEntityDrawerTemplate 响应式接入', () => {
  it('在抽屉内保留父实体请求与 actionRef 刷新', async () => {
    const ref = createRef<SubEntityDrawerRef<{ id: string }>>();
    const fetchList = vi.fn().mockResolvedValue({
      data: [{ id: 'container-1', name: '集装箱一' }],
      success: true,
    });
    renderWithApp(
      <SubEntityDrawerTemplate<{ id: string; name: string }, { id: string }>
        ref={ref}
        entityName="集装箱"
        drawerWidth={400}
        columns={[{ title: '名称', dataIndex: 'name', width: 320 }]}
        fetchList={fetchList}
        renderFormItems={() => null}
        canCreate={false}
        canUpdate={false}
        canRemove={false}
      />,
    );

    await act(async () => ref.current?.open({ id: 'order-1' }));
    expect(await screen.findByText('集装箱一')).toBeInTheDocument();
    expect(fetchList).toHaveBeenCalledWith({ id: 'order-1' });
    const calls = fetchList.mock.calls.length;
    await act(async () => ref.current?.reload());
    await waitFor(() =>
      expect(fetchList.mock.calls.length).toBeGreaterThan(calls),
    );
    fireEvent.click(screen.getByRole('button', { name: '关闭' }));
    await waitFor(() => {
      expect(screen.queryByText('集装箱一')).not.toBeInTheDocument();
    });
  });
});
