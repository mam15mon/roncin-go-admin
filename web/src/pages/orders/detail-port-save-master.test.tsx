import { act, fireEvent, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { saveFormDraft } from '@/components/layout/formDraft';
import {
  advanceVersion,
  draftKey,
  getOrderUpdateMock,
  getPortSaveState,
  renderDetail,
  resetPortSaveState,
  save,
} from './detail-port-save.fixtures';

const state = getPortSaveState();

describe('海运详情主单正式动作与港口草稿版本', () => {
  beforeEach(resetPortSaveState);

  it('主单正式动作推进主单版本并保留未保存港口', async () => {
    renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    advanceVersion('5');
    await act(async () => {
      await state.onOrderDataChanged?.('master');
    });
    expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
    await save();
    expect(getOrderUpdateMock().mock.calls[0][1]).toMatchObject({
      originLocationId: 'unsaved-port',
      expectedVersion: '5',
      seaMasterBill: { expectedCandidateVersion: '5' },
    });
  });

  it('旧港口草稿不能借主单正式动作采纳当前版本', async () => {
    saveFormDraft(draftKey, {
      version: '4',
      seaMasterBillExpectedCandidateVersion: '2',
      originLocationId: 'draft-port',
    });
    renderDetail();
    advanceVersion('5');
    await act(async () => {
      await state.onOrderDataChanged?.('master');
    });
    getOrderUpdateMock().mockRejectedValue({
      response: { status: 409, data: { message: '主单版本冲突' } },
    });
    await save();
    expect(getOrderUpdateMock().mock.calls[0][1]).toMatchObject({
      originLocationId: 'draft-port',
      seaMasterBill: { expectedCandidateVersion: '2' },
    });
    expect(screen.getByLabelText('起运港')).toHaveValue('draft-port');
    expect(getOrderUpdateMock()).toHaveBeenCalledOnce();
    expect(state.loadData).toHaveBeenCalledOnce();
  });

  it('主单动作成功后重读包含另一笔更新时仍保留旧港口版本', async () => {
    renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    // 本次正式动作将 MBL 从 4 推进到 5，重读前另一笔写入又推进到 6。
    advanceVersion('6');
    state.order.originLocationId = 'concurrent-port';
    await act(async () => {
      await state.onOrderDataChanged?.('master');
    });
    getOrderUpdateMock().mockRejectedValue({
      response: { status: 409, data: { message: '主单版本冲突' } },
    });
    await save();
    expect(getOrderUpdateMock().mock.calls[0][1]).toMatchObject({
      originLocationId: 'unsaved-port',
      expectedVersion: '4',
      seaMasterBill: { expectedCandidateVersion: '4' },
    });
    expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
    expect(getOrderUpdateMock()).toHaveBeenCalledOnce();
    expect(state.loadData).toHaveBeenCalledOnce();
  });

  it('主单动作后仅订单普通字段被并发修改时仍保留原订单版本', async () => {
    renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    advanceVersion('5');
    // 另一笔写入只改订单客户，MBL 仍是本次主单动作产生的版本。
    state.order = {
      ...state.order,
      version: '6',
      customerId: 'concurrent-customer',
    };
    await act(async () => {
      await state.onOrderDataChanged?.('master');
    });
    getOrderUpdateMock().mockRejectedValue({
      response: { status: 409, data: { message: '订单版本冲突' } },
    });
    await save();
    expect(getOrderUpdateMock().mock.calls[0][1]).toMatchObject({
      originLocationId: 'unsaved-port',
      customerId: 'customer-1',
      expectedVersion: '4',
      seaMasterBill: { expectedCandidateVersion: '5' },
    });
    expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
    expect(getOrderUpdateMock()).toHaveBeenCalledOnce();
    expect(state.loadData).toHaveBeenCalledOnce();
  });

  it('主单版本超过安全整数范围时仍精确推进一次', async () => {
    advanceVersion('9007199254740993');
    renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    advanceVersion('9007199254740994');
    await act(async () => {
      await state.onOrderDataChanged?.('master');
    });
    await save();
    expect(getOrderUpdateMock().mock.calls[0][1]).toMatchObject({
      originLocationId: 'unsaved-port',
      seaMasterBill: { expectedCandidateVersion: '9007199254740994' },
    });
  });
});
