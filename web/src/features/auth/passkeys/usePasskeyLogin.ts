import {
  type AuthenticationResponseJSON,
  startAuthentication,
  WebAuthnAbortService,
} from '@simplewebauthn/browser';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { passkeyServiceFinishPasskeyLogin } from '@/services/roncin/passkeyService';
import {
  authenticationOptions,
  PasskeyFlowError,
  passkeyConfigOptions,
  passkeyErrorMessage,
  passkeyLoginOptions,
  passkeyUnavailableReason,
} from './api';
import { useEnrollmentPreference } from './preferences';

export function usePasskeyLogin(rememberMe: boolean) {
  const mounted = useRef(true);
  const nativePending = useRef(false);
  const loginInFlight = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      if (nativePending.current) WebAuthnAbortService.cancelCeremony();
    };
  }, []);
  const config = useQuery(passkeyConfigOptions);
  const reason = config.isError
    ? '无法获取通行密钥配置，请稍后重试'
    : passkeyUnavailableReason(config.data);
  const canUse = !reason;
  const preference = useEnrollmentPreference();
  const options = useQuery({ ...passkeyLoginOptions, enabled: canUse });
  const mutation = useMutation({
    mutationFn: async (
      credentialPromise: Promise<AuthenticationResponseJSON>,
    ) => {
      let credential: AuthenticationResponseJSON;
      try {
        credential = await credentialPromise;
      } finally {
        nativePending.current = false;
      }
      if (!mounted.current)
        throw new PasskeyFlowError('登录页面已关闭，请重新登录');
      const response = await passkeyServiceFinishPasskeyLogin(
        {
          credentialJson: JSON.stringify(credential),
          rememberMe,
        },
        { skipErrorHandler: true },
      );
      if (!mounted.current)
        throw new PasskeyFlowError('登录页面已关闭，请重新登录');
      if (!response?.data?.id || response.success === false)
        throw new PasskeyFlowError('通行密钥登录未返回有效结果');
      preference.setEnrollAfterLogin(false);
      return response;
    },
    onError: () => {
      // Finish 会消费挑战；重试先准备新挑战，不再次自动调用浏览器认证。
      if (canUse && mounted.current)
        void options.refetch({ cancelRefetch: false });
    },
    meta: { silent: true },
  });
  return {
    canUse,
    reason,
    ...preference,
    login: () => {
      if (loginInFlight.current)
        return Promise.reject(
          new PasskeyFlowError('正在验证通行密钥，请勿重复操作'),
        );
      loginInFlight.current = true;
      let credentialPromise: Promise<AuthenticationResponseJSON>;
      try {
        if (!canUse) throw new PasskeyFlowError(reason);
        if (options.isFetching)
          throw new PasskeyFlowError('正在准备通行密钥验证，请稍后重试');
        // 先在原生点击栈启动浏览器验证，再交由 mutation 等待及提交。
        nativePending.current = true;
        credentialPromise = startAuthentication({
          optionsJSON: authenticationOptions(
            options.data,
            config.data?.rpId ?? '',
          ),
        });
      } catch (error) {
        nativePending.current = false;
        credentialPromise = Promise.reject(error);
      }
      return mutation.mutateAsync(credentialPromise).finally(() => {
        loginInFlight.current = false;
      });
    },
    prepareAgain: async () => {
      if (loginInFlight.current) return;
      const refreshedConfig = await config.refetch({ cancelRefetch: false });
      if (!passkeyUnavailableReason(refreshedConfig.data)) {
        await options.refetch({ cancelRefetch: false });
      }
    },
    isPending: mutation.isPending,
    isPreparing: options.isFetching,
    error: config.isError
      ? '无法获取通行密钥配置，请重新准备后重试'
      : mutation.error
        ? passkeyErrorMessage(mutation.error, '通行密钥登录失败，请重试')
        : options.isError
          ? passkeyErrorMessage(options.error, '通行密钥验证准备失败，请重试')
          : undefined,
  };
}
