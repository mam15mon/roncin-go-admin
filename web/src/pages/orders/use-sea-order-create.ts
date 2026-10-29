import { useMutation } from '@tanstack/react-query';
import { useLayoutEffect, useRef, useState } from 'react';
import {
  getRequestErrorMessage,
  getRequestErrorStatus,
} from '@/requestErrorConfig';
import { orderServiceCreateOrder } from '@/services/roncin/orderService';
import { generateUUID } from '@/utils/uuid';

type PendingCreate = { payload: API.CreateOrderRequest; createdId?: string };
type CreateState = {
  status: 'idle' | 'sending' | 'unknown' | 'success' | 'blocked';
  pending?: PendingCreate;
  error?: string;
};

function restore(key?: string): CreateState {
  if (!key) return { status: 'idle' };
  try {
    const raw = sessionStorage.getItem(key);
    if (!raw) return { status: 'idle' };
    const pending = JSON.parse(raw) as PendingCreate;
    if (!pending.payload?.idempotencyKey)
      throw new Error('待确认记录缺少请求标识');
    return { status: pending.createdId ? 'success' : 'unknown', pending };
  } catch {
    return {
      status: 'blocked',
      error:
        '无法读取待确认的创建记录，请保留当前页面并联系管理员核对订单，避免重复创建。',
    };
  }
}

/** 创建意图先持久化再发送；刷新与超时确认始终使用同一键和载荷。 */
export function useSeaOrderCreate(draftKey?: string) {
  const storageKey = draftKey
    ? `roncin:pending-order-create:${draftKey}`
    : undefined;
  const [state, setState] = useState<CreateState>(() => restore(storageKey));
  const pendingRef = useRef(state.pending);
  const inFlight = useRef(false);
  const epoch = useRef(0);
  useLayoutEffect(() => {
    epoch.current += 1;
    return () => {
      epoch.current += 1;
    };
  }, []);
  const mutation = useMutation({
    mutationFn: (payload: API.CreateOrderRequest) =>
      orderServiceCreateOrder(payload, { skipErrorHandler: true }),
    meta: { silent: true },
  });
  const submit = async (
    payload?: API.CreateOrderRequest,
  ): Promise<string | undefined> => {
    if (inFlight.current || state.status === 'blocked') return undefined;
    if (pendingRef.current?.createdId) return pendingRef.current.createdId;
    const currentEpoch = epoch.current;
    const current = () => currentEpoch === epoch.current;
    const confirming = Boolean(pendingRef.current);
    let pending = pendingRef.current;
    if (!pending) {
      if (!payload) return undefined;
      if (!storageKey)
        throw new Error(
          '当前用户或公司信息不完整，无法安全创建订单，请刷新后重试。',
        );
      pending = {
        // 与 HTTP JSON 载荷及刷新恢复采用同一序列化快照。
        payload: JSON.parse(
          JSON.stringify({ ...payload, idempotencyKey: generateUUID() }),
        ) as API.CreateOrderRequest,
      };
      try {
        sessionStorage.setItem(storageKey, JSON.stringify(pending));
      } catch {
        throw new Error(
          '无法暂存本次创建记录，请允许浏览器会话存储后重试；尚未发送创建请求。',
        );
      }
      pendingRef.current = pending;
    }
    inFlight.current = true;
    setState({ status: 'sending', pending });
    try {
      const response = await mutation.mutateAsync(pending.payload);
      if (!current()) return undefined;
      if (!response?.data?.id || response.success === false) {
        setState({ status: 'unknown', pending });
        throw new Error(
          '创建结果待确认，请点击「确认创建结果」重试，原提交内容已保留。',
        );
      }
      pending.createdId = response.data.id;
      // 写入成功后保留目标 ID；即使导航被中断，恢复时也不重复创建。
      try {
        if (storageKey)
          sessionStorage.setItem(storageKey, JSON.stringify(pending));
      } catch {
        /* 原请求快照仍可同键确认。 */
      }
      setState({ status: 'success', pending });
      return pending.createdId;
    } catch (error) {
      if (!current()) return undefined;
      const status = getRequestErrorStatus(error);
      // 创建接口在参数、认证、权限、业务前置条件失败时不写入；408/429 和未知网关报文仍待确认。
      const reason =
        (
          error as {
            data?: { reason?: string };
            response?: { data?: { reason?: string } };
          }
        )?.data?.reason ??
        (error as { response?: { data?: { reason?: string } } })?.response?.data
          ?.reason;
      const rejected =
        !confirming &&
        Boolean(reason) &&
        [400, 401, 403, 404, 409, 422].includes(status ?? 0) &&
        reason !== 'ORDER_IDEMPOTENCY_CONFLICT' &&
        reason !== 'DUPLICATE_SUBMIT' &&
        (error as { name?: string })?.name !== 'DuplicateSubmitError';
      if (rejected) {
        try {
          if (storageKey) sessionStorage.removeItem(storageKey);
        } catch {
          setState({ status: 'unknown', pending });
          throw new Error(
            '创建被拒绝，但待确认记录未能清理，请再次确认结果后重试。',
          );
        }
        pendingRef.current = undefined;
        setState({ status: 'idle' });
      } else setState({ status: 'unknown', pending });
      throw new Error(
        getRequestErrorMessage(
          error,
          rejected
            ? '创建失败，输入已保留，请修正后重试。'
            : '创建结果待确认，请点击「确认创建结果」，不会重复建单。',
        ),
      );
    } finally {
      if (current()) inFlight.current = false;
    }
  };
  const clear = () => {
    if (storageKey) sessionStorage.removeItem(storageKey);
    pendingRef.current = undefined;
  };
  return {
    status: mutation.isPending ? 'sending' : state.status,
    frozen: mutation.isPending || state.status !== 'idle',
    error: state.error,
    submit,
    clear,
  };
}
