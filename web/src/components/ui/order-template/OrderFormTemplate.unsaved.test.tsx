import type { ProFormInstance } from '@ant-design/pro-components';
import { ProFormText } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import type { MutableRefObject } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  confirmIfTabsDirty,
  isTabDirty,
} from '@/components/layout/tabCloseGuard';
import { OrderFormTemplate } from './OrderFormTemplate';
import type { OrderFormTemplateActions } from './types';

afterEach(() => {
  _clearAllTabCloseGuards();
  sessionStorage.clear();
});
describe('订单表单无持久草稿的未保存保护', () => {
  it('用户编辑即时标脏，外部变只读不撤销已有保护，resetTo后清理', async () => {
    const formRef: MutableRefObject<ProFormInstance | undefined> = {
      current: undefined,
    };
    const actionsRef: MutableRefObject<
      OrderFormTemplateActions<{ reference: string }> | undefined
    > = { current: undefined };
    const ui = (readonly: boolean) => (
      <OrderFormTemplate
        tabKey="/orders/sea-export"
        formRef={formRef}
        actionsRef={actionsRef}
        readonly={readonly}
        initialValues={{ reference: '初始值' }}
        sections={[
          {
            key: 'reference',
            title: '订单资料',
            content: <ProFormText name="reference" label="业务编号" />,
          },
        ]}
      />
    );
    const view = renderWithApp(ui(false));
    expect(isTabDirty('/orders/sea-export')).toBe(false);
    await act(async () =>
      formRef.current?.setFieldValue('reference', '初始化带入'),
    );
    expect(isTabDirty('/orders/sea-export')).toBe(false);
    fireEvent.change(screen.getByLabelText('业务编号'), {
      target: { value: '用户修改' },
    });
    expect(isTabDirty('/orders/sea-export')).toBe(true);
    view.rerender(ui(true));
    expect(isTabDirty('/orders/sea-export')).toBe(true);
    expect(formRef.current?.getFieldValue('reference')).toBe('用户修改');
    await act(async () =>
      actionsRef.current?.resetTo({ reference: '服务端新值' }),
    );
    expect(isTabDirty('/orders/sea-export')).toBe(false);
  });

  it('显式带入后同一事件内关闭仍确认，不生成详情草稿，重置后立即放行', async () => {
    const tabKey = '/orders/sea-export';
    const actionsRef: MutableRefObject<
      OrderFormTemplateActions<{ reference: string }> | undefined
    > = { current: undefined };
    const confirmation = vi.fn();
    const close = vi.fn();
    renderWithApp(
      <OrderFormTemplate
        tabKey={tabKey}
        draftScope="user:company"
        actionsRef={actionsRef}
        initialValues={{ reference: '初始值' }}
        submitter={false}
        sections={[
          {
            key: 'reference',
            title: '订单资料',
            content: <ProFormText name="reference" label="业务编号" />,
          },
        ]}
      />,
    );

    await act(async () => {
      actionsRef.current?.applyValues({ reference: '用户主动带入' });
      confirmIfTabsDirty([tabKey], close, confirmation, 'user:company');
      expect(confirmation).toHaveBeenCalledOnce();
      expect(close).not.toHaveBeenCalled();
    });
    expect(screen.getByLabelText('业务编号')).toHaveValue('用户主动带入');
    expect(sessionStorage.length).toBe(0);

    await act(async () => {
      actionsRef.current?.resetTo({ reference: '已保存值' });
      confirmIfTabsDirty([tabKey], close, confirmation, 'user:company');
      expect(close).toHaveBeenCalledOnce();
      expect(confirmation).toHaveBeenCalledOnce();
    });
    expect(screen.getByLabelText('业务编号')).toHaveValue('已保存值');
  });
});
