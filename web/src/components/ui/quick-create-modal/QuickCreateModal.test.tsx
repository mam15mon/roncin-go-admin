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

    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith({ name: '测试单位' }),
    );
    expect(await screen.findByText('创建往来单位失败')).toBeInTheDocument();
    expect(screen.getByLabelText('名称')).toHaveValue('测试单位');
  });

  it('附加动作渲染为第一个底部按钮，点击不触发表单校验与提交', async () => {
    const onSubmit = vi.fn();
    const onExtraClick = vi.fn();

    render(
      <App>
        <QuickCreateModal<{ name: string }, { id: string }>
          title="快捷创建"
          open
          onCancel={vi.fn()}
          onSubmit={onSubmit}
          okText="保存"
          centered
          extraAction={{
            text: '添加公司详情',
            onClick: onExtraClick,
          }}
        >
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </QuickCreateModal>
      </App>,
    );

    const footerButtons = screen.getAllByRole('button');
    const extraButton = screen.getByRole('button', { name: '添加公司详情' });
    // antd 两字按钮渲染为「取 消」「保 存」，顺序：附加动作在最前
    expect(footerButtons.indexOf(extraButton)).toBeLessThan(
      footerButtons.indexOf(screen.getByRole('button', { name: /取\s*消/ })),
    );
    expect(footerButtons.indexOf(extraButton)).toBeLessThan(
      footerButtons.indexOf(screen.getByRole('button', { name: /保\s*存/ })),
    );

    fireEvent.click(extraButton);
    expect(onExtraClick).toHaveBeenCalledTimes(1);
    // 表单为空且未校验：没有必填错误提示，也没有触发提交
    await waitFor(() =>
      expect(screen.queryByText('请输入名称')).not.toBeInTheDocument(),
    );
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
