import { type ProFormInstance, ProFormText } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { expect, it } from 'vitest';
import { OrderFormTemplate } from './OrderFormTemplate';

it('条件字段卸载、校验完成与清空错误时同步导航，统计仅限当前表单', async () => {
  const formRef: { current: ProFormInstance | undefined } = {
    current: undefined,
  };
  const template = (showConditional: boolean) => (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <div data-section-key="other-form">
          <div className="ant-form-item-has-error">其他表单错误</div>
        </div>
        <OrderFormTemplate
          formRef={formRef}
          enableCloseGuard={false}
          sections={[
            {
              key: 'base',
              title: '基本信息',
              content: (
                <ProFormText
                  name="customer"
                  label="客户"
                  rules={[{ required: true, message: '请选择客户' }]}
                />
              ),
            },
            {
              key: 'conditional',
              title: '条件分节',
              content: showConditional ? (
                <ProFormText
                  name="houseNo"
                  label="分单号"
                  rules={[{ required: true, message: '请输入分单号' }]}
                />
              ) : null,
            },
          ]}
        />
      </App>
    </ConfigProvider>
  );
  const { container, rerender } = render(template(true));
  const badge = () =>
    container.querySelector(
      '.roncin-form-anchor-nav-collapsed .ant-badge-count',
    );
  await act(async () => formRef.current?.submit());
  await waitFor(() => expect(badge()).toHaveTextContent('2'));

  rerender(template(false));
  await waitFor(() => expect(badge()).toHaveTextContent('1'));

  fireEvent.change(screen.getByLabelText('客户'), {
    target: { value: '客户甲' },
  });
  await waitFor(() => expect(badge()).not.toBeInTheDocument());
  // 程序写入的错误也应被同步；清零不能留下旧角标。
  await act(async () => {
    formRef.current?.setFields([{ name: 'customer', errors: ['客户已停用'] }]);
  });
  await waitFor(() => expect(badge()).toHaveTextContent('1'));
  await act(async () => {
    formRef.current?.setFields([{ name: 'customer', errors: [] }]);
  });
  await waitFor(() => expect(badge()).not.toBeInTheDocument());
});
