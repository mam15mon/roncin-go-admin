import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { SettingTableTemplate } from './SettingTableTemplate';

describe('SettingTableTemplate 响应式接入', () => {
  it('保留官方 request、行选择及显式纵向滚动配置', async () => {
    const query = vi.fn().mockResolvedValue({
      data: [{ id: 'unit-1', name: '票' }],
      total: 1,
      success: true,
    });
    const onSelectionChange = vi.fn();
    const { container } = renderWithApp(
      <SettingTableTemplate
        entityName="计费单位"
        columns={[{ title: '名称', dataIndex: 'name', width: 160 }]}
        query={query}
        renderFormItems={() => null}
        canCreate={false}
        canUpdate={false}
        scroll={{ y: 240 }}
        rowSelection={{ onChange: onSelectionChange }}
      />,
    );

    expect(await screen.findByText('票')).toBeInTheDocument();
    expect(query).toHaveBeenCalled();
    expect(container.querySelector('.ant-table-body')).toHaveStyle({
      maxHeight: '240px',
    });
    fireEvent.click(screen.getAllByRole('checkbox')[1]);
    await waitFor(() => {
      expect(onSelectionChange).toHaveBeenCalledWith(
        ['unit-1'],
        [expect.objectContaining({ id: 'unit-1', name: '票' })],
        expect.anything(),
      );
    });
  });
});
