import type { ProFormInstance } from '@ant-design/pro-components';
import { ProFormText } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import { getFormDraft, getFormDraftKey } from '@/components/layout/formDraft';
import { _clearAllTabCloseGuards } from '@/components/layout/tabCloseGuard';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import { OrderContainerRequestFields } from './order-plan-fields';

const identity = {
  tabKey: '/orders/sea-export',
  draftPathname: '/orders/sea-export/new',
  draftScope: 'user:company',
};
const draftKey = getFormDraftKey(
  identity.tabKey,
  identity.draftPathname,
  identity.draftScope,
);

function orderForm(containerRequests?: API.OrderContainerRequestInput[]) {
  const formRef: React.MutableRefObject<ProFormInstance | undefined> = {
    current: undefined,
  };
  const ui = (
    <OrderFormTemplate
      {...identity}
      formRef={formRef}
      initialValues={{
        version: '7',
        customerReferenceNo: '服务端编号',
        containerRequests,
      }}
      sections={[
        {
          key: 'business',
          title: '业务资料',
          content: (
            <>
              <ProFormText name="customerReferenceNo" label="客户业务编号" />
              <OrderContainerRequestFields
                options={[{ label: '20GP', value: 'spec-20gp' }]}
              />
            </>
          ),
        },
      ]}
    />
  );
  return { ui, formRef };
}

afterEach(() => {
  // 先卸载快照观察器，再清存储，避免上个用例的脏快照回写到下一用例。
  cleanup();
  sessionStorage.clear();
  _clearAllTabCloseGuards();
});

function editReference() {
  fireEvent.change(screen.getByLabelText('客户业务编号'), {
    target: { value: '未保存编号' },
  });
}

describe('新建订单计划箱量的草稿恢复', () => {
  it.each([undefined, []])(
    '无配箱计划 %j 时，只改业务编号并重开仍可编辑',
    async (containerRequests) => {
      const { ui, formRef } = orderForm(containerRequests);
      const view = renderWithApp(ui);
      editReference();
      await waitFor(() =>
        expect(getFormDraft(draftKey)).toMatchObject({
          customerReferenceNo: '未保存编号',
          containerRequests: containerRequests ?? null,
        }),
      );
      view.unmount();
      renderWithApp(ui);

      expect(screen.getByLabelText('客户业务编号')).toHaveValue('未保存编号');
      expect(formRef.current?.getFieldValue('version')).toBe('7');
      fireEvent.click(screen.getByRole('button', { name: /新增计划箱型箱量/ }));
      await waitFor(() =>
        expect(getFormDraft(draftKey)).toMatchObject({
          containerRequests: [{ containerSpecId: 'spec-20gp', quantity: 1 }],
        }),
      );
      fireEvent.change(screen.getByRole('spinbutton'), {
        target: { value: '3' },
      });
      await waitFor(() =>
        expect(getFormDraft(draftKey)).toMatchObject({
          containerRequests: [{ containerSpecId: 'spec-20gp', quantity: 3 }],
        }),
      );
      fireEvent.click(screen.getByRole('button', { name: /delete/ }));
      await waitFor(() =>
        expect(getFormDraft(draftKey)).toMatchObject({ containerRequests: [] }),
      );
    },
  );

  it('已有箱量时只改业务编号，重开保留计划记录和数量', async () => {
    const plan = [{ id: 'plan-1', containerSpecId: 'spec-20gp', quantity: 2 }];
    const { ui, formRef } = orderForm(plan);
    const view = renderWithApp(ui);
    editReference();
    await waitFor(() =>
      expect(getFormDraft(draftKey)).toMatchObject({ containerRequests: plan }),
    );
    view.unmount();
    renderWithApp(ui);

    expect(screen.getByLabelText('客户业务编号')).toHaveValue('未保存编号');
    expect(screen.getByRole('spinbutton')).toHaveValue('2');
    expect(formRef.current?.getFieldValue('containerRequests')).toEqual(plan);
  });

  it('显式清空箱量后重开保持空计划，不恢复服务端旧值', async () => {
    const { ui, formRef } = orderForm([
      { id: 'plan-1', containerSpecId: 'spec-20gp', quantity: 2 },
    ]);
    const view = renderWithApp(ui);
    editReference();
    await act(async () => {
      formRef.current?.setFieldValue('containerRequests', undefined);
    });
    await waitFor(() =>
      expect(getFormDraft(draftKey)).toMatchObject({ containerRequests: null }),
    );
    view.unmount();
    renderWithApp(ui);

    expect(screen.getByLabelText('客户业务编号')).toHaveValue('未保存编号');
    expect(screen.getByRole('spinbutton')).toBeDisabled();
    expect(formRef.current?.getFieldValue('containerRequests')).toBeNull();
  });
});
