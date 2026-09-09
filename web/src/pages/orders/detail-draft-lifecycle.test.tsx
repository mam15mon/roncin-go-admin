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
import {
  getFormDraft,
  getFormDraftKey,
  hasTabDraft,
  saveFormDraft,
} from '@/components/layout/formDraft';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { OrderAllowedAction } from '@/enums.generated';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import OrderDetailPage from './detail';

const routeState = vi.hoisted(() => ({
  params: { kind: 'sea-export', id: 'ord-A' },
}));

const templateLifecycleState = vi.hoisted(() => ({
  instanceId: 0,
  instancesMounted: [] as number[],
  instancesUnmounted: [] as number[],
  activeInstanceProps: null as any,
  restoreDraft: false,
}));

const detailTestState = vi.hoisted(() => ({
  loadData: vi.fn<(orderId?: string) => Promise<void>>(),
  refreshLockState: vi.fn<() => Promise<any>>(),
  allowedActions: [1] as number[],
  lockState: { isLocked: false } as API.OrderLockStateData | null,
  sectionReadonly: undefined as boolean | undefined,
  templateReadonly: undefined as boolean | undefined,
  customerReferenceNo: '服务端初始值',
  draftScope: 'user-1:org-1',
  orderAvailable: true,
}));

vi.mock('@umijs/max', () => ({
  useParams: () => routeState.params,
  useAccess: () => ({
    canOrder: () => true,
  }),
  Link: ({ children }: { children: React.ReactNode }) => <a>{children}</a>,
  history: { push: vi.fn() },
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceUpdateOrder: vi.fn(),
}));

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceGetSeaOrderChangeActions: vi
    .fn()
    .mockResolvedValue({ data: {} }),
}));

vi.mock('./use-order-detail-data', () => ({
  useOrderDetailData: (orderId?: string) => ({
    loading: false,
    order:
      orderId && detailTestState.orderAvailable
        ? {
            id: orderId,
            orderNo: `ORDER-${orderId}`,
            version: '1',
            customerReferenceNo:
              orderId === 'ord-B'
                ? 'B-服务端初始值'
                : detailTestState.customerReferenceNo,
            allowedActions: detailTestState.allowedActions,
          }
        : undefined,
    error: detailTestState.orderAvailable
      ? null
      : new Error('模拟真实详情刷新失败'),
    shippingDocs: [],
    personnel: [],
    serviceTypeOptions: [],
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: vi.fn().mockResolvedValue([]),
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: [],
    draftScope: detailTestState.draftScope,
    loadData: () => detailTestState.loadData(orderId),
  }),
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
      refresh: detailTestState.refreshLockState,
    }),
  };
});

vi.mock('@/components/ui/order-template/OrderFormTemplate', () => ({
  OrderFormTemplate: (props: {
    header?: React.ReactNode;
    footer?: React.ReactNode;
    readonly?: boolean;
    formRef?: React.MutableRefObject<ReturnType<typeof Form.useForm>[0]>;
    initialValues?: { customerReferenceNo?: string };
    onFinish?: (values: any) => Promise<any>;
    onDirtyChange?: (dirty: boolean) => void;
    dirty?: boolean;
    draftScope?: string;
    tabKey?: string;
  }) => {
    const idRef = React.useRef<number | undefined>(undefined);
    if (idRef.current === undefined) {
      idRef.current = ++templateLifecycleState.instanceId;
    }
    const [form] = Form.useForm();
    detailTestState.templateReadonly = props.readonly;
    templateLifecycleState.activeInstanceProps = props;

    React.useEffect(() => {
      const currentId = idRef.current;
      if (currentId === undefined) return;
      templateLifecycleState.instancesMounted.push(currentId);
      return () => {
        templateLifecycleState.instancesUnmounted.push(currentId);
      };
    }, []);

    React.useEffect(() => {
      if (props.formRef) {
        props.formRef.current = form;
      }
    }, [form, props.formRef]);

    React.useEffect(() => {
      if (!templateLifecycleState.restoreDraft || !props.draftScope) return;
      const pathname = `/orders/${routeState.params.kind}/${routeState.params.id}`;
      const draft = getFormDraft<{ customerReferenceNo?: string }>(
        getFormDraftKey(props.tabKey, pathname, props.draftScope),
      );
      if (!draft) return;
      form.setFieldsValue(draft);
      props.onDirtyChange?.(true);
    }, [form, props.draftScope, props.onDirtyChange, props.tabKey]);

    return (
      <Form
        form={form}
        initialValues={props.initialValues}
        onValuesChange={() => props.onDirtyChange?.(true)}
      >
        <Form.Item name="customerReferenceNo">
          <Input aria-label="客户参考号" />
        </Form.Item>
        {props.header}
        {props.footer}
      </Form>
    );
  },
}));

vi.mock('./templates', () => ({
  getAirTemplateSections: (props: { readonly?: boolean }) => {
    detailTestState.sectionReadonly = props.readonly;
    return [];
  },
  getSeaTemplateSections: (props: { readonly?: boolean }) => {
    detailTestState.sectionReadonly = props.readonly;
    return [];
  },
}));

vi.mock('./components/detail/OrderDetailHeader', () => ({
  default: (props: {
    moreMenuItems?: Array<{ key: string; onClick?: () => void }>;
  }) => (
    <div>
      <button
        type="button"
        onClick={() =>
          props.moreMenuItems
            ?.find((item) => item?.key === 'reload-data')
            ?.onClick?.()
        }
      >
        刷新数据
      </button>
    </div>
  ),
}));

const mockUpdateOrder = vi.mocked(orderServiceUpdateOrder);

describe('订单详情页草稿生命周期与记录身份', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    routeState.params = { kind: 'sea-export', id: 'ord-A' };
    templateLifecycleState.instanceId = 0;
    templateLifecycleState.instancesMounted = [];
    templateLifecycleState.instancesUnmounted = [];
    templateLifecycleState.activeInstanceProps = null;
    templateLifecycleState.restoreDraft = false;

    detailTestState.loadData.mockResolvedValue(undefined);
    detailTestState.refreshLockState.mockResolvedValue(null);
    detailTestState.allowedActions = [
      OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT,
    ];
    detailTestState.lockState = { isLocked: false } as API.OrderLockStateData;
    detailTestState.sectionReadonly = undefined;
    detailTestState.templateReadonly = undefined;
    detailTestState.customerReferenceNo = '服务端初始值';
    detailTestState.draftScope = 'user-1:org-1';
    detailTestState.orderAvailable = true;
  });

  it('详情 A 原地导航到 B 后，OrderFormTemplate 实例被重建且旧实例卸载', () => {
    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(templateLifecycleState.instancesMounted).toEqual([1]);
    expect(templateLifecycleState.instancesUnmounted).toEqual([]);

    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'A-未保存修改' },
    });
    expect(templateLifecycleState.activeInstanceProps.dirty).toBe(true);

    // 路由原地导航到 ord-B
    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );

    // 模板因 key={orderFormIdentity} 变化而被 React 整体重建：旧实例 1 卸载，新实例 2 挂载
    expect(templateLifecycleState.instancesUnmounted).toContain(1);
    expect(templateLifecycleState.instancesMounted).toEqual([1, 2]);
    expect(screen.getByLabelText('客户参考号')).toHaveValue('B-服务端初始值');
    expect(templateLifecycleState.activeInstanceProps.dirty).toBe(false);
  });

  it('详情 A 原地导航到 B 后，只恢复 B 自己的草稿并保持 dirty', async () => {
    const tabKey = resolveTabKey('/orders/sea-export/ord-B');
    saveFormDraft(
      getFormDraftKey(
        tabKey,
        '/orders/sea-export/ord-B',
        detailTestState.draftScope,
      ),
      { customerReferenceNo: 'B-自己的草稿' },
    );
    templateLifecycleState.restoreDraft = true;

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'A-未保存修改' },
    });
    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByLabelText('客户参考号')).toHaveValue('B-自己的草稿');
      expect(templateLifecycleState.activeInstanceProps.dirty).toBe(true);
    });
  });

  it('更新接口失败时保留当前草稿', async () => {
    const tabKey = resolveTabKey('/orders/sea-export/ord-A');
    const draftKey = getFormDraftKey(
      tabKey,
      '/orders/sea-export/ord-A',
      detailTestState.draftScope,
    );
    saveFormDraft(draftKey, { customerReferenceNo: 'TEMP-MODIFIED' });
    expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(true);

    mockUpdateOrder.mockRejectedValueOnce(new Error('保存失败'));

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    const formRef = templateLifecycleState.activeInstanceProps.formRef;
    const savePromise = templateLifecycleState.activeInstanceProps.onFinish(
      formRef.current.getFieldsValue(),
    );

    await expect(savePromise).resolves.toBe(false);
    expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(true);
  });

  it('更新接口成功后，即使后续 loadData 或锁状态刷新失败也已立即清理草稿', async () => {
    const tabKey = resolveTabKey('/orders/sea-export/ord-A');
    const draftKey = getFormDraftKey(
      tabKey,
      '/orders/sea-export/ord-A',
      detailTestState.draftScope,
    );
    saveFormDraft(draftKey, { customerReferenceNo: 'TEMP-MODIFIED' });
    expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(true);

    mockUpdateOrder.mockResolvedValueOnce({} as never);
    detailTestState.loadData.mockRejectedValueOnce(new Error('网络刷新失败'));

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    const formRef = templateLifecycleState.activeInstanceProps.formRef;
    await act(async () => {
      try {
        await templateLifecycleState.activeInstanceProps.onFinish(
          formRef.current.getFieldsValue(),
        );
      } catch {
        // 后续刷新错误
      }
    });

    // 即使 loadData 刷新抛错，更新成功后的草稿清理防线已经执行完毕
    expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(false);
  });

  it('显式刷新按真实 loadData 契约失败时保留草稿', async () => {
    const tabKey = resolveTabKey('/orders/sea-export/ord-A');
    const draftKey = getFormDraftKey(
      tabKey,
      '/orders/sea-export/ord-A',
      detailTestState.draftScope,
    );
    saveFormDraft(draftKey, { customerReferenceNo: 'DIRTY-DRAFT' });

    // 真实 useOrderDetailData 会吸收请求错误、清空当前 order 并正常 resolve。
    detailTestState.loadData.mockImplementationOnce(async () => {
      detailTestState.orderAvailable = false;
    });

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    const refreshBtn = screen.getByText('刷新数据');

    await act(async () => {
      fireEvent.click(refreshBtn);
    });

    expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(true);
    expect(screen.getByText('加载订单详情失败')).toBeInTheDocument();
  });

  it('显式刷新成功时清除草稿', async () => {
    const tabKey = resolveTabKey('/orders/sea-export/ord-A');
    const draftKey = getFormDraftKey(
      tabKey,
      '/orders/sea-export/ord-A',
      detailTestState.draftScope,
    );
    saveFormDraft(draftKey, { customerReferenceNo: 'DIRTY-DRAFT' });

    detailTestState.loadData.mockResolvedValueOnce(undefined);

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    await act(async () => {
      fireEvent.click(screen.getByText('刷新数据'));
    });

    await waitFor(() => {
      expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(false);
    });
  });

  it('底部重置修改按钮：清除当前草稿并回填 initialValues', () => {
    const tabKey = resolveTabKey('/orders/sea-export/ord-A');
    const draftKey = getFormDraftKey(
      tabKey,
      '/orders/sea-export/ord-A',
      detailTestState.draftScope,
    );
    saveFormDraft(draftKey, { customerReferenceNo: 'USER-EDITED' });

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    const undoBtn = screen.getByText('重置修改');
    act(() => {
      fireEvent.click(undoBtn);
    });

    expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(false);
    const input = screen.getByLabelText('客户参考号') as HTMLInputElement;
    expect(input.value).toBe('服务端初始值');
  });

  it('缺少编辑权限或业务锁单时，模板与分节均为 readonly', () => {
    detailTestState.allowedActions = [];
    detailTestState.lockState = { isLocked: true } as API.OrderLockStateData;

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(detailTestState.sectionReadonly).toBe(true);
    expect(detailTestState.templateReadonly).toBe(true);
    expect(screen.queryByText('重置修改')).not.toBeInTheDocument();
  });
});
