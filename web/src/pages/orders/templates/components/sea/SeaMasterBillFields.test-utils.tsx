import { ProForm, ProFormText } from '@ant-design/pro-components';
import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { SeaDocumentStructure } from '@/enums.generated';
import type { TemplateProps } from '../../types';
import { getSeaTransportFields } from './SeaTransportSection';

export const masterField = 'seaMasterBillMasterNo';
export const voyage: API.SeaTransportExecution = {
  id: 'te-1',
  version: '3',
  vesselName: 'EVER TEST',
  voyageNo: '001W',
  originLocationId: 'origin',
  originLocationName: '上海',
  dischargeLocationId: 'discharge',
  dischargeLocationName: '洛杉矶',
  transitLocationId: 'transit',
  transitLocationName: '釜山',
  etd: '2026-10-20T00:00:00+08:00',
  eta: '2026-10-25T00:00:00+08:00',
};
export function candidateResponse(
  overrides: Partial<API.SeaMasterBillCandidate> = {},
) {
  return {
    matched: true,
    candidate: {
      id: 'mbl-1',
      version: '5',
      masterNo: 'MBL123',
      shippingLineId: 'carrier-1',
      memberCount: 2,
      members: [
        {
          orderId: 'order-a',
          orderNo: 'SE-A',
          documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
        },
      ],
      batchNormalizedHouseNos: ['HBL-OLD'],
      transportExecutions: [voyage],
      ...overrides,
    },
  };
}
const search = async () => [];
const templateProps: TemplateProps = {
  locationOptions: [
    { value: 'destination', label: '本票最终目的地' },
    { value: 'other-origin', label: '宁波' },
  ],
  cargoCategoryOptions: [],
  currencyOptions: [],
  containerSpecOptions: [],
  personnelOptions: [],
  searchLocations: search,
  searchCustomers: search,
  searchShippingLines: search,
  searchBookingAgents: search,
  searchForeignAgents: search,
  searchShippingAgents: search,
  setCustomerCode: () => {},
};

export function renderVoyageForm(
  initialValues: Record<string, unknown> = {},
  { isDetail = false, disabled = false } = {},
) {
  let form!: FormInstance;
  function Harness() {
    const [instance] = Form.useForm();
    form = instance;
    const fields = getSeaTransportFields({ ...templateProps, isDetail });
    return (
      <ProForm
        form={instance}
        submitter={false}
        disabled={disabled}
        initialValues={{
          shippingLineId: 'carrier-1',
          [masterField]: 'MBL123',
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
          ...initialValues,
        }}
      >
        <ProFormText
          name="shippingLineId"
          label="船公司"
          rules={[{ required: true, message: '请选择船公司' }]}
        />
        {fields.master}
        {fields.vessel}
        {fields.ports}
        {fields.schedule}
      </ProForm>
    );
  }
  const result = renderWithClient(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Harness />
    </ConfigProvider>,
  );
  return {
    ...result,
    get form() {
      return form;
    },
  };
}
