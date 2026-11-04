import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { preview, renderEditor, state } from './HblEditor.fixtures';

describe('HBL 签发公司抬头变动', () => {
  it('签发英文名变化保留输入并要求确认刷新，补齐后才恢复输出', async () => {
    state.bundle.documents.houseBill = {
      ...state.bundle.documents.houseBill,
      issuerOrganizationNameEn: undefined,
    };
    const { queryClient } = renderEditor();
    const input = screen.getByLabelText('发货人 (Shipper)');
    fireEvent.change(input, { target: { value: '未保存输入' } });
    state.bundle = {
      ...state.bundle,
      documents: {
        ...state.bundle.documents,
        houseBill: {
          ...state.bundle.documents.houseBill,
          issuerOrganizationNameEn: 'New Confirmed Company Co., Ltd',
        },
      },
    };
    await act(async () =>
      queryClient.refetchQueries({ queryKey: ['orders', 'hbl-change-notice'] }),
    );
    expect(
      await screen.findByText('这个订单已经发生变动，请重新刷新。'),
    ).toBeVisible();
    expect(input).toHaveValue('未保存输入');
    fireEvent.click(screen.getByRole('button', { name: /刷新资料/ }));
    await waitFor(() =>
      expect(
        screen.getByRole('dialog', { name: '确认替换当前内容？' }),
      ).toBeVisible(),
    );
    fireEvent.click(screen.getByRole('button', { name: '保留当前内容' }));
    expect(input).toHaveValue('未保存输入');
    fireEvent.click(screen.getByRole('button', { name: /刷新资料/ }));
    fireEvent.click(await screen.findByRole('button', { name: '确定替换' }));
    await screen.findByDisplayValue('原发货人');
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0].model.issuerNameEn).toBe(
      'New Confirmed Company Co., Ltd',
    );
  });

  it('生成前发现抬头变化则拒绝旧基线，已开预览在变化提醒时关闭', async () => {
    const { queryClient } = renderEditor();
    await preview();
    state.bundle = {
      ...state.bundle,
      documents: {
        ...state.bundle.documents,
        houseBill: {
          ...state.bundle.documents.houseBill,
          issuerOrganizationNameEn: 'Updated Confirmed Issuer Co., Ltd',
        },
      },
    };
    await act(async () =>
      queryClient.refetchQueries({ queryKey: ['orders', 'hbl-change-notice'] }),
    );
    expect(
      await screen.findByText('这个订单已经发生变动，请重新刷新。'),
    ).toBeVisible();
    expect(screen.queryByRole('dialog', { name: 'HBL 预览' })).toBeNull();
    state.preview.mockClear();
    await act(async () =>
      fireEvent.click(
        screen.getAllByRole('button', { name: /预览／打印／下载/ })[0],
      ),
    );
    expect(state.preview).not.toHaveBeenCalled();
  });
});
