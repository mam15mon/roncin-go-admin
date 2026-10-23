import { act, fireEvent, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  billedFeeC,
  listFees,
  makeFee,
  makeProps,
  renderTabs,
  resetBulkActionMocks,
  unbilledFeeA,
} from './OrderFeeTableTabs.bulkActions.fixtures';

describe('OrderFeeTableTabs 选行与建账资格', () => {
  beforeEach(resetBulkActionMocks);

  it('全选只收集已保存有效行：未保存新行复选框禁用（A2 回归）', async () => {
    listFees.mockResolvedValue({
      data: [
        unbilledFeeA,
        makeFee({ id: 'new_1700000000000', feeName: '未保存新行' }),
      ],
    } as any);
    const { props } = makeProps('order-1');
    await renderTabs(props);
    await screen.findByText('海运费');

    const rowCheckbox = (rowText: string) =>
      within(
        screen.getByText(rowText).closest('tr') as HTMLTableRowElement,
      ).getByRole('checkbox');

    // 未保存行（new_ 前缀且无版本）不可勾选
    expect(rowCheckbox('未保存新行')).toBeDisabled();
    expect(rowCheckbox('海运费')).not.toBeDisabled();

    // sticky 将表头与表体放入不同 table，按当前业务表格容器定位全选入口。
    const tableWrapper = screen
      .getByText('海运费')
      .closest('.ant-table-wrapper') as HTMLElement;
    const header = tableWrapper.querySelector('thead');
    expect(header).not.toBeNull();
    const selectAll = within(header as HTMLTableSectionElement).getByRole(
      'checkbox',
    );
    await act(async () => {
      fireEvent.click(selectAll);
    });

    // rowSelection.onChange 首参为选中行 key 集合：全选只收集可勾选的已保存行
    expect(props.setSelectedReceivableFeeIds.mock.calls.at(-1)?.[0]).toEqual([
      'fee-a',
    ]);
  });

  it('生成账单：勾选含已建账行时显式拒绝并列出不合格笔数', async () => {
    listFees.mockResolvedValue({
      data: [unbilledFeeA, billedFeeC],
    } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a', 'fee-c'];
    await renderTabs(props);
    await screen.findByText('海运费');

    await act(async () => {
      screen.getByRole('button', { name: /生成账单（2）/ }).click();
    });

    expect(
      await screen.findByText(
        /选中费用中有 1 笔已建账，不能生成账单（已建账港杂费），请取消勾选后重试/,
      ),
    ).toBeVisible();
    expect(props.onOpenBillWorkbench).not.toHaveBeenCalled();
  });
  it('补审只阻断服务端标记的原费用，同票其他费用仍可建账', async () => {
    const pendingFee = makeFee({
      id: 'pending-source',
      version: '1',
      feeName: '待补审原费用',
      canEdit: false,
      canBill: false,
      hasPendingChange: true,
      blockedReasons: ['该费用有待审补录'],
    });
    listFees.mockResolvedValue({ data: [unbilledFeeA, pendingFee] } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a'];
    await renderTabs(props);
    await screen.findByText('待补审原费用');
    fireEvent.click(screen.getByRole('button', { name: /生成账单（1）/ }));
    expect(props.onOpenBillWorkbench).toHaveBeenCalledWith(['fee-a']);
  });
});
