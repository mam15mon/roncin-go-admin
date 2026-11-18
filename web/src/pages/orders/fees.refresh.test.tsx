import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  feeRouteState,
  feeTestState,
  renderFeesPage,
  rerenderFeesPage,
} from './fees.fixtures';

describe('订单费用页外部变动提示与刷新保护', () => {
  beforeEach(() => {
    feeRouteState.params = { kind: 'sea-export', id: 'order-A' };
    feeTestState.lockState = { isLocked: false };
    feeTestState.canCreateFee = true;
    feeTestState.canReadFee = true;
    feeTestState.canReadFinanceFees = false;
    feeTestState.canOperate = true;
    feeTestState.feeChanged = false;
  });

  it('检测到外部变动时持续提示，且不自动刷新或重建费用表', async () => {
    feeTestState.feeChanged = true;
    renderFeesPage();
    expect(
      await screen.findByText('这个订单已经发生变动，请重新刷新。'),
    ).toBeVisible();
    expect(screen.getByTestId('fee-reload-token')).toHaveTextContent('0');
  });

  it('无草稿时刷新直接重建两张费用表并同步检查参考点', async () => {
    renderFeesPage();
    await screen.findByTestId('fee-reload-token');
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    await waitFor(() =>
      expect(screen.getByTestId('fee-reload-token')).toHaveTextContent('1'),
    );
    expect(feeTestState.rebaseline).toHaveBeenCalled();
  });

  it('有未保存草稿时刷新先确认；选择继续编辑则不重建', async () => {
    renderFeesPage();
    const token = await screen.findByTestId('fee-reload-token');
    fireEvent.click(screen.getByRole('button', { name: '产生费用草稿' }));
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    expect(
      (await screen.findAllByText('刷新将丢弃未保存的费用修改')).length,
    ).toBeGreaterThan(0);
    expect(token).toHaveTextContent('0');
    fireEvent.click(await screen.findByRole('button', { name: /继续编辑/ }));
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(token).toHaveTextContent('0');
    expect(feeTestState.rebaseline).not.toHaveBeenCalled();
  });

  it('有未保存草稿时确认刷新才重建并同步检查参考点', async () => {
    renderFeesPage();
    await screen.findByTestId('fee-reload-token');
    fireEvent.click(screen.getByRole('button', { name: '产生费用草稿' }));
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));
    fireEvent.click(await screen.findByRole('button', { name: /确认刷新/ }));
    await waitFor(() =>
      expect(screen.getByTestId('fee-reload-token')).toHaveTextContent('1'),
    );
    expect(feeTestState.rebaseline).toHaveBeenCalled();
  });

  it('本人整表保存携带费用内容版本同步检查参考点', async () => {
    renderFeesPage();
    fireEvent.click(
      await screen.findByRole('button', { name: '通知整表保存' }),
    );
    expect(feeTestState.acceptOwnSave).toHaveBeenCalledWith('11');
    expect(feeTestState.rebaseline).not.toHaveBeenCalled();
    expect(feeTestState.refreshLockState).toHaveBeenCalledOnce();
    expect(screen.getByTestId('fee-reload-token')).toHaveTextContent('0');
  });

  it('保存后审核状态后台重读只关闭提交，不收起费用行', async () => {
    const { rerender } = renderFeesPage();
    expect(await screen.findByTestId('fee-rows-locked')).toHaveTextContent(
      'false',
    );
    feeTestState.lockStateLoading = true;
    feeTestState.lockStateRefreshing = true;
    rerenderFeesPage(rerender);
    expect(screen.getByTestId('fee-write-disabled')).toHaveTextContent('true');
    expect(screen.getByTestId('fee-rows-locked')).toHaveTextContent('false');

    // 首次加载（无已确认快照）仍按同步中锁定费用行。
    feeTestState.lockStateRefreshing = false;
    rerenderFeesPage(rerender);
    expect(screen.getByTestId('fee-rows-locked')).toHaveTextContent('true');
  });

  it('行内写入成功也同步审核状态而不整体重建费用表', async () => {
    renderFeesPage();
    fireEvent.click(
      await screen.findByRole('button', { name: '通知行内保存' }),
    );
    expect(feeTestState.refreshLockState).toHaveBeenCalledOnce();
    expect(feeTestState.rebaseline).toHaveBeenCalledOnce();
    expect(feeTestState.acceptOwnSave).not.toHaveBeenCalled();
    expect(screen.getByTestId('fee-reload-token')).toHaveTextContent('0');
  });

  it.each([
    { label: '快捷费目草稿', quickButton: '打开快捷费目' },
    { label: '快捷往来草稿', quickButton: '打开快捷往来单位' },
  ])(
    '审核同步及读取失败期间保留$label，真实锁定后仍关闭写入口',
    async ({ label, quickButton }) => {
      const { rerender } = renderFeesPage();
      fireEvent.click(await screen.findByRole('button', { name: quickButton }));
      const input = await screen.findByLabelText(label);
      fireEvent.change(input, { target: { value: '另一侧未保存草稿' } });
      fireEvent.click(screen.getByRole('button', { name: '通知整表保存' }));

      feeTestState.lockStateLoading = true;
      rerenderFeesPage(rerender);
      expect(screen.getByTestId('fee-write-disabled')).toHaveTextContent(
        'true',
      );
      expect(screen.getByLabelText(label)).toHaveValue('另一侧未保存草稿');
      feeTestState.lockStateLoading = false;
      feeTestState.lockState = null;
      feeTestState.lockStateError = new Error('读取审核状态失败');
      rerenderFeesPage(rerender);
      expect(screen.getByTestId('fee-write-disabled')).toHaveTextContent(
        'true',
      );
      expect(screen.getByLabelText(label)).toHaveValue('另一侧未保存草稿');
      feeTestState.lockStateError = null;
      feeTestState.lockState = { isLocked: false };
      rerenderFeesPage(rerender);
      expect(screen.getByTestId('fee-write-disabled')).toHaveTextContent(
        'false',
      );
      expect(screen.getByLabelText(label)).toHaveValue('另一侧未保存草稿');
      expect(screen.getByTestId('fee-reload-token')).toHaveTextContent('0');

      feeTestState.lockState = { isLocked: true };
      rerenderFeesPage(rerender);
      await waitFor(() =>
        expect(screen.queryByLabelText(label)).not.toBeInTheDocument(),
      );
    },
  );
});
