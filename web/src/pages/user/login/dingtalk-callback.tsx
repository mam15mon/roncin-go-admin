import { DingdingOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App, Button, Result, Spin } from 'antd';
import React, { startTransition, useEffect, useRef, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { useInitialState } from '@/app/AppProvider';
import { dingTalkLoginConfigOptions } from '@/app/dingtalkLoginConfig';
import { DingTalkLoginStatus } from '@/enums.generated';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import { history } from '@/router/history';
import { authServiceDingTalkLogin } from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';
import {
  type LoginOrganizationOption,
  OrganizationPicker,
  resolveLoginOrganizationOptions,
} from './components/organization-picker';
import styles from './index.module.less';

function storedRedirect(): string {
  const value = sessionStorage.getItem('dingtalk_login_redirect');
  sessionStorage.removeItem('dingtalk_login_redirect');
  if (!value?.startsWith('/') || value.startsWith('//')) return '/';
  return value;
}

export default function DingTalkCallback() {
  const { setInitialState } = useInitialState();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [failure, setFailure] = useState<string>();
  const [organizationSelection, setOrganizationSelection] = useState<{
    options: LoginOrganizationOption[];
    currentOrganizationId?: string;
  }>();
  const pendingRedirectRef = useRef('/');
  const handled = useRef(false);
  const login = useMutation({
    mutationFn: async (credentials: API.DingTalkLoginRequest) => {
      // 换码会消费 state Cookie；仅标记缓存失效，返回扫码页时再取新配置。
      await queryClient.invalidateQueries({
        queryKey: dingTalkLoginConfigOptions.queryKey,
        exact: true,
        refetchType: 'none',
      });
      const response = await authServiceDingTalkLogin(credentials, {
        skipErrorHandler: true,
      });
      if (!response?.data) throw new Error('钉钉认证未返回有效结果');
      if (
        !response.data.status ||
        (response.data.status ===
          DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED &&
          !response.data.currentUser)
      )
        throw new Error('钉钉认证未返回有效结果');
      return response.data;
    },
    meta: { silent: true },
    onSuccess: (data) => {
      if (
        data.status !==
          DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED ||
        !data.currentUser
      )
        return;
      startTransition(() =>
        setInitialState((current) => ({
          ...current,
          currentUser: data.currentUser,
        })),
      );
      const options = resolveLoginOrganizationOptions(
        undefined,
        data.currentUser,
      );
      if (options.length > 1) {
        pendingRedirectRef.current = storedRedirect();
        setOrganizationSelection({
          options,
          currentOrganizationId: data.currentUser.currentOrganization?.id,
        });
        return;
      }
      message.success('钉钉登录成功');
      window.location.replace(storedRedirect());
    },
    onError: (error) =>
      setFailure(getRequestErrorMessage(error, '钉钉登录失败')),
  });
  const { mutate } = login;
  useEffect(() => {
    if (handled.current) return;
    handled.current = true;
    const params = new URL(window.location.href).searchParams;
    const authCode = params.get('authCode') ?? '';
    const state = params.get('state') ?? '';
    if (!authCode || !state) {
      setFailure('钉钉未返回有效的登录凭证，请重新扫码');
      return;
    }
    mutate({ authCode, state });
  }, [mutate]);

  const status = login.data?.status;
  const blocked =
    status !== undefined &&
    status !== DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED;
  const title =
    status === DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_NOT_SYNCED
      ? '人员尚未同步'
      : status === DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_PENDING_ASSIGNMENT
        ? '尚未分配组织'
        : status === DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_DISABLED
          ? '账号已停用'
          : '钉钉认证未完成';
  return (
    <div className={styles.loginContainer} style={{ justifyContent: 'center' }}>
      <Helmet>
        <title>钉钉认证 - {Settings.title}</title>
      </Helmet>
      {organizationSelection ? (
        <div
          style={{
            width: '100%',
            padding: '0 16px',
            display: 'flex',
            justifyContent: 'center',
          }}
        >
          <OrganizationPicker
            options={organizationSelection.options}
            currentOrganizationId={organizationSelection.currentOrganizationId}
            onEnter={() => {
              message.success('钉钉登录成功');
              window.location.replace(pendingRedirectRef.current);
            }}
          />
        </div>
      ) : failure || blocked ? (
        <Result
          status={failure ? 'error' : 'info'}
          icon={<DingdingOutlined style={{ color: '#1677ff' }} />}
          title={failure ? '钉钉认证未完成' : title}
          subTitle={
            failure ||
            login.data?.message ||
            '请联系管理员同步人员并分配组织和角色后，再使用钉钉登录。'
          }
          extra={
            <Button type="primary" onClick={() => history.push('/user/login')}>
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
