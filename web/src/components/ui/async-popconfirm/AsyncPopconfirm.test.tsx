import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { Button } from 'antd';
import { expect, it, vi } from 'vitest';
import { AsyncPopconfirm } from './AsyncPopconfirm';

it('真实Popconfirm等待Promise，false与reject保持弹层，重试成功才关闭', async () => {
  let finish!: (value: boolean) => void;
  const onConfirm = vi
    .fn()
    .mockImplementationOnce(
      () =>
        new Promise<boolean>((resolve) => {
          finish = resolve;
        }),
    )
    .mockRejectedValueOnce(new Error('已由请求层提示'))
    .mockResolvedValueOnce(undefined);
  const onOpenChange = vi.fn();
  renderWithApp(
    <AsyncPopconfirm
      title="确认停用档案？"
      onConfirm={onConfirm}
      onOpenChange={onOpenChange}
      okText="提交停用"
    >
      <Button>停用</Button>
    </AsyncPopconfirm>,
  );
  const trigger = screen.getByRole('button', { name: /停\s*用/ });
  fireEvent.click(trigger);
  fireEvent.click(await screen.findByRole('button', { name: /提交停用/ }));
  await waitFor(() =>
    expect(screen.getByRole('button', { name: /提交停用/ })).toHaveClass(
      'ant-btn-loading',
    ),
  );
  finish(false);
  await waitFor(() =>
    expect(screen.getByRole('button', { name: /提交停用/ })).not.toHaveClass(
      'ant-btn-loading',
    ),
  );
  expect(screen.getByText('确认停用档案？')).toBeInTheDocument();
  expect(trigger).toHaveClass('ant-popover-open');
  expect(onOpenChange).not.toHaveBeenCalledWith(false);
  fireEvent.click(screen.getByRole('button', { name: /提交停用/ }));
  await waitFor(() => expect(onConfirm).toHaveBeenCalledTimes(2));
  await waitFor(() =>
    expect(screen.getByRole('button', { name: /提交停用/ })).not.toHaveClass(
      'ant-btn-loading',
    ),
  );
  expect(screen.getByText('确认停用档案？')).toBeInTheDocument();
  expect(trigger).toHaveClass('ant-popover-open');
  expect(onOpenChange).not.toHaveBeenCalledWith(false);
  fireEvent.click(screen.getByRole('button', { name: /提交停用/ }));
  await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  expect(onConfirm).toHaveBeenCalledTimes(3);
});
