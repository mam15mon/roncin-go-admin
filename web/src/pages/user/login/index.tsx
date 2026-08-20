import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { LoginForm, ProFormText } from '@ant-design/pro-components';
import { Helmet, useModel } from '@umijs/max';
import { App } from 'antd';
import React, { startTransition } from 'react';
import { authServiceLogin } from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';

function safeRedirect(value: string | null): string {
  if (!value?.startsWith('/') || value.startsWith('//')) return '/';
  const parsed = new URL(value, window.location.origin);
  if (parsed.origin !== window.location.origin) return '/';
  return `${parsed.pathname}${parsed.search}${parsed.hash}`;
}

export default function Login() {
  const { setInitialState } = useModel('@@initialState');
  const { message } = App.useApp();

  const handleSubmit = async (values: API.LoginRequest) => {
    const response = await authServiceLogin(values);
    if (!response.data) return;
    startTransition(() => {
      setInitialState((state) => ({ ...state, currentUser: response.data }));
    });
    message.success('登录成功');
    const redirect = new URL(window.location.href).searchParams.get('redirect');
    window.location.href = safeRedirect(redirect);
  };

  return (
    <>
      <Helmet>
        <title>登录 - {Settings.title}</title>
      </Helmet>
      <LoginForm<API.LoginRequest>
        title="Roncin"
        subTitle="货代业务管理后台"
        logo={<img alt="Roncin" src="/logo.svg" />}
        onFinish={handleSubmit}
      >
        <ProFormText
          name="username"
          fieldProps={{ size: 'large', prefix: <UserOutlined /> }}
          placeholder="用户名"
          rules={[{ required: true, message: '请输入用户名' }]}
        />
        <ProFormText.Password
          name="password"
          fieldProps={{ size: 'large', prefix: <LockOutlined /> }}
          placeholder="密码"
          rules={[{ required: true, message: '请输入密码' }]}
        />
      </LoginForm>
    </>
  );
}
