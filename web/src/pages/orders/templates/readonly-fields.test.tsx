import type { ProFormInstance } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, Button, ConfigProvider } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import { SeaOrderReferenceField } from './components/sea/SeaBasicInfoSection';
import { buildSeaPersonnelSection } from './components/sea/SeaPersonnelSection';
import type { TemplateProps } from './types';

const baseProps: TemplateProps = {
  isDetail: true,
  cargoCategoryOptions: [],
  locationOptions: [],
  searchLocations: async () => [],
  currencyOptions: [],
  containerSpecOptions: [],
  searchCustomers: async () => [],
  searchShippingLines: async () => [],
  searchBookingAgents: async () => [],
  searchForeignAgents: async () => [],
  searchShippingAgents: async () => [],
  setCustomerCode: () => {},
  personnelOptions: [{ userId: 'operator-1', displayName: '操作员甲' }],
};

describe('订单详情真实控件的只读约束', () => {
  it.each([
    { name: '已审核锁定', readonly: true, editingDisabled: false },
    { name: '保存结果待确认', readonly: false, editingDisabled: true },
  ])(
    '$name 禁止修改原生输入与人员，解冻后保留值并可保存',
    async ({ readonly, editingDisabled }) => {
      const formRef: { current: ProFormInstance | undefined } = {
        current: undefined,
      };
      const onFinish = vi.fn().mockResolvedValue(true);
      const onView = vi.fn();
      const renderForm = (locked: boolean, frozen: boolean) => (
        <ConfigProvider theme={{ token: { motion: false } }}>
          <App>
            <OrderFormTemplate
              formRef={formRef}
              readonly={locked}
              editingDisabled={frozen}
              enableCloseGuard={false}
              showAnchorNav={false}
              submitText="保存订单"
              initialValues={{
                customerReferenceNo: '客户原编号',
                internalReferenceNo: '内部原编号',
                operatorUserId: 'operator-1',
              }}
              onFinish={onFinish}
              sections={[
                {
                  key: 'references',
                  title: '业务编号',
                  content: (
                    <>
                      <SeaOrderReferenceField
                        name="customerReferenceNo"
                        isDetail
                      />
                      <SeaOrderReferenceField
                        name="internalReferenceNo"
                        isDetail
                      />
                      <Button disabled={false} onClick={onView}>
                        查看历史
                      </Button>
                    </>
                  ),
                },
                buildSeaPersonnelSection({ ...baseProps, readonly: locked }),
              ]}
            />
          </App>
        </ConfigProvider>
      );
      const { rerender } = render(renderForm(readonly, editingDisabled));
      expect(screen.getByLabelText('客户业务编号')).toBeDisabled();
      expect(screen.getByLabelText('企业内部编号')).toBeDisabled();
      for (const select of screen.getAllByRole('combobox'))
        expect(select).toBeDisabled();
      expect(screen.getByLabelText('客户业务编号')).toHaveValue('客户原编号');
      fireEvent.click(screen.getByRole('button', { name: '查看历史' }));
      expect(onView).toHaveBeenCalledOnce();
      await act(async () => {
        await formRef.current?.submit();
      });
      expect(onFinish).not.toHaveBeenCalled();

      rerender(renderForm(false, false));
      expect(screen.getByLabelText('客户业务编号')).toBeEnabled();
      expect(screen.getByLabelText('客户业务编号')).toHaveValue('客户原编号');
      expect(screen.getByLabelText('操作人员')).toBeEnabled();
      expect(screen.getByLabelText('业务人员')).toBeDisabled();
      expect(screen.getByLabelText('客服人员')).toBeDisabled();
      expect(screen.getByLabelText('创建人员')).toBeDisabled();
      fireEvent.change(screen.getByLabelText('客户业务编号'), {
        target: { value: '解锁后编号' },
      });
      fireEvent.click(screen.getByRole('button', { name: '保存订单' }));
      await waitFor(() =>
        expect(onFinish).toHaveBeenCalledWith(
          expect.objectContaining({
            customerReferenceNo: '解锁后编号',
            internalReferenceNo: '内部原编号',
            operatorUserId: 'operator-1',
          }),
        ),
      );
    },
  );
});
