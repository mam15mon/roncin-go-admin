import { ModalForm, ProFormText } from '@ant-design/pro-components';
import { Alert, App, Typography } from 'antd';
import React from 'react';
import { adminServiceResetUserPassword } from '@/services/roncin/adminService';

const { Text } = Typography;

interface ResetPasswordModalProps {
  user?: API.AdminUser;
  onClose: () => void;
  onReload: () => void;
}

export default function ResetPasswordModal({
  user,
  onClose,
  onReload,
}: ResetPasswordModalProps) {
  const { message } = App.useApp();

  return (
    <ModalForm<{ username?: string; password?: string }>
      title={
        user?.hasPassword
          ? `重置登录密码：${user?.displayName || user?.username || ''}`
          : `设置登录账密：${user?.displayName || user?.dingtalkName || user?.wecomName || ''}`
      }
      open={Boolean(user)}
      initialValues={user?.username ? { username: user.username } : undefined}
      modalProps={{
        destroyOnClose: true,
        width: 500,
        onCancel: onClose,
      }}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      onFinish={async (values) => {
        if (!user?.id) return false;
        await adminServiceResetUserPassword(
          { id: user.id },
          {
            id: user.id,
            password: values.password ?? '',
            username: values.username?.trim() || undefined,
          },
        );
        message.success(
          user.hasPassword
            ? '密码已重置，该用户现有在线会话已全部失效'
            : '登录账密已设置，该用户现可使用用户名与密码登录',
        );
        onClose();
        onReload();
        return true;
      }}
    >
      <Alert
        showIcon
        type={user?.hasPassword ? 'warning' : 'info'}
        title={user?.hasPassword ? '重置密码安全须知' : '设置登录账密'}
        description={
          user?.hasPassword
            ? '密码重置成功后，该用户的旧密码将立即失效，当前所有在线登录会话将被强制退出。'
            : '为该第三方账号设置专属登录用户名与密码后，用户可在未携带手机或扫码不便时使用账密备用登录。'
        }
        style={{ marginBottom: 16 }}
      />
      {(!user?.username || !user.hasPassword) && (
        <ProFormText
          name="username"
          label="登录用户名"
          placeholder="例如：zhangsan 或 logistics_op"
          fieldProps={{ maxLength: 64 }}
          rules={[
            { required: !user?.username, message: '请输入登录用户名' },
            { min: 3, max: 64, message: '用户名长度需在 3 至 64 个字符之间' },
            {
              pattern: /^[a-z0-9_.-]+$/,
              message: '用户名仅支持小写字母、数字、点号、下划线及连字符',
            },
          ]}
        />
      )}
      {user?.username && user.hasPassword && (
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            目标账号：
          </Text>
          <Text strong style={{ marginLeft: 4, fontFamily: 'monospace' }}>
            {user?.username}
          </Text>
        </div>
      )}
      <ProFormText
        name="password"
        label={user?.hasPassword ? '新登录密码' : '初始登录密码'}
        placeholder="请输入至少 12 位新密码"
        fieldProps={{ type: 'password' }}
        extra="密码至少 12 位，设置后请及时通知用户"
        rules={[{ required: true, min: 12, message: '密码至少 12 位' }]}
      />
    </ModalForm>
  );
}
