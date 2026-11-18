import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { editRemarks, outputAwb, renderAwb, state } from './AwbEditor.fixtures';

describe('航空制作只读草稿与保存快照输出', () => {
  it('只读重挂显示并保护原草稿，输出最新已保存正文', async () => {
    const first = renderAwb();
    await editRemarks('只可显示的未保存草稿');
    first.unmount();
    state.canUpdate = false;
    if (state.bundle.order) {
      state.bundle.order.version = '8';
      if (state.bundle.order.airExport?.house) {
        state.bundle.order.airExport.house.version = '9';
        state.bundle.order.airExport.house.content = {
          remarks: '当前已保存正文',
        };
      }
    }
    const restored = renderAwb();
    fireEvent.click(screen.getByText('签发与备注'));
    expect(screen.getByLabelText('单证备注')).toHaveValue(
      '只可显示的未保存草稿',
    );
    expect(screen.getByLabelText('单证备注')).toBeDisabled();
    outputAwb();
    expect(
      (await screen.findByText('关闭预览')).closest('[role="dialog"]'),
    ).toBeVisible();
    expect(state.preview.mock.calls.at(-1)?.[0].content.remarks).toBe(
      '当前已保存正文',
    );
    expect(state.save).not.toHaveBeenCalled();
    expect(screen.getByLabelText('单证备注')).toHaveValue(
      '只可显示的未保存草稿',
    );
    const observed = restored.client
      .getQueryCache()
      .find({ queryKey: ['air-orders', 'awb-check'], exact: false });
    expect(observed).toBeDefined();
    await act(async () => {
      restored.client.setQueryData(observed?.queryKey ?? [], {
        ...state.bundle,
        order: { ...state.bundle.order, version: '20' },
      });
    });
    await waitFor(() => expect(screen.queryByText('关闭预览')).toBeNull());
  });

  it('只读草稿存在时输出前发现新外变仍禁止输出', async () => {
    const first = renderAwb();
    await editRemarks('不允许输出的草稿');
    first.unmount();
    state.canUpdate = false;
    renderAwb();
    state.read.mockResolvedValueOnce({
      success: true,
      data: { ...state.bundle.order, version: '18' },
    });
    outputAwb();
    await screen.findAllByText('这个航空运单已经发生变动，请重新刷新。');
    expect(state.preview).not.toHaveBeenCalled();
    expect(
      screen.getByText('预览 / 打印 / 下载').closest('button'),
    ).toBeDisabled();
  });
});
