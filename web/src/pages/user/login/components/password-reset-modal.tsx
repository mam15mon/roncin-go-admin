import { ModalForm, ProFormText } from '@ant-design/pro-components';
import { useMutation } from '@tanstack/react-query';
import { Alert, Button, Form, Input } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import {
  authServiceConfirmPasswordReset,
  authServiceRequestPasswordResetCode,
} from '@/services/roncin/authService';
import { isWriteResponse } from '@/utils/api';

interface ResetValues {
  mobile: string;
  code: string;
  newPassword: string;
  confirmPassword: string;
}

interface PasswordResetModalProps {
  initialMobile?: string;
  onClose: () => void;
  onSuccess: (mobile: string) => void;
}

// 与服务端接受的国内手机号表示一致；仅作表单格式校验，账号归属由服务端判断。
function normalizedMobile(value: string): string {
  return value
    .replace(/[\p{White_Space}-]/gu, '')
    .replace(/^(?:\+86|0086)/u, '');
}

function cooldownMilliseconds(
  serverTime?: string,
  availableAt?: string,
): number {
  if (!serverTime || !availableAt) return 0;
  const remaining = Date.parse(availableAt) - Date.parse(serverTime);
  return Number.isFinite(remaining) ? Math.max(0, remaining) : 0;
}

function errorCooldown(error: unknown): number {
  const envelope = error as {
    data?: { metadata?: { serverTime?: string; resendAvailableAt?: string } };
    response?: {
      data?: { metadata?: { serverTime?: string; resendAvailableAt?: string } };
    };
  };
  const metadata = envelope.data?.metadata ?? envelope.response?.data?.metadata;
  return cooldownMilliseconds(
    metadata?.serverTime,
    metadata?.resendAvailableAt,
  );
}

export function PasswordResetModal({
  initialMobile,
  onClose,
  onSuccess,
}: PasswordResetModalProps) {
  const [form] = Form.useForm<ResetValues>();
  const [challenge, setChallenge] = useState<API.PasswordResetChallenge>();
  const [feedback, setFeedback] = useState<{ text: string; error: boolean }>();
  const [cooldownEnd, setCooldownEnd] = useState(0);
  const [remaining, setRemaining] = useState(0);
  const requestSequence = useRef(0);
  const sending = useRef(false);
  const confirming = useRef(false);
  const active = useRef(true);
  const sendCode = useMutation({
    mutationFn: (mobile: string) =>
      authServiceRequestPasswordResetCode(
        { mobile },
        { skipErrorHandler: true },
      ),
    gcTime: 0,
    meta: { silent: true },
  });

  const confirmReset = useMutation({
    mutationFn: (body: API.ConfirmPasswordResetRequest) =>
      authServiceConfirmPasswordReset(body, { skipErrorHandler: true }),
    gcTime: 0,
    meta: { silent: true },
  });

  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
      requestSequence.current += 1;
    };
  }, []);

  useEffect(() => {
    if (!cooldownEnd) return;
    const update = () =>
      setRemaining(
        Math.max(0, Math.ceil((cooldownEnd - performance.now()) / 1000)),
      );
    update();
    const timer = window.setInterval(update, 250);
    return () => window.clearInterval(timer);
  }, [cooldownEnd]);

  const startCooldown = (milliseconds: number) => {
    setRemaining(Math.ceil(milliseconds / 1000));
    setCooldownEnd(milliseconds > 0 ? performance.now() + milliseconds : 0);
  };

  const clearChallenge = () => {
    requestSequence.current += 1;
    setChallenge(undefined);
    setFeedback(undefined);
    startCooldown(0);
    form.setFieldsValue({ code: '', newPassword: '', confirmPassword: '' });
  };

  const close = () => {
    if (confirming.current || !active.current) return;
    active.current = false;
    clearChallenge();
    form.resetFields();
    onClose();
  };

  const requestCode = async () => {
    if (sending.current || confirming.current || remaining > 0) return;
    sending.current = true;
    let values: Pick<ResetValues, 'mobile'>;
    try {
      values = await form.validateFields(['mobile']);
    } catch {
      sending.current = false;
      return;
    }
    if (!active.current) {
      sending.current = false;
      return;
    }
    const mobile = values.mobile;
    const sequence = ++requestSequence.current;
    const current = () =>
      active.current &&
      sequence === requestSequence.current &&
      form.getFieldValue('mobile') === mobile;
    // 重发会使原挑战失效；结果未知时也不继续使用旧挑战。
    setChallenge(undefined);
    form.setFieldValue('code', '');
    setFeedback(undefined);
    try {
      const response = await sendCode.mutateAsync(mobile);
      if (!current()) return;
      if (!isWriteResponse(response) || !response?.data?.challengeToken) {
        setFeedback({
          error: true,
          text: '未能获取验证码，请稍后重试或使用钉钉扫码',
        });
        return;
      }
      setChallenge(response.data);
      startCooldown(
        cooldownMilliseconds(
          response.data.serverTime,
          response.data.resendAvailableAt,
        ),
      );
      setFeedback({
        error: false,
        text: '若该手机号对应已开通的钉钉账号，验证码将发送到本人钉钉，请前往钉钉查看。',
      });
    } catch (error) {
      if (!current()) return;
      startCooldown(errorCooldown(error));
      setFeedback({
        error: true,
        text: getRequestErrorMessage(
          error,
          '验证码发送结果暂时无法确认，请稍后重试或使用钉钉扫码',
        ),
      });
    } finally {
      sending.current = false;
      sendCode.reset();
    }
  };

  return (
    <ModalForm<ResetValues>
      title="设置 / 忘记密码"
      open
      form={form}
      initialValues={{ mobile: initialMobile ?? '' }}
      autoComplete="off"
      disabled={confirmReset.isPending}
      submitTimeout={60000}
      modalProps={{ destroyOnHidden: true, width: 480, onCancel: close }}
      onOpenChange={(open) => {
        if (!open) close();
      }}
      submitter={{
        searchConfig: { submitText: '设置密码并返回登录', resetText: '取消' },
        submitButtonProps: {
          'aria-label': '设置密码并返回登录',
          disabled: !challenge || sendCode.isPending,
        },
      }}
      onValuesChange={(changed) => {
        if ('mobile' in changed) clearChallenge();
      }}
      onFinish={async (values) => {
        if (!challenge?.challengeToken || sending.current || confirming.current)
          return false;
        confirming.current = true;
        const sequence = requestSequence.current;
        const current = () =>
          active.current &&
          sequence === requestSequence.current &&
          form.getFieldValue('mobile') === values.mobile;
        try {
          const response = await confirmReset.mutateAsync({
            challengeToken: challenge.challengeToken,
            code: values.code,
            newPassword: values.newPassword,
          });
          if (!current() || !isWriteResponse(response)) return false;
          onSuccess(values.mobile);
          return true;
        } catch (error) {
          if (!current()) return false;
          setFeedback({
            error: true,
            text: getRequestErrorMessage(error, '设置密码失败，请稍后重试'),
          });
          return false;
        } finally {
          confirming.current = false;
          confirmReset.reset();
        }
      }}
    >
      <Alert
        type="info"
        showIcon
        title="首次设置与找回使用同一流程"
        description="验证码发送到本人钉钉，5 分钟内有效，累计输错 5 次作废；重发后旧码失效。设置成功后，所有设备的登录会话将退出。未绑定钉钉的本地账号请联系管理员。"
        style={{ marginBottom: 16 }}
      />
      <ProFormText
        name="mobile"
        label="钉钉手机号"
        placeholder="请输入钉钉通讯录中的手机号"
        fieldProps={{ autoComplete: 'tel', inputMode: 'tel', maxLength: 32 }}
        rules={[
          { required: true, message: '请输入钉钉手机号' },
          {
            validator: (_, value: string) =>
              !value || /^1[3-9]\d{9}$/u.test(normalizedMobile(value))
                ? Promise.resolve()
                : Promise.reject(new Error('请输入有效的国内手机号')),
          },
        ]}
      />
      <Form.Item
        label="钉钉验证码"
        required
        extra="两次申请至少间隔 60 秒，刷新页面后仍受服务端限制。"
      >
        <div style={{ display: 'flex', gap: 8 }}>
          <Form.Item
            name="code"
            noStyle
            rules={[
              {
                required: true,
                pattern: /^\d{6}$/u,
                message: '请输入 6 位数字验证码',
              },
            ]}
          >
            <Input
              aria-label="钉钉验证码"
              placeholder="6 位数字验证码"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              style={{ flex: 1, minWidth: 0 }}
            />
          </Form.Item>
          <Button
            aria-label={remaining > 0 ? `${remaining} 秒后重发` : '发送验证码'}
            onClick={() => void requestCode()}
            loading={sendCode.isPending}
            disabled={remaining > 0}
          >
            {remaining > 0 ? `${remaining} 秒后重发` : '发送验证码'}
          </Button>
        </div>
      </Form.Item>
      <ProFormText.Password
        name="newPassword"
        label="新密码"
        placeholder="请输入至少 12 位新密码"
        fieldProps={{ autoComplete: 'new-password' }}
        rules={[
          { required: true, message: '请输入新密码' },
          {
            validator: (_, value: string) =>
              !value || new TextEncoder().encode(value.trim()).length >= 12
                ? Promise.resolve()
                : Promise.reject(new Error('密码至少 12 位')),
          },
        ]}
      />
      <ProFormText.Password
        name="confirmPassword"
        label="确认新密码"
        placeholder="请再次输入新密码"
        fieldProps={{ autoComplete: 'new-password' }}
        dependencies={['newPassword']}
        rules={[
          { required: true, message: '请再次输入新密码' },
          {
            validator: (_, value: string) =>
              !value || value === form.getFieldValue('newPassword')
                ? Promise.resolve()
                : Promise.reject(new Error('两次输入的密码不一致')),
          },
        ]}
      />
      {feedback && (
        <div role={feedback.error ? 'alert' : 'status'}>
          <Alert
            type={feedback.error ? 'error' : 'info'}
            showIcon
            title={feedback.text}
          />
        </div>
      )}
    </ModalForm>
  );
}
