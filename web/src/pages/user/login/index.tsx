import {
  ArrowRightOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  LoadingOutlined,
  LockOutlined,
  UserOutlined,
  WechatWorkOutlined,
} from '@ant-design/icons';
import { Helmet, useModel } from '@umijs/max';
import { App, Button, Checkbox, Divider, Form, Input } from 'antd';
import React, { startTransition, useEffect, useState } from 'react';
import {
  authServiceGetWeComLoginConfig,
  authServiceLogin,
} from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';
import { AnimatedCharacters } from './components/animated-characters';
import styles from './index.module.less';

function safeRedirect(value: string | null): string {
  if (!value?.startsWith('/') || value.startsWith('//')) return '/';
  const parsed = new URL(value, window.location.origin);
  if (parsed.origin !== window.location.origin) return '/';
  return `${parsed.pathname}${parsed.search}${parsed.hash}`;
}

export default function Login() {
  const { setInitialState } = useModel('@@initialState');
  const { message } = App.useApp();
  const [form] = Form.useForm<API.LoginRequest>();

  const [loading, setLoading] = useState(false);
  const [isTyping, setIsTyping] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [passwordValue, setPasswordValue] = useState('');
  const [wecomEnabled, setWecomEnabled] = useState(false);
  const [wecomLoading, setWecomLoading] = useState(false);

  useEffect(() => {
    authServiceGetWeComLoginConfig({ skipErrorHandler: true })
      .then((response) => setWecomEnabled(response.data?.enabled ?? false))
      .catch(() => setWecomEnabled(false));
  }, []);

  const handleSubmit = async (values: API.LoginRequest) => {
    setLoading(true);
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
    } finally {
      setLoading(false);
    }
  };

  const handleWeComLogin = async () => {
    setWecomLoading(true);
    try {
      const response = await authServiceGetWeComLoginConfig({
        skipErrorHandler: true,
      });
      if (!response.data?.enabled || !response.data.authorizeUrl) {
        message.warning('企业微信登录暂未启用');
        return;
      }
      const redirect = new URL(window.location.href).searchParams.get('redirect');
      sessionStorage.setItem('wecom_login_redirect', safeRedirect(redirect));
      window.location.assign(response.data.authorizeUrl);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '企业微信登录启动失败');
    } finally {
      setWecomLoading(false);
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

      {/* ── 右侧：认证表单区 ── */}
      <div className={styles.formSection}>
        <div className={styles.formCard}>
          {/* 移动端 Logo 展示 */}
          <div className={styles.mobileLogo}>
            <img src="/logo.svg" alt="Roncin" style={{ height: 32, width: 'auto' }} />
            <div>
              <span style={{ fontWeight: 800, fontSize: 18, letterSpacing: '0.08em', color: '#0f172a' }}>
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
                  prefix={<UserOutlined style={{ color: '#94a3b8', fontSize: 16, marginRight: 6 }} />}
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
                  prefix={<LockOutlined style={{ color: '#94a3b8', fontSize: 16, marginRight: 6 }} />}
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
              <Checkbox defaultChecked disabled={loading} style={{ fontSize: 13, color: '#64748b' }}>
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
                icon={loading ? <LoadingOutlined /> : <ArrowRightOutlined />}
                iconPosition="end"
              >
                {loading ? '登录中...' : '登录'}
              </Button>
            </Form.Item>
          </Form>

          {wecomEnabled && (
            <>
              <Divider plain className={styles.loginDivider}>
                或
              </Divider>
              <Button
                block
                size="large"
                icon={<WechatWorkOutlined />}
                loading={wecomLoading}
                disabled={loading || wecomLoading}
                className={styles.wecomButton}
                onClick={handleWeComLogin}
              >
                企业微信登录
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
