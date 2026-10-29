import type { ProFormInstance } from '@ant-design/pro-components';
import { ProFormText } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import React from 'react';
import { afterEach, expect, it } from 'vitest';
import { getFormDraft, getFormDraftKey } from '@/components/layout/formDraft';
import { _clearAllTabCloseGuards } from '@/components/layout/tabCloseGuard';
import { OrderFormTemplate } from './OrderFormTemplate';
import type { OrderFormTemplateActions } from './types';

const identity = {
  tabKey: 'orders',
  draftPathname: '/orders/sea-export/new',
  draftScope: 'user:org',
};
const draftKey = getFormDraftKey(
  identity.tabKey,
  identity.draftPathname,
  identity.draftScope,
);
afterEach(() => {
  sessionStorage.clear();
  _clearAllTabCloseGuards();
});

it('人员程序回填及清空进入草稿，重挂恢复与可见表单一致，resetTo 不重建草稿', async () => {
  const formRef: React.MutableRefObject<ProFormInstance | undefined> = {
    current: undefined,
  };
  const actionsRef: React.MutableRefObject<
    OrderFormTemplateActions<Record<string, unknown>> | undefined
  > = { current: undefined };
  const ui = (
    <OrderFormTemplate
      {...identity}
      formRef={formRef}
      actionsRef={actionsRef}
      initialValues={{
        customerId: 'old-customer',
        operatorUserId: 'old-operator',
      }}
      sections={[
        {
          key: 'entry',
          title: '录入',
          content: (
            <>
              <ProFormText name="customerId" label="客户" />
              <ProFormText name="operatorUserId" label="操作员" />
            </>
          ),
        },
      ]}
    />
  );
  const view = render(ui);
  expect(getFormDraft(draftKey)).toBeNull();
  fireEvent.change(screen.getByLabelText('客户'), {
    target: { value: 'new-customer' },
  });
  await act(async () => {
    formRef.current?.setFieldsValue({ operatorUserId: 'new-operator' });
  });
  await waitFor(() =>
    expect(getFormDraft(draftKey)).toMatchObject({
      customerId: 'new-customer',
      operatorUserId: 'new-operator',
    }),
  );
  await act(async () => {
    formRef.current?.setFieldValue('operatorUserId', undefined);
  });
  await waitFor(() =>
    expect(getFormDraft(draftKey)).toMatchObject({ operatorUserId: null }),
  );
  view.unmount();
  render(ui);
  expect(screen.getByLabelText('客户')).toHaveValue('new-customer');
  expect(screen.getByLabelText('操作员')).toHaveValue('');
  await act(async () => {
    actionsRef.current?.resetTo({
      customerId: 'saved-customer',
      operatorUserId: 'saved-operator',
    });
  });
  expect(getFormDraft(draftKey)).toBeNull();
  expect(screen.getByLabelText('操作员')).toHaveValue('saved-operator');
});
