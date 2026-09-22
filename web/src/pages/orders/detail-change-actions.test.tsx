import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, Form, Input } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderAllowedAction, OrderFlowStatus } from '@/enums.generated';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import OrderDetailPage from './detail';

const routeState = vi.hoisted(() => ({
  params: { kind: 'sea-export', id: 'ord-A' },
}));

const detailTestState = vi.hoisted(() => ({
  loadData: vi.fn<(orderId?: string) => Promise<void>>(),
  allowedActions: [1] as number[],
  lockState: { isLocked: false } as API.OrderLockStateData | null,
  sectionReadonly: undefined as boolean | undefined,
  templateReadonly: undefined as boolean | undefined,
  canRead: true,
  detailArgs: vi.fn(),
  canOperate: true,
  customerReferenceNo: '服务端初始值',
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useParams: () => routeState.params,
  };
});

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => detailTestState.canOperate,
    canOperateBusiness: true,
    canOrder: () => detailTestState.canRead,
  }),
}));

vi.mock('@/router/history', () => ({
  history: { push: vi.fn() },
}));

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceGetSeaOrderChangeActions: vi.fn(),
}));

vi.mock('./use-order-detail-data', () => ({
  useOrderDetailData: (orderId?: string) => {
    detailTestState.detailArgs(orderId);
    return {
      loading: false,
      order: orderId
        ? {
            id: orderId,
            orderNo: `ORDER-${orderId}`,
            version: '1',
            flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
            customerReferenceNo: detailTestState.customerReferenceNo,
            allowedActions: detailTestState.allowedActions,
          }
        : undefined,
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
      loadData: () => detailTestState.loadData(orderId),
    };
  },
}));

vi.mock('./use-order-lock-state', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('./use-order-lock-state')>();
  return {
    ...actual,
    useOrderLockState: () => ({
      state: detailTestState.lockState,
      loading: false,
      error: null,
      refresh: vi.fn().mockResolvedValue(null),
    }),
  };
});

vi.mock('@/components/ui/order-template/OrderFormTemplate', () => ({
  OrderFormTemplate: ({
    header,
    readonly,
    formRef,
    actionsRef,
    initialValues,
  }: {
    header: React.ReactNode;
    readonly?: boolean;
    formRef?: React.MutableRefObject<ReturnType<typeof Form.useForm>[0]>;
    actionsRef?: React.MutableRefObject<{
      resetTo: (values?: { customerReferenceNo?: string }) => void;
    } | null>;
    initialValues?: { customerReferenceNo?: string };
  }) => {
    const [form] = Form.useForm();
    detailTestState.templateReadonly = readonly;
    React.useEffect(() => {
      if (formRef) {
        formRef.current = form;
      }
    }, [form, formRef]);
    React.useImperativeHandle(
      actionsRef,
      () => ({
        resetTo: (values?: { customerReferenceNo?: string }) => {
          form.resetFields();
          if (values) form.setFieldsValue(values);
        },
      }),
      [form],
    );
    return (
      <Form form={form} initialValues={initialValues}>
        <Form.Item name="customerReferenceNo">
          <Input aria-label="客户参考号" />
        </Form.Item>
        {header}
      </Form>
    );
  },
}));

vi.mock('./templates', () => ({
  getSeaTemplateSections: (props: { readonly?: boolean }) => {
    detailTestState.sectionReadonly = props.readonly;
    return [];
  },
}));

vi.mock('./components/detail/OrderDetailHeader', () => ({
  default: (props: {
    businessActions?: React.ReactNode;
    moreMenuItems?: Array<{
      key?: React.Key;
      onClick?: () => void;
    }>;
    onSynchronizeLockChange?: () => Promise<void>;
  }) => (
    <div>
      {props.businessActions}
      <button
        type="button"
        onClick={() =>
          props.moreMenuItems
            ?.find((item) => item?.key === 'reload-data')
            ?.onClick?.()
        }
      >
        刷新动作资格
      </button>
      <button
        type="button"
        onClick={() => void props.onSynchronizeLockChange?.()}
      >
        同步锁单状态
      </button>
    </div>
  ),
}));

const mockGetChangeActions = vi.mocked(
  seaOrderChangeServiceGetSeaOrderChangeActions,
);

const getSplitButton = () => screen.getByRole('button', { name: /拆票/ });
const getReassignButton = () => screen.getByRole('button', { name: /改配/ });

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe('订单详情页拆票与改配动作隔离', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    detailTestState.canOperate = true;
    detailTestState.canRead = true;
    routeState.params = { kind: 'sea-export', id: 'ord-A' };
    detailTestState.loadData.mockResolvedValue(undefined);
    detailTestState.allowedActions = [
      OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT,
    ];
    detailTestState.lockState = { isLocked: false } as API.OrderLockStateData;
    detailTestState.sectionReadonly = undefined;
    detailTestState.templateReadonly = undefined;
    detailTestState.customerReferenceNo = '服务端初始值';
  });

  it('无当前业务类型查看权限时拒绝页面且不启动详情或动作查询', () => {
    detailTestState.canRead = false;
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    expect(screen.getByText('无权访问此业务类型')).toBeInTheDocument();
    expect(detailTestState.detailArgs).toHaveBeenCalledWith(undefined);
    expect(mockGetChangeActions).not.toHaveBeenCalled();
  });

  it.each([
    {
      name: '缺少编辑动作权限',
      canOperate: true,
      allowedActions: [] as number[],
      lockState: { isLocked: false } as API.OrderLockStateData,
    },
    {
      name: '订单已锁定',
      canOperate: true,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT],
      lockState: { isLocked: true } as API.OrderLockStateData,
    },
    {
      name: '当前工作台不是订单所属分公司',
      canOperate: false,
      allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT],
      lockState: { isLocked: false } as API.OrderLockStateData,
    },
  ])(
    '$name 时模板与分节使用同一完整只读值',
    async ({ allowedActions, lockState, canOperate }) => {
      detailTestState.canOperate = canOperate;
      detailTestState.allowedActions = allowedActions;
      detailTestState.lockState = lockState;
      mockGetChangeActions.mockResolvedValue({ data: {} } as never);

      render(
        <App>
          <OrderDetailPage />
        </App>,
      );

      expect(detailTestState.sectionReadonly).toBe(true);
      expect(detailTestState.templateReadonly).toBe(true);
      // 挂载期动作资格请求在 act 内落地，避免用例结束后迟到更新
      await act(async () => {
        await Promise.resolve();
        await Promise.resolve();
        await Promise.resolve();
      });
    },
  );

  it('已订舱订单有编辑动作且未锁单时模板与分节均可编辑', async () => {
    mockGetChangeActions.mockResolvedValue({ data: {} } as never);

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(detailTestState.sectionReadonly).toBe(false);
    expect(detailTestState.templateReadonly).toBe(false);
    // 挂载期动作资格请求在 act 内落地，避免用例结束后迟到更新
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
  });

  it('A 与 B 响应逆序返回时，仅展示当前订单 B 的动作资格', async () => {
    const requestA = deferred<any>();
    const requestB = deferred<any>();
    mockGetChangeActions
      .mockImplementationOnce(() => requestA.promise)
      .mockImplementationOnce(() => requestB.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => {
      expect(mockGetChangeActions).toHaveBeenCalledWith({ orderId: 'ord-A' });
    });

    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => {
      expect(mockGetChangeActions).toHaveBeenCalledWith({ orderId: 'ord-B' });
    });

    await act(async () => {
      requestB.resolve({
        data: {
          canSplit: false,
          splitBlockedReasons: ['B 不允许拆票'],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      });
    });
    await waitFor(() => {
      expect(getSplitButton()).toBeDisabled();
      expect(getReassignButton()).toBeEnabled();
    });

    await act(async () => {
      requestA.resolve({
        data: {
          canSplit: true,
          splitBlockedReasons: [],
          canReassign: false,
          reassignBlockedReasons: ['A 不允许改配'],
        },
      });
    });

    expect(getSplitButton()).toBeDisabled();
    expect(getReassignButton()).toBeEnabled();

    // 阻断原因进入按钮 Tooltip：B 的原因保持可见，A 的迟到结果不得覆盖。
    fireEvent.mouseEnter(getSplitButton());
    expect(await screen.findByText('B 不允许拆票')).toBeInTheDocument();
  });

  it('切换到 B 后立即清空 A 的动作资格，且 B 失败时不恢复 A 的状态', async () => {
    const requestB = deferred<any>();
    mockGetChangeActions
      .mockResolvedValueOnce({
        data: {
          canSplit: true,
          splitBlockedReasons: [],
          canReassign: false,
          reassignBlockedReasons: ['A 不允许改配'],
        },
      } as any)
      .mockImplementationOnce(() => requestB.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => {
      expect(getSplitButton()).toBeEnabled();
      expect(getReassignButton()).toBeDisabled();
    });

    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(getSplitButton()).toBeDisabled();
    expect(getReassignButton()).toBeDisabled();

    await act(async () => {
      requestB.reject(new Error('B 动作资格加载失败'));
    });

    await waitFor(() => {
      expect(getSplitButton()).toBeDisabled();
      expect(getReassignButton()).toBeDisabled();
    });
  });

  it('页面手工刷新会重新获取当前订单的动作资格', async () => {
    mockGetChangeActions
      .mockResolvedValueOnce({
        data: {
          canSplit: true,
          splitBlockedReasons: [],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      } as any)
      .mockResolvedValueOnce({
        data: {
          canSplit: false,
          splitBlockedReasons: ['刷新后不可拆票'],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      } as any);

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => expect(getSplitButton()).toBeEnabled());

    fireEvent.click(screen.getByRole('button', { name: '刷新动作资格' }));

    await waitFor(() => {
      expect(mockGetChangeActions).toHaveBeenCalledTimes(2);
      expect(getSplitButton()).toBeDisabled();
    });
  });

  it('订单 A 的旧同步任务结束后，不得使订单 B 当前动作响应失效', async () => {
    const loadOrderA = deferred<void>();
    const requestB = deferred<any>();
    detailTestState.loadData.mockImplementation((orderId) =>
      orderId === 'ord-A' ? loadOrderA.promise : Promise.resolve(),
    );
    mockGetChangeActions
      .mockResolvedValueOnce({
        data: {
          canSplit: true,
          splitBlockedReasons: [],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      } as any)
      .mockImplementationOnce(() => requestB.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => expect(getSplitButton()).toBeEnabled());

    fireEvent.click(screen.getByRole('button', { name: '同步锁单状态' }));
    await waitFor(() =>
      expect(detailTestState.loadData).toHaveBeenCalledWith('ord-A'),
    );

    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(mockGetChangeActions).toHaveBeenCalledWith({ orderId: 'ord-B' }),
    );

    await act(async () => {
      loadOrderA.resolve();
    });
    expect(mockGetChangeActions).toHaveBeenCalledTimes(2);

    await act(async () => {
      requestB.resolve({
        data: {
          canSplit: false,
          splitBlockedReasons: ['B 当前不可拆票'],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      });
    });

    await waitFor(() => {
      expect(getSplitButton()).toBeDisabled();
    });
  });

  it('锁状态同步和同订单后台加载不会覆盖当前未保存表单值', async () => {
    const pendingLoad = deferred<void>();
    detailTestState.loadData.mockImplementation(() => pendingLoad.promise);
    mockGetChangeActions.mockResolvedValue({ data: {} } as never);

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    const customerReferenceNo = await screen.findByLabelText('客户参考号');
    expect(customerReferenceNo).toHaveValue('服务端初始值');
    fireEvent.change(customerReferenceNo, {
      target: { value: '尚未保存的修改' },
    });

    fireEvent.click(screen.getByRole('button', { name: '同步锁单状态' }));
    await waitFor(() =>
      expect(screen.getByLabelText('客户参考号')).toHaveValue('尚未保存的修改'),
    );

    // 模拟同一订单的 loadData 返回新版对象；同步结束后的 rerender 不得把表单重置为服务端值。
    detailTestState.customerReferenceNo = '同步后的服务端值';
    await act(async () => {
      pendingLoad.resolve();
    });

    await waitFor(() =>
      expect(screen.getByLabelText('客户参考号')).toHaveValue('尚未保存的修改'),
    );
  });

  it('显式刷新数据在当前加载完成后才用最新服务端值重置表单', async () => {
    const pendingLoad = deferred<void>();
    detailTestState.loadData.mockImplementation(() => pendingLoad.promise);
    mockGetChangeActions.mockResolvedValue({ data: {} } as never);

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    const customerReferenceNo = await screen.findByLabelText('客户参考号');
    fireEvent.change(customerReferenceNo, {
      target: { value: '等待刷新前的修改' },
    });

    fireEvent.click(screen.getByRole('button', { name: '刷新动作资格' }));
    expect(screen.getByLabelText('客户参考号')).toHaveValue('等待刷新前的修改');

    detailTestState.customerReferenceNo = '刷新后的服务端值';
    await act(async () => {
      pendingLoad.resolve();
    });

    await waitFor(() =>
      expect(screen.getByLabelText('客户参考号')).toHaveValue(
        '刷新后的服务端值',
      ),
    );
  });
});
