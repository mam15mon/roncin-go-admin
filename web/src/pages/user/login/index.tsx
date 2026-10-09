import {
  ArrowRightOutlined,
  DingdingOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  LockOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { App, Button, Checkbox, Divider, Form, Input } from 'antd';
import React, { startTransition, useEffect, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { useInitialState } from '@/app/AppProvider';
import { QuarterRing } from '@/components/ui';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import {
  authServiceGetDingTalkLoginConfig,
  authServiceLogin,
} from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';
import { AnimatedCharacters } from './components/animated-characters';
import {
  type LoginOrganizationOption,
  OrganizationPicker,
  resolveLoginOrganizationOptions,
} from './components/organization-picker';
import styles from './index.module.less';

function safeRedirect(value: string | null): string {
  if (!value?.startsWith('/') || value.startsWith('//')) return '/';
  const parsed = new URL(value, window.location.origin);
  if (parsed.origin !== window.location.origin) return '/';
  return `${parsed.pathname}${parsed.search}${parsed.hash}`;
}

export default function Login() {
  const { setInitialState } = useInitialState();
  const { message } = App.useApp();
  const [form] = Form.useForm<API.LoginRequest>();

  const [loading, setLoading] = useState(false);
  const [isTyping, setIsTyping] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [passwordValue, setPasswordValue] = useState('');
  const [dingtalkEnabled, setDingtalkEnabled] = useState(false);
  const [dingtalkLoading, setDingtalkLoading] = useState(false);
  const [organizationOptions, setOrganizationOptions] =
    useState<LoginOrganizationOption[]>();
  const [pendingRedirect, setPendingRedirect] = useState('/');
  useEffect(() => {
    authServiceGetDingTalkLoginConfig({ skipErrorHandler: true })
      .then((response) => setDingtalkEnabled(response.data?.enabled ?? false))
      .catch(() => setDingtalkEnabled(false));
  }, []);

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
      const redirect = safeRedirect(
        new URL(window.location.href).searchParams.get('redirect'),
      );
      startTransition(() => {
        setInitialState((state) => ({ ...state, currentUser: response.data }));
      });
      // 多组织用户先选择进入组织：会话已按默认组织建立，
      // 选其他组织时由选择视图经 switch-organization 换发会话后进入。
      if (options.length > 1) {
        setPendingRedirect(redirect);
        setOrganizationOptions(options);
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

  const handleDingTalkLogin = async () => {
    setDingtalkLoading(true);
    try {
      const response = await authServiceGetDingTalkLoginConfig({
        skipErrorHandler: true,
      });
      if (!response.data?.enabled || !response.data.authorizeUrl) {
        message.warning('钉钉登录暂未启用');
        return;
      }
      const redirect = new URL(window.location.href).searchParams.get(
        'redirect',
      );
      sessionStorage.setItem('dingtalk_login_redirect', safeRedirect(redirect));
      window.location.assign(response.data.authorizeUrl);
    } catch (error) {
      message.error(getRequestErrorMessage(error, '钉钉登录启动失败'));
    } finally {
      setDingtalkLoading(false);
    }
  };

  return (
    <div className={styles.loginContainer}>
      <Helmet>
        <title>登录 - {Settings.title}</title>
      </Helmet>

      {/* ── 左侧：深色科技交互区 ── */}
      <div className={styles.heroSection}>
        {/* 弥散光晕 */}
        <div className={styles.glowTopLeft} />
        <div className={styles.glowBottomRight} />

        {/* 顶部品牌 Logo */}
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

        {/* 核心互动动画角色 */}
        <div className={styles.charactersWrapper}>
          <AnimatedCharacters
            isTyping={isTyping}
            showPassword={showPassword}
            passwordLength={passwordValue.length}
          />
        </div>
      </div>

      {/* ── 右侧：认证表单区 / 登录后组织选择区 ── */}
      <div className={styles.formSection}>
        {organizationOptions && organizationOptions.length > 1 ? (
          <OrganizationPicker
            options={organizationOptions}
            onEnter={() => {
              finishLogin(pendingRedirect);
            }}
          />
        ) : (
          <div className={styles.formCard}>
            {/* 移动端 Logo 展示 */}
            <div className={styles.mobileLogo}>
              <img
                src="/logo.svg"
                alt="Roncin"
                style={{ height: 32, width: 'auto' }}
              />
              <div>
                <span
                  style={{
                    fontWeight: 800,
                    fontSize: 18,
                    letterSpacing: '0.08em',
                    color: '#0f172a',
                  }}
                >
                  RONCIN
                </span>
              </div>
            </div>

            {/* 表单头部 */}
            <div>
              <h1 className={styles.headerTitle}>登录</h1>
            </div>

            {/* 登录表单 */}
            <Form<API.LoginRequest>
              form={form}
              layout="vertical"
              requiredMark={false}
              onFinish={handleSubmit}
              initialValues={{ username: '', password: '' }}
            >
              {/* 用户名 */}
              <Form.Item
                name="username"
                rules={[{ required: true, message: '请输入用户名' }]}
                style={{ marginBottom: 20 }}
              >
                <div>
                  <label className={styles.inputLabel} htmlFor="login_username">
                    账号
                  </label>
                  <Input
                    id="login_username"
                    size="large"
                    placeholder="用户名 / 邮箱"
                    prefix={
                      <UserOutlined
                        style={{
                          color: '#94a3b8',
                          fontSize: 16,
                          marginRight: 6,
                        }}
                      />
                    }
                    className={styles.pillInput}
                    disabled={loading}
                    onFocus={() => setIsTyping(true)}
                    onBlur={() => setIsTyping(false)}
                  />
                </div>
              </Form.Item>

              {/* 密码 */}
              <Form.Item
                name="password"
                rules={[{ required: true, message: '请输入密码' }]}
                style={{ marginBottom: 20 }}
              >
                <div>
                  <label className={styles.inputLabel} htmlFor="login_password">
                    密码
                  </label>
                  <Input.Password
                    id="login_password"
                    size="large"
                    placeholder="请输入密码"
                    prefix={
                      <LockOutlined
                        style={{
                          color: '#94a3b8',
                          fontSize: 16,
                          marginRight: 6,
                        }}
                      />
                    }
                    className={styles.pillInput}
                    disabled={loading}
                    iconRender={(visible) =>
                      visible ? (
                        <EyeOutlined style={{ color: '#64748b' }} />
                      ) : (
                        <EyeInvisibleOutlined style={{ color: '#94a3b8' }} />
                      )
                    }
                    visibilityToggle={{
                      visible: showPassword,
                      onVisibleChange: (visible) => setShowPassword(visible),
                    }}
                    onChange={(e) => setPasswordValue(e.target.value)}
                    onFocus={() => setIsTyping(true)}
                    onBlur={() => setIsTyping(false)}
                  />
                </div>
              </Form.Item>

              {/* 辅助操作栏 */}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: 24,
                  paddingLeft: 6,
                  paddingRight: 6,
                }}
              >
                <Checkbox
                  defaultChecked
                  disabled={loading}
                  style={{ fontSize: 13, color: '#64748b' }}
                >
                  保持登录
                </Checkbox>
              </div>

              {/* 登录操作按钮 */}
              <Form.Item style={{ marginBottom: 0 }}>
                <Button
                  type="primary"
                  htmlType="submit"
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

            {dingtalkEnabled && (
              <>
                <Divider plain className={styles.loginDivider}>
                  或
                </Divider>
                <div
                  style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
                >
                  <Button
                    block
                    size="large"
                    icon={<DingdingOutlined />}
                    loading={dingtalkLoading}
                    disabled={loading || dingtalkLoading}
                    onClick={handleDingTalkLogin}
                  >
                    钉钉登录
                  </Button>
                  <div
                    style={{
                      textAlign: 'center',
                      fontSize: 12,
                      color: '#8c8c8c',
                    }}
                  >
                    使用前请联系管理员同步人员并分配组织和角色
                  </div>
                </div>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
