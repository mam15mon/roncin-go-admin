import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import { App, ConfigProvider, Form, Input } from 'antd';
import React from 'react';
import { vi } from 'vitest';
import { getFormDraftKey } from '@/components/layout/formDraft';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { OrderAllowedAction, SeaDocumentStructure } from '@/enums.generated';
import {
  orderServiceGetOrder,
  orderServiceUpdateOrder,
} from '@/services/roncin/orderService';
import OrderDetailPage from './detail';
import type { OrderDetailFeaturesProps } from './order-kinds/types';
import { useSeaMasterBillCandidate } from './templates/components/sea/useSeaMasterBillCandidate';
import type { TemplateProps } from './templates/types';

const state = vi.hoisted(() => ({
  order: {} as API.Order,
  documents: {} as API.SeaOrderDocuments,
  loadData: vi.fn(),
  onOrderDataChanged: undefined as TemplateProps['onOrderDataChanged'],
  routeKind: 'sea-export',
  routeId: 'order-1',
}));
const pathname = '/orders/sea-export/order-1';
const draftScope = 'user-1:org-1';
export const draftKey = getFormDraftKey(
  resolveTabKey(pathname),
  pathname,
  draftScope,
);
export const ports = [
  ['originLocationId', '起运港'],
  ['dischargeLocationId', '卸货港'],
  ['transitLocationId', '中转港'],
] as const;

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
  orderServiceMatchSeaMasterBillCandidate: vi.fn(),
  orderServiceGetOrder: vi.fn(),
}));
vi.mock('./use-order-detail-data', () => ({
  useOrderDetailData: () => ({
    loading: false,
    error: null,
    order: state.order,
    seaDocuments: state.documents,
    shippingDocs: [],
    personnel: [],
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: async () => [],
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: [],
    draftScope,
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
vi.mock('./order-kinds/sea-export/SeaExportDetailFeatures', () => ({
  default: ({ children }: OrderDetailFeaturesProps) => children({}),
}));
vi.mock('./components/detail/OrderDetailHeader', () => ({
  default: ({
    onSave,
    onReset,
    saving,
    moreMenuItems,
  }: {
    onSave: () => void;
    onReset: () => void;
    saving: boolean;
    moreMenuItems?: Array<{ key?: React.Key; onClick?: () => void }>;
  }) => (
    <>
      <button type="button" onClick={onSave} disabled={saving}>
        保存修改
      </button>
      <button type="button" onClick={onReset}>
        重置修改
      </button>
      <button
        type="button"
        onClick={() =>
          moreMenuItems?.find((item) => item.key === 'reload-data')?.onClick?.()
        }
      >
        刷新数据
      </button>
    </>
  ),
}));
vi.mock('./templates', () => ({
  getSeaTemplateSections: (props: TemplateProps) => {
    state.onOrderDataChanged = props.onOrderDataChanged;
    return [
      { key: 'transport', title: '运输信息', content: <TransportFields /> },
    ];
  },
}));

/** 使用真实候选 Hook 和 Form，仅保留本次保存相关的可见字段。 */
function TransportFields() {
  useSeaMasterBillCandidate(true, []);
  return (
    <>
      <Form.Item name="seaMasterBillMasterNo" label="主单号">
        <Input />
      </Form.Item>
      <Form.Item name="shippingLineId" label="船公司">
        <Input />
      </Form.Item>
      {ports.map(([name, label]) => (
        <Form.Item key={name} name={name} label={label}>
          <Input />
        </Form.Item>
      ))}
    </>
  );
}

export function renderDetail() {
  return renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <OrderDetailPage />
      </App>
    </ConfigProvider>,
    { routerProps: { initialEntries: [pathname] } },
  );
}

export async function save() {
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: '保存修改' }));
  });
}

export function bundle() {
  return {
    order: state.order,
    seaDocuments: state.documents,
    shippingDocs: [],
    personnel: [],
  };
}

export function advanceVersion(version: string) {
  state.order = {
    ...state.order,
    version,
    seaMasterBill: { ...state.order.seaMasterBill, version },
  };
  state.documents = {
    ...state.documents,
    masterBill: { ...state.documents.masterBill, version },
  };
}

export function getPortSaveState() {
  return state;
}

export function resetPortSaveState() {
  vi.clearAllMocks();
  sessionStorage.clear();
  state.onOrderDataChanged = undefined;
  state.routeKind = 'sea-export';
  state.routeId = 'order-1';
  state.order = {
    id: 'order-1',
    orderNo: 'SE0001',
    organizationId: 'org-1',
    version: '4',
    createdAt: '2026-10-01T00:00:00Z',
    orderDate: '2026-10-01',
    customerId: 'customer-1',
    shippingLineId: 'carrier-1',
    originLocationId: 'origin-old',
    dischargeLocationId: 'discharge-old',
    transitLocationId: 'transit-old',
    seaDocumentLinkVersion: '1',
    seaDocumentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
    allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT],
    seaMasterBill: {
      masterBillId: 'mbl-1',
      masterNo: 'MBL123',
      shippingLineId: 'carrier-1',
      version: '4',
      memberCount: 1,
    },
  };
  state.documents = {
    documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
    linkVersion: '1',
    masterBill: { id: 'mbl-1', version: '4', content: {} },
  };
  state.loadData.mockImplementation(async () => bundle());
  vi.mocked(orderServiceUpdateOrder).mockResolvedValue({
    success: true,
    data: { id: 'order-1' },
  });
  vi.mocked(orderServiceGetOrder).mockImplementation(async () => ({
    data: state.order,
  }));
}

export function getOrderUpdateMock() {
  return vi.mocked(orderServiceUpdateOrder);
}

export function getOrderReadMock() {
  return vi.mocked(orderServiceGetOrder);
}
