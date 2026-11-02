import {
  DingdingOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { App, Button, Checkbox, Form, Input, Segmented, Tooltip } from 'antd';
import React, { startTransition, useCallback, useRef, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { useInitialState } from '@/app/AppProvider';
import { isDingTalkClient } from '@/app/dingtalkEnvironment';
import {
  formatPasskeyError,
  usePasskeyEnrollment,
  usePasskeyLogin,
} from '@/features/auth/passkeys';
import { clearOrderMasterDataCache } from '@/features/orders/options';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import { history } from '@/router/history';
import { authServiceLogin } from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';
import { AnimatedCharacters } from './components/animated-characters';
import { DingTalkEmbeddedLogin } from './components/dingtalk-embedded-login';
import { DingTalkQrLogin } from './components/dingtalk-qr-login';
import {
  type LoginOrganizationOption,
  OrganizationPicker,
  resolveLoginOrganizationOptions,
} from './components/organization-picker';
import { PasswordResetModal } from './components/password-reset-modal';
import styles from './index.module.less';
import { useRememberLogin } from './loginPreferences';
import { safeLoginRedirect } from './redirect';

export default function Login() {
  const { setInitialState } = useInitialState();
  const embedded = isDingTalkClient(navigator.userAgent);
  const { message } = App.useApp();
  const { rememberLogin, setRememberLogin } = useRememberLogin();
  const passkeys = usePasskeyLogin(rememberLogin);
  const { completeLogin, enrollmentPrompt, isEnrollmentOpen } =
    usePasskeyEnrollment();
  const queryClient = useQueryClient();
  const [resetPasswordOpen, setResetPasswordOpen] = useState(false);
  const [form] = Form.useForm<API.LoginRequest>();

  const [loading, setLoading] = useState(false);
  const busy = loading || passkeys.isPending || isEnrollmentOpen;
  const [embeddedSwitchLocked, setEmbeddedSwitchLocked] = useState(false);
  const embeddedSwitchLockedRef = useRef(false);
  const handleEmbeddedSwitchLock = useCallback((locked: boolean) => {
    embeddedSwitchLockedRef.current = locked;
    setEmbeddedSwitchLocked(locked);
  }, []);
  const [isTyping, setIsTyping] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [passwordValue, setPasswordValue] = useState('');
  const [loginMode, setLoginMode] = useState<'dingtalk' | 'password'>(
    'dingtalk',
  );
  const qrMode = loginMode === 'dingtalk' && !embedded;
  const [accountStep, setAccountStep] = useState<'account' | 'password'>(
    'account',
  );
  const previousAccount = useRef('');
  const usernameValue = Form.useWatch('username', form);
  const passwordInput = Form.useWatch('password', form);
  const [organizationSelection, setOrganizationSelection] = useState<{
    options: LoginOrganizationOption[];
    currentOrganizationId?: string;
  }>();
  const [pendingRedirect, setPendingRedirect] = useState('/');

  const finishLogin = (redirect: string) => {
    message.success('登录成功');
    window.location.assign(redirect);
  };

  const acceptAuthenticatedLogin = (
    response: API.LoginResponse | API.FinishPasskeyLoginResponse,
  ) => {
    const user = response.data;
    if (!user?.id) throw new Error('登录未返回有效用户，请重试');
    const options = resolveLoginOrganizationOptions(
      response.organizationChoices,
      user,
    );
    const redirect = safeLoginRedirect(
      new URL(window.location.href).searchParams.get('redirect'),
    );
    completeLogin(() => {
      startTransition(() => {
        setInitialState((state) => ({ ...state, currentUser: user }));
      });
      if (options.length > 1) {
        setPendingRedirect(redirect);
        setOrganizationSelection({
          options,
          currentOrganizationId: user.currentOrganization?.id,
        });
        return;
      }
      finishLogin(redirect);
    }, user.id);
  };

  const handleSubmit = async (values: API.LoginRequest) => {
    setLoading(true);
    try {
      const response = await authServiceLogin(
        { ...values, rememberMe: rememberLogin },
        { skipErrorHandler: true },
      );
      acceptAuthenticatedLogin(response);
    } catch (error) {
      message.error(getRequestErrorMessage(error, '登录失败，请稍后重试'));
    } finally {
      setLoading(false);
    }
  };

  const handlePasskeyLogin = () => {
    if (busy) return;
    setLoading(true);
    // 原生密钥请求由 login 在本次点击栈同步启动，后续仅等待验证结果。
    const pending = passkeys.login();
    void pending
      .then(acceptAuthenticatedLogin)
      .catch((error: unknown) => {
        message.error(formatPasskeyError(error, '通行密钥登录失败，请重试'));
      })
      .finally(() => setLoading(false));
  };

  const handleAccountStep = (values: API.LoginRequest) => {
    if (accountStep === 'password') {
      void handleSubmit(values);
      return;
    }
    if (previousAccount.current !== values.username) {
      form.setFieldValue('password', '');
      setPasswordValue('');
      setShowPassword(false);
    }
    previousAccount.current = values.username;
    setIsTyping(false);
    setAccountStep('password');
  };

  const handleDingTalkAuthenticated = useCallback((callbackPath: string) => {
    const redirect = new URL(window.location.href).searchParams.get('redirect');
    sessionStorage.setItem(
      'dingtalk_login_redirect',
      safeLoginRedirect(redirect),
    );
    history.replace(callbackPath);
  }, []);

  return (
    <div className={styles.loginContainer}>
      {enrollmentPrompt}
      <Helmet>
        <title>登录 - {Settings.title}</title>
      </Helmet>

      {/* 品牌与互动角色 */}
      <div className={styles.heroSection}>
        <div className={styles.brandLogoWrapper}>
          <img
            src="/images/logo-only.webp"
            alt="RONCIN"
            className={styles.brandLogoImg}
            onError={(e) => {
              (e.currentTarget as HTMLImageElement).src = '/logo.svg';
            }}
          />
        </div>

        {/* 保留角色跟随输入与密码可见性的交互 */}
        <div className={styles.charactersWrapper}>
          <AnimatedCharacters
            isTyping={isTyping}
            showPassword={showPassword}
            passwordLength={passwordValue.length}
          />
        </div>
      </div>

      {resetPasswordOpen && (
        <PasswordResetModal
          initialMobile={form.getFieldValue('username')}
          onClose={() => setResetPasswordOpen(false)}
          onSuccess={(mobile) => {
            clearOrderMasterDataCache();
            queryClient.clear();
            sessionStorage.removeItem('dingtalk_login_redirect');
            setInitialState((state) => ({ ...state, currentUser: undefined }));
            setOrganizationSelection(undefined);
            setPendingRedirect('/');
            form.setFieldsValue({ username: mobile, password: '' });
            setPasswordValue('');
            setShowPassword(false);
            setIsTyping(false);
            setLoginMode('password');
            setAccountStep('password');
            previousAccount.current = mobile;
            setResetPasswordOpen(false);
            message.success('密码设置成功，所有设备已退出，请重新登录');
          }}
        />
      )}

      {/* 认证主体与独立页脚 */}
      <div className={styles.formSection}>
        <main className={styles.formMain}>
          {organizationSelection && organizationSelection.options.length > 1 ? (
            <OrganizationPicker
              options={organizationSelection.options}
              currentOrganizationId={
                organizationSelection.currentOrganizationId
              }
              onEnter={() => {
                finishLogin(pendingRedirect);
              }}
            />
          ) : (
            <section className={styles.formCard} aria-label="登录">
              <Segmented<'dingtalk' | 'password'>
                name="login-mode"
                aria-label="登录方式"
                className={styles.loginModeSwitch}
                classNames={{
                  label: styles.switchLabel,
                  item: styles.switchItem,
                }}
                size="middle"
                shape="round"
                value={loginMode}
                options={[
                  { label: '账号登录', value: 'password' },
                  {
                    label: embedded ? '钉钉免登' : '扫码登录',
                    value: 'dingtalk',
                  },
                ]}
                disabled={busy || embeddedSwitchLocked}
                onChange={(mode) => {
                  if (embeddedSwitchLockedRef.current) return;
                  setIsTyping(false);
                  setShowPassword(false);
                  setLoginMode(mode);
                }}
              />
              <span className={styles.languageLabel}>简体中文</span>

              <div className={styles.formHeader}>
                <div className={styles.authIcon} aria-hidden="true">
                  <DingdingOutlined />
                </div>
                <h1 className={styles.headerTitle}>
                  {loginMode === 'dingtalk'
                    ? embedded
                      ? '钉钉客户端登录'
                      : '使用钉钉扫码'
                    : '登录 Roncin'}
                </h1>
                <div className={styles.headerSubtitle}>
                  {loginMode === 'password' && accountStep === 'password' ? (
                    <>
                      <span
                        className={styles.accountIdentity}
                        title={usernameValue}
                      >
                        {usernameValue}
                      </span>
                      <button
                        type="button"
                        className={styles.changeAccount}
                        disabled={busy}
                        onClick={() => {
                          setAccountStep('account');
                          setShowPassword(false);
                        }}
                      >
                        修改账号
                      </button>
                    </>
                  ) : loginMode === 'dingtalk' && embedded ? (
                    '使用当前钉钉身份进入工作台'
                  ) : (
                    '继续使用 Roncin 管理后台'
                  )}
                </div>
              </div>

              <div
                className={[
                  styles.authContent,
                  qrMode ? styles.qrAuthContent : styles.accountAuthContent,
                ].join(' ')}
              >
                {embedded ? (
                  loginMode === 'dingtalk' && (
                    <DingTalkEmbeddedLogin
                      onSwitchLockChange={handleEmbeddedSwitchLock}
                      onAccountLogin={() => {
                        if (!embeddedSwitchLockedRef.current)
                          setLoginMode('password');
                      }}
                    />
                  )
                ) : (
                  <DingTalkQrLogin
                    active={
                      loginMode === 'dingtalk' && !busy && !resetPasswordOpen
                    }
                    onAuthenticated={handleDingTalkAuthenticated}
                  />
                )}
                {loginMode === 'password' && (
                  <Form<API.LoginRequest>
                    form={form}
                    layout="vertical"
                    requiredMark={false}
                    onFinish={handleAccountStep}
                    initialValues={{ username: '', password: '' }}
                  >
                    <Form.Item
                      name="username"
                      hidden={accountStep !== 'account'}
                      rules={[{ required: true, message: '请输入账号' }]}
                      className={styles.accountField}
                    >
                      <Input
                        id="login_username"
                        aria-label="账号"
                        placeholder="钉钉手机号 / 本地账号"
                        autoComplete="username"
                        className={styles.cleanInput}
                        disabled={busy}
                        onFocus={() => setIsTyping(true)}
                        onBlur={() => setIsTyping(false)}
                      />
                    </Form.Item>
                    <Form.Item
                      name="password"
                      hidden={accountStep !== 'password'}
                      rules={[
                        {
                          required: accountStep === 'password',
                          message: '请输入密码',
                        },
                      ]}
                      className={styles.accountField}
                    >
                      <Input.Password
                        id="login_password"
                        aria-label="密码"
                        placeholder="请输入密码"
                        autoComplete="current-password"
                        className={styles.cleanInput}
                        disabled={busy}
                        iconRender={(visible) =>
                          visible ? <EyeOutlined /> : <EyeInvisibleOutlined />
                        }
                        visibilityToggle={{
                          visible: showPassword,
                          onVisibleChange: setShowPassword,
                        }}
                        onChange={(event) =>
                          setPasswordValue(event.target.value)
                        }
                        onFocus={() => setIsTyping(true)}
                        onBlur={() => setIsTyping(false)}
                      />
                    </Form.Item>
                    <Form.Item style={{ marginBottom: 0 }}>
                      <Button
                        type="primary"
                        htmlType="submit"
                        aria-label={
                          loading
                            ? '登录中...'
                            : accountStep === 'account'
                              ? '下一步'
                              : '登录'
                        }
                        loading={loading}
                        disabled={
                          busy ||
                          !(
                            accountStep === 'account'
                              ? usernameValue
                              : passwordInput
                          )?.trim()
                        }
                        className={styles.submitButton}
                      >
                        {loading
                          ? '登录中...'
                          : accountStep === 'account'
                            ? '下一步'
                            : '登录'}
                      </Button>
                    </Form.Item>
                  </Form>
                )}
              </div>

              {(!embedded || loginMode === 'password') && (
                <div
                  className={[
                    styles.loginOptions,
                    styles.passkeyOptions,
                    loginMode === 'password' ? styles.accountOptions : '',
                  ].join(' ')}
                >
                  <Checkbox
                    checked={rememberLogin}
                    onChange={(event) => setRememberLogin(event.target.checked)}
                    disabled={busy || embeddedSwitchLocked}
                    className={styles.rememberControl}
                    classNames={{
                      icon: styles.rememberIcon,
                      label: styles.rememberLabel,
                    }}
                  >
                    自动登录
                  </Checkbox>
                  <Tooltip
                    title={
                      passkeys.reason || '登录成功后，为当前设备添加通行密钥'
                    }
                  >
                    <span className={styles.passkeyOptionRow}>
                      <Checkbox
                        checked={passkeys.canUse && passkeys.enrollAfterLogin}
                        disabled={
                          busy || embeddedSwitchLocked || !passkeys.canUse
                        }
                        onChange={(event) =>
                          passkeys.setEnrollAfterLogin(event.target.checked)
                        }
                        className={styles.rememberControl}
                        classNames={{
                          icon: styles.rememberIcon,
                          label: styles.rememberLabel,
                        }}
                      >
                        开启免密登录(通行密钥)
                      </Checkbox>
                    </span>
                  </Tooltip>
                </div>
              )}
              {!embedded && passkeys.error && (
                <div className={styles.passkeyError} role="status">
                  <span>{passkeys.error}</span>
                  <Button
                    type="link"
                    size="small"
                    disabled={busy || passkeys.isPreparing}
                    onClick={() => {
                      void passkeys.prepareAgain().catch((error: unknown) => {
                        message.error(
                          formatPasskeyError(error, '通行密钥准备失败，请重试'),
                        );
                      });
                    }}
                  >
                    重新准备
                  </Button>
                </div>
              )}
              <div className={styles.cardActions}>
                {!embedded && (
                  <Tooltip
                    title={passkeys.reason || '使用已保存的通行密钥登录'}
                  >
                    <Button
                      type="link"
                      disabled={
                        busy || !passkeys.canUse || passkeys.isPreparing
                      }
                      loading={passkeys.isPending}
                      onClick={handlePasskeyLogin}
                      className={styles.cardAction}
                    >
                      免密登录
                    </Button>
                  </Tooltip>
                )}
                <Button
                  type="link"
                  disabled={busy || embeddedSwitchLocked}
                  onClick={() => setResetPasswordOpen(true)}
                  className={styles.cardAction}
                >
                  设置 / 忘记密码
                </Button>
              </div>
            </section>
          )}
        </main>
        <footer className={styles.formFooter}>
          © {new Date().getFullYear()} Roncin Global Logistics
        </footer>
      </div>
    </div>
  );
}
