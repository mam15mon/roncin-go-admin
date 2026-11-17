import type { ProFormInstance } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { afterEach, describe, expect, it } from 'vitest';
import {
  _clearAllTabCloseGuards,
  isTabDirty,
} from '@/components/layout/tabCloseGuard';
import { OrderFormTemplate } from '@/components/ui';
import type { OrderFormTemplateActions } from '@/components/ui/order-template/types';
import type { CreateOrderFormValues } from '../order-kinds/form-values';
import { airDocumentImportPatch } from './air-document-import';
import { AirWaybillFields } from './components/air/AirWaybillFields';

afterEach(() => {
  _clearAllTabCloseGuards();
});

describe('航空正文带入与真实订单未保存保护', () => {
  it('显式带入经模板动作同步标脏，深合并只改目标字段，普通切换不标脏', async () => {
    const tabKey = '/orders/air-export/new';
    const formRef: { current: ProFormInstance | undefined } = {
      current: undefined,
    };
    const actions: {
      current: OrderFormTemplateActions<CreateOrderFormValues> | undefined;
    } = { current: undefined };
    renderWithApp(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <OrderFormTemplate<CreateOrderFormValues>
          tabKey={tabKey}
          formRef={formRef}
          actionsRef={actions}
          submitter={false}
          showAnchorNav={false}
          initialValues={{
            goodsDescription: '订单品名',
            airHouseContent: {
              consigneeText: '原收货人',
              secondNotifyPartyText: '原第二通知人',
              remarks: '低频备注',
            },
          }}
          sections={[
            {
              key: 'airDocuments',
              title: '单证信息',
              content: (
                <AirWaybillFields
                  prefix={['airHouseContent']}
                  variant="order"
                  onImport={(...args) =>
                    actions.current?.applyValues(
                      airDocumentImportPatch(...args),
                    )
                  }
                />
              ),
            },
          ]}
        />
      </ConfigProvider>,
    );
    expect(isTabDirty(tabKey)).toBe(false);
    fireEvent.click(screen.getByText('第二通知人 (Second Notify Party) · 已填写'));
    expect(screen.getByRole('textbox', { name: '第二通知人' })).toHaveValue(
      '原第二通知人',
    );
    expect(isTabDirty(tabKey)).toBe(false);
    fireEvent.click(screen.getByRole('button', { name: /从订单货物信息带入/ }));
    await waitFor(() =>
      expect(screen.getByRole('textbox', { name: '运单品名' })).toHaveValue(
        '订单品名',
      ),
    );
    expect(isTabDirty(tabKey)).toBe(true);
    expect(
      formRef.current?.getFieldValue(['airHouseContent', 'consigneeText']),
    ).toBe('原收货人');
    expect(formRef.current?.getFieldValue(['airHouseContent', 'remarks'])).toBe(
      '低频备注',
    );
    await act(async () => {});
  });
});
