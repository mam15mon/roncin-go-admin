import fs from 'node:fs';
import path from 'node:path';
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
  activeHandleFinish: null as
    | ((values: any) => Promise<boolean | undefined>)
    | null,
  internalDirty: false,
  resetToCalls: 0,
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

vi.mock('@/components/ui/order-template/OrderFormTemplate', async () => {
  const formDraft = await import('@/components/layout/formDraft');
  return {
    OrderFormTemplate: (props: {
      header?: React.ReactNode;
      footer?: React.ReactNode;
      readonly?: boolean;
      formRef?: React.MutableRefObject<ReturnType<typeof Form.useForm>[0]>;
      actionsRef?: React.MutableRefObject<{
        resetTo: (values?: { customerReferenceNo?: string }) => void;
      } | null>;
      initialValues?: { customerReferenceNo?: string };
      onFinish?: (values: any) => Promise<any>;
      draftScope?: string;
      tabKey?: string;
      draftPathname?: string;
    }) => {
      const idRef = React.useRef<number | undefined>(undefined);
      if (idRef.current === undefined) {
        idRef.current = ++templateLifecycleState.instanceId;
      }
      const [form] = Form.useForm();
      const [internalDirty, setInternalDirty] = React.useState(false);
      detailTestState.templateReadonly = props.readonly;
      templateLifecycleState.activeInstanceProps = props;
      templateLifecycleState.internalDirty = internalDirty;

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

      const draftKey =
        props.tabKey && props.draftPathname && props.draftScope
          ? formDraft.getFormDraftKey(
              props.tabKey,
              props.draftPathname,
              props.draftScope,
            )
          : '';

      React.useImperativeHandle(
        props.actionsRef,
        () => ({
          resetTo: (values?: { customerReferenceNo?: string }) => {
            templateLifecycleState.resetToCalls += 1;
            formDraft.clearFormDraft(draftKey);
            form.resetFields();
            if (values) form.setFieldsValue(values);
            setInternalDirty(false);
          },
        }),
        [form, draftKey],
      );

      // 模拟真实模板：首次可编辑时机恢复本身份草稿并标记 dirty。
      React.useEffect(() => {
        if (props.readonly || !templateLifecycleState.restoreDraft) return;
        if (!draftKey) return;
        const draft = formDraft.getFormDraft<{
          customerReferenceNo?: string;
        }>(draftKey);
        if (!draft) return;
        form.setFieldsValue(draft);
        setInternalDirty(true);
      }, [form, draftKey, props.readonly]);

      // 模拟真实模板：提交成功由模板清当前草稿并复位 dirty。
      const handleFinish = async (values: any) => {
        const result = await props.onFinish?.(values);
        if (result !== false) {
          formDraft.clearFormDraft(draftKey);
          setInternalDirty(false);
        }
        return result;
      };
      templateLifecycleState.activeHandleFinish = handleFinish;

      return (
        <Form
          form={form}
          initialValues={props.initialValues}
          onValuesChange={() => setInternalDirty(true)}
          onFinish={handleFinish}
        >
          <Form.Item name="customerReferenceNo">
            <Input aria-label="客户参考号" />
          </Form.Item>
          {props.header}
          {props.footer}
        </Form>
      );
    },
  };
});

vi.mock('./templates', () => ({
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

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

describe('订单详情页草稿生命周期与记录身份', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    routeState.params = { kind: 'sea-export', id: 'ord-A' };
    templateLifecycleState.instanceId = 0;
    templateLifecycleState.instancesMounted = [];
    templateLifecycleState.instancesUnmounted = [];
    templateLifecycleState.activeInstanceProps = null;
    templateLifecycleState.activeHandleFinish = null;
    templateLifecycleState.internalDirty = false;
    templateLifecycleState.resetToCalls = 0;
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
    expect(templateLifecycleState.internalDirty).toBe(true);

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
    expect(templateLifecycleState.internalDirty).toBe(false);
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
      expect(templateLifecycleState.internalDirty).toBe(true);
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
    const finish = templateLifecycleState.activeHandleFinish;
    expect(finish).toBeTruthy();
    let saveResult: boolean | undefined;
    await act(async () => {
      saveResult = finish
        ? await finish(formRef.current.getFieldsValue())
        : undefined;
    });

    expect(saveResult).toBe(false);
    expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(true);
  });

  it('更新接口成功后立即清草稿并复位 dirty，后台刷新失败不改判保存结果', async () => {
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

    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'TEMP-MODIFIED' },
    });
    expect(templateLifecycleState.internalDirty).toBe(true);

    const formRef = templateLifecycleState.activeInstanceProps.formRef;
    const finish = templateLifecycleState.activeHandleFinish;
    expect(finish).toBeTruthy();
    let saveResult: boolean | undefined;
    await act(async () => {
      saveResult = finish
        ? await finish(formRef.current.getFieldsValue())
        : undefined;
    });

    // loadData 后台刷新失败不影响已落库的保存成功结果
    expect(saveResult).toBe(true);
    expect(hasTabDraft(tabKey, detailTestState.draftScope)).toBe(false);
    expect(templateLifecycleState.internalDirty).toBe(false);
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

  it('显式刷新成功时通过模板 resetTo 清草稿并回填最新服务端值', async () => {
    const tabKey = resolveTabKey('/orders/sea-export/ord-A');
    const draftKey = getFormDraftKey(
      tabKey,
      '/orders/sea-export/ord-A',
      detailTestState.draftScope,
    );
    saveFormDraft(draftKey, { customerReferenceNo: 'DIRTY-DRAFT' });

    detailTestState.loadData.mockResolvedValueOnce(undefined);
    detailTestState.customerReferenceNo = '刷新后的服务端值';

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
      expect(screen.getByLabelText('客户参考号')).toHaveValue(
        '刷新后的服务端值',
      );
      expect(templateLifecycleState.internalDirty).toBe(false);
      expect(templateLifecycleState.resetToCalls).toBe(1);
    });
  });

  it('同一订单连点两次刷新：最新刷新完成后，旧刷新迟到不得再次重置新输入', async () => {
    const firstLoad = deferred<void>();
    const secondLoad = deferred<void>();
    detailTestState.loadData
      .mockImplementationOnce(() => firstLoad.promise)
      .mockImplementationOnce(() => secondLoad.promise);

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    const refreshBtn = screen.getByText('刷新数据');
    await act(async () => {
      fireEvent.click(refreshBtn);
    });
    await act(async () => {
      fireEvent.click(refreshBtn);
    });

    // 最新刷新（第二次）先完成并按显式刷新语义重置一次
    await act(async () => {
      secondLoad.resolve();
    });
    await waitFor(() => {
      expect(templateLifecycleState.resetToCalls).toBe(1);
      expect(templateLifecycleState.internalDirty).toBe(false);
    });

    // 用户在重置之后开始新的输入
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: '重置后新输入的修改' },
    });
    expect(templateLifecycleState.internalDirty).toBe(true);

    // 旧刷新（第一次）最后迟到完成：不得再次 resetTo 抹掉新输入
    await act(async () => {
      firstLoad.resolve();
    });

    expect(screen.getByLabelText('客户参考号')).toHaveValue(
      '重置后新输入的修改',
    );
    expect(templateLifecycleState.internalDirty).toBe(true);
    expect(templateLifecycleState.resetToCalls).toBe(1);
  });

  it('A 发起刷新后切到 B：B 自己的刷新正常重置，A 迟到完成不影响 B', async () => {
    const loadA = deferred<void>();
    const loadB = deferred<void>();
    detailTestState.loadData
      .mockImplementationOnce(() => loadA.promise)
      .mockImplementationOnce(() => loadB.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    await act(async () => {
      fireEvent.click(screen.getByText('刷新数据'));
    });

    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );

    // B 上编辑后发起自己的刷新并成功，回填 B 的服务端值
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'B-编辑中的修改' },
    });
    await act(async () => {
      fireEvent.click(screen.getByText('刷新数据'));
    });
    await act(async () => {
      loadB.resolve();
    });
    await waitFor(() => {
      expect(templateLifecycleState.resetToCalls).toBe(1);
      expect(screen.getByLabelText('客户参考号')).toHaveValue('B-服务端初始值');
    });

    // A 的旧刷新最后完成：不得再次重置 B
    await act(async () => {
      loadA.resolve();
    });

    expect(screen.getByLabelText('客户参考号')).toHaveValue('B-服务端初始值');
    expect(templateLifecycleState.resetToCalls).toBe(1);
  });

  it('A 发起刷新后途经 B 回到 A：旧刷新迟到完成不重置新 A 实例的输入', async () => {
    const pendingLoad = deferred<void>();
    detailTestState.loadData.mockImplementationOnce(() => pendingLoad.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    await act(async () => {
      fireEvent.click(screen.getByText('刷新数据'));
    });

    // A → B → 回 A：期间未再点刷新，字符串身份与刷新序号均未变化
    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );
    routeState.params = { kind: 'sea-export', id: 'ord-A' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );

    // 在重新挂载的 A 实例上输入新内容
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'A-往返后的新输入' },
    });
    expect(templateLifecycleState.internalDirty).toBe(true);

    // 旧 A 的刷新此时才迟到完成：字符串身份相同，但令牌已被身份往返作废
    await act(async () => {
      pendingLoad.resolve();
    });

    expect(screen.getByLabelText('客户参考号')).toHaveValue('A-往返后的新输入');
    expect(templateLifecycleState.internalDirty).toBe(true);
    expect(templateLifecycleState.resetToCalls).toBe(0);
  });

  it('A 的显式刷新在导航到 B 后完成时，不得重置 B 的表单与 dirty', async () => {
    const pendingLoad = deferred<void>();
    detailTestState.loadData.mockImplementationOnce(() => pendingLoad.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'A-未保存修改' },
    });
    await act(async () => {
      fireEvent.click(screen.getByText('刷新数据'));
    });

    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(screen.getByLabelText('客户参考号')).toHaveValue('B-服务端初始值');
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'B-未保存修改' },
    });
    expect(templateLifecycleState.internalDirty).toBe(true);

    // A 发起的 loadData 在导航到 B 之后才结束
    await act(async () => {
      pendingLoad.resolve();
    });

    expect(screen.getByLabelText('客户参考号')).toHaveValue('B-未保存修改');
    expect(templateLifecycleState.internalDirty).toBe(true);
    expect(templateLifecycleState.resetToCalls).toBe(0);
  });

  it('底部重置修改按钮：通过模板 resetTo 清草稿并回填 initialValues', () => {
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
    expect(templateLifecycleState.internalDirty).toBe(false);
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

  it('页面源码不再包含草稿键计算、直接清草稿或受控 dirty 代码', () => {
    const source = fs.readFileSync(
      path.resolve(process.cwd(), 'src/pages/orders/detail.tsx'),
      'utf-8',
    );
    expect(source).not.toContain('getFormDraftKey');
    expect(source).not.toContain('clearFormDraft');
    expect(source).not.toContain('formDirtyState');
    expect(source).not.toContain('activeOrderFormIdentityRef');
    expect(source).not.toContain('setIsFormDirty');
    expect(source).not.toContain('onDirtyChange');
    expect(source).not.toContain('dirty={');
  });
});
