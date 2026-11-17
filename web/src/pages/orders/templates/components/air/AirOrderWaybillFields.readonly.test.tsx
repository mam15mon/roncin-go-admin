import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { AirOrderWaybillLayout } from './AirOrderWaybillFields';
import { AirWaybillFields } from './AirWaybillFields';

describe('航空只读正文布局', () => {
  it('父表单禁用仍可切换通知人，只读文本不注册整票字段', async () => {
    let form!: FormInstance;
    function Editor() {
      const [current] = Form.useForm();
      form = current;
      return (
        <ConfigProvider theme={{ token: { motion: false } }}>
          <ProForm
            form={form}
            disabled
            submitter={false}
            initialValues={{ privateValue: '本票原稿' }}
          >
            <AirOrderWaybillLayout
              content={{
                shipperText: '主单发货人',
                notifyPartyText: '第一通知人',
                secondNotifyPartyText: '第二通知人内容',
              }}
            />
          </ProForm>
        </ConfigProvider>
      );
    }
    renderWithApp(<Editor />);
    expect(
      screen.getByRole('textbox', { name: '发货人名称与地址' }),
    ).toHaveValue('主单发货人');
    expect(
      screen.getByRole('textbox', { name: '发货人名称与地址' }),
    ).toHaveAttribute('readonly');
    fireEvent.click(screen.getByText('第二通知人 (Second Notify Party) · 已填写'));
    expect(screen.getByRole('textbox', { name: '第二通知人' })).toHaveValue(
      '第二通知人内容',
    );
    expect(screen.queryByRole('button', { name: /带入/ })).toBeNull();
    await act(async () => expect(await form.validateFields()).toEqual({}));
    expect(form.getFieldValue('privateValue')).toBe('本票原稿');
    expect(form.getFieldValue('shipperText')).toBeUndefined();
  });

  it('缺少明确标脏回调时不展示带入入口', () => {
    renderWithApp(
      <ProForm submitter={false}>
        <AirWaybillFields prefix={['content']} variant="order" />
      </ProForm>,
    );
    expect(screen.queryByRole('button', { name: /带入/ })).toBeNull();
  });
});
