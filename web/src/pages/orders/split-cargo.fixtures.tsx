import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { MemoryRouter } from 'react-router';
import { beforeEach, expect, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import SeaOrderSplitPage from './split';

const splitState = vi.hoisted(() => ({
  context: {} as API.SeaOrderSplitContextData,
  cargoAccess: true,
  locked: false,
}));

const splitServices = vi.hoisted(() => ({
  context: vi.fn(),
  preview: vi.fn(),
  execute: vi.fn(),
  listCargo: vi.fn(),
  addCargo: vi.fn(),
  push: vi.fn(),
}));

export { splitServices, splitState };

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: (_business: number, operation: string) =>
      !operation.startsWith('cargo_item.') || splitState.cargoAccess,
  }),
}));

vi.mock('react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router')>()),
  useParams: () => ({ id: 'split-order' }),
}));
vi.mock('@/router/history', () => ({
  history: {
    get location() {
      return window.location;
    },
    push: splitServices.push,
  },
}));
vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceGetSeaOrderSplitContext: splitServices.context,
  seaOrderChangeServicePreviewSeaOrderSplit: splitServices.preview,
  seaOrderChangeServiceExecuteSeaOrderSplit: splitServices.execute,
}));
vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceGetOrderLockState: async () => ({
    data: {
      orderId: 'split-order',
      businessType: OrderBusinessType.BUSINESS_TYPE_SE,
      isLocked: splitState.locked,
      orderVersion: splitState.context.orderVersion,
    },
  }),
}));
vi.mock('@/services/roncin/orderCargoItemService', () => ({
  orderCargoItemServiceListCargoItems: splitServices.listCargo,
  orderCargoItemServiceAddCargoItem: splitServices.addCargo,
  orderCargoItemServiceUpdateCargoItem: vi.fn(),
  orderCargoItemServiceRemoveCargoItem: vi.fn(),
}));
// 这些分节不参与本组的货物写入/分配/基线生命周期，保留实际货物抽屉与分配表。
vi.mock('./components/split/SplitResultsSection', () => ({
  default: () => null,
}));
vi.mock('./components/split/SplitFeesSection', () => ({ default: () => null }));

export function makeSplitContext(
  withCargo = false,
): API.SeaOrderSplitContextData {
  return {
    orderId: 'split-order',
    orderNo: 'SE2026110600123',
    orderVersion: '1',
    currentLinkVersion: '1',
    documentStructure: 'HOUSE',
    currentMasterBill: { id: 'master-1', masterNo: 'MBL123' },
    currentHouseBill: { id: 'house-1', houseNo: 'HBL123', version: '1' },
    containers: [
      { id: 'container-1', containerNo: 'MSCU1234567', version: '1' },
    ],
    cargoItems: withCargo
      ? [
          {
            id: 'cargo-1',
            cargoName: '测试机器',
            packageCount: 100,
            grossWeightKg: '2000',
            volumeCbm: '15',
            version: '1',
          },
        ]
      : [],
    draftFees: [],
    attachments: [],
  };
}

export function setupSplitCargoTests() {
  beforeEach(() => {
    vi.clearAllMocks();
    splitState.context = makeSplitContext();
    splitState.cargoAccess = true;
    splitState.locked = false;
    splitServices.context.mockImplementation(async () => ({
      data: splitState.context,
    }));
    splitServices.listCargo.mockImplementation(async () => ({
      data: (splitState.context.cargoItems ?? []).map((item) => ({
        ...item,
        grossWeightKg: Number(item.grossWeightKg),
        volumeCbm: Number(item.volumeCbm),
      })),
    }));
    splitServices.addCargo.mockImplementation(
      async (_params, values: API.AddCargoItemRequest) => {
        splitState.context = {
          ...splitState.context,
          orderVersion: '2',
          cargoItems: [
            {
              id: 'cargo-1',
              cargoName: values.cargoName,
              packageCount: values.packageCount,
              grossWeightKg: String(values.grossWeightKg),
              volumeCbm: String(values.volumeCbm),
              version: '1',
            },
          ],
        };
        return { success: true, data: { id: 'cargo-1' } };
      },
    );
    splitServices.preview.mockImplementation(
      async (_params, request: API.PreviewSeaOrderSplitRequest) => {
        const allocations = request.results.map(
          (result) => result.cargoAllocations?.[0],
        );
        const isValid =
          allocations.length === 2 &&
          allocations.every((item) => (item?.packageCount ?? 0) > 0) &&
          allocations.reduce(
            (total, item) => total + (item?.packageCount ?? 0),
            0,
          ) === 100;
        return {
          data: {
            isValid,
            conservationPassed: isValid,
            baseline: {
              packageCount: 100,
              grossWeightKg: '2000',
              volumeCbm: '15',
            },
            allocated: { packageCount: isValid ? 100 : 0 },
          },
        };
      },
    );
    splitServices.execute.mockResolvedValue({
      success: true,
      data: {
        splitEventId: 'event-1',
        originalOrder: { orderId: 'split-order' },
        createdOrders: [{ orderId: 'created-1', orderNo: 'SE2026110600124' }],
      },
    });
  });
}

export function renderSplit() {
  return renderWithClient(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <MemoryRouter>
        <App>
          <SeaOrderSplitPage />
        </App>
      </MemoryRouter>
    </ConfigProvider>,
  );
}

export async function pageReady() {
  const sourceButton = (await screen.findByText('录入实际货物')).closest(
    'button',
  ) as HTMLButtonElement;
  await waitFor(() => expect(sourceButton).toBeEnabled());
  return sourceButton;
}

export async function submitActualCargo() {
  fireEvent.click(await pageReady());
  const addButton = (await screen.findByText('添加货物明细')).closest(
    'button',
  ) as HTMLButtonElement;
  await waitFor(() => expect(addButton).toBeVisible());
  fireEvent.click(addButton);
  const nameInput = await screen.findByPlaceholderText('请输入货名');
  const dialog = nameInput.closest('[role="dialog"]') as HTMLElement;
  await waitFor(() => expect(dialog).toBeVisible());
  fireEvent.change(within(dialog).getByLabelText('货物名称'), {
    target: { value: '测试机器' },
  });
  fireEvent.change(within(dialog).getByLabelText('件数'), {
    target: { value: '100' },
  });
  fireEvent.change(within(dialog).getByLabelText('毛重 (KG)'), {
    target: { value: '2000' },
  });
  fireEvent.change(within(dialog).getByLabelText('体积 (CBM)'), {
    target: { value: '15' },
  });
  const submit = within(dialog).getByRole('button', { name: '确 定' });
  expect(submit).toBeVisible();
  fireEvent.click(submit);
  await waitFor(() => expect(splitServices.addCargo).toHaveBeenCalled());
  return dialog;
}

export async function addActualCargo() {
  await submitActualCargo();
  await screen.findByText('拆票数据需要刷新');
  const drawer = screen
    .getByText('订单货物明细 - SE2026110600123')
    .closest('[role="dialog"]') as HTMLElement;
  fireEvent.click(within(drawer).getByRole('button', { name: '关闭' }));
  await waitFor(() => expect(drawer).not.toBeVisible());
}

export function getCargoAllocationRows() {
  const cargoCard = screen
    .getByText('测试机器')
    .closest('.ant-card') as HTMLElement;
  return {
    original: within(cargoCard)
      .getByText('原票 (SE2026110600123)')
      .closest('tr') as HTMLElement,
    created: within(cargoCard)
      .getByText('拆出新票 1')
      .closest('tr') as HTMLElement,
  };
}

export function editCargoAllocation() {
  const { original, created } = getCargoAllocationRows();
  const fields = within(original).getAllByRole('textbox');
  fireEvent.change(within(original).getByRole('spinbutton'), {
    target: { value: '60' },
  });
  fireEvent.change(fields[0], { target: { value: '1200' } });
  fireEvent.change(fields[1], { target: { value: '9' } });
  fireEvent.click(within(created).getByText('填入剩余'));
}
