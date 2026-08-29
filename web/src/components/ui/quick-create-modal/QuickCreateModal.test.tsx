import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App, Form, Input } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { QuickCreateModal } from './QuickCreateModal';

describe('QuickCreateModal', () => {
  it('提交失败时显示错误并保留表单内容', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new Error('创建往来单位失败'));

    render(
      <App>
        <QuickCreateModal<{ name: string }, { id: string }>
          title="快捷创建"
          open
          onCancel={vi.fn()}
          onSubmit={onSubmit}
        >
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </QuickCreateModal>
      </App>,
    );

    fireEvent.change(screen.getByLabelText('名称'), {
      target: { value: '测试单位' },
    });
    fireEvent.click(screen.getByRole('button', { name: '保存并选用' }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith({ name: '测试单位' }));
    expect(await screen.findByText('创建往来单位失败')).toBeInTheDocument();
    expect(screen.getByLabelText('名称')).toHaveValue('测试单位');
  });
});
