import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { AirDocumentSection } from './components/air/AirDocumentSection';
import { revealSeaFormErrors } from './sea-form-error-reveal';
import type { TemplateProps } from './types';

const search = async () => [];
const props: TemplateProps = {
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

function setup() {
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
            airDocumentMode: 'HOUSE',
            airHouseNo: 'H-1',
            airHouseContent: { shipperText: '本票发货人' },
            airExport: {
              documentMode: 'HOUSE',
              master: {
                id: 'm-1',
                version: '3',
                content: { waybillContent: { shipperText: '共享主单发货人' } },
              },
            },
          }}
        >
          <AirDocumentSection props={props} />
        </ProForm>
      </ConfigProvider>
    );
  }
  renderWithApp(<Editor />);
  return () => form;
}

describe('航空订单就近单证', () => {
  it('主分单切换保留本票输入，只读主单正文不进入整票提交', async () => {
    const read = setup();
    fireEvent.change(
      screen.getByRole('textbox', { name: '发货人名称与地址' }),
      { target: { value: '本票未保存修改' } },
    );
    fireEvent.click(screen.getByRole('tab', { name: /^主运单 \(MAWB\)/ }));
    const master = screen.getByRole('tabpanel', { name: /^主运单 \(MAWB\)/ });
    const readonlyShipper = within(master).getByRole('textbox', {
      name: '发货人名称与地址',
    });
    expect(readonlyShipper).toHaveValue('共享主单发货人');
    expect(readonlyShipper).toHaveAttribute('readonly');
    fireEvent.click(screen.getByRole('tab', { name: /^分运单 \(HAWB\)/ }));
    expect(
      screen.getByRole('textbox', { name: '发货人名称与地址' }),
    ).toHaveValue('本票未保存修改');
    let values!: Record<string, unknown>;
    await act(async () => {
      values = await read().validateFields();
    });
    expect(values.airHouseContent).toMatchObject({
      shipperText: '本票未保存修改',
    });
    expect(values.content).toBeUndefined();
    expect(read().getFieldValue(['airExport', 'master', 'version'])).toBe('3');
  });

  it('隐藏分单的低频错误会同时切页签并展开，运输字段错误不切单证', async () => {
    setup();
    fireEvent.click(screen.getByRole('tab', { name: /^主运单 \(MAWB\)/ }));
    act(() => revealSeaFormErrors([{ name: ['airMaster', 'flightNo'] }]));
    expect(screen.getByRole('tab', { name: /^主运单 \(MAWB\)/ })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    act(() => revealSeaFormErrors([{ name: ['airHouseContent', 'currency'] }]));
    await waitFor(() =>
      expect(screen.getByRole('tab', { name: /^分运单 \(HAWB\)/ })).toHaveAttribute(
        'aria-selected',
        'true',
      ),
    );
    expect(
      await screen.findByRole('textbox', { name: '运单币种' }),
    ).toBeVisible();
  });
});
