import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import {
  masterResponse,
  masterServices,
  renderMaster,
  setupMasterTests,
} from './air-master-test.fixtures';

setupMasterTests();

async function openEdit() {
  const button = await screen.findByRole('button', { name: '修改共同资料' });
  await waitFor(() => expect(button).toBeEnabled());
  fireEvent.click(button);
  return screen.findByLabelText('测试主单正文');
}

async function openRestore() {
  const button = await screen.findByRole('button', {
    name: '恢复关联已终止订单',
  });
  fireEvent.click(button);
  const order = await screen.findByLabelText('选择空运订单');
  fireEvent.mouseDown(order);
  fireEvent.click(await screen.findByText(/AE001/));
  fireEvent.change(screen.getByLabelText('关联原因'), {
    target: { value: '恢复操作' },
  });
}

describe('航空主单提交后的会话与输入保护', () => {
  it('共享保存成功后聚合刷新在途的新输入保留，旧完成不得关闭编辑稿', async () => {
    renderMaster();
    const remarks = await openEdit();
    fireEvent.change(screen.getByLabelText('修改原因'), {
      target: { value: '修订' },
    });
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    const submit = await screen.findByRole('button', {
      name: '确认影响并提交',
    });
    const refresh = Promise.withResolvers<API.GetAirMasterWaybillResponse>();
    masterServices.update.mockResolvedValueOnce({
      success: true,
      data: masterResponse('2').data,
    });
    masterServices.get.mockReturnValueOnce(refresh.promise);
    fireEvent.click(submit);
    await waitFor(() => expect(masterServices.get).toHaveBeenCalledTimes(2));
    expect(remarks).toBeVisible();
    fireEvent.change(remarks, { target: { value: '保存后的新正文' } });
    await act(async () => refresh.resolve(masterResponse('2')));
    expect(remarks).toHaveValue('保存后的新正文');
    expect(remarks).toBeVisible();
    expect(getAnyDirtyTabKey()).toBeDefined();
  });

  it('恢复关联冻结打开时主单版本，后台新查询不能借给旧恢复稿', async () => {
    const view = renderMaster();
    await openRestore();
    masterServices.get.mockResolvedValue(masterResponse('2'));
    await act(async () =>
      view.queryClient.invalidateQueries({
        queryKey: ['air-orders', 'master'],
      }),
    );
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(masterServices.link).toHaveBeenCalledOnce());
    expect(masterServices.link.mock.calls[0][1]).toMatchObject({
      expectedMasterVersion: '1',
      expectedOrderVersion: '3',
      orderId: 'order',
    });
  });

  it('恢复成功后刷新等待期间的新原因输入保留，不被旧提交关闭', async () => {
    renderMaster();
    await openRestore();
    const refresh = Promise.withResolvers<API.GetAirMasterWaybillResponse>();
    masterServices.link.mockResolvedValueOnce({
      success: true,
      data: masterResponse('2').data,
    });
    masterServices.get.mockReturnValueOnce(refresh.promise);
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() => expect(masterServices.get).toHaveBeenCalledTimes(2));
    const reason = screen.getByLabelText('关联原因');
    fireEvent.change(reason, { target: { value: '恢复后的新输入' } });
    await act(async () => refresh.resolve(masterResponse('2')));
    expect(reason).toBeVisible();
    expect(reason).toHaveValue('恢复后的新输入');
    expect(getAnyDirtyTabKey()).toBeDefined();
  });
  it.each(['共同资料', '恢复关联'])(
    '%s成功且没有新输入时官方生命周期正常关闭',
    async (operation) => {
      renderMaster();
      const response = { success: true, data: masterResponse('2').data };
      if (operation === '共同资料') {
        await openEdit();
        fireEvent.change(screen.getByLabelText('修改原因'), {
          target: { value: '修订' },
        });
        fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
        const submit = await screen.findByRole('button', {
          name: '确认影响并提交',
        });
        masterServices.update.mockResolvedValueOnce(response);
        fireEvent.click(submit);
        await waitFor(() =>
          expect(
            screen.queryByLabelText('测试主单正文'),
          ).not.toBeInTheDocument(),
        );
      } else {
        await openRestore();
        masterServices.link.mockResolvedValueOnce(response);
        fireEvent.click(screen.getByRole('button', { name: '确 定' }));
        await waitFor(() =>
          expect(screen.queryByLabelText('关联原因')).not.toBeInTheDocument(),
        );
      }
    },
  );
});
