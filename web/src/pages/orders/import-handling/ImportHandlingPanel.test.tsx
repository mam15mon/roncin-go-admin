import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OrderClosureStatus } from '@/enums.generated';
import {
  flush,
  order,
  renderPanel,
  response,
  services,
  setupTests,
} from './ImportHandlingPanel.fixtures';

setupTests();
describe('本票进口服务维护', () => {
  it('wire省略false展示不适用，承接任意子集只提交服务快照和打开版本', async () => {
    const { onSaved } = renderPanel();
    expect(await screen.findAllByText('不适用')).toHaveLength(5);
    fireEvent.click(screen.getByRole('button', { name: '维护承接与完成情况' }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '承接换单' }));
    expect(
      await within(dialog).findByRole('textbox', { name: /换单实际完成时间/ }),
    ).toHaveValue('');
    fireEvent.change(
      within(dialog).getByRole('textbox', { name: '换单参考号' }),
      { target: { value: 'EX-001' } },
    );
    const saved = response('8');
    if (saved.data?.steps?.[1])
      saved.data.steps[1] = {
        ...saved.data.steps[1],
        undertaken: true,
        referenceNo: 'EX-001',
      };
    services.save.mockResolvedValue(saved);
    services.get.mockResolvedValue(saved);
    fireEvent.click(
      within(dialog).getByRole('button', { name: '保存服务资料' }),
    );
    await waitFor(() => expect(services.save).toHaveBeenCalledTimes(1));
    expect(services.save.mock.calls[0][1]).toMatchObject({
      orderId: 'order',
      expectedVersion: '7',
      steps: expect.arrayContaining([
        {
          kind: 2,
          undertaken: true,
          occurredAt: undefined,
          responsibleUserId: undefined,
          referenceNo: 'EX-001',
          notes: '',
          attachmentIds: [],
        },
      ]),
    });
    expect(services.save.mock.calls[0][1].steps).toHaveLength(5);
    expect(services.save.mock.calls[0][1]).not.toHaveProperty('flowStatus');
    await waitFor(() =>
      expect(onSaved).toHaveBeenCalledWith(
        expect.objectContaining({
          orderId: 'order',
          previousVersion: '7',
          saved: expect.objectContaining({
            orderId: 'order',
            orderVersion: '8',
          }),
        }),
      ),
    );
  });

  it('取消承接先确认，保留完成历史需要原因，取消确认不提交', async () => {
    const initial = response();
    if (initial.data?.steps?.[1])
      initial.data.steps[1] = {
        ...initial.data.steps[1],
        undertaken: true,
        occurredAt: '2026-11-16T00:00:00Z',
        referenceNo: 'OLD',
      };
    services.get.mockResolvedValue(initial);
    renderPanel();
    const open = await screen.findByRole('button', {
      name: '维护承接与完成情况',
    });
    await waitFor(() => expect(open).toBeEnabled());
    fireEvent.click(open);
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '承接换单' }));
    fireEvent.click(
      within(dialog).getByRole('button', { name: '保存服务资料' }),
    );
    expect(
      await within(dialog).findByText('撤回、取消或纠正已完成服务必须填写原因'),
    ).toBeVisible();
    expect(services.save).not.toHaveBeenCalled();
    fireEvent.change(
      within(dialog).getByRole('textbox', { name: '变更原因' }),
      { target: { value: '客户取消委托' } },
    );
    await waitFor(() =>
      expect(
        within(dialog).queryByText('撤回、取消或纠正已完成服务必须填写原因'),
      ).toBeNull(),
    );
    fireEvent.click(
      within(dialog).getByRole('button', { name: '保存服务资料' }),
    );
    const confirmation = await screen.findAllByText('确认取消承接：换单？');
    expect(confirmation.length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '保留承接' }));
    await flush();
    expect(services.save).not.toHaveBeenCalled();
    expect(
      within(dialog).getByRole('textbox', { name: '变更原因' }),
    ).toHaveValue('客户取消委托');
  });

  it('保存失败保留输入，已有单据CLOSED必须重开，主编辑版本不匹配禁止编辑', async () => {
    const view = renderPanel();
    const open = await screen.findByRole('button', {
      name: '维护承接与完成情况',
    });
    await waitFor(() => expect(open).toBeEnabled());
    fireEvent.click(open);
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '承接派送' }));
    fireEvent.change(
      await within(dialog).findByRole('textbox', { name: '派送备注' }),
      { target: { value: '保留输入' } },
    );
    fireEvent.click(
      within(dialog).getByRole('button', { name: '保存服务资料' }),
    );
    await waitFor(() => expect(services.save).toHaveBeenCalledTimes(1));
    expect(
      within(dialog).getByRole('textbox', { name: '派送备注' }),
    ).toHaveValue('保留输入');
    view.unmount();
    renderPanel({
      order: {
        ...order,
        closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
      },
    });
    expect(
      await screen.findByText('订单已完结，请先重开订单再变更进口服务。'),
    ).toBeVisible();
    expect(
      screen.getByRole('button', { name: '维护承接与完成情况' }),
    ).toBeDisabled();
  });
});
