import type { ProFormInstance } from '@ant-design/pro-components';
import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen } from '@testing-library/react';
import { App, ConfigProvider, Form, Input } from 'antd';
import { useRef } from 'react';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import type { OrderFormTemplateActions } from '@/components/ui/order-template/types';
import type { CreateOrderFormValues } from '../../../order-kinds/form-values';
import { SeaFinalDestinationField } from './SeaFinalDestinationField';

export const options = [
  { value: 'singapore', label: '新加坡' },
  { value: 'jakarta', label: '雅加达' },
  { value: 'bangkok', label: '曼谷' },
];
const search = async () => options;

export function renderDestination({
  initialValues = {},
  isDetail = false,
  draftPathname,
}: {
  initialValues?: Record<string, unknown>;
  isDetail?: boolean;
  draftPathname?: string;
} = {}) {
  const handles: {
    form?: ProFormInstance;
    actions?: OrderFormTemplateActions<CreateOrderFormValues>;
  } = {};
  function Harness() {
    const formRef = useRef<ProFormInstance | undefined>(undefined);
    const actionsRef = useRef<
      OrderFormTemplateActions<CreateOrderFormValues> | undefined
    >(undefined);
    Object.defineProperties(handles, {
      form: { get: () => formRef.current, configurable: true },
      actions: { get: () => actionsRef.current, configurable: true },
    });
    return (
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <OrderFormTemplate<CreateOrderFormValues>
            formRef={formRef}
            actionsRef={actionsRef}
            initialValues={initialValues}
            tabKey={draftPathname ? 'sea-destination' : undefined}
            draftPathname={draftPathname}
            draftScope="user-1:org-1"
            submitter={false}
            showAnchorNav={false}
            sections={[
              {
                key: 'transport',
                title: '共同航程与本票目的地',
                content: (
                  <>
                    <Form.Item name="dischargeLocationId" label="卸货港">
                      <Input />
                    </Form.Item>
                    <SeaFinalDestinationField
                      isDetail={isDetail}
                      locationOptions={options}
                      searchLocations={search}
                    />
                  </>
                ),
              },
            ]}
          />
        </App>
      </ConfigProvider>
    );
  }
  return { ...renderWithClient(<Harness />), handles };
}

export async function changeDischarge(value: string) {
  await act(async () => {
    fireEvent.change(screen.getByLabelText('卸货港'), { target: { value } });
  });
}

export async function editDestination() {
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: '更改最终目的港' }));
  });
}

export async function chooseDestination(label: string) {
  fireEvent.mouseDown(screen.getByLabelText('最终目的港'));
  const option = await screen.findByText(label, {
    selector: '.ant-select-item-option-content',
  });
  await act(async () => fireEvent.click(option));
}
