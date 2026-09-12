import { DingdingOutlined } from '@ant-design/icons';
import { Helmet, useModel } from '@umijs/max';
import { App, Button, Result, Select, Space, Spin } from 'antd';
import React, { startTransition, useEffect, useRef, useState } from 'react';
import { DingTalkLoginStatus } from '@/enums.generated';
import {
  authServiceDingTalkLogin,
  authServiceGetDingTalkInvitationInfo,
  authServiceRegisterDingTalkUser,
} from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';
import {
  type LoginOrganizationOption,
  OrganizationPicker,
  resolveLoginOrganizationOptions,
} from './components/organization-picker';
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
  const [registrationOrganizations, setRegistrationOrganizations] = useState<
    API.OrganizationChoice[]
  >([]);
  const [selectedOrganizationId, setSelectedOrganizationId] = useState('');
  const [organizationOptions, setOrganizationOptions] =
    useState<LoginOrganizationOption[]>();
  const [invitationToken, setInvitationToken] = useState('');
  const [invitationInfo, setInvitationInfo] =
    useState<API.DingTalkInvitationPublicInfo>();
  const pendingRedirectRef = useRef('/');
  const handled = useRef(false);

  useEffect(() => {
    const token =
      sessionStorage.getItem('dingtalk_invitation_token') ||
      new URL(window.location.href).searchParams.get('invite') ||
      '';
    if (token) {
      setInvitationToken(token);
      authServiceGetDingTalkInvitationInfo(
        { token },
        { skipErrorHandler: true },
      )
        .then((response) => {
          if (response.data) setInvitationInfo(response.data);
        })
        .catch(() => {});
    }
  }, []);

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
          response.data?.status ===
            DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_REGISTRATION_REQUIRED &&
          response.data.displayName
        ) {
          setRegistrationName(response.data.displayName);
          setRegistrationOrganizations(
            response.data.registrationOrganizations ?? [],
          );
          return;
        }
        if (
          response.data?.status !==
            DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED ||
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
        // 钉钉登录响应不携带组织候选列表，回退到 principal organizations；
        // 多组织用户先选择进入组织，单组织直接进入。
        const options = resolveLoginOrganizationOptions(
          undefined,
          response.data.currentUser,
        );
        if (options.length > 1) {
          pendingRedirectRef.current = storedRedirect();
          setOrganizationOptions(options);
          return;
        }
        message.success('钉钉登录成功');
        sessionStorage.removeItem('dingtalk_invitation_token');
        window.location.replace(storedRedirect());
      })
      .catch((error) => {
        setFailure(loginFailure(error));
      });
  }, [message, setInitialState]);

  const confirmRegistration = async () => {
    setRegistrationLoading(true);
    try {
      const activeToken =
        invitationToken ||
        sessionStorage.getItem('dingtalk_invitation_token') ||
        undefined;
      const response = await authServiceRegisterDingTalkUser(
        activeToken
          ? { invitationToken: activeToken }
          : selectedOrganizationId
            ? { organizationId: selectedOrganizationId }
            : {},
        { skipErrorHandler: true },
      );
      if (!response.data) {
        setFailure({ message: '钉钉注册未返回申请结果' });
        return;
      }
      setRegisteredName(response.data.displayName ?? registrationName);
      setRegistrationName('');
      sessionStorage.removeItem('dingtalk_invitation_token');
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
          subTitle={
            invitationInfo?.organizationName
              ? `${registeredName}，企业身份验证已通过。申请已转交【${invitationInfo.organizationName}】管理员审批，授权完成后即可使用钉钉登录。`
              : `${registeredName}，企业身份验证已通过。请等待管理员重新确认所属组织和角色，授权完成后即可使用钉钉登录。`
          }
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
          subTitle={
            invitationToken
              ? `已确认 ${registrationName} 属于本企业。您正通过【${invitationInfo?.organizationName || '专属通道'}】申请入职，确认后将转交该分公司管理员审批。`
              : registrationOrganizations.length > 1
                ? `已确认 ${registrationName} 属于本企业。请选择要加入的公司；不选择时由总部审批。`
                : `已确认 ${registrationName} 属于本企业。确认注册后将提交管理员分配所属组织和角色。`
          }
          extra={
            <Space orientation="vertical" size={16}>
              {!invitationToken && registrationOrganizations.length > 1 && (
                <div>
                  <div
                    style={{
                      marginBottom: 8,
                      fontSize: 14,
                      color: 'rgba(0, 0, 0, 0.88)',
                    }}
                  >
                    要加入的公司
                  </div>
                  <Select
                    value={selectedOrganizationId}
                    disabled={registrationLoading}
                    onChange={setSelectedOrganizationId}
                    style={{ width: 280, display: 'block' }}
                    options={[
                      { label: '默认（总部审批）', value: '' },
                      ...registrationOrganizations.map((organization) => ({
                        label: `${organization.organizationName} (${organization.organizationCode})`,
                        value: organization.organizationId,
                      })),
                    ]}
                  />
                </div>
              )}
              <Space>
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
              </Space>
            </Space>
          }
        />
      ) : organizationOptions ? (
        <div
          style={{
            width: '100%',
            maxWidth: 520,
            padding: '0 16px',
            display: 'flex',
            justifyContent: 'center',
          }}
        >
          <OrganizationPicker
            options={organizationOptions}
            onEnter={() => {
              message.success('钉钉登录成功');
              window.location.replace(pendingRedirectRef.current);
            }}
          />
        </div>
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
