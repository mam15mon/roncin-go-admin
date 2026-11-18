import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, render, screen } from '@testing-library/react';
import { App, Form, Input } from 'antd';
import React from 'react';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { afterEach, beforeEach, vi } from 'vitest';
import { _clearAllTabCloseGuards } from '@/components/layout/tabCloseGuard';
import { OrderBusinessType, SeaDocumentStructure } from '@/enums.generated';
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
const navigation = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock('@/router/history', () => ({ history: navigation }));
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

export {
  createOrder,
  deferred,
  getOrder,
  input,
  latestProps,
  mount,
  navigate,
  navigation,
  permissions,
  ready,
  reference,
  sourceOrder,
};
