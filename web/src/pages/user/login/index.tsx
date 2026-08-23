import {
  ArrowRightOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  LoadingOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Helmet, useModel } from '@umijs/max';
import { App, Button, Checkbox, Form, Input } from 'antd';
import React, { startTransition, useState } from 'react';
import { authServiceLogin } from '@/services/roncin/authService';
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

  return (
    <div className={styles.loginContainer}>
      <Helmet>
        <title>登录 - {Settings.title}</title>
      </Helmet>

      {/* ── 左侧：深色科技交互区 (Decorative & Animated Hero Section) ── */}
      <div className={styles.heroSection}>
        {/* 点阵网格背景 */}
        <div className={styles.dotGridBackground} />

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
              // 降级使用 SVG
              (e.currentTarget as HTMLImageElement).src = '/logo.svg';
            }}
          />
          <div>
            <div className={styles.brandTitle}>RONCIN</div>
            <div className={styles.brandSubtitle}>LOGISTICS INTELLIGENCE</div>
          </div>
        </div>

        {/* 核心互动动画角色 */}
        <div className={styles.charactersWrapper}>
          <AnimatedCharacters
            isTyping={isTyping}
            showPassword={showPassword}
            passwordLength={passwordValue.length}
          />
        </div>

        {/* 底部功能标识 */}
        <div className={styles.heroFooterBadge}>
          <SafetyCertificateOutlined style={{ color: '#38bdf8' }} />
          <span>多级组织隔离 · 角色权限边界保护</span>
        </div>
      </div>

      {/* ── 右侧：现代化认证表单区 (Auth Form Section) ── */}
      <div className={styles.formSection}>
        <div className={styles.formCard}>
          {/* 移动端 Logo 展示 */}
          <div className={styles.mobileLogo}>
            <img src="/logo.svg" alt="Roncin" style={{ height: 32, width: 'auto' }} />
            <div>
              <span style={{ fontWeight: 800, fontSize: 18, letterSpacing: '0.08em', color: '#0f172a' }}>
                RONCIN
              </span>
              <span style={{ fontSize: 10, color: '#64748b', display: 'block' }}>货代智能协同管理平台</span>
            </div>
          </div>

          {/* 表单头部 */}
          <div>
            <h1 className={styles.headerTitle}>欢迎回来 👋</h1>
            <p className={styles.headerSubtitle}>国际货运代理与供应链协同管理平台</p>
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
              rules={[{ required: true, message: '请输入用户名（登录账号）' }]}
              style={{ marginBottom: 20 }}
            >
              <div>
                <label className={styles.inputLabel} htmlFor="login_username">
                  访问账号
                </label>
                <Input
                  id="login_username"
                  size="large"
                  placeholder="用户名 / 邮箱地址"
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
              rules={[{ required: true, message: '请输入登录密码' }]}
              style={{ marginBottom: 20 }}
            >
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <label className={styles.inputLabel} htmlFor="login_password">
                    验证密码
                  </label>
                </div>
                <Input.Password
                  id="login_password"
                  size="large"
                  placeholder="••••••••"
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
              <span style={{ fontSize: 12, color: '#1677ff', cursor: 'pointer', fontWeight: 500 }}>
                安全验证模式
              </span>
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
                {loading ? '授权验证中...' : '授权并登录'}
              </Button>
            </Form.Item>
          </Form>

          {/* 底部安全与协议区 */}
          <div className={styles.footerWrapper}>
            <div className={styles.footerSecurityBadge}>
              <SafetyCertificateOutlined style={{ color: '#1677ff', fontSize: 13 }} />
              <span>受限访问区域 · 仅限授权人员</span>
            </div>
            <div className={styles.footerLinks}>
              <span>服务协议</span>
              <span>·</span>
              <span>隐私条款</span>
              <span>·</span>
              <span>运维支持</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
