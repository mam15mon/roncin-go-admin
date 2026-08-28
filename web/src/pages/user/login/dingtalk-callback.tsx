import { DingdingOutlined } from '@ant-design/icons';
import { Helmet, useModel } from '@umijs/max';
import { App, Button, Result, Spin } from 'antd';
import React, { startTransition, useEffect, useRef, useState } from 'react';
import {
  authServiceDingTalkLogin,
  authServiceRegisterDingTalkUser,
} from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';
import styles from './index.module.less';

interface LoginError extends Error {
  response?: { data?: { message?: string } };
  data?: { message?: string };
}

interface LoginFailure {
  message: string;
}

function loginFailure(error: unknown): LoginFailure {
  const requestError = error as LoginError;
  const data = requestError.data ?? requestError.response?.data;
  return {
    message: data?.message ?? requestError.message ?? '钉钉登录失败',
  };
}

const dingTalkLoginStatusAuthenticated = 1;
const dingTalkLoginStatusRegistrationRequired = 2;

function storedRedirect(): string {
  const value = sessionStorage.getItem('dingtalk_login_redirect');
  sessionStorage.removeItem('dingtalk_login_redirect');
  if (!value?.startsWith('/') || value.startsWith('//')) return '/';
  return value;
}

export default function DingTalkCallback() {
  const { setInitialState } = useModel('@@initialState');
  const { message } = App.useApp();
  const [failure, setFailure] = useState<LoginFailure>();
  const [registrationName, setRegistrationName] = useState('');
  const [registeredName, setRegisteredName] = useState('');
  const [registrationLoading, setRegistrationLoading] = useState(false);
  const handled = useRef(false);

  useEffect(() => {
    if (handled.current) return;
    handled.current = true;
    const params = new URL(window.location.href).searchParams;
    const authCode = params.get('authCode') ?? '';
    const state = params.get('state') ?? '';
    if (!authCode || !state) {
      setFailure({ message: '钉钉未返回有效的登录凭证，请重新扫码' });
      return;
    }
    authServiceDingTalkLogin({ authCode, state }, { skipErrorHandler: true })
      .then((response) => {
        if (
          response.data?.status === dingTalkLoginStatusRegistrationRequired &&
          response.data.displayName
        ) {
          setRegistrationName(response.data.displayName);
          return;
        }
        if (
          response.data?.status !== dingTalkLoginStatusAuthenticated ||
          !response.data.currentUser
        ) {
          setFailure({ message: '钉钉认证未返回有效结果' });
          return;
        }
        startTransition(() => {
          setInitialState((current) => ({
            ...current,
            currentUser: response.data?.currentUser,
          }));
        });
        message.success('钉钉登录成功');
        window.location.replace(storedRedirect());
      })
      .catch((error) => {
        setFailure(loginFailure(error));
      });
  }, [message, setInitialState]);

  const confirmRegistration = async () => {
    setRegistrationLoading(true);
    try {
      const response = await authServiceRegisterDingTalkUser(
        {},
        { skipErrorHandler: true },
      );
      if (!response.data) {
        setFailure({ message: '钉钉注册未返回申请结果' });
        return;
      }
      setRegisteredName(response.data.displayName ?? registrationName);
      setRegistrationName('');
    } catch (error) {
      setFailure(loginFailure(error));
      setRegistrationName('');
    } finally {
      setRegistrationLoading(false);
    }
  };

  return (
    <div className={styles.loginContainer} style={{ justifyContent: 'center' }}>
      <Helmet>
        <title>钉钉认证 - {Settings.title}</title>
      </Helmet>
      {registeredName ? (
        <Result
          status="success"
          title="入职或返聘申请已提交"
          subTitle={`${registeredName}，企业身份验证已通过。请等待管理员重新确认所属组织和角色，授权完成后即可使用钉钉登录。`}
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
      ) : registrationName ? (
        <Result
          status="info"
          icon={<DingdingOutlined style={{ color: '#1677ff' }} />}
          title="钉钉身份验证完成"
          subTitle={`已确认 ${registrationName} 属于本企业。确认注册后将提交管理员分配所属组织和角色。`}
          extra={
            <>
              <Button
                type="primary"
                loading={registrationLoading}
                onClick={confirmRegistration}
              >
                确认注册
              </Button>
              <Button
                disabled={registrationLoading}
                onClick={() => {
                  window.location.href = '/user/login';
                }}
              >
                取消并返回登录
              </Button>
            </>
          }
        />
      ) : failure ? (
        <Result
          status="error"
          icon={<DingdingOutlined style={{ color: '#1677ff' }} />}
          title="钉钉认证未完成"
          subTitle={failure.message}
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
