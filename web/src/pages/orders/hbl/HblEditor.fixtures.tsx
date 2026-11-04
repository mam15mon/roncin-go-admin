import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { beforeEach, vi } from 'vitest';
import { MasterDataKind } from '@/enums.generated';
import HblEditor from './HblEditor';
import type { HblBundle } from './model';

export { state };

const state = vi.hoisted(() => ({
  locked: false,
  canUpdate: true,
  canContainers: true,
  bundle: {} as HblBundle,
  read: vi.fn(),
  documents: vi.fn(),
  save: vi.fn(),
  containers: vi.fn(),
  shared: vi.fn(),
  preview: vi.fn(),
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: (_business: number, action: string) =>
      action === 'update'
        ? state.canUpdate
        : action === 'container.read'
          ? state.canContainers
          : true,
  }),
}));
vi.mock('../use-order-lock-state', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../use-order-lock-state')>()),
  useOrderLockState: () => ({
    state: { isLocked: state.locked },
    loading: false,
    error: null,
    refresh: async () => null,
  }),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: state.read,
}));
vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceGetSeaOrderDocuments: state.documents,
  seaDocumentServiceUpdateSeaHouseBill: state.save,
}));
vi.mock('@/services/roncin/orderContainerService', () => ({
  orderContainerServiceListContainers: state.containers,
}));
vi.mock('@/services/roncin/seaSharedContainerService', () => ({
  seaSharedContainerServiceListSeaSharedContainers: state.shared,
}));
vi.mock('@/features/orders/options', () => ({
  getMasterDataOptions: async () => [
    {
      id: '20gp',
      code: '20GP',
      kind: MasterDataKind.MASTER_DATA_KIND_CONTAINER_SPEC,
    },
  ],
}));
vi.mock('./pdf', () => ({
  default: (props: {
    variant: string;
    model: { houseNo: string; content?: API.SeaBillContent };
    onClose: () => void;
  }) => {
    state.preview(props);
    return (
      <div role="dialog" aria-label="HBL 预览">
        <span>{props.variant}</span>
        <span>{props.model.houseNo}</span>
        <button type="button" onClick={props.onClose}>
          关闭预览
        </button>
      </div>
    );
  },
}));

beforeEach(() => {
  vi.clearAllMocks();
  window.sessionStorage.clear();
  state.locked = false;
  state.canUpdate = true;
  state.canContainers = true;
  state.bundle = {
    order: {
      id: 'order-1',
      orderNo: 'SE001',
      organizationId: 'org-1',
      version: '1',
      bookingNo: 'BOOK001',
      originLocationName: 'SHANGHAI',
      shipmentType: 1,
    },
    documents: {
      orderId: 'order-1',
      linkVersion: '2',
      allowedActions: [4],
      houseBill: {
        id: 'house-1',
        houseNo: 'HBL001',
        version: '3',
        issuerSource: 1,
        issuerOrganizationId: 'issuer-company-1',
        issuerOrganizationName: '融迅北京',
        issuerOrganizationNameEn: 'CONFIRMED ISSUER COMPANY LIMITED',
        printTemplate: 'RONCIN_SELF',
        content: { shipperText: '原发货人', goodsDescriptionText: '普通货物' },
      },
    },
  };
  state.read.mockImplementation(async () => ({
    success: true,
    data: state.bundle.order,
  }));
  state.documents.mockImplementation(async () => ({
    success: true,
    data: state.bundle.documents,
  }));
  state.save.mockImplementation(
    async (_params, input: API.UpdateSeaHouseBillRequest) => {
      state.bundle = {
        ...state.bundle,
        documents: {
          ...state.bundle.documents,
          linkVersion: '3',
          houseBill: {
            ...state.bundle.documents.houseBill,
            content: input.houseBill.content,
            printDetails: input.houseBill.printDetails,
            version: '4',
          },
        },
      };
      return { success: true, data: state.bundle.documents.houseBill };
    },
  );
  state.containers.mockResolvedValue({ success: true, data: [] });
  state.shared.mockResolvedValue({ success: true, data: [], total: 0 });
});

export function renderEditor() {
  const queryClient = createTestQueryClient();
  const router = createMemoryRouter(
    [
      {
        path: '/orders/sea-export/:id/hbl',
        element: <HblEditor initial={state.bundle} scope="user-1:org-1" />,
      },
      { path: '*', element: <div>已离开</div> },
    ],
    { initialEntries: ['/orders/sea-export/order-1/hbl'] },
  );
  const view = render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </App>
    </ConfigProvider>,
  );
  return { ...view, queryClient, router };
}

export async function preview() {
  await act(async () =>
    fireEvent.click(
      screen
        .getAllByText('预览／打印／下载')[0]
        .closest('button') as HTMLElement,
    ),
  );
  await screen.findByRole('dialog', { name: 'HBL 预览' });
}

export async function save() {
  await act(async () =>
    fireEvent.click(screen.getAllByRole('button', { name: /保存制作资料/ })[0]),
  );
  await waitFor(() => expect(state.save).toHaveBeenCalled());
}
