import { renderWithApp as render } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  OrderAllowedAction,
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import OrderDetailPage from './detail';
import {
  confirmOrderClosure,
  confirmOrderFlow,
  confirmOrderTermination,
} from './order-detail-transitions';

const testState = vi.hoisted(() => ({
  canOperate: true,
  loading: false,
  lockState: { isLocked: false } as API.OrderLockStateData | null,
  order: {} as API.Order,
  loadData: vi.fn(),
  refreshLockState: vi.fn(),
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useParams: () => ({ kind: 'sea-export', id: 'order-1' }),
    Link: ({ children }: { children: React.ReactNode }) => <a>{children}</a>,
  };
});

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => testState.canOperate,
    canOrder: () => true,
  }),
}));

vi.mock('@/router/history', () => ({
  history: {
    get location() {
      return window.location;
    },
    push: vi.fn(),
  },
}));

vi.mock('./order-kinds/registry', () => ({
  getOrderKindDefinition: () => ({
    kind: 'sea-export',
    title: '海运出口',
    navigationTitle: '海运出口',
    businessType: 1,
    form: {
      buildDetailInitialValues: () => ({}),
      buildSections: () => [],
      buildUpdatePayload: () => ({}),
    },
  }),
}));

vi.mock('./use-order-detail-data', () => ({
  useOrderDetailData: () => ({
    loading: testState.loading,
    error: null,
    order: testState.order,
    seaDocuments: {
      documentStructure: 2,
      linkVersion: '1',
      masterBill: { id: 'mbl-1', version: '1', content: {} },
    },
    shippingDocs: [],
    personnel: [],
    serviceTypeOptions: [],
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: vi.fn().mockResolvedValue([]),
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: [],
    draftScope: 'user-1:org-1',
    loadData: testState.loadData,
  }),
}));

vi.mock('./use-order-lock-state', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('./use-order-lock-state')>();
  return {
    ...actual,
    useOrderLockState: () => ({
      state: testState.lockState,
      loading: false,
      error: null,
      refresh: testState.refreshLockState,
    }),
  };
});

vi.mock('@/components/ui/order-template/OrderFormTemplate', () => ({
  OrderFormTemplate: ({
    header,
    prependSections,
  }: {
    header: React.ReactNode;
    prependSections: Array<{
      key: string;
      extra?: React.ReactNode;
      content?: React.ReactNode;
    }>;
  }) => (
    <div>
      {header}
      <input aria-label="未保存表单" defaultValue="" />
      {prependSections.map((section) => (
        <React.Fragment key={section.key}>
          {section.extra}
          {section.content}
        </React.Fragment>
      ))}
    </div>
  ),
}));

vi.mock('./components/detail/OrderDetailHeader', () => ({
  default: () => <div>订单页头</div>,
}));

vi.mock('./order-detail-transitions', () => ({
  confirmOrderFlow: vi.fn(),
  confirmOrderTermination: vi.fn(),
  confirmOrderClosure: vi.fn(),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceUpdateOrder: vi.fn(),
}));

const mockConfirmFlow = vi.mocked(confirmOrderFlow);
const mockConfirmTermination = vi.mocked(confirmOrderTermination);
const mockConfirmClosure = vi.mocked(confirmOrderClosure);

describe('订单详情状态卡流转编排', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    testState.canOperate = true;
    testState.loading = false;
    testState.lockState = { isLocked: false } as API.OrderLockStateData;
    testState.loadData.mockResolvedValue(undefined);
    testState.refreshLockState.mockResolvedValue(null);
    mockConfirmFlow.mockReturnValue(true);
    mockConfirmTermination.mockReturnValue(true);
    mockConfirmClosure.mockReturnValue(true);
    testState.order = {
      id: 'order-1',
      orderNo: 'SE0001',
      organizationId: 'org-1',
      version: '3',
      flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
      terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_TRANSITION_FLOW],
      allowedTargetFlowStatuses: [OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED],
    } as API.Order;
  });

  it('点击合法主流程节点打开固定目标确认且防止重复弹窗', () => {
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    const button = screen.getByRole('button', { name: '流转到已订舱' });
    fireEvent.click(button);
    fireEvent.click(button);

    expect(mockConfirmFlow).toHaveBeenCalledOnce();
    expect(mockConfirmFlow).toHaveBeenCalledWith(
      expect.anything(),
      testState.order,
      OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
      expect.any(Function),
      expect.objectContaining({
        afterClose: expect.any(Function),
        canSubmit: expect.any(Function),
      }),
    );
  });

  it('锁定订单仍可从退关中执行完成退关，不被业务写门禁拦截', async () => {
    testState.lockState = { isLocked: true } as API.OrderLockStateData;
    testState.order = {
      ...testState.order,
      terminationStatus:
        OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
      allowedActions: [
        OrderAllowedAction.ORDER_ALLOWED_ACTION_COMPLETE_TERMINATION,
        OrderAllowedAction.ORDER_ALLOWED_ACTION_CANCEL_TERMINATION,
      ],
      allowedTargetFlowStatuses: [],
    };
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: '选择退关操作' }));
    fireEvent.click(await screen.findByRole('menuitem', { name: '完成退关' }));

    expect(mockConfirmTermination).toHaveBeenCalledWith(
      expect.anything(),
      testState.order,
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
      expect.any(Function),
      expect.objectContaining({ canSubmit: expect.any(Function) }),
    );
  });

  it('锁单阻止主流程和发起退关，但允许完结订单', () => {
    testState.lockState = { isLocked: true } as API.OrderLockStateData;
    testState.order = {
      ...testState.order,
      allowedActions: [
        OrderAllowedAction.ORDER_ALLOWED_ACTION_TRANSITION_FLOW,
        OrderAllowedAction.ORDER_ALLOWED_ACTION_START_TERMINATION,
        OrderAllowedAction.ORDER_ALLOWED_ACTION_CLOSE,
      ],
    };
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(screen.getByRole('button', { name: '流转到已订舱' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '发起退关' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '完结订单' }));
    expect(mockConfirmClosure).toHaveBeenCalledOnce();
  });

  it('草稿收到完结动作即可直接打开完结确认，不要求先走主流程', () => {
    testState.order = {
      ...testState.order,
      flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_CLOSE],
    };
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    expect(screen.getByRole('button', { name: '完结订单' })).toBeEnabled();
    fireEvent.click(screen.getByRole('button', { name: '完结订单' }));
    expect(mockConfirmClosure).toHaveBeenCalledOnce();
  });

  it('工作台不符时无法完结或流转', () => {
    testState.canOperate = false;
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(screen.getByRole('button', { name: '完结订单' })).toBeDisabled();
    expect(
      screen.queryByRole('button', { name: '流转到已订舱' }),
    ).toBeDisabled();
    expect(mockConfirmFlow).not.toHaveBeenCalled();
  });

  it('提交前重新核对最新允许动作，并在成功回调刷新详情和锁状态', async () => {
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '流转到已订舱' }));

    const [, , , onCompleted, options] = mockConfirmFlow.mock.calls[0];
    expect(options?.canSubmit?.()).toBe(true);
    await onCompleted();
    expect(testState.loadData).toHaveBeenCalledOnce();
    expect(testState.refreshLockState).toHaveBeenCalledOnce();

    testState.canOperate = false;
    expect(options?.canSubmit?.()).toBe(false);
  });

  it('状态刷新进入加载态时保留未保存的表单实例', () => {
    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    fireEvent.change(screen.getByRole('textbox', { name: '未保存表单' }), {
      target: { value: '待保存内容' },
    });

    testState.loading = true;
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );
    expect(screen.getByRole('textbox', { name: '未保存表单' })).toHaveValue(
      '待保存内容',
    );
  });

  it('待审核补录禁用完结按钮并显示原因，投影刷新后恢复入口', async () => {
    testState.order = {
      ...testState.order,
      flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_RELEASED,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_CLOSE],
      feeReview: { hasPendingSupplement: true, hasPendingChange: true },
    };
    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    const close = screen.getByRole('button', { name: '完结订单' });
    expect(close).toBeDisabled();
    fireEvent.mouseOver(close.parentElement as HTMLElement);
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      '存在待审批费用变更，请先处理后再结案',
    );
    fireEvent.click(close);
    expect(mockConfirmClosure).not.toHaveBeenCalled();

    testState.order = {
      ...testState.order,
      feeReview: { hasPendingSupplement: false },
    };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '完结订单' }));
    expect(mockConfirmClosure).toHaveBeenCalledOnce();
    await act(async () => {});
  });

  it('结案确认打开后投影变为待审，提交前重新检查阻断', async () => {
    testState.order = {
      ...testState.order,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_CLOSE],
      feeReview: { hasPendingSupplement: false },
    };
    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '完结订单' }));
    const options = mockConfirmClosure.mock.calls[0][4];
    expect(options?.canSubmit?.()).toBe(true);
    testState.order = {
      ...testState.order,
      feeReview: { hasPendingSupplement: true, hasPendingChange: true },
    };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await act(async () => {
      expect(options?.canSubmit?.()).toBe(false);
    });
    expect(
      await screen.findByText('存在待审批费用变更，请先处理后再结案'),
    ).toBeInTheDocument();
    expect(testState.loadData).not.toHaveBeenCalled();
  });

  it('已结案有待审补录仍显示反结案动作', () => {
    testState.order = {
      ...testState.order,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_REOPEN],
      feeReview: { hasPendingSupplement: true, hasPendingChange: true },
    };
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '反结案' }));
    expect(mockConfirmClosure).toHaveBeenCalledWith(
      expect.anything(),
      testState.order,
      OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
      expect.any(Function),
      expect.anything(),
    );
    expect(mockConfirmClosure.mock.calls[0][4]?.canSubmit?.()).toBe(true);
  });

  it('已锁定订单不开放反结案，陈旧服务端动作也不能绕过', () => {
    testState.lockState = { isLocked: true };
    testState.order = {
      ...testState.order,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_REOPEN],
    };
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    expect(screen.getByRole('button', { name: '反结案' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '反结案' }));
    expect(mockConfirmClosure).not.toHaveBeenCalled();
  });

  it.each([true, null])(
    '反结案确认打开后审核锁改变或丢失会阻止提交：%s',
    async (isLocked) => {
      testState.order = {
        ...testState.order,
        closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
        allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_REOPEN],
      };
      const { rerender } = render(
        <App>
          <OrderDetailPage />
        </App>,
      );
      fireEvent.click(screen.getByRole('button', { name: '反结案' }));
      const options = mockConfirmClosure.mock.calls[0][4];
      expect(options?.canSubmit?.()).toBe(true);
      testState.lockState = isLocked === null ? null : { isLocked };
      rerender(
        <App>
          <OrderDetailPage />
        </App>,
      );
      await act(async () => expect(options?.canSubmit?.()).toBe(false));
      expect(
        await screen.findByText(
          isLocked === null
            ? '订单锁定状态加载失败，请刷新后重试'
            : '订单已审核，请先撤销审核后再反结案',
        ),
      ).toBeInTheDocument();
    },
  );

  it('未提供费用投影时提交结案，由服务端复核准入', () => {
    testState.order = {
      ...testState.order,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_CLOSE],
    };
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '完结订单' }));
    expect(mockConfirmClosure).toHaveBeenCalledOnce();
  });
  it('只有待审批更正也阻止结案，不依赖补录标志', async () => {
    testState.order = {
      ...testState.order,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_CLOSE],
      feeReview: { hasPendingCorrection: true, hasPendingChange: true },
    };
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    expect(screen.getByRole('button', { name: '完结订单' })).toBeDisabled();
    await act(async () => {});
  });
});
