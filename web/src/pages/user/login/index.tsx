import {
  ArrowRightOutlined,
  DingdingOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  LockOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { App, Button, Checkbox, Form, Input, Segmented } from 'antd';
import React, { startTransition, useCallback, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { useInitialState } from '@/app/AppProvider';
import { isDingTalkClient } from '@/app/dingtalkEnvironment';
import { QuarterRing } from '@/components/ui';
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
import { safeLoginRedirect } from './redirect';

export default function Login() {
  const { setInitialState } = useInitialState();
  const embedded = isDingTalkClient(navigator.userAgent);
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [resetPasswordOpen, setResetPasswordOpen] = useState(false);
  const [form] = Form.useForm<API.LoginRequest>();

  const [loading, setLoading] = useState(false);
  const [isTyping, setIsTyping] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [passwordValue, setPasswordValue] = useState('');
  const [loginMode, setLoginMode] = useState<'dingtalk' | 'password'>(
    'dingtalk',
  );
  const [organizationSelection, setOrganizationSelection] = useState<{
    options: LoginOrganizationOption[];
    currentOrganizationId?: string;
  }>();
  const [pendingRedirect, setPendingRedirect] = useState('/');

  const finishLogin = (redirect: string) => {
    message.success('登录成功');
    window.location.assign(redirect);
  };

  const handleSubmit = async (values: API.LoginRequest) => {
    setLoading(true);
    try {
      const response = await authServiceLogin(values, {
        skipErrorHandler: true,
      });
      if (!response.data) return;
      const options = resolveLoginOrganizationOptions(
        response.organizationChoices,
        response.data,
      );
      const redirect = safeLoginRedirect(
        new URL(window.location.href).searchParams.get('redirect'),
      );
      startTransition(() => {
        setInitialState((state) => ({ ...state, currentUser: response.data }));
      });
      // 多组织用户先选择进入组织：会话已按候选顺序建立，
      // 选其他组织时由选择视图经 switch-organization 换发会话后进入。
      if (options.length > 1) {
        setPendingRedirect(redirect);
        setOrganizationSelection({
          options,
          currentOrganizationId: response.data.currentOrganization?.id,
        });
        return;
      }
      finishLogin(redirect);
    } catch (error) {
      // 登录失败已跳过全局 401 跳转，这里展示服务端业务提示（如「用户名或密码错误」）。
      message.error(getRequestErrorMessage(error, '登录失败，请稍后重试'));
    } finally {
      setLoading(false);
    }
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
            setResetPasswordOpen(false);
            message.success('密码设置成功，所有设备已退出，请重新登录');
          }}
        />
      )}

      {/* 认证主体与独立页脚 */}
      <div className={styles.formSection}>
        <div className={styles.mobileLogo}>
          <img src="/images/logo-mark.png" alt="" />
          <span>RONCIN</span>
        </div>
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
                block
                name="login-mode"
                aria-label="登录方式"
                className={styles.loginModeSwitch}
                size="large"
                value={loginMode}
                options={[
                  {
                    label: embedded ? '钉钉免登' : '钉钉扫码',
                    value: 'dingtalk',
                  },
                  { label: '账号登录', value: 'password' },
                ]}
                disabled={loading}
                onChange={(mode) => {
                  setIsTyping(false);
                  setShowPassword(false);
                  setLoginMode(mode);
                }}
              />

              <div className={styles.formHeader}>
                <div className={styles.authIcon} aria-hidden="true">
                  {loginMode === 'dingtalk' ? (
                    <DingdingOutlined />
                  ) : (
                    <UserOutlined />
                  )}
                </div>
                <h1 className={styles.headerTitle}>
                  {loginMode === 'dingtalk'
                    ? embedded
                      ? '钉钉客户端登录'
                      : '钉钉扫码登录'
                    : '账号登录'}
                </h1>
                <p className={styles.headerSubtitle}>
                  {loginMode === 'dingtalk'
                    ? embedded
                      ? '使用当前钉钉身份进入工作台'
                      : '打开手机钉钉「扫一扫」快速登录'
                    : '使用钉钉手机号或本地账号与密码登录'}
                </p>
              </div>

              <div className={styles.authContent}>
                {embedded ? (
                  loginMode === 'dingtalk' && (
                    <DingTalkEmbeddedLogin
                      onAccountLogin={() => setLoginMode('password')}
                    />
                  )
                ) : (
                  <DingTalkQrLogin
                    active={
                      loginMode === 'dingtalk' && !loading && !resetPasswordOpen
                    }
                    onAuthenticated={handleDingTalkAuthenticated}
                  />
                )}
                {loginMode === 'password' && (
                  <Form<API.LoginRequest>
                    form={form}
                    layout="vertical"
                    requiredMark={false}
                    onFinish={handleSubmit}
                    initialValues={{ username: '', password: '' }}
                  >
                    {/* 用户名 */}
                    <Form.Item style={{ marginBottom: 18 }}>
                      <div>
                        <label
                          className={styles.inputLabel}
                          htmlFor="login_username"
                        >
                          账号
                        </label>
                        <Form.Item
                          name="username"
                          noStyle
                          rules={[{ required: true, message: '请输入账号' }]}
                        >
                          <Input
                            id="login_username"
                            size="large"
                            placeholder="钉钉手机号 / 本地账号"
                            autoComplete="username"
                            prefix={
                              <UserOutlined
                                style={{
                                  color: '#94a3b8',
                                  fontSize: 16,
                                  marginRight: 6,
                                }}
                              />
                            }
                            className={styles.cleanInput}
                            disabled={loading}
                            onFocus={() => setIsTyping(true)}
                            onBlur={() => setIsTyping(false)}
                          />
                        </Form.Item>
                      </div>
                    </Form.Item>

                    {/* 密码 */}
                    <Form.Item style={{ marginBottom: 18 }}>
                      <div>
                        <label
                          className={styles.inputLabel}
                          htmlFor="login_password"
                        >
                          密码
                        </label>
                        <Form.Item
                          name="password"
                          noStyle
                          rules={[{ required: true, message: '请输入密码' }]}
                        >
                          <Input.Password
                            id="login_password"
                            size="large"
                            placeholder="请输入密码"
                            autoComplete="current-password"
                            prefix={
                              <LockOutlined
                                style={{
                                  color: '#94a3b8',
                                  fontSize: 16,
                                  marginRight: 6,
                                }}
                              />
                            }
                            className={styles.cleanInput}
                            disabled={loading}
                            iconRender={(visible) =>
                              visible ? (
                                <EyeOutlined style={{ color: '#64748b' }} />
                              ) : (
                                <EyeInvisibleOutlined
                                  style={{ color: '#94a3b8' }}
                                />
                              )
                            }
                            visibilityToggle={{
                              visible: showPassword,
                              onVisibleChange: (visible) =>
                                setShowPassword(visible),
                            }}
                            onChange={(e) => setPasswordValue(e.target.value)}
                            onFocus={() => setIsTyping(true)}
                            onBlur={() => setIsTyping(false)}
                          />
                        </Form.Item>
                      </div>
                    </Form.Item>

                    {/* 辅助操作栏 */}
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        marginBottom: 20,
                      }}
                    >
                      <Checkbox
                        defaultChecked
                        disabled={loading}
                        style={{ fontSize: 13, color: '#475569' }}
                      >
                        保持登录状态
                      </Checkbox>
                      <Button
                        type="link"
                        disabled={loading}
                        onClick={() => setResetPasswordOpen(true)}
                        style={{ paddingInline: 0 }}
                      >
                        设置 / 忘记密码
                      </Button>
                    </div>

                    {/* 登录操作按钮 */}
                    <Form.Item style={{ marginBottom: 0 }}>
                      <Button
                        type="primary"
                        htmlType="submit"
                        aria-label={loading ? '登录中...' : '登录'}
                        loading={loading}
                        disabled={loading}
                        className={styles.submitButton}
                        icon={
                          loading ? (
                            <QuarterRing size={16} strokeWidth="2px" />
                          ) : (
                            <ArrowRightOutlined />
                          )
                        }
                        iconPlacement="end"
                      >
                        {loading ? '登录中...' : '登录'}
                      </Button>
                    </Form.Item>
                  </Form>
                )}
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
