import { useMutation, useQuery } from '@tanstack/react-query';
import { App, Button, Result, Spin } from 'antd';
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import { useInitialState } from '@/app/AppProvider';
import { DingTalkLoginStatus } from '@/enums.generated';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import {
  authServiceDingTalkEmbeddedLogin,
  authServiceGetDingTalkEmbeddedConfig,
} from '@/services/roncin/authService';
import { requestEmbeddedAuthCode } from '../dingtalkEmbeddedSdk';
import { safeLoginRedirect } from '../redirect';
import {
  OrganizationPicker,
  resolveLoginOrganizationOptions,
} from './organization-picker';

interface Props {
  onAccountLogin: () => void;
  /** 会写会话 Cookie 时及组织选择期间，禁止开始其他登录。 */
  onSwitchLockChange?: (locked: boolean) => void;
}

/** 尝试 Promise 与订阅分开：StrictMode 重订阅，不重复消费一次性授权码。 */
export function DingTalkEmbeddedLogin({
  onAccountLogin,
  onSwitchLockChange,
}: Props) {
  const { initialState, setInitialState } = useInitialState();
  const existingUser = useRef(initialState?.currentUser);
  const { message } = App.useApp();
  const [attemptNumber, setAttemptNumber] = useState(0);
  const [result, setResult] = useState<API.DingTalkLoginResult>();
  const [failure, setFailure] = useState<string>();
  const live = useRef(false);
  const switchLockedRef = useRef(false);
  const [switchLocked, setSwitchLocked] = useState(false);
  const attempt = useRef<{
    number: number;
    promise: Promise<API.DingTalkLoginResult | undefined>;
  }>(undefined);
  const target = useRef(
    safeLoginRedirect(
      new URL(window.location.href).searchParams.get('redirect'),
    ),
  );
  const { refetch } = useQuery({
    queryKey: ['auth', 'dingtalk-embedded-config'],
    queryFn: () =>
      authServiceGetDingTalkEmbeddedConfig({ skipErrorHandler: true }),
    enabled: false,
    gcTime: 0,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retryOnMount: false,
    meta: { silent: true },
  });
  const { mutateAsync } = useMutation({
    mutationFn: (credentials: API.DingTalkEmbeddedLoginRequest) =>
      authServiceDingTalkEmbeddedLogin(credentials, { skipErrorHandler: true }),
    meta: { silent: true },
  });
  const callbacks = useRef({ setInitialState, message, onSwitchLockChange });
  callbacks.current = { setInitialState, message, onSwitchLockChange };
  const changeSwitchLock = useCallback((locked: boolean) => {
    // 同步门禁先于 React 提交，堵住请求发出与按钮禁用之间的同 tick 窗口。
    switchLockedRef.current = locked;
    callbacks.current.onSwitchLockChange?.(locked);
    setSwitchLocked(locked);
  }, []);

  useLayoutEffect(() => {
    live.current = true;
    return () => {
      live.current = false;
      switchLockedRef.current = false;
      callbacks.current.onSwitchLockChange?.(false);
    };
  }, []);

  useEffect(() => {
    let subscribed = true;
    const isActive = () =>
      live.current && attempt.current?.number === attemptNumber;
    if (attempt.current?.number !== attemptNumber) {
      const promise = (async () => {
        if (existingUser.current?.id)
          return {
            status: DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED,
            currentUser: existingUser.current,
          };
        // 不取消会写 Cookie 的配置请求；显式重试与仍在途的同 key 查询串行。
        const configResponse = await refetch({ cancelRefetch: false });
        if (!isActive()) return;
        if (configResponse.error) throw configResponse.error;
        const config = configResponse.data?.data;
        if (!config) throw new Error('钉钉免登配置加载失败，请重试');
        if (config.enabled !== true)
          throw new Error('钉钉免登尚未启用，请联系管理员或使用账号登录');
        const expiresAt = Date.parse(config.expiresAt ?? '');
        if (!config.corpId || !config.state || !Number.isFinite(expiresAt))
          throw new Error('钉钉免登配置不完整，请联系管理员');
        if (expiresAt <= Date.now())
          throw new Error('钉钉免登配置已过期，请重试');
        const authCode = await requestEmbeddedAuthCode(config.corpId, isActive);
        if (!isActive()) return;
        if (expiresAt <= Date.now())
          throw new Error('钉钉免登配置已过期，请重试');
        changeSwitchLock(true);
        let authenticated = false;
        try {
          const response = await mutateAsync({ authCode, state: config.state });
          const data = response?.data;
          if (
            !data ||
            ![
              DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED,
              DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_NOT_SYNCED,
              DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_PENDING_ASSIGNMENT,
              DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_DISABLED,
            ].some((status) => status === data.status) ||
            (data.status ===
              DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED &&
              !data.currentUser?.id)
          )
            throw new Error('钉钉认证未返回有效结果，请重试');
          authenticated =
            data.status ===
            DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED;
          return data;
        } finally {
          // 已建立会话继续锁定直到导航或明确选完组织，失败响应落定才恢复切换。
          if (isActive() && !authenticated) changeSwitchLock(false);
        }
      })();
      attempt.current = { number: attemptNumber, promise };
    }
    void attempt.current.promise.then(
      (data) => {
        if (!subscribed || !isActive() || !data) return;
        setResult(data);
        if (
          data.status ===
            DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED &&
          data.currentUser
        ) {
          changeSwitchLock(true);
          callbacks.current.setInitialState((current) => ({
            ...current,
            currentUser: data.currentUser,
          }));
          if (
            resolveLoginOrganizationOptions(undefined, data.currentUser)
              .length <= 1
          ) {
            callbacks.current.message.success('钉钉登录成功');
            // 认证后整页重建工作台，沿用新会话及组织权限。
            window.location.replace(target.current);
          }
        }
      },
      (error: unknown) => {
        if (subscribed && isActive())
          setFailure(
            getRequestErrorMessage(
              error,
              error instanceof Error && error.message.startsWith('钉钉')
                ? error.message
                : '钉钉免登失败，请重试',
            ),
          );
      },
    );
    return () => {
      subscribed = false;
    };
  }, [attemptNumber, refetch, mutateAsync, changeSwitchLock]);

  const options = resolveLoginOrganizationOptions(
    undefined,
    result?.status === DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED
      ? result.currentUser
      : undefined,
  );
  if (options.length > 1) {
    return (
      <OrganizationPicker
        options={options}
        currentOrganizationId={result?.currentUser?.currentOrganization?.id}
        onEnter={() => {
          if (!live.current || attempt.current?.number !== attemptNumber)
            return;
          message.success('钉钉登录成功');
          window.location.replace(target.current);
        }}
      />
    );
  }
  const blocked =
    result &&
    result.status !== DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED;
  const title =
    result?.status === DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_NOT_SYNCED
      ? '人员尚未同步'
      : result?.status ===
          DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_PENDING_ASSIGNMENT
        ? '尚未分配组织'
        : result?.status === DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_DISABLED
          ? '账号已停用'
          : '钉钉免登未完成';
  return (
    <section aria-label="钉钉客户端免登">
      {failure || blocked ? (
        <Result
          status={failure ? 'error' : 'info'}
          title={title}
          subTitle={
            failure ||
            result?.message ||
            '请联系管理员同步人员并分配组织和角色后重试。'
          }
          extra={
            <Button
              onClick={() => {
                attempt.current = undefined;
                setFailure(undefined);
                setResult(undefined);
                setAttemptNumber((current) => current + 1);
              }}
            >
              重试免登
            </Button>
          }
        />
      ) : (
        <div role="status" style={{ padding: '32px 0', textAlign: 'center' }}>
          <Spin />
          <p>正在使用当前钉钉身份登录…</p>
        </div>
      )}
      <Button
        block
        disabled={switchLocked}
        onClick={() => {
          if (switchLockedRef.current) return;
          live.current = false;
          onAccountLogin();
        }}
      >
        使用账号登录
      </Button>
    </section>
  );
}
