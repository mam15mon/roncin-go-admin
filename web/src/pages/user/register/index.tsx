import {
  CheckCircleOutlined,
  DingdingOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { Helmet, Link } from '@umijs/max';
import { Alert, App, Button, Steps } from 'antd';
import React, { useEffect, useState } from 'react';
import { authServiceGetDingTalkRegistrationConfig } from '@/services/roncin/authService';
import Settings from '../../../../config/defaultSettings';
import styles from '../login/index.module.less';

export default function Register() {
  const { message } = App.useApp();
  const [enabled, setEnabled] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    authServiceGetDingTalkRegistrationConfig({ skipErrorHandler: true })
      .then((response) => setEnabled(response.data?.enabled ?? false))
      .catch(() => setEnabled(false));
  }, []);

  const startRegistration = async () => {
    setLoading(true);
    try {
      const response = await authServiceGetDingTalkRegistrationConfig({
        skipErrorHandler: true,
      });
      if (!response.data?.enabled || !response.data.authorizeUrl) {
        message.warning('钉钉注册暂未启用');
        return;
      }
      sessionStorage.setItem('dingtalk_auth_mode', 'register');
      window.location.assign(response.data.authorizeUrl);
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : '钉钉注册启动失败',
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.loginContainer}>
      <Helmet>
        <title>注册 - {Settings.title}</title>
      </Helmet>
      <div className={styles.heroSection}>
        <div className={styles.glowTopLeft} />
        <div className={styles.glowBottomRight} />
        <div className={styles.brandLogoWrapper}>
          <img
            src="/images/logo-only.webp"
            alt="RONCIN"
            className={styles.brandLogoImg}
            onError={(event) => {
              event.currentTarget.src = '/logo.svg';
            }}
          />
        </div>
        <div
          style={{
            position: 'relative',
            zIndex: 10,
            maxWidth: 440,
            padding: 32,
            color: '#fff',
          }}
        >
          <SafetyCertificateOutlined
            style={{ fontSize: 52, color: '#69b1ff', marginBottom: 24 }}
          />
          <h1 style={{ margin: 0, color: '#fff', fontSize: 34 }}>
            企业成员注册
          </h1>
          <p
            style={{
              marginTop: 16,
              color: '#cbd5e1',
              fontSize: 16,
              lineHeight: 1.8,
            }}
          >
            先验证钉钉企业身份，再由管理员分配所属公司、部门和角色。注册阶段不需要设置用户名或密码。
          </p>
        </div>
      </div>
      <div className={styles.formSection}>
        <div className={styles.formCard}>
          <div className={styles.mobileLogo}>
            <img
              src="/logo.svg"
              alt="Roncin"
              style={{ height: 32, width: 'auto' }}
            />
            <span
              style={{ fontWeight: 800, fontSize: 18, letterSpacing: '0.08em' }}
            >
              RONCIN
            </span>
          </div>
          <h1 className={styles.headerTitle}>创建账号</h1>
          <Steps
            current={0}
            size="small"
            responsive={false}
            items={[
              { title: '企业验证', icon: <DingdingOutlined /> },
              { title: '管理员授权', icon: <TeamOutlined /> },
              { title: '完成', icon: <CheckCircleOutlined /> },
            ]}
            style={{ marginBottom: 28 }}
          />
          <Alert
            showIcon
            type="info"
            title="请使用本企业钉钉账号扫码"
            description="系统会核验钉钉返回的企业标识。其他企业的账号会被拒绝，且不会生成用户或注册记录。"
            style={{ marginBottom: 24 }}
          />
          <Button
            type="primary"
            size="large"
            block
            icon={<DingdingOutlined />}
            loading={loading}
            disabled={!enabled || loading}
            className={styles.submitButton}
            onClick={startRegistration}
          >
            {enabled ? '钉钉扫码验证并注册' : '钉钉注册暂未启用'}
          </Button>
          <div
            style={{
              marginTop: 20,
              textAlign: 'center',
              color: '#64748b',
              fontSize: 13,
            }}
          >
            已有账号？<Link to="/user/login">返回登录</Link>
          </div>
        </div>
      </div>
    </div>
  );
}
