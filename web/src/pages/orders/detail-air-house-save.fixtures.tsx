import { renderWithDataRouter } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, ConfigProvider, Form, type FormInstance, Input } from 'antd';
import React, { type ComponentType } from 'react';
import { expect, vi } from 'vitest';
import { OrderAllowedAction, OrderBusinessType } from '@/enums.generated';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import type { OrderDetailFeaturesProps } from './order-kinds/types';
import type { TemplateProps } from './templates/types';

vi.mock('./order-kinds/air-export/AirExportDetailFeatures', () => ({
  default: ({ children }: OrderDetailFeaturesProps) => children({}),
}));
vi.mock('./templates/air-template', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('./templates/air-template')>();
  return {
    ...actual,
    getAirTemplateSections: (props: TemplateProps) => [
      {
        key: 'metadata',
        title: '空运订单资料',
        content: <AirSaveMetadata />,
      },
      // 保留真实状态、原因框与正文逻辑，只裁剪无关主数据及共享主单工作台。
      ...actual
        .getAirTemplateSections(props)
        .filter((section) => section.key === 'airDocuments'),
    ],
  };
});

const state = vi.hoisted(() => ({
  order: {} as API.Order,
  routeKind: 'air-export',
  routeId: 'order-1',
  form: undefined as FormInstance | undefined,
  loadData: vi.fn(),
}));
let OrderDetailPage: ComponentType;

vi.mock('react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router')>()),
  useParams: () => ({ kind: state.routeKind, id: state.routeId }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: () => true,
  }),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceUpdateOrder: vi.fn(),
  orderServiceGetOrder: vi.fn(),
}));
vi.mock('./use-order-detail-data', () => ({
  useOrderDetailData: () => ({
    loading: false,
    error: null,
    order: state.order,
    seaDocuments: undefined,
    shippingDocs: [],
    personnel: [],
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: async () => [],
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: [],
    draftScope: 'user-1:org-1',
    loadData: state.loadData,
  }),
}));
vi.mock('./use-order-lock-state', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./use-order-lock-state')>()),
  useOrderLockState: () => ({
    state: { isLocked: false },
    loading: false,
    error: null,
    refresh: async () => null,
  }),
}));
vi.mock('./components/detail/OrderDetailHeader', () => ({
  default: ({ onSave, saving }: { onSave: () => void; saving: boolean }) => (
    <button type="button" onClick={onSave} disabled={saving}>
      保存修改
    </button>
  ),
}));

export function renderAirSaveDetail() {
  return renderWithDataRouter(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <OrderDetailPage />
      </App>
    </ConfigProvider>,
    {
      routerProps: {
        initialEntries: [`/orders/${state.routeKind}/${state.routeId}`],
      },
    },
  );
}

export function getAirUpdateMock() {
  return vi.mocked(orderServiceUpdateOrder);
}

export async function saveAirDetail() {
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: '保存修改' }));
  });
}

function AirSaveMetadata() {
  state.form = Form.useFormInstance();
  return (
    <>
      <Form.Item name="airExport" hidden>
        <span />
      </Form.Item>
      <Form.Item name="airMasterId" hidden>
        <span />
      </Form.Item>
      <Form.Item name="airMasterExpectedVersion" hidden>
        <span />
      </Form.Item>
      <Form.Item name="customerReferenceNo" label="客户参考号">
        <Input />
      </Form.Item>
    </>
  );
}

export async function resetAirSaveState(houseStatus = 'DRAFT') {
  OrderDetailPage = (await import('./detail')).default;
  vi.clearAllMocks();
  sessionStorage.clear();
  state.form = undefined;
  state.routeId = 'order-1';
  state.routeKind = 'air-export';
  state.order = {
    id: 'order-1',
    version: '4',
    organizationId: 'org-1',
    businessType: OrderBusinessType.BUSINESS_TYPE_AE,
    customerId: 'customer-1',
    orderDate: '2026-10-01',
    allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT],
    orderNo: 'AE0001',
    seaMasterBill: undefined,
    customerReferenceNo: 'initial-reference',
    airExport: {
      documentMode: 'HOUSE',
      active: true,
      house: {
        id: 'hawb-1',
        houseNo: 'HAWB-001',
        status: houseStatus,
        version: '2',
        content: { remarks: 'initial-content' },
      },
      linkId: 'link-1',
      linkVersion: '1',
      master: {
        id: 'mawb-1',
        organizationId: 'org-1',
        businessType: 'AE',
        version: '3',
        content: {},
      },
    },
  };
  state.loadData.mockImplementation(async () => ({
    order: state.order,
    shippingDocs: [],
    personnel: [],
  }));
  getAirUpdateMock().mockImplementation(async (_params, payload) => ({
    success: true,
    data: commitAirSave(payload),
  }));
}

export function commitAirSave(payload: API.UpdateOrderRequest): API.Order {
  const previous = state.order.airExport;
  const input = payload.airExport;
  const changed =
    previous?.house?.status !== input?.house?.status ||
    JSON.stringify(previous?.house?.content) !==
      JSON.stringify(input?.house?.content ?? previous?.house?.content);
  state.order = {
    ...state.order,
    version: String(BigInt(payload.expectedVersion ?? '0') + 1n),
    customerReferenceNo: payload.customerReferenceNo,
    airExport: {
      ...previous,
      linkVersion: String(
        BigInt(previous?.linkVersion ?? '0') + (changed ? 1n : 0n),
      ),
      house: previous?.house
        ? {
            ...previous.house,
            status: input?.house?.status,
            content: input?.house?.content ?? previous.house.content,
            version: String(
              BigInt(previous.house.version ?? '0') + (changed ? 1n : 0n),
            ),
          }
        : undefined,
    },
  };
  return state.order;
}

export async function setHouseStatus(label: '已签发' | '资料待完善') {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: '本票分单状态' }));
  fireEvent.click(
    await screen.findByText(label, {
      selector: '.ant-select-item-option-content',
    }),
  );
}

export function rerenderAirSavePage(
  view: ReturnType<typeof renderAirSaveDetail>,
) {
  view.rerender(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <OrderDetailPage />
      </App>
    </ConfigProvider>,
  );
}

export function getAirSaveState() {
  return state;
}

export async function openAirHouseRemarks() {
  const house = screen.getByRole('tabpanel', { name: '分运单 HAWB' });
  const more = within(house)
    .getByText('更多运单资料')
    .closest('[role="button"]');
  if (!more) throw new Error('未找到更多运单资料入口');
  await waitFor(() => expect(more).toBeVisible());
  if (more.getAttribute('aria-expanded') !== 'true') fireEvent.click(more);
  const trigger = within(house)
    .getByText('签发与备注')
    .closest('[role="button"]');
  if (!trigger) throw new Error('未找到签发与备注入口');
  await waitFor(() => expect(trigger).toBeVisible());
  if (trigger.getAttribute('aria-expanded') !== 'true')
    fireEvent.click(trigger);
  return screen.findByLabelText('单证备注');
}
