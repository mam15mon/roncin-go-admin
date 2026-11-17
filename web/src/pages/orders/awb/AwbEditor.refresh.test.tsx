import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { editRemarks, renderAwb, state } from './AwbEditor.fixtures';

describe('航空制作显式刷新', () => {
  it('确认覆盖后读取失败，旧输入与原基线保留', async () => {
    state.read.mockRejectedValueOnce(new Error('受控读取失败'));
    renderAwb();
    const input = await editRemarks('刷新失败保留稿');
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    fireEvent.click(await screen.findByRole('button', { name: '确认刷新' }));
    await screen.findByText('受控读取失败');
    expect(input).toHaveValue('刷新失败保留稿');
    expect(state.save).not.toHaveBeenCalled();
  });
  it('刷新等待中继续输入，迟到刷新不覆盖新稿', async () => {
    let finish: ((response: API.GetOrderResponse) => void) | undefined;
    state.read.mockImplementationOnce(
      () =>
        new Promise<API.GetOrderResponse>((resolve) => {
          finish = resolve;
        }),
    );
    renderAwb();
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    await waitFor(() => expect(state.read).toHaveBeenCalledOnce());
    const input = await editRemarks('刷新途中继续的新输入');
    const updated = structuredClone(state.bundle.order);
    if (!updated?.airExport?.house) throw new Error('测试分单缺失');
    updated.version = '8';
    updated.airExport.house.content = { remarks: '后台较新稿' };
    await act(async () => finish?.({ success: true, data: updated }));
    expect(input).toHaveValue('刷新途中继续的新输入');
    expect(state.save).not.toHaveBeenCalled();
  });
});
