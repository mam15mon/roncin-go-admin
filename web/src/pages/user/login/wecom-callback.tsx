import { WechatWorkOutlined } from '@ant-design/icons';
import { Helmet, useModel } from '@umijs/max';
import { App, Button, Result, Spin } from 'antd';
import React, { startTransition, useEffect, useState } from 'react';
import { authServiceWeComLogin } from '@/services/roncin/authService';
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
    '企业微信登录失败'
  );
}

function storedRedirect(): string {
  const value = sessionStorage.getItem('wecom_login_redirect');
  sessionStorage.removeItem('wecom_login_redirect');
  if (!value?.startsWith('/') || value.startsWith('//')) return '/';
  return value;
}

export default function WeComCallback() {
  const { setInitialState } = useModel('@@initialState');
  const { message } = App.useApp();
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    const params = new URL(window.location.href).searchParams;
    const code = params.get('code') ?? '';
    const state = params.get('state') ?? '';
    if (!code || !state) {
      setErrorMessage('企业微信未返回有效的登录凭证，请重新扫码');
      return;
    }
    authServiceWeComLogin(
      { code, state },
      { skipErrorHandler: true },
    )
      .then((response) => {
        if (!response.data) {
          setErrorMessage('企业微信登录未返回用户信息');
          return;
        }
        startTransition(() => {
          setInitialState((current) => ({ ...current, currentUser: response.data }));
        });
        message.success('企业微信登录成功');
        const targetUrl = storedRedirect();
        if (window.top && window.top !== window.self) {
          window.top.location.replace(targetUrl);
        } else {
          window.location.replace(targetUrl);
        }
      })
      .catch((error) => {
        setErrorMessage(loginErrorMessage(error));
      });
  }, [message, setInitialState]);

  const isEmbedded = typeof window !== 'undefined' && window.top !== window.self;

  return (
    <div
      className={isEmbedded ? undefined : styles.loginContainer}
      style={
        isEmbedded
          ? {
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              minHeight: 320,
              background: '#ffffff',
            }
          : { justifyContent: 'center' }
      }
    >
      <Helmet>
        <title>企业微信登录 - {Settings.title}</title>
      </Helmet>
      {errorMessage ? (
        <Result
          status="info"
          icon={<WechatWorkOutlined style={{ color: '#2b7ffc' }} />}
          title="企业微信登录未完成"
          subTitle={errorMessage}
          extra={
            <Button
              type="primary"
              onClick={() => {
                if (isEmbedded && window.top) {
                  window.top.location.reload();
                } else {
                  window.location.href = '/user/login';
                }
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
