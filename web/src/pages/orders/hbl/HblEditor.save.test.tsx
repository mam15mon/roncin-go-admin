import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderEditor, save, state } from './HblEditor.fixtures';

describe('HBL 制作保存与输入保护', () => {
  it('修改保存同一 HBL，保留原始版本并提交印字、正文，成功后可继续编辑', async () => {
    renderEditor();
    fireEvent.change(screen.getByLabelText('发货人 (Shipper)'), {
      target: { value: '制作修改发货人' },
    });
    fireEvent.change(screen.getByLabelText('签发地点'), {
      target: { value: 'BEIJING' },
    });
    await save();
    expect(state.save.mock.calls[0]).toEqual([
      { orderId: 'order-1', id: 'house-1' },
      expect.objectContaining({
        expectedVersion: '3',
        expectedLinkVersion: '2',
        houseBill: expect.objectContaining({
          id: 'house-1',
          content: expect.objectContaining({ shipperText: '制作修改发货人' }),
          printDetails: expect.objectContaining({ issuePlaceText: 'BEIJING' }),
        }),
      }),
    ]);
    await waitFor(() =>
      expect(
        screen.getAllByRole('button', { name: /保存制作资料/ })[0],
      ).toBeEnabled(),
    );
    expect(screen.getByLabelText('发货人 (Shipper)')).toHaveValue(
      '制作修改发货人',
    );
  });
  it('写失败返回空响应保留输入和提交基线，刷新取消不会覆盖', async () => {
    state.save.mockResolvedValueOnce(undefined);
    renderEditor();
    const input = screen.getByLabelText('发货人 (Shipper)');
    fireEvent.change(input, { target: { value: '失败后保留' } });
    await save();
    await waitFor(() =>
      expect(
        screen.getAllByRole('button', { name: /保存制作资料/ })[0],
      ).toBeEnabled(),
    );
    fireEvent.click(screen.getByRole('button', { name: /刷新资料/ }));
    const keep = await screen.findByText('保留当前内容', {
      selector: 'button span',
    });
    await act(async () => fireEvent.click(keep));
    expect(input).toHaveValue('失败后保留');
    await save();
    expect(state.save.mock.calls[1][1]).toMatchObject({
      expectedVersion: '3',
      expectedLinkVersion: '2',
    });
  });
});
