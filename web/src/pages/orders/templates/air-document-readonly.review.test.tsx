import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { AirDocumentSection } from './components/air/AirDocumentSection';
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
  readonly: true,
};

describe('航空只读主单注册与通知人', () => {
  it('父表单禁用仍可切第二通知人，只读框不注册到本票提交及版本基线', async () => {
    let form!: FormInstance;
    function Editor() {
      const [current] = Form.useForm();
      form = current;
      return (
        <ConfigProvider theme={{ token: { motion: false } }}>
          <ProForm
            form={form}
            disabled
            submitter={false}
            initialValues={{
              airMasterExpectedVersion: '4',
              airExport: {
                documentMode: 'DIRECT',
                master: {
                  id: 'master',
                  version: '4',
                  content: {
                    waybillContent: {
                      shipperText: '只读发货人',
                      notifyPartyText: '只读通知人',
                      secondNotifyPartyText: '只读第二通知人',
                    },
                  },
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
    const shipper = screen.getByRole('textbox', { name: '发货人名称与地址' });
    expect(shipper).toHaveValue('只读发货人');
    expect(shipper).toHaveAttribute('readonly');
    expect(shipper).toHaveAttribute('rows', '4');
    fireEvent.click(screen.getByText('第二通知人 (Second Notify Party) · 已填写'));
    expect(screen.getByRole('textbox', { name: '第二通知人' })).toHaveValue(
      '只读第二通知人',
    );
    expect(
      screen.queryByRole('button', { name: /带入/ }),
    ).not.toBeInTheDocument();
    let values!: Record<string, unknown>;
    await act(async () => {
      values = await form.validateFields();
    });
    expect(values.airMaster).toBeUndefined();
    expect(values.airHouseContent).toBeUndefined();
    expect(form.getFieldValue('airMasterExpectedVersion')).toBe('4');
  });
});
