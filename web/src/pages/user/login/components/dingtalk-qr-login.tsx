import { ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { QRCode, Spin } from 'antd';
import {
  type ReactNode,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from 'react';
import {
  CALLBACK_PATH,
  dingTalkLoginConfigOptions,
  type QRConfig,
} from '@/app/dingtalkLoginConfig';
import { isRecord } from '@/app/dingtalkLoginProtocol';
import { DingTalkQRCodeStatus } from '@/enums.generated';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import { authServicePollDingTalkQrCode } from '@/services/roncin/authService';
import styles from '../index.module.less';

type Props = {
  active?: boolean;
  onAuthenticated: (callbackPath: string) => void;
};

const {
  DING_TALK_QR_CODE_STATUS_WAITING: WAITING,
  DING_TALK_QR_CODE_STATUS_SCANNED: SCANNED,
  DING_TALK_QR_CODE_STATUS_CONFIRMED: CONFIRMED,
  DING_TALK_QR_CODE_STATUS_EXPIRED: EXPIRED,
} = DingTalkQRCodeStatus;

type PollResult =
  | { status: typeof WAITING | typeof SCANNED | typeof EXPIRED }
  | { status: typeof CONFIRMED; authCode: string };

function parsePollResponse(response: unknown): PollResult {
  if (
    !isRecord(response) ||
    response.success === false ||
    !isRecord(response.data)
  ) {
    throw new Error('钉钉扫码状态获取失败，请刷新重试');
  }
  const data = response.data;
  if (
    data.status === WAITING ||
    data.status === SCANNED ||
    data.status === EXPIRED
  ) {
    return { status: data.status };
  }
  if (
    data.status === CONFIRMED &&
    typeof data.authCode === 'string' &&
    data.authCode.trim()
  ) {
    return { status: CONFIRMED, authCode: data.authCode };
  }
  throw new Error('钉钉扫码状态不正确，请刷新重试');
}

function QrRefreshCard({
  children,
  onRefresh,
  disabled = false,
  title,
}: {
  children: ReactNode;
  onRefresh: () => void;
  disabled?: boolean;
  title?: string;
}) {
  return (
    <button
      type="button"
      className={styles.qrCard}
      aria-label="刷新二维码"
      onClick={onRefresh}
      disabled={disabled}
      title={title}
    >
      {children}
    </button>
  );
}

function QrRefreshNotice({ title }: { title: string }) {
  return (
    <div className={styles.qrStatus}>
      <ReloadOutlined className={styles.qrStatusIcon} />
      <span>{title}</span>
      <span className={styles.qrRetry}>点击刷新</span>
    </div>
  );
}

function subscribeVisibility(listener: () => void) {
  document.addEventListener('visibilitychange', listener);
  return () => document.removeEventListener('visibilitychange', listener);
}

function DingTalkCode({
  config,
  active,
  onAuthenticated,
  onRefresh,
  onUnavailable,
  retireCode,
}: Props & {
  config: QRConfig;
  onRefresh: () => void;
  onUnavailable: (state: string) => void;
  retireCode: React.RefObject<(() => void) | null>;
}) {
  const queryClient = useQueryClient();
  const activeRef = useRef(active);
  activeRef.current = active;
  const retired = useRef(false);
  const handled = useRef(false);
  const unavailableRef = useRef(onUnavailable);
  unavailableRef.current = onUnavailable;
  const authenticatedRef = useRef(onAuthenticated);
  authenticatedRef.current = onAuthenticated;
  const [status, setStatus] = useState<
    'ready' | 'expired' | 'authenticating' | 'failed'
  >('ready');
  const queryKey = ['auth', 'dingtalk-qr-poll', config.state] as const;
  const poll = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      const result = parsePollResponse(
        await authServicePollDingTalkQrCode(
          { state: config.state },
          { signal, skipErrorHandler: true },
        ),
      );
      // 已切换登录方式或隐藏时到达的授权码不得在返回后补做认证。
      if (
        result.status === CONFIRMED &&
        (!activeRef.current || signal.aborted || retired.current)
      ) {
        retired.current = true;
        setStatus('expired');
        unavailableRef.current(config.state);
        void queryClient.cancelQueries({
          queryKey: ['auth', 'dingtalk-qr-poll', config.state],
          exact: true,
        });
      }
      return result;
    },
    enabled:
      active &&
      status === 'ready' &&
      !retired.current &&
      config.expiresAt > Date.now(),
    refetchInterval: (query) =>
      query.state.status === 'error' ||
      query.state.data?.status === CONFIRMED ||
      query.state.data?.status === EXPIRED
        ? false
        : 1000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retryOnMount: false,
    staleTime: 0,
    gcTime: 0,
    meta: { silent: true },
  });

  useLayoutEffect(() => {
    const retire = () => {
      retired.current = true;
      void queryClient.cancelQueries({
        queryKey: ['auth', 'dingtalk-qr-poll', config.state],
        exact: true,
      });
    };
    retireCode.current = retire;
    return () => {
      retireCode.current = null;
    };
  }, [config.state, queryClient, retireCode]);

  useLayoutEffect(() => {
    if (!active) {
      void queryClient.cancelQueries({
        queryKey: ['auth', 'dingtalk-qr-poll', config.state],
        exact: true,
      });
    }
  }, [active, config.state, queryClient]);

  useEffect(() => {
    const expire = () => {
      if (retired.current || handled.current || Date.now() < config.expiresAt)
        return;
      retired.current = true;
      setStatus('expired');
      unavailableRef.current(config.state);
    };
    const timeout = window.setTimeout(
      expire,
      Math.max(0, config.expiresAt - Date.now()),
    );
    window.addEventListener('focus', expire);
    document.addEventListener('visibilitychange', expire);
    expire();
    return () => {
      window.clearTimeout(timeout);
      window.removeEventListener('focus', expire);
      document.removeEventListener('visibilitychange', expire);
    };
  }, [config]);

  useEffect(() => {
    if (retired.current || handled.current) return;
    if (poll.isError) {
      retired.current = true;
      setStatus('failed');
      unavailableRef.current(config.state);
      return;
    }
    const result = poll.data;
    if (result?.status === EXPIRED) {
      retired.current = true;
      setStatus('expired');
      unavailableRef.current(config.state);
    } else if (
      result?.status === CONFIRMED &&
      activeRef.current &&
      Date.now() < config.expiresAt
    ) {
      handled.current = true;
      retired.current = true;
      unavailableRef.current(config.state);
      setStatus('authenticating');
      const callback = new URL(CALLBACK_PATH, window.location.origin);
      callback.searchParams.set('authCode', result.authCode);
      callback.searchParams.set('state', config.state);
      try {
        authenticatedRef.current(`${callback.pathname}${callback.search}`);
      } catch {
        setStatus('failed');
      }
    }
  }, [poll.data, poll.isError, config]);

  if (status === 'authenticating') {
    return (
      <QrRefreshCard onRefresh={onRefresh} disabled>
        <div className={styles.qrStatus}>
          <Spin />
          <span>扫码成功，正在认证</span>
        </div>
      </QrRefreshCard>
    );
  }
  if (status === 'failed') {
    return (
      <QrRefreshCard onRefresh={onRefresh}>
        <QrRefreshNotice title="二维码加载失败" />
      </QrRefreshCard>
    );
  }
  if (status === 'expired' || config.expiresAt <= Date.now()) {
    return (
      <QrRefreshCard onRefresh={onRefresh}>
        <QrRefreshNotice title="二维码已失效" />
      </QrRefreshCard>
    );
  }
  return (
    <QrRefreshCard onRefresh={onRefresh}>
      <div
        className={styles.qrDisplay}
        role="img"
        aria-label="钉钉扫码登录二维码"
      >
        <QRCode
          type="svg"
          value={config.qrCodeUrl}
          size={180}
          bordered={false}
          bgColor="#ffffff"
          color="#181c1f"
        />
      </div>
      {poll.data?.status === SCANNED && (
        <div className={styles.qrScanned}>
          <Spin />
          <span>已扫码，请在手机钉钉上确认</span>
        </div>
      )}
    </QrRefreshCard>
  );
}

export function DingTalkQrLogin({ active = true, onAuthenticated }: Props) {
  const visible = useSyncExternalStore(
    subscribeVisibility,
    () => document.visibilityState !== 'hidden',
    () => true,
  );
  const foreground = active && visible;
  const [retiredState, setRetiredState] = useState<string>();
  const queryClient = useQueryClient();
  const query = useQuery({
    ...dingTalkLoginConfigOptions,
    enabled: foreground,
  });
  const wasActive = useRef(foreground);
  const unavailableState = useRef<string | undefined>(undefined);
  const currentState = useRef<string | undefined>(undefined);
  currentState.current = query.data?.enabled ? query.data.qr.state : undefined;
  const retireCode = useRef<(() => void) | null>(null);

  const refresh = useCallback(() => {
    if (query.isFetching) return;
    retireCode.current?.();
    // 新配置会写 state Cookie；先退役旧轮询，再串行获取新尝试。
    setRetiredState(query.data?.enabled ? query.data.qr.state : undefined);
    unavailableState.current = undefined;
    void query.refetch({ cancelRefetch: false });
  }, [query]);

  useLayoutEffect(() => {
    const returning = foreground && !wasActive.current;
    wasActive.current = foreground;
    const state = queryClient.getQueryState(
      dingTalkLoginConfigOptions.queryKey,
    );
    if (
      returning &&
      ((state?.data?.enabled &&
        unavailableState.current === state.data.qr.state) ||
        state?.status === 'error' ||
        state?.isInvalidated ||
        (state?.data?.enabled && state.data.qr.expiresAt <= Date.now()))
    )
      refresh();
  }, [foreground, queryClient, refresh]);

  return (
    <div className={styles.qrLogin} hidden={!active}>
      <div className={styles.qrContent} aria-live="polite">
        {query.isFetching ||
        query.isPending ||
        (!query.isError &&
          query.data?.enabled &&
          query.data.qr.state === retiredState) ? (
          <QrRefreshCard onRefresh={refresh} disabled>
            <div className={styles.qrStatus}>
              <Spin />
              <span>正在加载钉钉二维码</span>
            </div>
          </QrRefreshCard>
        ) : query.isError ? (
          <QrRefreshCard
            onRefresh={refresh}
            title={getRequestErrorMessage(
              query.error,
              '获取钉钉登录配置失败，请重试',
            )}
          >
            <QrRefreshNotice title="二维码加载失败" />
          </QrRefreshCard>
        ) : query.data.enabled ? (
          <DingTalkCode
            key={query.data.qr.state}
            config={query.data.qr}
            active={foreground}
            retireCode={retireCode}
            onUnavailable={(state) => {
              if (currentState.current === state)
                unavailableState.current = state;
            }}
            onAuthenticated={onAuthenticated}
            onRefresh={refresh}
          />
        ) : (
          <QrRefreshCard onRefresh={refresh} disabled>
            <div className={styles.qrStatus}>
              <span>钉钉登录暂未启用，请联系管理员</span>
            </div>
          </QrRefreshCard>
        )}
      </div>
    </div>
  );
}
