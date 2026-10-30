import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  getFormDraft,
  hasFormDraft,
  saveFormDraft,
} from '@/components/layout/formDraft';
import { orderErrorReasons } from '@/errorReasons.generated';
import {
  advanceVersion,
  draftKey,
  getOrderUpdateMock,
  getPortSaveState,
  ports,
  renderDetail,
  resetPortSaveState,
  save,
} from './detail-port-save.fixtures';
import type { OrderDetailFormValues } from './order-kinds/form-values';

const state = getPortSaveState();

describe('海运详情修改港口的版本与草稿', () => {
  beforeEach(resetPortSaveState);

  it.each(ports)(
    '修改 %s 时提交未注册的主单版本并成功保存',
    async (name, label) => {
      renderDetail();
      fireEvent.change(screen.getByLabelText(label), {
        target: { value: 'port-new' },
      });
      await save();
      await waitFor(() => expect(state.loadData).toHaveBeenCalledOnce());
      expect(getOrderUpdateMock()).toHaveBeenCalledWith(
        { id: 'order-1' },
        expect.objectContaining({
          expectedVersion: '4',
          [name]: 'port-new',
          seaMasterBill: expect.objectContaining({
            expectedCandidateVersion: '4',
          }),
        }),
        { skipErrorHandler: true },
      );
      expect(hasFormDraft(draftKey)).toBe(false);
    },
  );

  it('连续保存使用成功刷新后的主单版本', async () => {
    getOrderUpdateMock().mockImplementation(async (_params, payload) => {
      state.order = {
        ...state.order,
        originLocationId: payload.originLocationId,
      };
      advanceVersion(String(Number(state.order.version) + 1));
      return { data: { id: 'order-1' } };
    });
    renderDetail();
    for (const port of ['port-first', 'port-second']) {
      fireEvent.change(screen.getByLabelText('起运港'), {
        target: { value: port },
      });
      await save();
      await waitFor(() =>
        expect(screen.getByRole('button', { name: '保存修改' })).toBeEnabled(),
      );
    }
    expect(
      getOrderUpdateMock().mock.calls.map(([, payload]) => ({
        port: payload.originLocationId,
        orderVersion: payload.expectedVersion,
        masterVersion: payload.seaMasterBill?.expectedCandidateVersion,
      })),
    ).toEqual([
      { port: 'port-first', orderVersion: '4', masterVersion: '4' },
      { port: 'port-second', orderVersion: '5', masterVersion: '5' },
    ]);
  });

  it('恢复旧版本草稿仍由服务端拒绝，保留港口输入且不自动重试', async () => {
    saveFormDraft(draftKey, {
      version: '4',
      seaMasterBillExpectedCandidateVersion: '2',
      originLocationId: 'draft-port',
    });
    getOrderUpdateMock().mockRejectedValue({
      response: {
        status: 409,
        data: {
          reason: orderErrorReasons.SEA_MASTER_BILL_STATUS_CONFLICT,
          message: '海运主单状态或版本已被修改，请刷新后重试',
        },
      },
    });
    renderDetail();
    await waitFor(() =>
      expect(screen.getByLabelText('起运港')).toHaveValue('draft-port'),
    );
    await save();
    expect(
      await screen.findAllByText('这个订单已经发生变动，请重新刷新。'),
    ).not.toHaveLength(0);
    expect(
      getOrderUpdateMock().mock.calls[0][1].seaMasterBill
        ?.expectedCandidateVersion,
    ).toBe('2');
    expect(getOrderUpdateMock()).toHaveBeenCalledOnce();
    expect(state.loadData).not.toHaveBeenCalled();
    expect(screen.getByLabelText('起运港')).toHaveValue('draft-port');
    expect(
      getFormDraft<Partial<OrderDetailFormValues>>(draftKey),
    ).toMatchObject({
      seaMasterBillExpectedCandidateVersion: '2',
      originLocationId: 'draft-port',
    });
  });

  it('缺少原主单版本的草稿不能借用最新版本，明确重置后才可保存', async () => {
    saveFormDraft(draftKey, { version: '4', originLocationId: 'draft-port' });
    renderDetail();
    await save();
    expect(
      await screen.findByText('订单草稿信息不完整，输入已保留，请刷新后重试。'),
    ).toBeInTheDocument();
    expect(getOrderUpdateMock()).not.toHaveBeenCalled();
    expect(screen.getByLabelText('起运港')).toHaveValue('draft-port');
    expect(hasFormDraft(draftKey)).toBe(true);
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '重置修改' }));
    });
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'reset-port' },
    });
    await save();
    expect(
      getOrderUpdateMock().mock.calls[0][1].seaMasterBill
        ?.expectedCandidateVersion,
    ).toBe('4');
  });

  it('显式刷新清除旧草稿后使用当前主单版本', async () => {
    saveFormDraft(draftKey, {
      version: '4',
      seaMasterBillExpectedCandidateVersion: '2',
      originLocationId: 'draft-port',
    });
    renderDetail();
    advanceVersion('5');
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '刷新数据' }));
    });
    const dialog = await screen.findByRole('dialog');
    await act(async () => {
      fireEvent.click(within(dialog).getByRole('button', { name: '确定刷新' }));
    });
    expect(screen.getByLabelText('起运港')).toHaveValue('origin-old');
    expect(hasFormDraft(draftKey)).toBe(false);
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'refreshed-port' },
    });
    await save();
    expect(getOrderUpdateMock().mock.calls[0][1]).toMatchObject({
      originLocationId: 'refreshed-port',
      expectedVersion: '5',
      seaMasterBill: { expectedCandidateVersion: '5' },
    });
  });

  it.each(['mode', 'house'] as const)(
    '%s 动作不采纳其他写入推进的主单版本',
    async (document) => {
      renderDetail();
      fireEvent.change(screen.getByLabelText('起运港'), {
        target: { value: 'unsaved-port' },
      });
      // 模式和分单动作不校验或推进 MBL；查询晚到的新版本不能替换编辑基线。
      advanceVersion('5');
      await act(async () => {
        await state.onOrderDataChanged?.(document);
      });
      getOrderUpdateMock().mockRejectedValue({
        response: { status: 409, data: { message: '主单版本冲突' } },
      });
      await save();
      expect(getOrderUpdateMock().mock.calls[0][1]).toMatchObject({
        originLocationId: 'unsaved-port',
        expectedVersion: '5',
        seaMasterBill: { expectedCandidateVersion: '4' },
      });
      expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
      expect(state.loadData).toHaveBeenCalledOnce();
    },
  );
});
