import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  editCargoAllocation,
  getCargoAllocationRows,
  makeSplitContext,
  pageReady,
  renderSplit,
  setupSplitCargoTests,
  splitServices,
  splitState,
} from './split-cargo.fixtures';

setupSplitCargoTests();

describe('拆票基线刷新与编辑保护', () => {
  it('取消刷新保留编辑，确认刷新才重新采纳基线并清除编辑', async () => {
    splitState.context = makeSplitContext(true);
    renderSplit();
    await pageReady();
    editCargoAllocation();
    const note = screen.getByPlaceholderText(
      '可填写本次拆票说明（将永久记录在不可变拆票事件历史中）',
    );
    fireEvent.change(note, { target: { value: '保留我的拆票说明' } });
    fireEvent.click(
      screen.getByText('刷新数据').closest('button') as HTMLButtonElement,
    );
    const cancel = (await screen.findByText('保留编辑')).closest(
      'button',
    ) as HTMLButtonElement;
    await waitFor(() => expect(cancel).toBeVisible());
    fireEvent.click(cancel);
    expect(note).toHaveValue('保留我的拆票说明');
    expect(splitServices.context).toHaveBeenCalledTimes(1);

    fireEvent.click(
      screen.getByText('刷新数据').closest('button') as HTMLButtonElement,
    );
    const confirm = (await screen.findByText('确认刷新')).closest(
      'button',
    ) as HTMLButtonElement;
    await waitFor(() => expect(confirm).toBeVisible());
    fireEvent.click(confirm);
    await waitFor(() => expect(splitServices.context).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(note).toHaveValue(''));
    const { original } = getCargoAllocationRows();
    expect(original.querySelector('input')).toHaveValue('100');
  });

  it('后台重查发现变化只提示刷新，不替换分配输入或提交基线', async () => {
    splitState.context = makeSplitContext(true);
    const { queryClient } = renderSplit();
    await pageReady();
    editCargoAllocation();
    const note = screen.getByPlaceholderText(
      '可填写本次拆票说明（将永久记录在不可变拆票事件历史中）',
    );
    fireEvent.change(note, { target: { value: '后台刷新不能覆盖' } });
    splitState.context = { ...splitState.context, orderVersion: '9' };
    await act(async () => {
      await queryClient.invalidateQueries({
        queryKey: ['order-split', 'context'],
      });
    });
    expect(await screen.findByText('拆票数据需要刷新')).toBeInTheDocument();
    expect(note).toHaveValue('后台刷新不能覆盖');
    const { original } = getCargoAllocationRows();
    expect(original.querySelector('input')).toHaveValue('60');
    expect(screen.getByText('确认执行拆票').closest('button')).toBeDisabled();
    expect(splitServices.execute).not.toHaveBeenCalled();
  });

  it('订单审核锁定时实际货物录入与拆票均不可执行', async () => {
    splitState.locked = true;
    renderSplit();
    await screen.findByText('海运出口订单已审核，如需修改请先撤销审核');
    expect(
      (await screen.findByText('录入实际货物')).closest('button'),
    ).toBeDisabled();
    expect(screen.getByText('确认执行拆票').closest('button')).toBeDisabled();
    expect(splitServices.listCargo).not.toHaveBeenCalled();
    expect(splitServices.preview).not.toHaveBeenCalled();
  });
});
