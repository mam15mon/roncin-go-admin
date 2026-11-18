import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import {
  editRemarks,
  outputAwb,
  renderAwb,
  saveAwb,
  state,
} from './AwbEditor.fixtures';
import { savedHouseOrder } from './model.fixtures';

describe('航空制作保存与输入保护', () => {
  it('只编辑修改原因时也登记实时未保存保护', () => {
    renderAwb('MAWB');
    fireEvent.change(screen.getByLabelText('修改原因'), {
      target: { value: '尚未提交的更正说明' },
    });
    expect(getAnyDirtyTabKey()).toBeTruthy();
  });
  it('切换后重挂恢复正文并立即提示已加载的外部变动', async () => {
    const first = renderAwb();
    await editRemarks('仍打开页签的草稿');
    first.unmount();
    state.bundle = {
      ...state.bundle,
      order: { ...state.bundle.order, version: '9' },
      master: { ...state.bundle.master, version: '8' },
    };
    renderAwb();
    fireEvent.click(screen.getByText('签发与备注'));
    expect(screen.getByLabelText('单证备注')).toHaveValue('仍打开页签的草稿');
    expect(
      screen.getByText('这个航空运单已经发生变动，请重新刷新。'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '保存正文' })).toBeDisabled();
    expect(state.save).not.toHaveBeenCalled();
  });
  it('MAWB同版本重挂只比较独立主单，不把来源订单误判为变化', async () => {
    const first = renderAwb('MAWB');
    await editRemarks('主单临时正文');
    first.unmount();
    renderAwb('MAWB');
    fireEvent.click(screen.getByText('签发与备注'));
    expect(screen.getByLabelText('单证备注')).toHaveValue('主单临时正文');
    expect(
      screen.queryByText('这个航空运单已经发生变动，请重新刷新。'),
    ).toBeNull();
    expect(screen.getByRole('button', { name: '保存正文' })).toBeEnabled();
  });
  it('HAWB保存只提交明确正文和旧基线，独立保存后才可输出', async () => {
    state.save.mockImplementation(
      async (_params, payload: API.UpdateOrderRequest) => {
        const saved = savedHouseOrder(
          state.bundle,
          payload.airExport?.house?.content ?? {},
        );
        state.bundle = { ...state.bundle, order: saved };
        return { success: true, data: saved };
      },
    );
    renderAwb();
    await editRemarks('本票新正文');
    saveAwb();
    await waitFor(() => expect(state.save).toHaveBeenCalledOnce());
    expect(state.save.mock.calls[0][1]).toMatchObject({
      expectedVersion: '4',
      airExport: {
        expectedMasterVersion: '2',
        expectedLinkVersion: '3',
        house: { expectedVersion: '5', content: { remarks: '本票新正文' } },
      },
    });
    expect(state.save.mock.calls[0][1]).not.toHaveProperty('notes');
    expect(state.save.mock.calls[0][1].airExport).not.toHaveProperty('master');
    await screen.findByText('航空运单正文已保存');
    outputAwb();
    await screen.findByRole('dialog', { name: '航空PDF预览' });
    expect(state.preview).toHaveBeenLastCalledWith(
      expect.objectContaining({
        version: '6',
        content: expect.objectContaining({ remarks: '本票新正文' }),
      }),
    );
  });
  it('409/失败保留输入且不能生成新稿', async () => {
    state.save.mockRejectedValueOnce(new Error('已被更新，请刷新后重试'));
    renderAwb();
    const input = await editRemarks('冲突草稿');
    saveAwb();
    await screen.findByText('请刷新后核对保存结果，当前输入已保留。');
    expect(input).toHaveValue('冲突草稿');
    expect(
      screen.getByRole('button', { name: '预览 / 打印 / 下载' }),
    ).toBeDisabled();
    expect(state.preview).not.toHaveBeenCalled();
  });
  it('成功响应含后续版本时不借新基线或覆盖旧输入', async () => {
    state.save.mockImplementationOnce(
      async (_params, payload: API.UpdateOrderRequest) => {
        const saved = savedHouseOrder(
          state.bundle,
          payload.airExport?.house?.content ?? {},
        );
        saved.version = '6';
        return { success: true, data: saved };
      },
    );
    renderAwb();
    const input = await editRemarks('旧版本草稿');
    saveAwb();
    await screen.findByText('请刷新后核对保存结果，当前输入已保留。');
    expect(input).toHaveValue('旧版本草稿');
    expect(state.preview).not.toHaveBeenCalled();
  });
  it('保存途中继续输入，迟到成功不覆盖新稿', async () => {
    let finish: ((response: API.UpdateOrderResponse) => void) | undefined;
    state.save.mockImplementationOnce(
      () =>
        new Promise<API.UpdateOrderResponse>((resolve) => {
          finish = resolve;
        }),
    );
    renderAwb();
    await editRemarks('提交中的旧稿');
    saveAwb();
    await waitFor(() => expect(state.save).toHaveBeenCalledOnce());
    const sent = state.save.mock.calls[0][1] as API.UpdateOrderRequest;
    const input = await editRemarks('保存途中继续的新稿');
    await act(async () =>
      finish?.({
        success: true,
        data: savedHouseOrder(
          state.bundle,
          sent.airExport?.house?.content ?? {},
        ),
      }),
    );
    await screen.findByText('请刷新后核对保存结果，当前输入已保留。');
    expect(input).toHaveValue('保存途中继续的新稿');
    expect(state.preview).not.toHaveBeenCalled();
  });
  it('MAWB先确认共享影响，保持打开的运输副本并采用独立主单基线', async () => {
    state.previewImpact.mockResolvedValueOnce({
      success: true,
      canExecute: true,
      inputFingerprint: 'impact-1',
      members: [{ orderNo: 'AE001' }],
    });
    state.saveMaster.mockImplementationOnce(
      async (_params, payload: API.UpdateAirMasterWaybillRequest) => {
        const master = {
          ...state.bundle.master,
          content: payload.input,
          version: '3',
          contentHash: 'saved-master',
        };
        state.bundle.master = master;
        return { success: true, data: master };
      },
    );
    renderAwb('MAWB');
    await editRemarks('共享主单新正文');
    fireEvent.change(screen.getByLabelText('修改原因'), {
      target: { value: '更正印字' },
    });
    saveAwb();
    await screen.findByText('影响 1 票订单');
    expect(state.saveMaster).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '确认影响并保存' }));
    await screen.findByText('航空运单正文已保存');
    expect(state.saveMaster.mock.calls[0][1]).toMatchObject({
      expectedVersion: '2',
      inputFingerprint: 'impact-1',
      input: {
        masterNo: '78112345675',
        originAirportId: 'airport-1',
        waybillContent: { remarks: '共享主单新正文' },
      },
    });
    outputAwb();
    await screen.findByRole('dialog', { name: '航空PDF预览' });
    expect(state.preview).toHaveBeenLastCalledWith(
      expect.objectContaining({ target: 'MAWB', version: '3' }),
    );
  });
});
