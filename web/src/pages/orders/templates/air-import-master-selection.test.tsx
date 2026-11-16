import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  airExportServiceGetAirMasterWaybill,
  airExportServiceListAirMasterWaybills,
} from '@/services/roncin/airExportService';
import { AirMasterTransportSection } from './components/air/AirMasterFields';
import type { TemplateProps } from './types';

vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: vi.fn(),
  airExportServiceListAirMasterWaybills: vi.fn(),
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListAirlines: vi.fn().mockResolvedValue({ data: [] }),
}));
const search = async () => [];
const props: TemplateProps = {
  businessType: OrderBusinessType.BUSINESS_TYPE_AI,
  orderKind: 'air-import',
  cargoCategoryOptions: [],
  locationOptions: [],
  searchLocations: search,
  currencyOptions: [],
  containerSpecOptions: [],
  searchCustomers: search,
  searchShippingLines: search,
  searchBookingAgents: search,
  searchForeignAgents: search,
  searchShippingAgents: search,
  setCustomerCode: () => {},
  personnelOptions: [],
};
const master: API.AirMasterWaybill = {
  id: 'master-ai',
  organizationId: 'company',
  businessType: 'AI',
  version: '4',
  documentMode: 'HOUSE',
  canAttach: true,
  content: { masterNo: '78112345675' },
};
function setup(query = 'masterId=master-ai') {
  let form!: FormInstance;
  function Editor() {
    const [current] = Form.useForm();
    form = current;
    return (
      <ConfigProvider theme={{ token: { motion: false } }}>
        <ProForm
          form={form}
          submitter={false}
          initialValues={{
            airMasterMode: 'NEW',
            airDocumentMode: 'HOUSE',
          }}
        >
          <AirMasterTransportSection props={props} />
        </ProForm>
      </ConfigProvider>
    );
  }
  const view = renderWithApp(<Editor />, {
    user: { id: 'user', currentOrganization: { id: 'company' } },
    routerProps: { initialEntries: [`/orders/air-import/new?${query}`] },
  });
  return { ...view, readForm: () => form };
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(airExportServiceGetAirMasterWaybill).mockResolvedValue({
    data: master,
  });
  vi.mocked(airExportServiceListAirMasterWaybills).mockResolvedValue({
    data: [master],
  });
});
describe('AI建票主单预选与候选', () => {
  it('AI候选携真实类型，预选读取独立缓存且后台重读不覆盖已采纳基线', async () => {
    const view = setup();
    await waitFor(() =>
      expect(view.readForm().getFieldValue('airMasterId')).toBe('master-ai'),
    );
    expect(view.readForm().getFieldValue('airMasterExpectedVersion')).toBe('4');
    // 同号核对按 AI 业务类型查询，只命中进口主单。
    expect(airExportServiceListAirMasterWaybills).toHaveBeenCalledWith(
      expect.objectContaining({ businessType: 'AI', keyword: '78112345675' }),
      expect.anything(),
    );
    vi.mocked(airExportServiceGetAirMasterWaybill).mockResolvedValue({
      data: { ...master, version: '5' },
    });
    await act(async () =>
      view.queryClient.refetchQueries({
        queryKey: ['air-orders', 'preselected-master'],
      }),
    );
    expect(view.readForm().getFieldValue('airMasterExpectedVersion')).toBe('4');
  });
  it.each([
    { businessType: 'AE' },
    { organizationId: 'other' },
    { id: 'other' },
    { businessType: undefined },
  ])('预选主单缺失或错误身份不可写入表单 %j', async (patch) => {
    vi.mocked(airExportServiceGetAirMasterWaybill).mockResolvedValue({
      data: { ...master, ...patch },
    });
    const view = setup();
    expect(
      await screen.findByText('所选主单加载失败，请重新输入主运单号。'),
    ).toBeVisible();
    expect(view.readForm().getFieldValue('airMasterId')).toBeUndefined();
    expect(
      view.readForm().getFieldValue('airMasterExpectedVersion'),
    ).toBeUndefined();
  });
});
