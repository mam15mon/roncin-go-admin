import { renderWithApp } from '@root/tests/renderWithApp';
import { act, cleanup, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import FinanceFeeDetailPage from './detail';

const get = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetFeeLedgerOrderDetail: get,
}));
vi.mock('react-router', async (original) => ({
  ...(await original<typeof import('react-router')>()),
  useParams: () => ({ orderId: 'order-id' }),
}));
vi.mock('@/features/orders/fee-correction', () => ({
  FeeCorrectionSection: () => <div>真实订单费用更正入口</div>,
}));
const user: API.CurrentUser = {
  id: 'user',
  currentOrganization: { id: 'company', kind: 2 },
  permissionCapabilities: [
    { key: 'system.finance.fee.read', dataScope: 'organization' },
  ],
};
afterEach(() => {
  cleanup();
  vi.resetAllMocks();
});

describe('四类财务费用详情的公司与目标读取', () => {
  it.each([undefined, false])(
    '失败或省略成功标志%s的封套夹带旧资料也不能显示为详情',
    async (success) => {
      get.mockResolvedValue({
        success,
        data: {
          orderId: 'order-id',
          organizationId: 'company',
          orderNo: '失败旧资料',
        },
      });
      renderWithApp(<FinanceFeeDetailPage />, { user });
      expect(
        await screen.findByText('费用台账详情加载失败，请刷新重试'),
      ).toBeVisible();
      expect(screen.queryByText('失败旧资料')).not.toBeInTheDocument();
    },
  );
  it.each([
    ['SI', '海运进口'],
    ['AI', '空运进口'],
  ])('%s合法当前公司详情显示中文品类', async (businessType, title) => {
    get.mockResolvedValue({
      success: true,
      data: {
        orderId: 'order-id',
        orderNo: 'IMPORT-DETAIL',
        organizationId: 'company',
        businessType,
      },
    });
    renderWithApp(<FinanceFeeDetailPage />, { user });
    expect(await screen.findByText('IMPORT-DETAIL')).toBeVisible();
    expect(screen.getByText(title)).toBeVisible();
    expect(screen.getByText('真实订单费用更正入口')).toBeVisible();
  });

  it.each([undefined, 1])(
    '身份未就绪或系统工作区%s不发经营请求',
    async (kind) => {
      renderWithApp(<FinanceFeeDetailPage />, {
        user: {
          ...user,
          currentOrganization: kind ? { id: 'system', kind } : undefined,
        },
      });
      await act(async () => {});
      expect(get).not.toHaveBeenCalled();
      expect(
        screen.queryByText('真实订单费用更正入口'),
      ).not.toBeInTheDocument();
    },
  );

  it.each([
    { orderId: 'other-id', organizationId: 'company' },
    { orderId: 'order-id', organizationId: 'other-company' },
  ])('拒绝不属于当前目标的响应：%j', async (identity) => {
    get.mockResolvedValue({
      success: true,
      data: { ...identity, orderNo: '不可显示的其他资料' },
    });
    renderWithApp(<FinanceFeeDetailPage />, { user });
    expect(
      await screen.findByText('费用台账详情加载失败，请刷新重试'),
    ).toBeVisible();
    expect(screen.queryByText('不可显示的其他资料')).not.toBeInTheDocument();
    expect(screen.queryByText('真实订单费用更正入口')).not.toBeInTheDocument();
  });
});
