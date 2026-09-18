import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderCommissionVisibilityMode } from '@/enums.generated';
import OrderCommissionSummaryCell from './OrderCommissionSummaryCell';
import OrderCommissionSummaryModal from './OrderCommissionSummaryModal';

const accessState = vi.hoisted(() => ({
  canReadFinanceCommissions: false,
}));

const pageState = vi.hoisted(() => ({
  pathname: '/orders/sea-export',
}));

const historyState = vi.hoisted(() => ({
  push: vi.fn(),
}));

const listQueryState = vi.hoisted(() => ({
  queryOrderList: vi.fn(),
}));

vi.mock('@umijs/max', () => ({
  useLocation: () => ({ pathname: pageState.pathname }),
  useAccess: () => ({
    canOrder: () => true,
    canCreateEnterpriseResources: true,
    canReadFinanceCommissions: accessState.canReadFinanceCommissions,
  }),
  useModel: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '测试组织' },
      },
    },
  }),
  history: { push: historyState.push },
}));

vi.mock('../list-query', () => ({
  queryOrderList: listQueryState.queryOrderList,
}));

vi.mock('../list-resources', () => ({
  useOrderListResources: () => ({
    masterOptions: [],
    ports: [],
    airports: [],
    customerMap: {},
    containerSpecOptions: [],
    containerSpecMap: {},
    searchCustomers: vi.fn(),
    searchOrderPorts: vi.fn(),
    searchOrderCarriers: vi.fn(),
    searchOrderPersonnel: vi.fn(),
  }),
}));

vi.mock('@/services/roncin/orderTagService', () => ({
  orderTagServiceListOrderTagOptions: vi.fn().mockResolvedValue({ tags: [] }),
}));

import OrderListPage from '../list';

const employeeMode =
  OrderCommissionVisibilityMode.ORDER_COMMISSION_VISIBILITY_MODE_EMPLOYEE;
const organizationMode =
  OrderCommissionVisibilityMode.ORDER_COMMISSION_VISIBILITY_MODE_ORGANIZATION;

const employeeFactsSummary: API.OrderCommissionSummary = {
  visibilityMode: employeeMode,
  baseCurrency: 'CNY',
  hasDraftCommission: true,
  draftCommissionCount: 1,
  draftCommissionAmount: '80.00',
  hasPaidCommission: true,
  paidCommissionCount: 2,
  paidCommissionAmount: '120.00',
};

const employeeEmptySummary: API.OrderCommissionSummary = {
  visibilityMode: employeeMode,
};

const organizationSummary: API.OrderCommissionSummary = {
  visibilityMode: organizationMode,
  baseCurrency: 'CNY',
  hasExpectedOpportunity: true,
  expectedOpportunityCount: 3,
  hasConfirmedCommission: true,
  confirmedCommissionCount: 1,
  confirmedCommissionAmount: '260.50',
  hasPaidCommission: true,
  paidCommissionCount: 2,
  paidCommissionAmount: '120.00',
  hasPendingDecrease: true,
  pendingDecreaseCount: 1,
  pendingDecreaseAmount: '30.00',
};

describe('OrderCommissionSummaryCell', () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('commission_summary 缺省时渲染占位符且不渲染任何 Tag', () => {
    const { container } = render(<OrderCommissionSummaryCell />);

    expect(container.querySelector('.ant-tag')).toBeNull();
    expect(container.textContent).toBe('-');
  });

  it('EMPLOYEE 模式渲染本人事实：待确认数量与已发金额并存', () => {
    render(<OrderCommissionSummaryCell summary={employeeFactsSummary} />);

    expect(screen.getByText('待确认 ¥80.00')).toBeInTheDocument();
    expect(screen.getByText('已发 ¥120.00')).toBeInTheDocument();
    expect(screen.queryByText('待确认 1')).not.toBeInTheDocument();
  });

  it('EMPLOYEE 模式无任何事实时渲染仅针对本人的「我暂无提成」空态', () => {
    const { container } = render(
      <OrderCommissionSummaryCell summary={employeeEmptySummary} />,
    );

    expect(screen.getByText('我暂无提成')).toBeInTheDocument();
    expect(container.textContent).not.toContain('整票');
  });

  it('ORGANIZATION 模式渲染全员汇总事实，无事实时保持中性占位', () => {
    const { container, rerender } = render(
      <OrderCommissionSummaryCell summary={organizationSummary} />,
    );

    expect(screen.getByText('预计 3')).toBeInTheDocument();
    expect(screen.getByText('待发 ¥260.50')).toBeInTheDocument();
    expect(screen.getByText('待冲减 ¥30.00')).toBeInTheDocument();
    expect(screen.queryByText('我暂无提成')).not.toBeInTheDocument();

    // visibility_mode 切换为组织后按新数据渲染，不残留本人视图内容。
    rerender(
      <OrderCommissionSummaryCell
        summary={{ visibilityMode: organizationMode }}
      />,
    );
    expect(container.textContent).toBe('-');
    expect(screen.queryByText('我暂无提成')).not.toBeInTheDocument();
  });

  it('多事实并存并列展示，待冲减是带警示图标的独立风险标识', () => {
    const mixed: API.OrderCommissionSummary = {
      visibilityMode: employeeMode,
      baseCurrency: 'CNY',
      hasExpectedOpportunity: true,
      expectedOpportunityCount: 1,
      hasDraftCommission: true,
      draftCommissionCount: 1,
      hasConfirmedCommission: true,
      confirmedCommissionCount: 1,
      hasPaidCommission: true,
      paidCommissionCount: 1,
      hasPendingDecrease: true,
      pendingDecreaseCount: 1,
    };
    const { container } = render(
      <OrderCommissionSummaryCell summary={mixed} />,
    );

    expect(screen.getByText('预计 1')).toBeInTheDocument();
    expect(screen.getByText('待确认 1')).toBeInTheDocument();
    expect(screen.getByText('待发 1')).toBeInTheDocument();
    expect(screen.getByText('已发 1')).toBeInTheDocument();
    expect(screen.getByText('待冲减 1')).toBeInTheDocument();

    const tags = container.querySelectorAll('.ant-tag');
    expect(tags).toHaveLength(5);
    const decreaseTag = Array.from(tags).find((tag) =>
      tag.textContent?.includes('待冲减'),
    );
    expect(decreaseTag?.querySelector('.anticon-warning')).not.toBeNull();
  });

  it('非 CNY 金额显式跟随币种代码，不做币种合并', () => {
    render(
      <OrderCommissionSummaryCell
        summary={{
          visibilityMode: organizationMode,
          baseCurrency: 'USD',
          hasPaidCommission: true,
          paidCommissionCount: 1,
          paidCommissionAmount: '120.00',
        }}
      />,
    );

    expect(screen.getByText('已发 120.00 USD')).toBeInTheDocument();
  });

  it('点击事实 Tag 触发下钻回调并携带裁剪投影', () => {
    const onOpen = vi.fn();
    render(
      <OrderCommissionSummaryCell
        summary={employeeFactsSummary}
        onOpen={onOpen}
      />,
    );

    fireEvent.click(screen.getByText('待确认 ¥80.00'));
    expect(onOpen).toHaveBeenCalledWith(employeeFactsSummary);
  });
});

describe('OrderCommissionSummaryModal', () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('open=false 时不渲染任何内容', () => {
    const { container } = render(
      <OrderCommissionSummaryModal
        open={false}
        orderNo="SE202609180001"
        summary={organizationSummary}
        onClose={vi.fn()}
      />,
    );

    expect(container.textContent).toBe('');
  });

  it('组织视图下钻展示全员事实、语义说明与台账入口', () => {
    render(
      <OrderCommissionSummaryModal
        open
        orderNo="SE202609180001"
        summary={organizationSummary}
        canOpenLedger
        onClose={vi.fn()}
        onOpenLedger={vi.fn()}
      />,
    );

    expect(screen.getByText('提成摘要 · SE202609180001')).toBeInTheDocument();
    expect(screen.getByText('组织全员视图')).toBeInTheDocument();
    expect(screen.getByText('3 笔')).toBeInTheDocument();
    expect(screen.getByText('1 笔 · ¥260.50')).toBeInTheDocument();
    expect(screen.getByText('2 笔 · ¥120.00')).toBeInTheDocument();
    expect(screen.getByText('1 笔 · ¥30.00')).toBeInTheDocument();
    // 语义铁律：PAID 不宣称整票结清；冲减草稿不等于已扣回；预计非应发承诺。
    expect(
      screen.getByText('仅表示对应提成单已发放，不代表整票结清'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('冲减建议待处理，尚未实际扣回'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('尚未生成有效提成单的估算机会，非应发承诺'),
    ).toBeInTheDocument();
    expect(screen.getByText(/订单组织本位币（CNY）口径/)).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /前往提成台账查看明细/ }),
    ).toBeInTheDocument();
  });

  it('EMPLOYEE 模式本人空态不提供台账入口时不下钻同事信息', () => {
    render(
      <OrderCommissionSummaryModal
        open
        orderNo="SE202609180002"
        summary={employeeEmptySummary}
        canOpenLedger={false}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText('本人视图')).toBeInTheDocument();
    expect(screen.getByText('我暂无提成')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /前往提成台账查看明细/ }),
    ).not.toBeInTheDocument();
  });
});

describe('海运出口订单列表提成摘要接线', () => {
  beforeEach(() => {
    pageState.pathname = '/orders/sea-export';
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('列表行渲染本人提成事实，点击后打开下钻弹窗', async () => {
    listQueryState.queryOrderList.mockResolvedValue({
      data: [
        {
          id: 'order-1',
          orderNo: 'SE202609180001',
          rawRecord: { commissionSummary: employeeFactsSummary },
        },
      ],
      total: 1,
      success: true,
    });

    render(
      <App>
        <OrderListPage />
      </App>,
    );

    expect(await screen.findByText('待确认 ¥80.00')).toBeInTheDocument();
    expect(screen.getByText('已发 ¥120.00')).toBeInTheDocument();

    fireEvent.click(screen.getByText('待确认 ¥80.00'));
    expect(
      await screen.findByText('提成摘要 · SE202609180001'),
    ).toBeInTheDocument();
    expect(screen.getByText('本人视图')).toBeInTheDocument();
  });

  it('commission_summary 缺省的行不崩溃也不渲染空壳 Tag', async () => {
    listQueryState.queryOrderList.mockResolvedValue({
      data: [
        {
          id: 'order-2',
          orderNo: 'SE202609180003',
          rawRecord: {},
        },
      ],
      total: 1,
      success: true,
    });

    const { container } = render(
      <App>
        <OrderListPage />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText('SE202609180003')).toBeInTheDocument();
    });
    const commissionTags = Array.from(
      container.querySelectorAll('.ant-tag'),
    ).filter((tag) => tag.textContent === '我暂无提成');
    expect(commissionTags).toHaveLength(0);
  });
});
