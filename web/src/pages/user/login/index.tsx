import { LockOutlined, SafetyCertificateOutlined, UserOutlined } from '@ant-design/icons';
import { LoginForm, ProFormText } from '@ant-design/pro-components';
import { Helmet, useModel } from '@umijs/max';
import { App, Space, Typography } from 'antd';
import React, { startTransition } from 'react';
import { authServiceLogin } from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';

const { Text } = Typography;

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
    try {
      const response = await authServiceLogin(values, { skipErrorHandler: true });
      if (!response.data) return;
      startTransition(() => {
        setInitialState((state) => ({ ...state, currentUser: response.data }));
      });
      message.success('登录成功');
      const redirect = new URL(window.location.href).searchParams.get('redirect');
      window.location.href = safeRedirect(redirect);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '登录失败，请稍后重试');
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100vh',
        overflow: 'auto',
        backgroundColor: '#f8fafc',
      }}
    >
      <Helmet>
        <title>登录 - {Settings.title}</title>
      </Helmet>
      <div style={{ flex: '1', padding: '32px 0' }}>
        <LoginForm<API.LoginRequest>
          title="Roncin Admin"
          subTitle="国际货运代理与供应链协同管理平台"
          logo={<img alt="Roncin" src="/logo.svg" />}
          submitter={{
            searchConfig: {
              submitText: '登 录',
            },
            submitButtonProps: {
              size: 'large',
              style: {
                width: '100%',
                fontWeight: 600,
              },
            },
          }}
          onFinish={handleSubmit}
        >
          <div style={{ marginTop: 24 }}>
            <ProFormText
              name="username"
              fieldProps={{
                size: 'large',
                prefix: <UserOutlined style={{ color: '#94a3b8' }} />,
              }}
              placeholder="请输入用户名（登录账号）"
              rules={[{ required: true, message: '请输入用户名' }]}
            />
            <ProFormText.Password
              name="password"
              fieldProps={{
                size: 'large',
                prefix: <LockOutlined style={{ color: '#94a3b8' }} />,
              }}
              placeholder="请输入登录密码"
              rules={[{ required: true, message: '请输入密码' }]}
            />
          </div>
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              marginTop: 16,
              marginBottom: 16,
            }}
          >
            <Space size={6}>
              <SafetyCertificateOutlined style={{ color: '#1677ff', fontSize: 13 }} />
              <Text type="secondary" style={{ fontSize: 12 }}>
                多级组织隔离 · 角色权限边界保护
              </Text>
            </Space>
          </div>
        </LoginForm>
      </div>
    </div>
  );
}
