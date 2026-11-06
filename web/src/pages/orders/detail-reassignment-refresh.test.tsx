import { renderWithApp as render } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App, Form, Input } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderFlowStatus } from '@/enums.generated';
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
  closureStatus: 1,
  masterNo: 'OLDMBL123',
  version: '1',
  dirty: false,
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
  history: {
    get location() {
      return window.location;
    },
    push: vi.fn(),
  },
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
            version: detailTestState.version,
            seaMasterBill: {
              masterNo: detailTestState.masterNo,
              version: detailTestState.version,
            },
            seaDocumentLinkVersion: '1',
            seaDocumentStructure: 2,
            flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
            closureStatus: detailTestState.closureStatus,
            customerReferenceNo: detailTestState.customerReferenceNo,
            allowedActions: detailTestState.allowedActions,
          }
        : undefined,
      seaDocuments: {
        documentStructure: 2,
        linkVersion: '1',
        masterBill: {
          id: 'mbl-1',
          version: detailTestState.version,
          content: { remark: detailTestState.masterNo },
        },
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
      loadData: async () => {
        await detailTestState.loadData(orderId);
        return {
          order: { id: orderId, version: '1' },
          seaDocuments: { documentStructure: 2, linkVersion: '1' },
          shippingDocs: [],
          personnel: [],
        };
      },
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
        <Form.Item name="seaMasterBillMasterNo">
          <Input aria-label="母单号" />
        </Form.Item>
        <Form.Item name="version">
          <Input aria-label="内部订单版本" />
        </Form.Item>
        <Form.Item name="seaMasterBillContentVersion">
          <Input aria-label="内部单证版本" />
        </Form.Item>
        <Form.Item name={['seaMasterBillContent', 'remark']}>
          <Input aria-label="单证备注" />
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
    businessMenuItems?: import('antd').MenuProps['items'];
    moreMenuItems?: Array<{
      key?: React.Key;
      onClick?: () => void;
    }>;
    onSynchronizeLockChange?: () => Promise<void>;
  }) => (
    <div>
      {props.businessMenuItems?.map((item) => {
        const action = item as {
          key: string;
          label: React.ReactNode;
          disabled?: boolean;
          onClick?: () => void;
        };
        return (
          <button
            key={action.key}
            type="button"
            disabled={action.disabled}
            onClick={action.onClick}
          >
            {action.label}
          </button>
        );
      })}
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

vi.mock('@/components/layout/tabCloseGuard', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/components/layout/tabCloseGuard')
  >()),
  isTabDirty: () => detailTestState.dirty,
}));

vi.mock('./components/drawers/SeaOrderReassignmentModal', () => ({
  default: ({
    open,
    onSuccess,
  }: {
    open: boolean;
    onSuccess: () => Promise<void>;
  }) =>
    open ? (
      <button
        type="button"
        onClick={() =>
          void onSuccess().catch((error: Error) => {
            detailTestState.customerReferenceNo = error.message;
          })
        }
      >
        模拟改配已成功
      </button>
    ) : null,
}));

describe('改配完成后的详情表单同步', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.params = { kind: 'sea-export', id: 'ord-A' };
    detailTestState.masterNo = 'OLDMBL123';
    detailTestState.version = '1';
    detailTestState.customerReferenceNo = '服务端初始值';
    detailTestState.dirty = false;
    detailTestState.loadData.mockResolvedValue(undefined);
    mockGetChangeActions.mockResolvedValue({ data: { canReassign: true } });
  });

  it('改配后无需手动刷新即可回填母单、单证内容和提交基线', async () => {
    const pending = deferred<void>();
    detailTestState.loadData.mockImplementation(() => pending.promise);
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => expect(getReassignButton()).toBeEnabled());
    expect(screen.getByLabelText('母单号')).toHaveValue('OLDMBL123');
    fireEvent.click(getReassignButton());
    fireEvent.click(screen.getByRole('button', { name: '模拟改配已成功' }));
    await waitFor(() =>
      expect(detailTestState.loadData).toHaveBeenCalledOnce(),
    );
    detailTestState.masterNo = 'NEWMBL456';
    detailTestState.version = '2';
    await act(async () => pending.resolve());
    await waitFor(() =>
      expect(screen.getByLabelText('母单号')).toHaveValue('NEWMBL456'),
    );
    expect(screen.getByLabelText('单证备注')).toHaveValue('NEWMBL456');
    expect(screen.getByLabelText('内部订单版本')).toHaveValue('2');
    expect(screen.getByLabelText('内部单证版本')).toHaveValue('2');
  });

  it('有未保存输入时保留输入和原版本，提示显式刷新', async () => {
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => expect(getReassignButton()).toBeEnabled());
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: '未保存修改' },
    });
    detailTestState.dirty = true;
    detailTestState.masterNo = 'NEWMBL456';
    detailTestState.version = '2';
    fireEvent.click(getReassignButton());
    fireEvent.click(screen.getByRole('button', { name: '模拟改配已成功' }));
    await waitFor(() => expect(mockGetChangeActions).toHaveBeenCalledTimes(2));
    expect(screen.getByLabelText('客户参考号')).toHaveValue('未保存修改');
    expect(screen.getByLabelText('母单号')).toHaveValue('OLDMBL123');
    expect(screen.getByLabelText('内部订单版本')).toHaveValue('1');
    expect(detailTestState.customerReferenceNo).toContain('详情页有未保存修改');
  });

  it('刷新失败不清空详情内容或推进表单版本', async () => {
    detailTestState.loadData.mockRejectedValue(new Error('读取失败'));
    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => expect(getReassignButton()).toBeEnabled());
    fireEvent.click(getReassignButton());
    fireEvent.click(screen.getByRole('button', { name: '模拟改配已成功' }));
    await waitFor(() => expect(mockGetChangeActions).toHaveBeenCalledTimes(2));
    expect(screen.getByLabelText('母单号')).toHaveValue('OLDMBL123');
    expect(screen.getByLabelText('内部订单版本')).toHaveValue('1');
    expect(detailTestState.customerReferenceNo).toContain('详情刷新未完成');
  });
});
