import { ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Spin } from 'antd';
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import {
  CALLBACK_PATH,
  DINGTALK_ORIGIN,
  dingTalkLoginConfigOptions,
  type FrameConfig,
} from '@/app/dingtalkLoginConfig';
import {
  alignPreloadedDingTalkFrame,
  claimPreloadedDingTalkFrame,
  discardPreloadedDingTalkFrame,
} from '@/app/dingtalkPreloadedFrame';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import styles from '../index.module.less';

type Props = {
  active?: boolean;
  onAuthenticated: (callbackPath: string) => void;
};

function DingTalkFrame({
  config,
  active = true,
  onAuthenticated,
  onExpired,
  onUnavailable,
  retireFrame,
}: Props & {
  config: FrameConfig;
  onExpired: () => void;
  onUnavailable: (state: string) => void;
  retireFrame: React.RefObject<(() => void) | null>;
}) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const slotRef = useRef<HTMLDivElement>(null);
  const lifecycle = useRef(0);
  const activeRef = useRef(active);
  activeRef.current = active;
  const updateVisibility = useRef<() => void>(() => {});
  const onUnavailableRef = useRef(onUnavailable);
  onUnavailableRef.current = onUnavailable;
  const retired = useRef(false);
  const handled = useRef(false);
  const onAuthenticatedRef = useRef(onAuthenticated);
  onAuthenticatedRef.current = onAuthenticated;
  // 经 ref 消费回调，父组件重渲染不会重置到期定时器与消息监听。
  const onExpiredRef = useRef(onExpired);
  onExpiredRef.current = onExpired;
  const [status, setStatus] = useState<
    | 'navigating'
    | 'displayed'
    | 'slow'
    | 'expired'
    | 'failed'
    | 'authenticating'
  >('navigating');
  const [failure, setFailure] = useState('钉钉二维码加载失败，请刷新重试');

  useLayoutEffect(() => {
    const slot = slotRef.current;
    if (!slot) return;
    const generation = ++lifecycle.current;
    let mounted = true;
    let stopAlign = () => {};
    const destroy = () => {
      stopAlign();
      iframeRef.current?.remove();
      iframeRef.current = null;
    };
    retireFrame.current = () => {
      retired.current = true;
      destroy();
    };
    updateVisibility.current = () => {
      stopAlign();
      const frame = iframeRef.current;
      if (!frame || retired.current) return;
      if (Date.now() >= config.expiresAt) {
        expire();
        return;
      }
      const visible = activeRef.current;
      frame.hidden = !visible;
      frame.tabIndex = visible ? 0 : -1;
      frame.setAttribute('aria-hidden', String(!visible));
      if (!visible) frame.style.visibility = 'hidden';
      else if (frame.parentElement === slot) frame.style.visibility = 'visible';
      else stopAlign = alignPreloadedDingTalkFrame(frame, slot);
    };
    const onLoad = () => {
      if (!mounted || retired.current) return;
      if (Date.now() >= config.expiresAt) {
        expire();
      } else {
        // load 只结束导航提示，不代表跨源二维码已就绪。
        setStatus('displayed');
      }
    };
    const onError = () => {
      if (!mounted || retired.current) return;
      retired.current = true;
      destroy();
      setStatus('failed');
      onUnavailableRef.current(config.state);
    };
    const expire = () => {
      if (
        mounted &&
        !retired.current &&
        !handled.current &&
        Date.now() >= config.expiresAt
      ) {
        retired.current = true;
        destroy();
        setStatus('expired');
        onUnavailableRef.current(config.state);
        onExpiredRef.current();
      }
    };
    const timeout = window.setTimeout(
      expire,
      Math.max(0, config.expiresAt - Date.now()),
    );
    const onVisibility = () => {
      if (document.visibilityState === 'visible') expire();
    };
    const onMessage = (event: MessageEvent<unknown>) => {
      const source = iframeRef.current?.contentWindow;
      if (
        !mounted ||
        !activeRef.current ||
        retired.current ||
        handled.current ||
        Date.now() >= config.expiresAt ||
        event.origin !== DINGTALK_ORIGIN ||
        !source ||
        event.source !== source ||
        typeof event.data !== 'object' ||
        event.data === null ||
        Array.isArray(event.data)
      )
        return;
      const data = event.data as Record<string, unknown>;
      if (data.success === false) {
        retired.current = true;
        destroy();
        setFailure('钉钉扫码未完成，请刷新二维码后重试');
        setStatus('failed');
        onUnavailableRef.current(config.state);
        return;
      }
      if (data.success !== true || typeof data.redirectUrl !== 'string') return;
      let redirect: URL;
      try {
        redirect = new URL(data.redirectUrl);
      } catch {
        return;
      }
      const authCode = redirect.searchParams.get('authCode');
      if (
        redirect.origin !== config.callback.origin ||
        redirect.pathname !== config.callback.pathname ||
        redirect.username ||
        redirect.password ||
        redirect.searchParams.getAll('state').length !== 1 ||
        redirect.searchParams.get('state') !== config.state ||
        redirect.searchParams.getAll('authCode').length !== 1 ||
        !authCode?.trim()
      )
        return;
      onUnavailableRef.current(config.state);
      handled.current = true;
      retired.current = true;
      destroy();
      setStatus('authenticating');
      // 只导航到本站固定回调，绝不直接打开 message 提供的 redirectUrl。
      const callback = new URL(CALLBACK_PATH, window.location.origin);
      callback.searchParams.set('authCode', authCode);
      callback.searchParams.set('state', config.state);
      try {
        onAuthenticatedRef.current(`${callback.pathname}${callback.search}`);
      } catch {
        setFailure('无法继续钉钉登录，请刷新页面后重试');
        setStatus('failed');
      }
    };
    window.addEventListener('message', onMessage);
    window.addEventListener('focus', expire);
    document.addEventListener('visibilitychange', onVisibility);
    let frame = iframeRef.current;
    if (!retired.current && Date.now() < config.expiresAt) {
      frame ??= claimPreloadedDingTalkFrame(config) ?? null;
      if (!frame) {
        frame = document.createElement('iframe');
        frame.title = '钉钉扫码登录二维码';
        frame.src = config.src;
        frame.referrerPolicy = 'no-referrer';
        frame.className = styles.qrFrame;
        slot.appendChild(frame);
      }
      iframeRef.current = frame;
      frame.addEventListener('load', onLoad);
      frame.addEventListener('error', onError);
      // 固定 HTML 事件只记录导航结果；接管后改用可清理的原生监听。
      frame.removeAttribute('onload');
      frame.removeAttribute('onerror');
      if (frame.dataset.navigation === 'failed') onError();
      else {
        updateVisibility.current();
        if (frame.dataset.navigation === 'loaded') onLoad();
      }
    }
    expire();
    return () => {
      mounted = false;
      retireFrame.current = null;
      updateVisibility.current = () => {};
      stopAlign();
      if (frame) {
        frame.style.visibility = 'hidden';
        frame.removeEventListener('load', onLoad);
        frame.removeEventListener('error', onError);
      }
      // StrictMode 会立即重演 setup；保留同一浏览上下文，真实卸载则在微任务销毁。
      queueMicrotask(() => {
        if (lifecycle.current === generation) destroy();
      });
      window.clearTimeout(timeout);
      window.removeEventListener('message', onMessage);
      window.removeEventListener('focus', expire);
      document.removeEventListener('visibilitychange', onVisibility);
    };
  }, [config, retireFrame]);

  useLayoutEffect(() => {
    updateVisibility.current();
  }, [active]);

  useEffect(() => {
    if (status !== 'navigating') return;
    const timeout = window.setTimeout(() => {
      // 没有二维码就绪事件：慢导航只提示重试，不停用可能仍可扫码的实例。
      setStatus('slow');
    }, 15000);
    return () => window.clearTimeout(timeout);
  }, [status]);

  if (status === 'expired' || status === 'failed') {
    return (
      <Alert
        type={status === 'expired' ? 'warning' : 'error'}
        showIcon
        title={status === 'expired' ? '二维码已过期，请刷新' : failure}
      />
    );
  }
  if (status === 'authenticating') {
    return (
      <div className={styles.qrPlaceholder}>
        <Spin />
        <span>扫码成功，正在认证</span>
      </div>
    );
  }
  return (
    <div className={styles.qrFrameContent}>
      {status === 'slow' && (
        <Alert
          type="warning"
          showIcon
          title="钉钉二维码加载较慢，请检查网络或刷新二维码"
        />
      )}
      {status === 'navigating' && (
        <div className={styles.qrLoading}>
          <Spin size="small" />
          <span>正在加载钉钉二维码</span>
        </div>
      )}
      <div ref={slotRef} className={styles.qrFrameWrapper} />
    </div>
  );
}

export function DingTalkQrLogin({ active = true, onAuthenticated }: Props) {
  const [retiredState, setRetiredState] = useState<string>();
  const lastAutoRefreshAt = useRef(0);
  const queryClient = useQueryClient();
  const query = useQuery({ ...dingTalkLoginConfigOptions, enabled: active });
  const activeRef = useRef(active);
  activeRef.current = active;
  const wasActive = useRef(active);
  const unavailableState = useRef<string | undefined>(undefined);
  const retireFrame = useRef<(() => void) | null>(null);

  const refresh = useCallback(() => {
    if (query.isFetching) return;
    retireFrame.current?.();
    discardPreloadedDingTalkFrame();
    // 先同步退役实例，再发出会覆盖 state Cookie 的配置请求。
    setRetiredState(query.data?.enabled ? query.data.frame.state : undefined);
    unavailableState.current = undefined;
    void query.refetch({ cancelRefetch: false });
  }, [query]);

  const refreshOnExpired = useCallback(() => {
    if (!activeRef.current) return;
    // 客户端时钟快于服务端时新配置会一到手就过期，限流防止自动刷新演变成请求风暴。
    if (Date.now() - lastAutoRefreshAt.current < 1000) return;
    lastAutoRefreshAt.current = Date.now();
    refresh();
  }, [refresh]);

  useLayoutEffect(() => {
    const returning = active && !wasActive.current;
    wasActive.current = active;
    const state = queryClient.getQueryState(
      dingTalkLoginConfigOptions.queryKey,
    );
    if (
      returning &&
      ((state?.data?.enabled &&
        unavailableState.current === state.data.frame.state) ||
        state?.status === 'error' ||
        state?.isInvalidated ||
        (state?.data?.enabled && state.data.frame.expiresAt <= Date.now()))
    )
      refresh();
  }, [active, queryClient, refresh]);

  return (
    <div className={styles.qrLogin} hidden={!active}>
      <div className={styles.qrContent} aria-live="polite">
        {query.isFetching ||
        query.isPending ||
        (!query.isError &&
          query.data?.enabled &&
          query.data.frame.state === retiredState) ? (
          <div className={styles.qrPlaceholder}>
            <Spin />
            <span>正在获取钉钉登录配置</span>
          </div>
        ) : query.isError ? (
          <Alert
            type="error"
            showIcon
            title={getRequestErrorMessage(
              query.error,
              '获取钉钉登录配置失败，请重试',
            )}
          />
        ) : query.data.enabled ? (
          <DingTalkFrame
            key={query.data.frame.state}
            config={query.data.frame}
            active={active}
            retireFrame={retireFrame}
            onUnavailable={(state) => {
              unavailableState.current = state;
            }}
            onAuthenticated={onAuthenticated}
            onExpired={refreshOnExpired}
          />
        ) : (
          <Alert type="info" showIcon title="钉钉登录暂未启用，请联系管理员" />
        )}
      </div>

      <div className={styles.qrFooter}>
        <Button
          type="link"
          size="small"
          icon={<ReloadOutlined />}
          loading={query.isFetching}
          disabled={query.isFetching}
          className={styles.qrRefreshBtn}
          onClick={refresh}
        >
          刷新二维码
        </Button>
        <div className={styles.qrHintGroup}>
          <p className={styles.qrHint}>
            使用前请联系管理员同步人员并分配组织和角色
          </p>
        </div>
      </div>
    </div>
  );
}
