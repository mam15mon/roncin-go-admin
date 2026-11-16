import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { App, ConfigProvider, Form, type FormInstance } from 'antd';
import { vi } from 'vitest';
import { AirWaybillFields } from './AirWaybillFields';

export function renderAirOrderContent(
  content: API.AirWaybillContent = {},
  extras: Record<string, unknown> = {},
) {
  let form!: FormInstance;
  const imported = vi.fn(
    (prefix: (string | number)[], field: string, value: string) => {
      form.setFieldValue([...prefix, field], value);
      form.setFields([{ name: [...prefix, field], touched: true }]);
    },
  );
  function Editor({ target = 'order-1' }: { target?: string }) {
    const [current] = Form.useForm();
    form = current;
    return (
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <ProForm
            form={form}
            submitter={false}
            initialValues={{
              id: target,
              customerId: 'customer-1',
              foreignAgentId: 'agent-1',
              goodsDescription: '订单原始品名',
              content,
              ...extras,
            }}
          >
            <AirWaybillFields
              prefix={['content']}
              variant="order"
              onImport={imported}
            />
          </ProForm>
        </App>
      </ConfigProvider>
    );
  }
  const view = renderWithApp(<Editor />, {
    user: { id: 'user-1', currentOrganization: { id: 'company-1' } },
  });
  return { ...view, imported, readForm: () => form, Editor };
}
