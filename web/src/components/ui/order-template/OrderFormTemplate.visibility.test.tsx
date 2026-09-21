import { type ProFormInstance, ProFormText } from '@ant-design/pro-components';
import { act, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { expect, it } from 'vitest';
import { getFormDraftKey } from '@/components/layout/formDraft';
import { OrderFormTemplate } from './OrderFormTemplate';

it('分节配置动态变化时使用当前表单值判断，新增后置分节正常展示', async () => {
  const formRef: { current: ProFormInstance | undefined } = {
    current: undefined,
  };
  const sections = [
    { key: 'base', title: '基本信息', content: <ProFormText name="enabled" /> },
    {
      key: 'optional',
      title: '可选内容',
      content: '条件区块',
      visible: (values: Record<string, unknown>) => values.enabled === 'yes',
    },
  ];
  const renderTemplate = (appended: boolean) => (
    <App>
      <OrderFormTemplate
        formRef={formRef}
        enableCloseGuard={false}
        submitter={false}
        initialValues={{ enabled: 'no' }}
        sections={sections}
        appendSections={
          appended
            ? [{ key: 'extra', title: '后置区块', content: '后置内容' }]
            : []
        }
      />
    </App>
  );
  const { rerender } = render(renderTemplate(false));
  expect(screen.queryByText('条件区块')).not.toBeInTheDocument();
  rerender(renderTemplate(true));
  expect(screen.getByText('后置内容')).toBeInTheDocument();
  await act(async () => {
    formRef.current?.setFieldsValue({ enabled: 'yes' });
  });
  await waitFor(() => expect(screen.getByText('条件区块')).toBeInTheDocument());
  rerender(renderTemplate(false));
  await waitFor(() => expect(screen.getByText('条件区块')).toBeInTheDocument());
  expect(screen.queryByText('后置内容')).not.toBeInTheDocument();
});

it('初始加载结束后按恢复的草稿判断可见性，不沿用初始值', async () => {
  const draftKey = getFormDraftKey('visibility', '/visibility', 'review');
  sessionStorage.setItem(draftKey, JSON.stringify({ enabled: 'no' }));
  const formRef: { current: ProFormInstance | undefined } = {
    current: undefined,
  };
  const renderTemplate = (loading: boolean) => (
    <App>
      <OrderFormTemplate
        loading={loading}
        formRef={formRef}
        tabKey="visibility"
        draftPathname="/visibility"
        draftScope="review"
        enableCloseGuard={false}
        submitter={false}
        initialValues={{ enabled: 'yes' }}
        sections={[
          {
            key: 'base',
            title: '基本信息',
            content: <ProFormText name="enabled" />,
          },
          {
            key: 'optional',
            title: '可选内容',
            content: '草稿条件区块',
            visible: (values) => values.enabled === 'yes',
          },
        ]}
      />
    </App>
  );
  try {
    const { rerender } = render(renderTemplate(true));
    rerender(renderTemplate(false));
    expect(formRef.current?.getFieldValue('enabled')).toBe('no');
    expect(screen.queryByText('草稿条件区块')).not.toBeInTheDocument();
    await act(async () => {
      formRef.current?.resetFields();
    });
    expect(screen.getByText('草稿条件区块')).toBeInTheDocument();
  } finally {
    sessionStorage.removeItem(draftKey);
  }
});
