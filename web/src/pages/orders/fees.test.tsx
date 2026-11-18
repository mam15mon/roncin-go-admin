import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import {
  feeRouteState,
  feeTestState,
  renderFeesPage,
  rerenderFeesPage,
} from './fees.fixtures';

describe('订单费用页跨订单状态隔离', () => {
  const listTaxableServices = feeTestState.listTaxableServices;

  beforeEach(() => {
    feeRouteState.params = { kind: 'sea-export', id: 'order-A' };
    feeTestState.lockState = { isLocked: false };
    feeTestState.canCreateFee = true;
    feeTestState.canReadFee = true;
    feeTestState.canLock = true;
    feeTestState.feeOptions.mockClear();
    feeTestState.canOperate = true;
    feeTestState.hasPendingSupplement = false;
    vi.clearAllMocks();
  });

  it('A 的快捷费目迟到响应不得覆盖 B 当前弹窗数据', async () => {
    let resolveA!: (value: any) => void;
    let resolveB!: (value: any) => void;
    const responseA = new Promise<any>((resolve) => {
      resolveA = resolve;
    });
    const responseB = new Promise<any>((resolve) => {
      resolveB = resolve;
    });
    listTaxableServices
      .mockImplementationOnce(() => responseA)
      .mockImplementationOnce(() => responseB);

    const { rerender } = renderFeesPage();
    fireEvent.click(screen.getByRole('button', { name: '打开快捷费目' }));
    await waitFor(() => expect(listTaxableServices).toHaveBeenCalledTimes(1));

    feeRouteState.params = { kind: 'sea-export', id: 'order-B' };
    rerenderFeesPage(rerender);
    fireEvent.click(screen.getByRole('button', { name: '打开快捷费目' }));
    await waitFor(() => expect(listTaxableServices).toHaveBeenCalledTimes(2));

    resolveB({ data: [{ id: 'taxable-B' }] });
    await waitFor(() =>
      expect(screen.getByTestId('quick-fee-modal')).toHaveTextContent(
        'true|taxable-B',
      ),
    );

    resolveA({ data: [{ id: 'taxable-A' }] });
    await waitFor(() =>
      expect(screen.getByTestId('quick-fee-modal')).toHaveTextContent(
        'true|taxable-B',
      ),
    );
    expect(screen.getByTestId('quick-fee-modal')).not.toHaveTextContent(
      'taxable-A',
    );
  });

  it('同一页面实例从 A 切到 B 时关闭工作台并清空 A 的费用编辑状态', async () => {
    const { rerender } = renderFeesPage();

    fireEvent.click(screen.getByRole('button', { name: '准备当前订单费用' }));
    fireEvent.click(screen.getByRole('button', { name: '打开快捷费目' }));
    fireEvent.click(screen.getByRole('button', { name: '打开快捷往来单位' }));

    expect(screen.getByTestId('fee-table-state')).toHaveTextContent(
      'order-A|receivable-order-A|payable-order-A',
    );
    expect(screen.getByTestId('fee-summary')).toHaveTextContent('1:120|1:20');
    expect(screen.getByTestId('finance-summary-board')).toHaveTextContent(
      'true|2|120,20',
    );
    expect(screen.getByTestId('bill-workbench')).toHaveTextContent(
      'true|receivable-order-A|organization-order-A|订单 ORDER-A',
    );
    expect(
      screen.getByRole('button', { name: '提交账单' }),
    ).toBeInTheDocument();
    expect(screen.getByTestId('quick-fee-modal')).toHaveTextContent('true');
    // 从应付侧发起：默认往来角色为供应商。
    expect(screen.getByTestId('quick-partner-modal')).toHaveTextContent(
      `true|${PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}`,
    );

    feeRouteState.params = { kind: 'sea-export', id: 'order-B' };
    rerenderFeesPage(rerender);

    await waitFor(() => {
      expect(screen.getByTestId('fee-table-state')).toHaveTextContent(
        'order-B||',
      );
      expect(screen.getByTestId('fee-summary')).toHaveTextContent('0:0|0:0');
      expect(screen.getByTestId('finance-summary-board')).toHaveTextContent(
        'false|0|',
      );
      expect(screen.getByTestId('bill-workbench')).toHaveTextContent(
        'false|||订单 ORDER-B',
      );
      expect(
        screen.queryByRole('button', { name: '提交账单' }),
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('quick-fee-modal')).toHaveTextContent('false');
      expect(screen.getByTestId('quick-partner-modal')).toHaveTextContent(
        'false',
      );
    });
  });
});

describe('订单费用页统一补录入口', () => {
  beforeEach(() => {
    feeRouteState.params = { kind: 'sea-export', id: 'order-A' };
    feeTestState.lockState = { isLocked: false };
    feeTestState.canCreateFee = true;
    feeTestState.canReadFee = true;
    feeTestState.canLock = true;
    feeTestState.canOperate = true;
    feeTestState.hasPendingSupplement = false;
    vi.clearAllMocks();
  });
  it('仅审核人读取补审列表，不显示订单审核区或加载普通费用候选', async () => {
    feeTestState.canReadFee = false;
    renderFeesPage();
    await waitFor(() =>
      expect(feeTestState.listCorrections).toHaveBeenCalled(),
    );
    expect(feeTestState.feeOptions).toHaveBeenCalledWith(undefined);
    expect(screen.queryByText('订单审核')).not.toBeInTheDocument();
    expect(screen.getByText('费用补录补审')).toBeInTheDocument();
    expect(
      screen.queryByTestId('finance-summary-board'),
    ).not.toBeInTheDocument();
  });
  it('无费用读取和对应品类审核权限时拒绝进入', () => {
    feeTestState.canReadFee = false;
    feeTestState.canLock = false;
    renderFeesPage();
    expect(screen.getByText('无权访问此业务类型')).toBeInTheDocument();
    expect(feeTestState.listCorrections).not.toHaveBeenCalled();
  });
  it('锁后只保留一个补录入口，待审时不能另建申请', async () => {
    feeTestState.lockState = { isLocked: true };
    feeTestState.hasPendingSupplement = true;
    renderFeesPage();
    expect(screen.queryByText('订单审核')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '锁定订单' }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '费用补录与补审' }));
    expect(
      await screen.findByRole('button', { name: '申请费用补录' }),
    ).toBeDisabled();
    expect(
      screen.queryByRole('button', { name: '费用补录' }),
    ).not.toBeInTheDocument();
  });
  it('已锁定无待审时按费用权限开放补录，跨公司隐藏申请入口', async () => {
    feeTestState.lockState = { isLocked: true };
    const page = renderFeesPage();
    fireEvent.click(screen.getByRole('button', { name: '费用补录与补审' }));
    expect(
      await screen.findByRole('button', { name: '申请费用补录' }),
    ).toBeEnabled();
    page.unmount();
    feeTestState.canOperate = false;
    renderFeesPage();
    fireEvent.click(screen.getByRole('button', { name: '费用补录与补审' }));
    expect(
      screen.queryByRole('button', { name: '申请费用补录' }),
    ).not.toBeInTheDocument();
  });
});
