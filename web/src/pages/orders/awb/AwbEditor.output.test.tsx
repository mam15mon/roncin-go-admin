import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  editRemarks,
  makeImportBundle,
  outputAwb,
  renderAwb,
  state,
} from './AwbEditor.fixtures';

describe('航空制作输出前复核', () => {
  it('未保存稿明确阻止输出，刷新取消保留输入', async () => {
    renderAwb();
    const input = await editRemarks('未保存稿');
    outputAwb();
    await screen.findByText('请先保存正文后再输出。');
    expect(state.preview).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    await screen.findAllByText('刷新会覆盖尚未保存的航空运单正文，是否继续？');
    fireEvent.click(screen.getByRole('button', { name: '保留编辑' }));
    expect(input).toHaveValue('未保存稿');
  });
  it('生成前公司英文名变更即使Master版本未变仍阻止旧稿输出', async () => {
    renderAwb();
    state.read.mockResolvedValueOnce({
      success: true,
      data: {
        ...state.bundle.order,
        airExport: {
          ...state.bundle.order?.airExport,
          master: {
            ...state.bundle.master,
            organizationNameEn: 'CHANGED REAL COMPANY',
          },
        },
      },
    });
    outputAwb();
    await screen.findByText('这个航空运单已经发生变动，请重新刷新。');
    expect(state.preview).not.toHaveBeenCalled();
  });
  it('读取权限可输出保存快照但没有编辑按钮', async () => {
    state.canUpdate = false;
    renderAwb();
    expect(
      screen.queryByRole('button', { name: '保存正文' }),
    ).not.toBeInTheDocument();
    outputAwb();
    expect(
      (await screen.findByText('关闭预览')).closest('[role="dialog"]'),
    ).toBeVisible();
  });
  it('MAWB无经营公司办理权不开放保存', () => {
    state.canOperate = false;
    renderAwb('MAWB');
    expect(
      screen.queryByRole('button', { name: '保存正文' }),
    ).not.toBeInTheDocument();
  });
  it('进口分单只输出已保存外部印发人，缺失时仍可维护', async () => {
    makeImportBundle();
    renderAwb('HAWB', { businessType: OrderBusinessType.BUSINESS_TYPE_AI });
    expect(
      screen.getByText('预览 / 打印 / 下载').closest('button'),
    ).toBeDisabled();
    const input = await editRemarks('可维护的外部归档稿');
    expect(input).toBeEnabled();
    expect(state.preview).not.toHaveBeenCalled();
  });
  it('旧组件卸载后的迟到生成前检查不触发输出，A→B→A重新装载独立快照', async () => {
    let finish: ((response: API.GetOrderResponse) => void) | undefined;
    state.read.mockImplementationOnce(
      () =>
        new Promise<API.GetOrderResponse>((resolve) => {
          finish = resolve;
        }),
    );
    const first = renderAwb();
    outputAwb();
    await waitFor(() => expect(state.read).toHaveBeenCalledOnce());
    first.unmount();
    renderAwb();
    await act(async () =>
      finish?.({ success: true, data: state.bundle.order }),
    );
    expect(state.preview).not.toHaveBeenCalled();
    outputAwb();
    expect(
      (await screen.findByText('关闭预览')).closest('[role="dialog"]'),
    ).toBeVisible();
  });
});
