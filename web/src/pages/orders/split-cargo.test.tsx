import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  addActualCargo,
  editCargoAllocation,
  renderSplit,
  setupSplitCargoTests,
  splitServices,
  splitState,
  submitActualCargo,
} from './split-cargo.fixtures';

setupSplitCargoTests();

describe('拆票实际货物录入', () => {
  it('已有实际箱但无货物时解释来源，补录刷新后可分配并执行守恒拆票', async () => {
    renderSplit();
    expect(
      await screen.findByText('尚未登记可分配的实际货物明细'),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/委托件重尺和实际箱记录不会自动生成货物明细/),
    ).toBeInTheDocument();
    expect(screen.getByText('MSCU1234567')).toBeInTheDocument();
    expect(screen.getByText('确认执行拆票').closest('button')).toBeDisabled();

    await addActualCargo();
    // 成功写入只标记基线过期，不通过后台重查直接丢弃当前分配。
    expect(splitServices.context).toHaveBeenCalledTimes(1);
    fireEvent.click(
      screen.getByText('刷新拆票数据').closest('button') as HTMLButtonElement,
    );
    await screen.findByText('测试机器');
    editCargoAllocation();

    const execute = screen
      .getByText('确认执行拆票')
      .closest('button') as HTMLButtonElement;
    await waitFor(() => expect(execute).toBeEnabled());
    fireEvent.click(execute);
    const confirm = (await screen.findByText('确认执行')).closest(
      'button',
    ) as HTMLButtonElement;
    await waitFor(() => expect(confirm).toBeVisible());
    fireEvent.click(confirm);

    await waitFor(() =>
      expect(splitServices.execute).toHaveBeenCalledWith(
        { orderId: 'split-order' },
        expect.objectContaining({
          expectedVersions: expect.objectContaining({
            orderVersion: '2',
            cargoItemVersions: { 'cargo-1': '1' },
          }),
          results: [
            expect.objectContaining({
              resultRole: 'ORIGINAL',
              cargoAllocations: [
                {
                  cargoItemId: 'cargo-1',
                  packageCount: 60,
                  grossWeightKg: '1200',
                  volumeCbm: '9',
                },
              ],
            }),
            expect.objectContaining({
              resultRole: 'CREATED',
              cargoAllocations: [
                {
                  cargoItemId: 'cargo-1',
                  packageCount: 40,
                  grossWeightKg: '800.000',
                  volumeCbm: '6.000000',
                },
              ],
            }),
          ],
        }),
      ),
    );
    await waitFor(() =>
      expect(splitServices.push).toHaveBeenCalledWith(
        '/orders/sea-export/split-order',
      ),
    );
  });

  it('没有货物权限时说明补录方式并禁止打开实际货物接口', async () => {
    splitState.cargoAccess = false;
    renderSplit();
    expect(
      await screen.findByText(
        '当前没有实际货物录入权限，请联系有权限的操作人员补录。',
      ),
    ).toBeInTheDocument();
    const open = screen
      .getByText('录入实际货物')
      .closest('button') as HTMLButtonElement;
    expect(open).toBeDisabled();
    fireEvent.click(open);
    expect(splitServices.listCargo).not.toHaveBeenCalled();
    expect(splitServices.addCargo).not.toHaveBeenCalled();
  });

  it('货物写入失败保留表单输入，不标记成功或刷新拆票基线', async () => {
    splitServices.addCargo.mockResolvedValue(undefined);
    renderSplit();
    const dialog = await submitActualCargo();
    await waitFor(() =>
      expect(
        within(dialog).getByRole('button', { name: '确 定' }),
      ).not.toHaveClass('ant-btn-loading'),
    );
    expect(dialog).toBeVisible();
    expect(within(dialog).getByLabelText('货物名称')).toHaveValue('测试机器');
    expect(screen.queryByText('拆票数据需要刷新')).not.toBeInTheDocument();
    expect(splitServices.context).toHaveBeenCalledTimes(1);
    expect(splitState.context.cargoItems).toEqual([]);
  });
});
