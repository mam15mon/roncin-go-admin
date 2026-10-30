import { ProForm, ProFormText } from '@ant-design/pro-components';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, it, vi } from 'vitest';

it('隐藏占位框有内部标识，正常提交仅包含注册的业务字段', async () => {
  const submit = vi.fn().mockResolvedValue(true);
  const { container } = render(
    <ProForm onFinish={submit}>
      <ProFormText name="businessName" label="业务名称" />
    </ProForm>,
  );
  const guard = container.querySelector<HTMLInputElement>(
    'input[name="__pro_form_implicit_submit_guard"]',
  );
  expect(guard).toHaveAttribute('type', 'text');
  expect(guard).toHaveStyle({ display: 'none' });
  fireEvent.change(screen.getByLabelText('业务名称'), {
    target: { value: '表单验证' },
  });
  fireEvent.click(screen.getByRole('button', { name: /提 交|提交/ }));
  await waitFor(() => expect(submit).toHaveBeenCalledTimes(1));
  expect(submit.mock.calls[0][0]).toEqual({ businessName: '表单验证' });
});
