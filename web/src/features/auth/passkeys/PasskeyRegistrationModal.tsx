import { ModalForm, ProFormText } from '@ant-design/pro-components';
import {
  type RegistrationResponseJSON,
  startRegistration,
  WebAuthnAbortService,
} from '@simplewebauthn/browser';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Form, Space } from 'antd';
import { useEffect, useRef } from 'react';
import { passkeyServiceFinishPasskeyRegistration } from '@/services/roncin/passkeyService';
import { isWriteResponse } from '@/utils/api';
import {
  PasskeyFlowError,
  passkeyConfigOptions,
  passkeyErrorMessage,
  passkeyKeys,
  passkeyRegistrationOptions,
  passkeyUnavailableReason,
  registrationOptions,
} from './api';

type Props = {
  open: boolean;
  userId: string;
  onRegistered: () => void;
  onSkip: () => void;
  skipText?: string;
};

export function PasskeyRegistrationModal({
  open,
  userId,
  onRegistered,
  onSkip,
  skipText = '暂不添加，继续登录',
}: Props) {
  const completed = useRef(false);
  const openRef = useRef(open);
  openRef.current = open;
  const mounted = useRef(true);
  const nativePending = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      if (nativePending.current) WebAuthnAbortService.cancelCeremony();
    };
  }, []);
  const nativeCredential = useRef<Promise<RegistrationResponseJSON> | null>(
    null,
  );
  const [form] = Form.useForm<{ name: string }>();
  const nameValue = Form.useWatch('name', form);
  const queryClient = useQueryClient();
  const config = useQuery({ ...passkeyConfigOptions, enabled: open });
  const reason = config.isError
    ? '无法获取通行密钥配置，请稍后重试'
    : passkeyUnavailableReason(config.data);
  const canUse = !reason;
  const options = useQuery({
    ...passkeyRegistrationOptions(userId),
    enabled: open && canUse,
  });
  const registration = useMutation({
    mutationFn: async ({
      name,
      credentialPromise,
    }: {
      name: string;
      credentialPromise: Promise<RegistrationResponseJSON>;
    }) => {
      let credential: RegistrationResponseJSON;
      try {
        credential = await credentialPromise;
      } finally {
        nativePending.current = false;
      }
      if (!mounted.current || !openRef.current)
        throw new PasskeyFlowError('注册页面已关闭，请重新添加');
      const response = await passkeyServiceFinishPasskeyRegistration(
        {
          credentialJson: JSON.stringify(credential),
          name,
        },
        { skipErrorHandler: true },
      );
      if (!mounted.current || !openRef.current)
        throw new PasskeyFlowError('注册页面已关闭，请重新添加');
      if (!isWriteResponse(response, true))
        throw new PasskeyFlowError('通行密钥未保存成功，请重试');
      return response;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: passkeyKeys.credentials(userId),
      });
    },
    onError: () => {
      // 已提交的挑战只能使用一次，用户下次确认前先准备新的挑战。
      if (canUse && mounted.current)
        void options.refetch({ cancelRefetch: false });
    },
    meta: { silent: true },
  });
  const startNativeRegistration = () => {
    if (
      nativeCredential.current ||
      !canUse ||
      options.isFetching ||
      registration.isPending ||
      !form.getFieldValue('name')?.trim()
    )
      return;
    try {
      // 不能等异步表单校验后才启动 Safari 的原生密钥界面。
      nativePending.current = true;
      nativeCredential.current = startRegistration({
        optionsJSON: registrationOptions(options.data, config.data?.rpId ?? ''),
      });
    } catch (error) {
      nativePending.current = false;
      nativeCredential.current = Promise.reject(error);
    }
    // 表单校验稍后消费同一个 Promise，先观察拒绝以避免中间阶段未处理错误。
    void nativeCredential.current.catch(() => undefined);
  };

  const error = registration.error ?? options.error;
  return (
    <ModalForm<{ name: string }>
      title="开启免密登录（通行密钥）"
      form={form}
      open={open}
      width={420}
      initialValues={{ name: '我的通行密钥' }}
      onOpenChange={(next) => {
        if (!next && !completed.current && !registration.isPending) onSkip();
      }}
      modalProps={{
        destroyOnHidden: true,
        maskClosable: false,
        closable: !registration.isPending,
      }}
      submitter={{
        searchConfig: { submitText: '添加通行密钥', resetText: skipText },
        submitButtonProps: {
          disabled:
            !canUse ||
            options.isFetching ||
            !options.data ||
            !nameValue?.trim(),
          onClick: startNativeRegistration,
        },
        resetButtonProps: { disabled: registration.isPending, onClick: onSkip },
      }}
      onFinish={async ({ name }) => {
        try {
          const credentialPromise = nativeCredential.current;
          nativeCredential.current = null;
          if (!credentialPromise) return false;
          await registration.mutateAsync({
            name: name.trim(),
            credentialPromise,
          });
          if (!mounted.current || !openRef.current) return false;
          completed.current = true;
          onRegistered();
          return true;
        } catch {
          return false;
        }
      }}
    >
      <Space
        orientation="vertical"
        size={12}
        style={{ width: '100%', marginBottom: 16 }}
      >
        <div>添加后可使用指纹、面容、屏锁或安全密钥登录 Roncin。</div>
        {reason && <Alert type="info" showIcon title={reason} />}
        {(config.isError || options.isError) && (
          <Button
            disabled={
              registration.isPending || config.isFetching || options.isFetching
            }
            onClick={() => {
              if (config.isError) void config.refetch({ cancelRefetch: false });
              else void options.refetch({ cancelRefetch: false });
            }}
          >
            重新准备通行密钥
          </Button>
        )}
        {error && (
          <Alert
            type="error"
            showIcon
            title={passkeyErrorMessage(error, '通行密钥添加失败，请重试')}
          />
        )}
      </Space>
      <ProFormText
        name="name"
        label="通行密钥名称"
        placeholder="例如：办公电脑"
        fieldProps={{
          maxLength: 100,
          onKeyDown: (event) => {
            if (event.key === 'Enter') startNativeRegistration();
          },
        }}
        rules={[
          { required: true, whitespace: true, message: '请输入通行密钥名称' },
        ]}
      />
    </ModalForm>
  );
}
