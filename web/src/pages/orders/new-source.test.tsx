import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, Form, Input } from 'antd';
import React from 'react';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getFormDraftKey,
  getFormDraftScope,
  saveFormDraft,
} from '@/components/layout/formDraft';
import { _clearAllTabCloseGuards } from '@/components/layout/tabCloseGuard';
import { OrderBusinessType, SeaDocumentStructure } from '@/enums.generated';
import { history } from '@/router/history';
import {
  orderServiceCreateOrder,
  orderServiceGetOrder,
} from '@/services/roncin/orderService';
import NewOrderPage from './new';
import { seaExportDefinition } from './order-kinds/sea-export/definition';
import type { TemplateProps } from './templates/types';

const permissions = vi.hoisted(() => ({
  create: true,
  read: true,
  organization: true,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: (_type: unknown, operation: 'create' | 'read') =>
      permissions[operation],
    canOperateOrganization: () => permissions.organization,
  }),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        displayName: '操作员',
        currentOrganization: { id: 'org-1' },
      },
    },
  }),
}));
vi.mock('@/router/history', () => ({ history: { push: vi.fn() } }));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceCreateOrder: vi.fn(),
  orderServiceGetOrder: vi.fn(),
}));
vi.mock('./use-order-create-options', () => {
  const options = {
    loading: false,
    error: null,
    retry: vi.fn(),
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: vi.fn().mockResolvedValue([]),
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: [],
  };
  return { useOrderCreateOptions: () => options };
});
vi.mock('./components/OrderPageHeader', () => ({
  default: ({ actions }: { actions?: React.ReactNode }) => <div>{actions}</div>,
}));

const getOrder = vi.mocked(orderServiceGetOrder);
const createOrder = vi.mocked(orderServiceCreateOrder);
let latestProps: TemplateProps;

function sourceOrder(id: string, masterNo = `MBL-${id}`): API.Order {
  return {
    id,
    organizationId: 'org-1',
    orderNo: `SE-${id}`,
    businessType: OrderBusinessType.BUSINESS_TYPE_SE,
    shippingLineId: 'line-1',
    seaDocumentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
    customerId: '来源客户',
    goodsDescription: '来源货物',
    destinationLocationId: '来源目的地',
    seaMasterBill: {
      masterBillId: `mbl-${id}`,
      masterNo,
      shippingLineId: 'line-1',
      shippingLineName: '来源船公司',
      transportExecutionId: `te-${id}`,
      version: '1',
      transportExecutionVersion: '2',
    },
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((done, fail) => {
    resolve = done;
    reject = fail;
  });
  return { promise, resolve, reject };
}

function mount(path = '/orders/sea-export/new?sourceOrderId=A') {
  const client = createTestQueryClient();
  const router = createMemoryRouter(
    [{ path: '/orders/:kind/new', element: <NewOrderPage /> }],
    { initialEntries: [path] },
  );
  render(
    <App>
      <QueryClientProvider client={client}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </App>,
  );
  return { client, router };
}

const input = (label: string) =>
  screen.getByRole('textbox', { name: label }) as HTMLInputElement;
const reference = () => input('客户业务编号');
async function ready() {
  await screen.findByRole('textbox', { name: '客户业务编号' });
}
async function navigate(
  router: ReturnType<typeof createMemoryRouter>,
  path: string,
) {
  await act(async () => {
    await router.navigate(path);
  });
  await ready();
}

describe('连续新增分单的来源与资源生命周期', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    _clearAllTabCloseGuards();
    permissions.create = true;
    permissions.read = true;
    permissions.organization = true;
    getOrder.mockImplementation(async ({ id }) => ({ data: sourceOrder(id) }));
    createOrder.mockResolvedValue(undefined as never);
    // 保留真实模板、Form、草稿与路由，只缩小字段树以集中验证页面编排。
    vi.spyOn(seaExportDefinition.form, 'buildSections').mockImplementation(
      (props) => {
        latestProps = props;
        return [
          {
            key: 'identity',
            title: '建单资料',
            content: (
              <>
                <Form.Item name="customerReferenceNo" label="客户业务编号">
                  <Input />
                </Form.Item>
                <Form.Item name="shippingLineId" label="船公司">
                  <Input />
                </Form.Item>
                <Form.Item name="seaMasterBillMasterNo" label="主单号">
                  <Input />
                </Form.Item>
                <Form.Item name="customerId" label="客户">
                  <Input />
                </Form.Item>
                <Form.Item name="goodsDescription" label="货物">
                  <Input />
                </Form.Item>
                <Form.Item name="destinationLocationId" label="目的地">
                  <Input />
                </Form.Item>
              </>
            ),
          },
        ];
      },
    );
  });
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    sessionStorage.clear();
    _clearAllTabCloseGuards();
  });

  it('本次来源读取成功前不挂表单，只预填主单共享身份，不复制本票资料', async () => {
    const request = deferred<API.GetOrderResponse>();
    getOrder.mockReturnValueOnce(request.promise);
    mount();
    expect(screen.getByText('正在读取来源主单...')).toBeInTheDocument();
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    await act(async () => request.resolve({ data: sourceOrder('A') }));
    await ready();
    expect(input('主单号')).toHaveValue('MBL-A');
    expect(input('船公司')).toHaveValue('line-1');
    for (const label of ['客户', '货物', '目的地'])
      expect(input(label)).toHaveValue('');
    expect(latestProps.seaHouseOrderCreateSource).toEqual({
      orderId: 'A',
      orderNo: 'SE-A',
      masterBillId: 'mbl-A',
      masterNo: 'MBL-A',
      shippingLineId: 'line-1',
      shippingLineName: '来源船公司',
      transportExecutionId: 'te-A',
    });
    expect(getOrder).toHaveBeenCalledWith(
      { id: 'A' },
      { skipErrorHandler: true },
    );
  });

  it('来源A的迟到读取不能回填已经进入的B表单', async () => {
    const request = deferred<API.GetOrderResponse>();
    getOrder.mockReturnValueOnce(request.promise);
    const { router } = mount();
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    fireEvent.change(reference(), { target: { value: 'B本次输入' } });
    await act(async () => request.resolve({ data: sourceOrder('A') }));
    expect(reference()).toHaveValue('B本次输入');
    expect(input('主单号')).toHaveValue('MBL-B');
    expect(latestProps.seaHouseOrderCreateSource?.orderId).toBe('B');
  });

  it('无读取/创建/当前公司能力及空重复来源参数都阻断且不请求来源', async () => {
    for (const operation of ['read', 'create', 'organization'] as const) {
      permissions[operation] = false;
      mount();
      expect(
        screen.getByText(
          operation === 'create'
            ? '无权新建此类订单'
            : '无权读取来源订单或新建分单',
        ),
      ).toBeInTheDocument();
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      cleanup();
      permissions[operation] = true;
    }
    for (const search of [
      '?sourceOrderId=',
      '?sourceOrderId=A&sourceOrderId=A',
    ]) {
      mount(`/orders/sea-export/new${search}`);
      expect(screen.getByText('无法从此订单新增分单')).toBeInTheDocument();
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      cleanup();
    }
    expect(getOrder).not.toHaveBeenCalled();
  });

  it('来源读取失败给出重试，DIRECT及跨公司来源均不退化为空白建单', async () => {
    getOrder.mockRejectedValueOnce(new Error('来源已失效'));
    mount();
    await screen.findByText('来源已失效');
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '重新加载来源订单' }));
    await ready();
    cleanup();
    for (const order of [
      {
        ...sourceOrder('A'),
        seaDocumentStructure:
          SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
      },
      { ...sourceOrder('A'), organizationId: 'org-2' },
    ]) {
      getOrder.mockResolvedValueOnce({ data: order });
      mount();
      await screen.findByText('来源订单加载失败');
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      cleanup();
    }
  });

  it('普通新建及来源A/B分别恢复草稿，无关query变化保留当前实例', async () => {
    const { router } = mount('/orders/sea-export/new');
    await ready();
    fireEvent.change(reference(), { target: { value: '普通草稿' } });
    await navigate(router, '/orders/sea-export/new?sourceOrderId=A');
    expect(reference()).toHaveValue('');
    fireEvent.change(reference(), { target: { value: 'A草稿' } });
    await navigate(router, '/orders/sea-export/new?extra=1&sourceOrderId=A');
    expect(reference()).toHaveValue('A草稿');
    expect(getOrder).toHaveBeenCalledTimes(1);
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    expect(reference()).toHaveValue('');
    expect(input('主单号')).toHaveValue('MBL-B');
    fireEvent.change(reference(), { target: { value: 'B草稿' } });
    await navigate(router, '/orders/sea-export/new?sourceOrderId=A');
    expect(reference()).toHaveValue('A草稿');
    await navigate(router, '/orders/sea-export/new');
    expect(reference()).toHaveValue('普通草稿');
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    expect(reference()).toHaveValue('B草稿');
  });

  it('来源加载完成后恢复本来源草稿，后台重查成功或失败均不覆盖输入和初始来源', async () => {
    saveFormDraft(
      getFormDraftKey(
        '/orders/sea-export',
        '/orders/sea-export/new?sourceOrderId=A',
        getFormDraftScope('user-1', 'org-1'),
      ),
      {
        customerReferenceNo: 'A恢复草稿',
        seaMasterBillMasterNo: '草稿改选主单',
      },
    );
    const { client } = mount();
    await ready();
    expect(reference()).toHaveValue('A恢复草稿');
    expect(input('主单号')).toHaveValue('草稿改选主单');
    fireEvent.change(reference(), { target: { value: '本次输入' } });
    getOrder.mockResolvedValueOnce({ data: sourceOrder('A', '后台新主单') });
    await act(async () => {
      await client.invalidateQueries({
        queryKey: ['orders', 'sea-house-create-source'],
      });
    });
    expect(reference()).toHaveValue('本次输入');
    expect(input('主单号')).toHaveValue('草稿改选主单');
    expect(latestProps.seaHouseOrderCreateSource?.masterNo).toBe('MBL-A');
    getOrder.mockRejectedValueOnce(new Error('后台重查失败'));
    await act(async () => {
      await client.invalidateQueries({
        queryKey: ['orders', 'sea-house-create-source'],
      });
    });
    expect(reference()).toHaveValue('本次输入');
  });

  it('返回已有缓存来源仍等待本次GET；失败阻断，重试使用当前来源身份', async () => {
    const { router } = mount();
    await ready();
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    const request = deferred<API.GetOrderResponse>();
    getOrder.mockReturnValueOnce(request.promise);
    await act(async () => {
      await router.navigate('/orders/sea-export/new?sourceOrderId=A');
    });
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    await act(async () => request.reject(new Error('缓存来源已失效')));
    await screen.findByText('缓存来源已失效');
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    getOrder.mockResolvedValueOnce({
      data: {
        ...sourceOrder('A', '最新主单'),
        seaMasterBill: {
          ...sourceOrder('A').seaMasterBill,
          masterNo: '最新主单',
          transportExecutionId: 'te-new',
        },
      },
    });
    fireEvent.click(screen.getByRole('button', { name: '重新加载来源订单' }));
    await ready();
    expect(input('主单号')).toHaveValue('最新主单');
    expect(latestProps.seaHouseOrderCreateSource?.transportExecutionId).toBe(
      'te-new',
    );
  });

  it('失败重试沿用幂等键，切换来源及往返重新生成；旧请求成功不导航或清新草稿', async () => {
    const { router } = mount();
    await ready();
    fireEvent.change(reference(), { target: { value: 'A草稿' } });
    fireEvent.click(screen.getByRole('button', { name: /创建订单/ }));
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /创建订单/ })).not.toHaveClass(
        'ant-btn-loading',
      ),
    );
    fireEvent.click(screen.getByRole('button', { name: /创建订单/ }));
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(2));
    const pending = deferred<API.CreateOrderResponse>();
    createOrder.mockReturnValueOnce(pending.promise);
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    fireEvent.change(reference(), { target: { value: 'B草稿' } });
    fireEvent.click(screen.getByRole('button', { name: /创建订单/ }));
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(3));
    await navigate(router, '/orders/sea-export/new?sourceOrderId=A');
    expect(reference()).toHaveValue('A草稿');
    await act(async () => pending.resolve({ data: { id: 'late-B' } }));
    expect(history.push).not.toHaveBeenCalled();
    expect(screen.queryByText('创建订单成功')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /创建订单/ }));
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(4));
    const keys = createOrder.mock.calls.map(
      ([payload]) => payload.idempotencyKey,
    );
    expect(keys[0]).toBeTruthy();
    expect(keys[0]).toBe(keys[1]);
    expect(keys[2]).not.toBe(keys[0]);
    expect(keys[3]).not.toBe(keys[0]);
    expect(keys[3]).not.toBe(keys[2]);
    expect(reference()).toHaveValue('A草稿');
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    expect(reference()).toHaveValue('B草稿');
  });

  it('切换来源后旧创建失败不反馈错误，当前创建失败仍保留输入与重试能力', async () => {
    const pending = deferred<API.CreateOrderResponse>();
    createOrder.mockReturnValueOnce(pending.promise);
    const { router } = mount();
    await ready();
    fireEvent.click(screen.getByRole('button', { name: /创建订单/ }));
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(1));
    expect(createOrder.mock.calls[0][1]).toEqual({ skipErrorHandler: true });
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    fireEvent.change(reference(), { target: { value: 'B未保存输入' } });
    await act(async () => pending.reject(new Error('A迟到失败')));
    expect(screen.queryByText('A迟到失败')).not.toBeInTheDocument();
    createOrder.mockRejectedValueOnce(new Error('当前创建失败'));
    fireEvent.click(screen.getByRole('button', { name: /创建订单/ }));
    await screen.findByText('当前创建失败');
    expect(screen.getAllByText('当前创建失败')).toHaveLength(1);
    expect(reference()).toHaveValue('B未保存输入');
    expect(history.push).not.toHaveBeenCalled();
    createOrder.mockResolvedValueOnce(undefined as never);
    fireEvent.click(screen.getByRole('button', { name: /创建订单/ }));
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(3));
    expect(createOrder.mock.calls[2][0].idempotencyKey).toBe(
      createOrder.mock.calls[1][0].idempotencyKey,
    );
    expect(reference()).toHaveValue('B未保存输入');
  });
});
