import { ProFormRadio } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import { afterEach, expect, it, vi } from 'vitest';
import {
  clearFormDraft,
  getFormDraft,
  getFormDraftKey,
} from '@/components/layout/formDraft';
import {
  _clearAllTabCloseGuards,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import { OrderFormTemplate } from '@/components/ui';
import type { OrderFormTemplateActions } from '@/components/ui/order-template/types';
import { OrderContainerRequestFields } from './order-plan-fields';
import { SeaContainerPlanFields } from './templates/components/sea/SeaTransportSection';

afterEach(() => {
  _clearAllTabCloseGuards();
  sessionStorage.clear();
});

it('散杂货主动清空既有箱量计划也通过订单表单事件保护', async () => {
  render(
    <App>
      <OrderFormTemplate<Record<string, unknown>>
        tabKey="/orders/sea-export"
        initialValues={{
          shipmentType: 3,
          containerRequests: [{ containerSpecId: '40gp', quantity: 2 }],
        }}
        sections={[
          {
            key: 'transport',
            title: '运输计划',
            content: (
              <>
                <ProFormRadio.Group
                  name="shipmentType"
                  label="托运类型"
                  options={[{ value: 3, label: '散杂货' }]}
                />
                <SeaContainerPlanFields options={[]} />
              </>
            ),
          },
        ]}
      />
    </App>,
  );
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.click(await screen.findByRole('button', { name: '清空箱量计划' }));
  await waitFor(() => expect(getAnyDirtyTabKey()).toBe('/orders/sea-export'));
  expect(
    screen.queryByRole('button', { name: '清空箱量计划' }),
  ).not.toBeInTheDocument();
});

it('计划箱型箱量真实用户编辑进入订单模板草稿与未保存生命周期，初始化和重置不误标脏', async () => {
  const actionsRef: React.MutableRefObject<
    OrderFormTemplateActions<Record<string, unknown>> | undefined
  > = { current: undefined };
  const draftKey = getFormDraftKey(
    '/orders/sea-export',
    '/orders/sea-export/new',
    'user:beijing',
  );
  clearFormDraft(draftKey);
  const finish = vi.fn().mockResolvedValue(false);
  render(
    <App>
      <OrderFormTemplate<Record<string, unknown>>
        tabKey="/orders/sea-export"
        draftPathname="/orders/sea-export/new"
        draftScope="user:beijing"
        actionsRef={actionsRef}
        onFinish={finish}
        initialValues={{
          containerRequests: [{ containerSpecId: '40gp', quantity: 2 }],
        }}
        sections={[
          {
            key: 'transport',
            title: '运输计划',
            content: (
              <OrderContainerRequestFields
                options={[{ value: '40gp', label: '40GP' }]}
              />
            ),
          },
        ]}
      />
    </App>,
  );
  const quantity = await screen.findByRole('spinbutton');
  expect(quantity).toHaveValue('2');
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.change(quantity, { target: { value: '3' } });
  fireEvent.blur(quantity);
  await waitFor(() => expect(getAnyDirtyTabKey()).toBe('/orders/sea-export'));
  expect(getFormDraft(draftKey)).toMatchObject({
    containerRequests: [{ containerSpecId: '40gp', quantity: 3 }],
  });
  fireEvent.click(screen.getByRole('button', { name: /提\s*交/ }));
  await waitFor(() => expect(finish).toHaveBeenCalledOnce());
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
  act(() => actionsRef.current?.resetTo({ containerRequests: [] }));
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.click(screen.getByRole('button', { name: /新增计划箱型箱量/ }));
  await waitFor(() => expect(getAnyDirtyTabKey()).toBe('/orders/sea-export'));
  expect(getFormDraft(draftKey)).toMatchObject({
    containerRequests: [{ containerSpecId: '40gp', quantity: 1 }],
  });
});
