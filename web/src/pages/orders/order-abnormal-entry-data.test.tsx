import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { useRef, useState } from 'react';
import {
  createMemoryRouter,
  RouterProvider,
  useLocation,
  useParams,
} from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { InitialState } from '@/app/AppProvider';
import { appInitialStateContext } from '@/app/AppProvider';
import { getOrderPersonnelOptions } from '@/features/orders/options';
import { bindRouter } from '@/router/history';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import type { AbnormalCasePanelRef } from './abnormal-case-panel';
import { getOrderKindDefinition } from './order-kinds/registry';
import { useOrderAbnormalEntry } from './use-order-abnormal-entry';
import { useOrderDetailData } from './use-order-detail-data';

vi.mock('@/features/orders/options', () => ({
  getOrderPersonnelOptions: vi.fn(),
}));
vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceGetSeaOrderDocuments: vi
    .fn()
    .mockResolvedValue({
      data: {
        documentStructure: 2,
        linkVersion: '1',
        masterBill: { id: 'mbl-1', version: '1', content: {} },
      },
    }),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: vi.fn(),
}));
vi.mock('@/services/roncin/orderPersonnelService', () => ({
  orderPersonnelServiceListPersonnel: vi.fn(),
}));
vi.mock('@/services/roncin/orderShippingDocumentService', () => ({
  orderShippingDocumentServiceListShippingDocuments: vi.fn(),
}));
vi.mock('./common', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./common')>()),
  fetchOrderMasterData: vi.fn().mockResolvedValue({
    cargoCategoryOptions: [],
    seaLocationOptions: [],
    airLocationOptions: [],
    currencyOptions: [],
    masterOptions: [],
  }),
}));

const openPanel = vi.fn();

function Entry() {
  const { kind, id } = useParams();
  const definition = getOrderKindDefinition(kind);
  const data = useOrderDetailData(id, definition);
  const panelRef = useRef<AbnormalCasePanelRef>({ open: openPanel });
  const location = useLocation();
  useOrderAbnormalEntry({
    order: data.order,
    orderId: id,
    kind,
    loading: data.loading,
    error: Boolean(data.error),
    canRead: true,
    panelRef,
  });
  return (
    <>
      <div>{data.order?.orderNo}</div>
      <div data-testid="personnel-count">{data.personnel.length}</div>
      <div data-testid="option-count">{data.personnelOptions.length}</div>
      <div data-testid="query">{location.search}</div>
    </>
  );
}

function renderEntry(kind: string, extraPermissions: string[] = []) {
  const code = kind === 'sea-export' ? 'se' : 'ae';
  const user = (extras: string[]): API.CurrentUser => ({
    id: 'employee',
    currentOrganization: { id: 'company', kind: 2 },
    permissionCapabilities: ['read', 'abnormal_case.read', ...extras].map(
      (operation) => ({
        key: `business.order.${code}.${operation}`,
        dataScope: 'organization',
      }),
    ),
  });
  const router = createMemoryRouter(
    [{ path: '/orders/:kind/:id', element: <Entry /> }],
    { initialEntries: [`/orders/${kind}/order-1?panel=abnormal`] },
  );
  bindRouter(router);
  function Fixture() {
    const [initialState, setInitialState] = useState<InitialState | undefined>({
      currentUser: user(extraPermissions),
    });
    return (
      <appInitialStateContext.Provider
        value={{
          initialState,
          setInitialState,
          refresh: () => {},
          loading: false,
        }}
      >
        <button
          type="button"
          onClick={() => setInitialState({ currentUser: user([]) })}
        >
          收回人员权限
        </button>
        <RouterProvider router={router} />
      </appInitialStateContext.Provider>
    );
  }
  return render(
    <QueryClientProvider client={createTestQueryClient()}>
      <Fixture />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(orderServiceGetOrder).mockResolvedValue({
    data: { id: 'order-1', orderNo: '订单001' },
  });
  vi.mocked(orderPersonnelServiceListPersonnel).mockResolvedValue({
    data: [{ id: 'personnel-1', userId: 'employee' }],
  });
  vi.mocked(getOrderPersonnelOptions).mockResolvedValue([
    { userId: 'employee', displayName: '员工甲' },
  ]);
});

describe('异常直达的真实详情权限链', () => {
  it.each(['sea-export', 'air-export'])(
    '%s 仅订单读取和异常读取即可加载并打开面板，不调用无权的人员接口',
    async (kind) => {
      renderEntry(kind);
      expect(await screen.findByText('订单001')).toBeInTheDocument();
      await waitFor(() => expect(openPanel).toHaveBeenCalledTimes(1));
      expect(screen.getByTestId('query')).toBeEmptyDOMElement();
      expect(orderPersonnelServiceListPersonnel).not.toHaveBeenCalled();
      expect(getOrderPersonnelOptions).not.toHaveBeenCalled();
      expect(screen.getByTestId('personnel-count')).toHaveTextContent('0');
      expect(screen.getByTestId('option-count')).toHaveTextContent('0');
    },
  );

  it('只有人员读取时加载原人员，不调用要求创建权限的人员候选', async () => {
    renderEntry('sea-export', ['personnel.read']);
    expect(await screen.findByText('订单001')).toBeInTheDocument();
    expect(orderPersonnelServiceListPersonnel).toHaveBeenCalledOnce();
    expect(getOrderPersonnelOptions).not.toHaveBeenCalled();
    expect(screen.getByTestId('personnel-count')).toHaveTextContent('1');
  });

  it('只有创建权限时可拉人员候选，不读取未获授权的订单人员', async () => {
    renderEntry('sea-export', ['create']);
    expect(await screen.findByText('订单001')).toBeInTheDocument();
    expect(orderPersonnelServiceListPersonnel).not.toHaveBeenCalled();
    expect(getOrderPersonnelOptions).toHaveBeenCalledOnce();
    expect(screen.getByTestId('personnel-count')).toHaveTextContent('0');
    expect(screen.getByTestId('option-count')).toHaveTextContent('1');
  });

  it('同一用户和订单收回人员权限后立即切换缓存，不继续展示旧人员', async () => {
    renderEntry('sea-export', ['personnel.read', 'create']);
    expect(await screen.findByText('订单001')).toBeInTheDocument();
    expect(screen.getByTestId('personnel-count')).toHaveTextContent('1');
    expect(screen.getByTestId('option-count')).toHaveTextContent('1');
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '收回人员权限' }));
    });
    await waitFor(() =>
      expect(screen.getByTestId('personnel-count')).toHaveTextContent('0'),
    );
    expect(screen.getByTestId('option-count')).toHaveTextContent('0');
    expect(orderPersonnelServiceListPersonnel).toHaveBeenCalledOnce();
    expect(getOrderPersonnelOptions).toHaveBeenCalledOnce();
    expect(orderServiceGetOrder).toHaveBeenCalledTimes(2);
  });
});
