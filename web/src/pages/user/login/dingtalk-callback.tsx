import { DingdingOutlined } from '@ant-design/icons';
import { Helmet, useModel } from '@umijs/max';
import { App, Button, Result, Spin } from 'antd';
import React, { startTransition, useEffect, useState } from 'react';
import { authServiceDingTalkLogin } from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';
import styles from './index.module.less';

interface LoginError extends Error {
  response?: { data?: { message?: string } };
  data?: { message?: string };
}

function loginErrorMessage(error: unknown): string {
  const requestError = error as LoginError;
  return (
    requestError.data?.message ??
    requestError.response?.data?.message ??
    requestError.message ??
    '钉钉登录失败'
  );
}

function storedRedirect(): string {
  const value = sessionStorage.getItem('dingtalk_login_redirect');
  sessionStorage.removeItem('dingtalk_login_redirect');
  if (!value?.startsWith('/') || value.startsWith('//')) return '/';
  return value;
}

export default function DingTalkCallback() {
  const { setInitialState } = useModel('@@initialState');
  const { message } = App.useApp();
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    const params = new URL(window.location.href).searchParams;
    const authCode = params.get('authCode') ?? '';
    const state = params.get('state') ?? '';
    if (!authCode || !state) {
      setErrorMessage('钉钉未返回有效的登录凭证，请重新扫码');
      return;
    }
    authServiceDingTalkLogin({ authCode, state }, { skipErrorHandler: true })
      .then((response) => {
        if (!response.data) {
          setErrorMessage('钉钉登录未返回用户信息');
          return;
        }
        startTransition(() => {
          setInitialState((current) => ({
            ...current,
            currentUser: response.data,
          }));
        });
        message.success('钉钉登录成功');
        window.location.replace(storedRedirect());
      })
      .catch((error) => {
        setErrorMessage(loginErrorMessage(error));
      });
  }, [message, setInitialState]);

  return (
    <div className={styles.loginContainer} style={{ justifyContent: 'center' }}>
      <Helmet>
        <title>钉钉登录 - {Settings.title}</title>
      </Helmet>
      {errorMessage ? (
        <Result
          status="info"
          icon={<DingdingOutlined style={{ color: '#1677ff' }} />}
          title="钉钉登录未完成"
          subTitle={errorMessage}
          extra={
            <Button
              type="primary"
              onClick={() => {
                window.location.href = '/user/login';
              }}
            >
              返回登录
            </Button>
          }
        />
      ) : (
        <Spin size="large" />
      )}
    </div>
  );
}
