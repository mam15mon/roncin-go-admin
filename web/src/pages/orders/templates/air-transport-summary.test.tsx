import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, screen } from '@testing-library/react';
import { Form, type FormInstance } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { getAirTemplateSections } from './air-template';
import { AirMasterTransportSection } from './components/air/AirMasterFields';
import type { TemplateProps } from './types';

vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: vi.fn(),
  airExportServiceListAirMasterWaybills: vi
    .fn()
    .mockResolvedValue({ success: true }),
}));
const search = async () => [];
const props: TemplateProps = {
  businessType: OrderBusinessType.BUSINESS_TYPE_AI,
  orderKind: 'air-import',
  cargoCategoryOptions: [],
  locationOptions: [],
  currencyOptions: [],
  containerSpecOptions: [],
  searchLocations: search,
  searchCustomers: search,
  searchShippingLines: search,
  searchBookingAgents: search,
  searchForeignAgents: search,
  searchShippingAgents: search,
  setCustomerCode: () => {},
  personnelOptions: [],
  isDetail: true,
};

describe('航空运输摘要', () => {
  it('已保存运输快照可刷新展示，保持整票原主单版本基线与私有输入', async () => {
    let form!: FormInstance;
    const master: API.AirMasterWaybill = {
      id: 'master',
      version: '4',
      airlineName: '测试航司',
      originAirportName: '上海浦东',
      destinationAirportName: '新加坡',
      content: {
        masterNo: '78112345675',
        flightNo: 'MU100',
        declaredPackages: 0,
        etd: '2026-11-16T00:00:00Z',
      },
    };
    function Editor({ snapshot }: { snapshot: API.AirMasterWaybill }) {
      const [current] = Form.useForm();
      form = current;
      return (
        <ProForm
          form={form}
          submitter={false}
          initialValues={{
            airDocumentMode: 'HOUSE',
            airMasterExpectedVersion: '4',
            bookingNo: '未保存订舱号',
            airExport: { documentMode: 'HOUSE', master },
          }}
        >
          <AirMasterTransportSection
            props={{ ...props, airTransportSnapshot: snapshot }}
          />
        </ProForm>
      );
    }
    const view = renderWithApp(<Editor snapshot={master} />);
    expect(screen.getByText('MU100')).toBeVisible();
    expect(screen.getByText('上海浦东')).toBeVisible();
    expect(screen.getByText('新加坡')).toBeVisible();
    view.rerender(
      <Editor
        snapshot={{
          ...master,
          version: '5',
          content: { ...master.content, flightNo: 'MU200' },
        }}
      />,
    );
    expect(screen.getByText('MU200')).toBeVisible();
    expect(form.getFieldValue('airMasterExpectedVersion')).toBe('4');
    expect(form.getFieldValue('bookingNo')).toBe('未保存订舱号');
    expect(form.getFieldValue(['airExport', 'master', 'version'])).toBe('4');
    expect(screen.getByText('承运与主单')).toBeVisible();
    await act(async () => {});
  });

  it('进口使用承运文案，单证与货物卡保持海空一致命名', () => {
    expect(
      getAirTemplateSections(props).map((section) => section.title),
    ).toEqual([
      '业务与客户',
      '承运与运输',
      '货物与商业',
      '单证信息',
      '内部人员',
    ]);
  });
});
