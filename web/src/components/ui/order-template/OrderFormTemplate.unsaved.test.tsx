import type { ProFormInstance } from '@ant-design/pro-components';
import { ProFormText } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import type { MutableRefObject } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  getFormDraft,
  getFormDraftKey,
  hasFormDraft,
} from '@/components/layout/formDraft';
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
  it.each(['用户继续键入', '主动带入'] as const)(
    '保存等待期间%s后保留输入、草稿和离开保护，下一次无新编辑保存才清除',
    async (editMode) => {
      const tabKey = '/orders/sea-export';
      const draftPathname = '/orders/sea-export/create';
      const draftScope = 'user:company';
      const draftKey = getFormDraftKey(tabKey, draftPathname, draftScope);
      const formRef: MutableRefObject<ProFormInstance | undefined> = {
        current: undefined,
      };
      const actionsRef: MutableRefObject<
        OrderFormTemplateActions<{ reference: string }> | undefined
      > = { current: undefined };
      let finishSave!: (saved: boolean) => void;
      const onFinish = vi.fn(
        () =>
          new Promise<boolean>((resolve) => {
            finishSave = resolve;
          }),
      );
      renderWithApp(
        <OrderFormTemplate
          tabKey={tabKey}
          draftPathname={draftPathname}
          draftScope={draftScope}
          formRef={formRef}
          actionsRef={actionsRef}
          onFinish={onFinish}
          initialValues={{ reference: '初始值' }}
          sections={[
            {
              key: 'reference',
              title: '订单资料',
              content: <ProFormText name="reference" label="业务编号" />,
            },
          ]}
        />,
      );
      fireEvent.change(screen.getByLabelText('业务编号'), {
        target: { value: '本次提交' },
      });
      await act(async () => formRef.current?.submit());
      await waitFor(() => expect(onFinish).toHaveBeenCalledOnce());
      if (editMode === '用户继续键入')
        fireEvent.change(screen.getByLabelText('业务编号'), {
          target: { value: '尚未提交的新值' },
        });
      else
        await act(async () =>
          actionsRef.current?.applyValues({ reference: '尚未提交的新值' }),
        );
      await act(async () => finishSave(true));
      expect(screen.getByLabelText('业务编号')).toHaveValue('尚未提交的新值');
      expect(isTabDirty(tabKey)).toBe(true);
      expect(getFormDraft(draftKey)).toMatchObject({
        reference: '尚未提交的新值',
      });
      await act(async () => formRef.current?.submit());
      await waitFor(() => expect(onFinish).toHaveBeenCalledTimes(2));
      await act(async () => finishSave(true));
      expect(isTabDirty(tabKey)).toBe(false);
      expect(hasFormDraft(draftKey)).toBe(false);
    },
  );
});
