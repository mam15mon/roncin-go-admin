import { act, fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  getFormDraft,
  getFormDraftKey,
  saveFormDraft,
} from '@/components/layout/formDraft';
import { preview, renderEditor, state } from './HblEditor.fixtures';
import { buildHblFormValues } from './model';

describe('HBL 制作输出边界', () => {
  it('审核锁单也能选择正本和电放输出，不要求先保存初始化印字', async () => {
    state.locked = true;
    renderEditor();
    expect(
      screen.getAllByRole('button', { name: /保存制作资料/ })[0],
    ).toBeDisabled();
    expect(screen.getByRole('button', { name: '返回订单' })).toBeEnabled();
    fireEvent.click(
      screen.getByText('正本', { selector: '.ant-segmented-item-label' }),
    );
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0]).toMatchObject({
      variant: 'original',
      model: { houseNo: 'HBL001', printDetails: { bookingNoText: 'BOOK001' } },
    });
    fireEvent.click(screen.getByRole('button', { name: '关闭预览' }));
    fireEvent.click(
      screen.getByText('电放', { selector: '.ant-segmented-item-label' }),
    );
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0].variant).toBe('surrendered');
    fireEvent.click(screen.getByRole('button', { name: '关闭预览' }));
    fireEvent.click(
      screen.getByText('副本', { selector: '.ant-segmented-item-label' }),
    );
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0].variant).toBe('copy');
    fireEvent.click(screen.getByRole('button', { name: '关闭预览' }));
    fireEvent.click(
      screen.getByText('草稿', { selector: '.ant-segmented-item-label' }),
    );
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0].variant).toBe('draft');
    expect(state.save).not.toHaveBeenCalled();
  });
  it('已锁定时不恢复待保存草稿，仍可输出已保存正文并保留原草稿', async () => {
    const draftKey = getFormDraftKey(
      '/orders/sea-export',
      '/orders/sea-export/order-1/hbl',
      'user-1:org-1',
    );
    saveFormDraft(draftKey, {
      ...buildHblFormValues(state.bundle),
      content: { shipperText: '不可用作已保存内容的草稿' },
    });
    state.locked = true;
    renderEditor();
    await preview();
    expect(state.preview.mock.calls.at(-1)?.[0].model.content.shipperText).toBe(
      '原发货人',
    );
    expect(getFormDraft(draftKey)).toMatchObject({
      content: { shipperText: '不可用作已保存内容的草稿' },
    });
    expect(state.save).not.toHaveBeenCalled();
  });
  it('外部 HBL 版本变化只提醒，保留输入并阻止输出旧资料', async () => {
    const { queryClient } = renderEditor();
    const input = screen.getByLabelText('发货人 (Shipper)');
    fireEvent.change(input, { target: { value: '尚未保存' } });
    state.bundle = {
      ...state.bundle,
      documents: {
        ...state.bundle.documents,
        linkVersion: '8',
        houseBill: { ...state.bundle.documents.houseBill, version: '9' },
      },
    };
    await act(async () =>
      queryClient.refetchQueries({
        queryKey: ['orders', 'hbl-change-notice'],
        type: 'all',
      }),
    );
    expect(
      await screen.findByText('这个订单已经发生变动，请重新刷新。'),
    ).toBeVisible();
    expect(input).toHaveValue('尚未保存');
    await act(async () =>
      fireEvent.click(
        screen
          .getAllByText('预览／打印／下载')[0]
          .closest('button') as HTMLElement,
      ),
    );
    expect(state.preview).not.toHaveBeenCalled();
    expect(input).toHaveValue('尚未保存');
  });
});
