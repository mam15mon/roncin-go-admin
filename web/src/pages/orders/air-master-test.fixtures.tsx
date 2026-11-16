import { renderWithApp } from '@root/tests/renderWithApp';
import { cleanup } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { afterEach, beforeEach, vi } from 'vitest';
import { _clearAllTabCloseGuards } from '@/components/layout/tabCloseGuard';
import AirMasterPage from './air-master';

const masterServices = vi.hoisted(() => ({
  get: vi.fn(),
  canOrder: vi.fn(),
  preview: vi.fn(),
  update: vi.fn(),
  link: vi.fn(),
  candidates: vi.fn(),
  push: vi.fn(),
}));

export { masterServices };

vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: masterServices.get,
  airExportServicePreviewAirMasterWaybillUpdate: masterServices.preview,
  airExportServiceUpdateAirMasterWaybill: masterServices.update,
  airExportServiceLinkAirMasterWaybill: masterServices.link,
  airExportServiceListAirLinkCandidates: masterServices.candidates,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOrder: masterServices.canOrder }),
}));
vi.mock('@/router/history', async (original) => ({
  ...(await original<typeof import('@/router/history')>()),
  history: {
    location: { pathname: '/orders/air-export/masters/master' },
    push: masterServices.push,
  },
}));
vi.mock('react-router', async (original) => ({
  ...(await original<typeof import('react-router')>()),
  useParams: () => ({ masterId: 'master' }),
}));
vi.mock('@/components/ui', async (original) => ({
  ...(await original<typeof import('@/components/ui')>()),
  ResponsiveProTable: () => null,
}));
vi.mock('./order-kinds/air-export/AirWeightPanel', () => ({
  default: () => null,
}));
// 字段组件另有真实控件测试；这里保留官方表单绑定，验证页面的快照和请求生命周期。
vi.mock('./templates/components/air/AirMasterFields', async () => {
  const { ProFormText } = await import('@ant-design/pro-components');
  return {
    AirMasterFields: () => (
      <ProFormText name={['input', 'flightNo']} label="测试运输航班" />
    ),
  };
});
vi.mock('./templates/components/air/AirWaybillFields', async () => {
  const { ProFormTextArea } = await import('@ant-design/pro-components');
  return {
    AirWaybillFields: ({ prefix }: { prefix: (string | number)[] }) => (
      <ProFormTextArea name={[...prefix, 'remarks']} label="测试主单正文" />
    ),
  };
});

export function masterResponse(
  version = '1',
  extra: Partial<API.AirMasterWaybill> = {},
): API.GetAirMasterWaybillResponse {
  return {
    data: {
      id: 'master',
      organizationId: 'org',
      businessType: 'AE',
      version,
      documentMode: 'HOUSE',
      canAttach: true,
      content: {
        masterNo: '78112345675',
        flightNo: `运输航班-${version}`,
        waybillContent: { remarks: `主单正文-${version}` },
      },
      ...extra,
    },
    members: [],
    history: [],
  };
}

export function setupMasterTests() {
  beforeEach(() => {
    vi.clearAllMocks();
    masterServices.canOrder.mockImplementation(() => true);
    vi.spyOn(document, 'hasFocus').mockReturnValue(false);
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    masterServices.get.mockResolvedValue(masterResponse());
    masterServices.preview.mockResolvedValue({
      inputFingerprint: 'preview-fingerprint',
      canExecute: true,
      members: [],
    });
    masterServices.update.mockResolvedValue(undefined);
    masterServices.link.mockResolvedValue(undefined);
    masterServices.candidates.mockResolvedValue({
      data: [
        {
          orderId: 'order',
          orderNo: 'AE001',
          orderVersion: '3',
          houseNo: 'H001',
        },
      ],
      total: 1,
    });
  });
  afterEach(() => {
    cleanup();
    _clearAllTabCloseGuards();
    vi.useRealTimers();
    vi.restoreAllMocks();
  });
}

export function renderMaster(kind = 'air-export') {
  return renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <AirMasterPage />
      </App>
    </ConfigProvider>,
    {
      user: { id: 'user', currentOrganization: { id: 'org' } },
      routerProps: { initialEntries: [`/orders/${kind}/masters/master`] },
    },
  );
}
