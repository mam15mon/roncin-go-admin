import { renderWithApp } from '@root/tests/renderWithApp';
import { act, cleanup } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { afterEach, beforeEach, vi } from 'vitest';
import { _clearAllTabCloseGuards } from '@/components/layout/tabCloseGuard';
import {
  OrderBusinessType,
  OrderClosureStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import ImportHandlingPanel, {
  type ImportHandlingSaveResult,
} from './ImportHandlingPanel';
import { IMPORT_SERVICES } from './model';

const services = vi.hoisted(() => ({
  get: vi.fn(),
  save: vi.fn(),
  events: vi.fn(),
  people: vi.fn(),
  files: vi.fn(),
  partners: vi.fn(),
  allowed: true,
}));

export { services };

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetImportHandling: services.get,
  orderServiceSaveImportHandling: services.save,
  orderServiceListImportHandlingEvents: services.events,
  orderServiceListPersonnelOptions: services.people,
}));
vi.mock('@/services/roncin/orderAttachmentService', () => ({
  orderAttachmentServiceListAttachments: services.files,
}));
vi.mock('@/features/partners', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/features/partners')>()),
  searchPartnerOptions: services.partners,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => services.allowed,
    canReadPartners: true,
    canOperateOrganization: (organizationId?: string) =>
      organizationId === 'company',
  }),
}));
vi.mock('@/components/ui', async (original) => ({
  ...(await original<typeof import('@/components/ui')>()),
  ResponsiveProTable: () => null,
}));

export const order: API.Order = {
  id: 'order',
  organizationId: 'company',
  businessType: OrderBusinessType.BUSINESS_TYPE_SI,
  version: '7',
  terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
  closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
};
export function response(
  version = '7',
  steps?: API.ImportHandlingStep[],
): API.GetImportHandlingResponse {
  return {
    success: true,
    data: {
      orderId: 'order',
      orderVersion: version,
      steps: steps ?? IMPORT_SERVICES.map(({ kind }) => ({ kind })),
    },
  };
}
export function setupTests() {
  beforeEach(() => {
    vi.clearAllMocks();
    services.allowed = true;
    vi.spyOn(document, 'hasFocus').mockReturnValue(true);
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    services.get.mockResolvedValue(response());
    services.save.mockResolvedValue(undefined);
    services.events.mockResolvedValue({
      success: true,
      data: { items: [], total: 0 },
    });
    services.people.mockResolvedValue({ data: [] });
    services.files.mockResolvedValue({ data: [] });
    services.partners.mockResolvedValue([]);
  });
  afterEach(() => {
    cleanup();
    _clearAllTabCloseGuards();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });
}
export function renderPanel(
  extra: {
    expectedVersion?: string;
    order?: API.Order;
    onSaved?: (result: ImportHandlingSaveResult) => Promise<void>;
  } = {},
) {
  const onSaved = extra.onSaved ?? vi.fn().mockResolvedValue(undefined);
  const ui = (
    version: string = extra.expectedVersion ?? '7',
    targetOrder = extra.order ?? order,
  ) => (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <ImportHandlingPanel
        order={targetOrder}
        expectedVersion={version}
        onSaved={onSaved}
      />
    </ConfigProvider>
  );
  const view = renderWithApp(ui(), {
    user: { id: 'user', currentOrganization: { id: 'company' } },
    routerProps: {
      initialEntries: [
        `/orders/${extra.order?.businessType === OrderBusinessType.BUSINESS_TYPE_AI ? 'air-import' : 'sea-import'}/order`,
      ],
    },
  });
  return { ...view, onSaved, ui };
}
export async function flush() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}
