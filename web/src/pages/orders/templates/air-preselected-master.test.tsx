import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, screen, waitFor } from '@testing-library/react';
import { Form, type FormInstance } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { airExportServiceGetAirMasterWaybill } from '@/services/roncin/airExportService';
import { AirMasterTransportSection } from './components/air/AirMasterFields';
import type { TemplateProps } from './types';

vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: vi.fn(),
  airExportServiceListAirMasterWaybills: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListAirlines: vi.fn().mockResolvedValue({ data: [] }),
}));

const search = async () => [];
const props: TemplateProps = {
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

function setup(query: string, cachedMaster?: API.GetAirMasterWaybillResponse) {
  let current!: FormInstance;
  function Editor({ ready }: { ready: boolean }) {
    const [form] = Form.useForm();
    current = form;
    return (
      <ProForm
        form={form}
        submitter={false}
        initialValues={{ airMasterMode: 'NEW', airDocumentMode: 'HOUSE' }}
      >
        {ready && <AirMasterTransportSection props={props} />}
      </ProForm>
    );
  }
  const view = renderWithApp(<Editor ready={!cachedMaster} />, {
    user: { id: 'operator', currentOrganization: { id: 'company' } },
    routerProps: { initialEntries: [`/orders/air-export/new?${query}`] },
  });
  if (cachedMaster) {
    view.queryClient.setQueryData(
      ['air-export', 'master', 'master', { organizationId: 'company' }],
      cachedMaster,
    );
    view.rerender(<Editor ready />);
  }
  return { readForm: () => current, queryClient: view.queryClient };
}

describe('航空主单预选与显式模式', () => {
  beforeEach(() => {
    vi.mocked(airExportServiceGetAirMasterWaybill).mockResolvedValue({
      data: {
        id: 'master',
        version: '3',
        documentMode: 'HOUSE',
        canAttach: true,
        content: { masterNo: '78112345675' },
      },
    });
  });

  it('URL明确DIRECT时保留用户选择并阻止关联HOUSE主单', async () => {
    const { readForm } = setup('masterId=master&documentMode=DIRECT');
    await waitFor(() =>
      expect(readForm().getFieldValue('airMasterId')).toBe('master'),
    );
    expect(readForm().getFieldValue('airDocumentMode')).toBe('DIRECT');
    expect(
      screen.getByRole('radio', { name: '直单（DIRECT，无 HAWB）' }),
    ).toBeChecked();
    await act(async () => {
      await expect(
        readForm().validateFields(['airMasterId']),
      ).rejects.toMatchObject({
        errorFields: [
          { errors: ['所选主单不允许按当前单证模式建票，请重新选择'] },
        ],
      });
    });
  });

  it('未指定模式时采用主单已固定的DIRECT模式', async () => {
    vi.mocked(airExportServiceGetAirMasterWaybill).mockResolvedValue({
      data: {
        id: 'master',
        version: '3',
        documentMode: 'DIRECT',
        canAttach: true,
      },
    });
    const { readForm } = setup('masterId=master');
    await waitFor(() =>
      expect(readForm().getFieldValue('airDocumentMode')).toBe('DIRECT'),
    );
    await act(async () => {
      await expect(readForm().validateFields(['airMasterId'])).resolves.toEqual(
        {
          airMasterId: 'master',
        },
      );
    });
  });

  it('从主单页进入建票时不共用不同形态的详情缓存', async () => {
    const cachedMaster: API.GetAirMasterWaybillResponse = {
      data: {
        id: 'master',
        version: '2',
        documentMode: 'HOUSE',
        canAttach: true,
      },
      members: [{ orderId: 'original-order' }],
    };
    const { readForm, queryClient } = setup(
      'masterId=master&documentMode=HOUSE',
      cachedMaster,
    );
    await waitFor(() =>
      expect(readForm().getFieldValue('airMasterExpectedVersion')).toBe('3'),
    );
    expect(readForm().getFieldValue('airMasterId')).toBe('master');
    expect(
      queryClient.getQueryData([
        'air-export',
        'master',
        'master',
        { organizationId: 'company' },
      ]),
    ).toEqual(cachedMaster);
  });
});
