import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { beforeEach, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { getOrderKindDefinitionByBusinessType } from '../order-kinds/registry';
import AwbEditor from './AwbEditor';
import type { AwbBundle, AwbIdentity, AwbTarget } from './model';
import { aeIdentity, awbFixture } from './model.fixtures';
import type { AwbPdfModel } from './pdf/types';

export { state };

const state = vi.hoisted(() => ({
  bundle: {} as AwbBundle,
  canUpdate: true,
  canOperate: true,
  read: vi.fn(),
  readMaster: vi.fn(),
  save: vi.fn(),
  previewImpact: vi.fn(),
  saveMaster: vi.fn(),
  preview: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => state.canOperate,
    canOrder: (_type: number, action: string) =>
      action === 'update' ? state.canUpdate : true,
  }),
}));
vi.mock('../use-order-lock-state', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../use-order-lock-state')>()),
  useOrderLockState: () => ({
    state: { isLocked: false },
    loading: false,
    error: null,
    refresh: async () => null,
  }),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: state.read,
  orderServiceUpdateOrder: state.save,
}));
vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: state.readMaster,
  airExportServicePreviewAirMasterWaybillUpdate: state.previewImpact,
  airExportServiceUpdateAirMasterWaybill: state.saveMaster,
}));
vi.mock('./pdf/AwbPdfPreview', () => ({
  default: (props: { model: AwbPdfModel; onClose: () => void }) => {
    state.preview(props.model);
    return (
      <div role="dialog" aria-label="航空PDF预览">
        <span>{props.model.content.remarks}</span>
        <button type="button" onClick={props.onClose}>
          关闭预览
        </button>
      </div>
    );
  },
}));

beforeEach(() => {
  vi.clearAllMocks();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  state.bundle = awbFixture();
  state.canUpdate = true;
  state.canOperate = true;
  state.read.mockImplementation(async () => ({
    success: true,
    data: structuredClone(state.bundle.order),
  }));
  state.readMaster.mockImplementation(async () => ({
    success: true,
    data: structuredClone(state.bundle.master),
    blockedReasons: [],
  }));
});

export function renderAwb(
  target: AwbTarget = 'HAWB',
  identityPatch?: Partial<AwbIdentity>,
) {
  const client = createTestQueryClient();
  const identity = { ...aeIdentity, target, ...identityPatch };
  const definition = getOrderKindDefinitionByBusinessType(
    identity.businessType,
  );
  if (!definition) throw new Error('测试品类缺失');
  const path = `/orders/${definition.kind}/${target === 'MAWB' ? 'masters/master-1/awb' : 'order-1/awb/HAWB'}`;
  const router = createMemoryRouter(
    [
      {
        path: '*',
        element: (
          <AwbEditor
            initial={structuredClone(state.bundle)}
            identity={identity}
            definition={definition}
          />
        ),
      },
    ],
    { initialEntries: [path] },
  );
  const view = render(
    <ConfigProvider>
      <App>
        <QueryClientProvider client={client}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </App>
    </ConfigProvider>,
  );
  return { ...view, client, router };
}
export async function editRemarks(text: string) {
  if (!screen.queryByLabelText('单证备注'))
    fireEvent.click(screen.getByText('签发与备注'));
  const input = await screen.findByLabelText('单证备注');
  fireEvent.change(input, { target: { value: text } });
  return input;
}
export function saveAwb() {
  fireEvent.click(screen.getByRole('button', { name: '保存正文' }));
}
export function outputAwb() {
  fireEvent.click(screen.getByRole('button', { name: '预览 / 打印 / 下载' }));
}
export function makeImportBundle() {
  state.bundle.master.businessType = 'AI';
  if (state.bundle.order?.airExport?.master) {
    state.bundle.order.businessType = OrderBusinessType.BUSINESS_TYPE_AI;
    state.bundle.order.airExport.master.businessType = 'AI';
  }
}
