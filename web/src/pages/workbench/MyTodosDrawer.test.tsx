import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { formatDate } from '@/utils/format';
import MyTodosDrawer from './MyTodosDrawer';

const mocks = vi.hoisted(() => ({
  fees: vi.fn(),
  abnormal: vi.fn(),
  push: vi.fn(),
  refresh: vi.fn(),
}));
vi.mock('@/services/roncin/workbenchService', () => ({
  workbenchServiceListMyUnbilledFees: mocks.fees,
  workbenchServiceListMyOpenAbnormalCases: mocks.abnormal,
}));
vi.mock('@/router/history', () => ({ history: { push: mocks.push } }));

const user: API.CurrentUser = {
  id: 'operator',
  currentOrganization: { id: 'org-a', kind: 2 },
  permissionCapabilities: ['se', 'ae'].flatMap((kind) =>
    ['read', 'fee.read', 'abnormal_case.read'].map((operation) => ({
      key: `business.order.${kind}.${operation}`,
      dataScope: 'organization',
    })),
  ),
};

const fee: API.WorkbenchUnbilledFee = {
  feeId: 'fee-1',
  orderId: 'order-se',
  orderNo: 'SE-001',
  businessType: 'SE',
  customerName: '客户甲',
  feeName: '海运费',
  settlementPartyName: '结算客户甲',
  direction: 'RECEIVABLE',
  currency: 'USD',
  totalAmount: '15362.50000000',
  createdAt: '2026-10-26T07:18:00Z',
};

function renderDrawer(
  initialKind: 'fees' | 'abnormal' = 'fees',
  currentUser = user,
) {
  return renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <MyTodosDrawer
        initialKind={initialKind}
        onClose={vi.fn()}
        onRefresh={mocks.refresh}
      />
    </ConfigProvider>,
    { user: currentUser },
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.fees.mockResolvedValue({ success: true, data: [], total: '0' });
  mocks.abnormal.mockResolvedValue({ success: true, data: [], total: '0' });
});

describe('作业待办分页与办理入口', () => {
  it('同票费用逐笔显示原币金额、方向、时间和阻塞条件，SE/AE 均直达费用页', async () => {
    mocks.fees.mockResolvedValue({
      success: true,
      total: '3',
      data: [
        { ...fee, orderLocked: true },
        {
          ...fee,
          feeId: 'fee-2',
          feeName: '报关费',
          direction: 'PAYABLE',
          currency: 'CNY',
          totalAmount: '-80.00000000',
          hasPendingChange: true,
          blockedReasons: ['源费用更正待补审，暂不可建账'],
          terminationStatus: 'TERMINATED',
          closureStatus: 'CLOSED',
        },
        {
          ...fee,
          feeId: 'fee-3',
          businessType: 'AE',
          orderId: 'order-ae',
          orderNo: 'AE-001',
        },
      ],
    });
    renderDrawer();
    expect(await screen.findAllByText('15,362.50')).toHaveLength(2);
    expect(screen.getAllByText('SE-001')).toHaveLength(2);
    expect(screen.getByText('-80.00')).toBeInTheDocument();
    expect(screen.getByText('应付')).toBeInTheDocument();
    expect(screen.getAllByText('应收')).toHaveLength(2);
    expect(screen.getByText('共 3 笔')).toBeInTheDocument();
    expect(
      screen.getAllByText(formatDate(fee.createdAt, 'minute')),
    ).toHaveLength(3);
    expect(screen.queryByText('2026-10-26T07:18:00Z')).not.toBeInTheDocument();
    expect(
      screen.getByText('审核已锁定（禁止修改，符合条件可建账）'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('源费用更正待补审，暂不可建账'),
    ).toBeInTheDocument();
    expect(screen.getByText('已结案')).toBeInTheDocument();
    const links = screen.getAllByRole('button', { name: '进入费用页' });
    expect(links[0]).toBeEnabled();
    fireEvent.click(links[0]);
    expect(mocks.push).toHaveBeenLastCalledWith(
      '/orders/sea-export/order-se/fees',
    );
    fireEvent.click(links[2]);
    expect(mocks.push).toHaveBeenLastCalledWith(
      '/orders/air-export/order-ae/fees',
    );
  });

  it('服务端翻页，切换分类回第一页，同票异常逐件显示名称并附带直达意图', async () => {
    mocks.fees.mockResolvedValue({ success: true, total: '45', data: [fee] });
    mocks.abnormal.mockResolvedValue({
      success: true,
      total: '2',
      data: [
        {
          id: 'case-1',
          orderId: 'order-se',
          orderNo: 'SE-001',
          businessType: 'SE',
          abnormalCaseName: '已停用的缺件类型',
          markedAt: fee.createdAt,
          orderLocked: true,
          businessWriteBlockedReason: '订单已锁定',
        },
        {
          id: 'case-2',
          orderId: 'order-ae',
          orderNo: 'AE-001',
          businessType: 'AE',
          abnormalCaseName: '舱位异常',
          markedAt: fee.createdAt,
        },
      ],
    });
    renderDrawer();
    await screen.findByText('SE-001');
    fireEvent.click(screen.getByTitle('2'));
    await waitFor(() =>
      expect(mocks.fees).toHaveBeenLastCalledWith({ page: 2, pageSize: 20 }),
    );
    fireEvent.click(screen.getByRole('tab', { name: '进行中异常' }));
    expect(await screen.findByText('已停用的缺件类型')).toBeInTheDocument();
    expect(screen.getByText('只读：订单已锁定')).toBeInTheDocument();
    expect(screen.getByText('共 2 件')).toBeInTheDocument();
    expect(mocks.abnormal).toHaveBeenCalledWith({ page: 1, pageSize: 20 });
    const links = screen.getAllByRole('button', { name: '查看异常' });
    fireEvent.click(links[0]);
    expect(mocks.push).toHaveBeenLastCalledWith(
      '/orders/sea-export/order-se?panel=abnormal',
    );
    fireEvent.click(links[1]);
    expect(mocks.push).toHaveBeenLastCalledWith(
      '/orders/air-export/order-ae?panel=abnormal',
    );
    fireEvent.click(screen.getByRole('tab', { name: '未建账费用' }));
    await waitFor(() =>
      expect(mocks.fees).toHaveBeenLastCalledWith({ page: 1, pageSize: 20 }),
    );
    await screen.findByText('SE-001');
  });

  it.each(['undefined', 'reject', 'unsuccessful'])(
    '请求失败 %s 提供重试，成功空结果显示空态',
    async (mode) => {
      if (mode === 'reject')
        mocks.fees.mockRejectedValueOnce(new Error('失败'));
      else
        mocks.fees.mockResolvedValueOnce(
          mode === 'undefined' ? undefined : { success: false },
        );
      renderDrawer();
      expect(await screen.findByText('待办明细加载失败')).toBeInTheDocument();
      expect(screen.queryByText('当前没有此类待办')).not.toBeInTheDocument();
      expect(screen.queryByText('共 0 笔')).not.toBeInTheDocument();
      fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
      expect(await screen.findByText('当前没有此类待办')).toBeInTheDocument();
      expect(mocks.refresh).toHaveBeenCalledTimes(1);
    },
  );

  it('只有审核入口和异常读取权限时隐藏费用分类且不请求费用，刷新重取异常', async () => {
    renderDrawer('abnormal', {
      ...user,
      permissionCapabilities: [
        'business.order.se.read',
        'business.order.se.lock',
        'business.order.se.abnormal_case.read',
      ].map((key) => ({ key, dataScope: 'organization' })),
    });
    await screen.findByText('当前没有此类待办');
    expect(
      screen.queryByRole('tab', { name: '未建账费用' }),
    ).not.toBeInTheDocument();
    expect(mocks.fees).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '刷新待办' }));
    await waitFor(() => expect(mocks.abnormal).toHaveBeenCalledTimes(2));
    await act(async () => {});
    expect(mocks.refresh).toHaveBeenCalledTimes(1);
    expect(
      within(screen.getByRole('dialog')).getByText('当前没有此类待办'),
    ).toBeInTheDocument();
  });
});
