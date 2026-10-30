import { useLayoutEffect, useRef, useState } from 'react';
import { orderErrorReasons } from '@/errorReasons.generated';
import {
  getRequestErrorMessage,
  getRequestErrorStatus,
  type RequestError,
} from '@/requestErrorConfig';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import { generateUUID } from '@/utils/uuid';

type SaveStatus = 'idle' | 'saving' | 'unknown' | 'refresh';
// 只收敛真正的并发冲突，号码占用和业务限制仍展示服务端具体原因。
const concurrentChangeReasons = new Set<string>([
  orderErrorReasons.ORDER_STATUS_CONFLICT,
  orderErrorReasons.SEA_MASTER_BILL_STATUS_CONFLICT,
  orderErrorReasons.SEA_HOUSE_BILL_STATUS_CONFLICT,
  orderErrorReasons.SEA_DOCUMENT_STRUCTURE_CONFLICT,
  orderErrorReasons.SEA_DOCUMENT_VERSION_CONFLICT,
  orderErrorReasons.SEA_MASTER_BILL_CONFLICT,
  orderErrorReasons.SEA_HOUSE_BILL_CONFLICT,
]);
/** 一个提交意图冻结同一载荷；已确认写入后只允许重读，不重复写入。 */
export function useSeaOrderSave(
  identity: string | undefined,
  reload: (isCurrent: () => boolean) => Promise<void>,
) {
  const [state, setState] = useState<{ identity?: string; status: SaveStatus }>(
    { status: 'idle' },
  );
  const pending = useRef<
    | { identity: string; payload: API.UpdateOrderRequest; committed: boolean }
    | undefined
  >(undefined);
  const activeIdentity = useRef(identity);
  activeIdentity.current = identity;
  const status = state.identity === identity ? state.status : 'idle';
  const savingRef = useRef(false);
  const epoch = useRef(0);
  useLayoutEffect(() => {
    epoch.current += 1;
    setState({ identity, status: 'idle' });
    pending.current = undefined;
    savingRef.current = false;
    return () => {
      epoch.current += 1;
    };
  }, [identity]);
  const submit = async (payload?: API.UpdateOrderRequest): Promise<boolean> => {
    if (!identity || savingRef.current) return false;
    const currentIdentity = identity;
    const startedEpoch = epoch.current;
    const isCurrent = () =>
      activeIdentity.current === currentIdentity &&
      epoch.current === startedEpoch;
    let request =
      pending.current?.identity === identity ? pending.current : undefined;
    if (!request) {
      if (!payload) return false;
      request = {
        identity,
        payload: structuredClone({
          ...payload,
          idempotencyKey: generateUUID(),
        }),
        committed: false,
      };
      pending.current = request;
    }
    savingRef.current = true;
    setState({ identity, status: 'saving' });
    try {
      if (!request.committed) {
        try {
          const response = await orderServiceUpdateOrder(
            { id: request.payload.id },
            request.payload,
            { skipErrorHandler: true },
          );
          if (!response?.data?.id || response.success === false)
            throw new Error('未能确认保存结果，请重试同一次提交');
          request.committed = true;
        } catch (error) {
          const requestError =
            error && typeof error === 'object'
              ? (error as RequestError)
              : undefined;
          const status = requestError
            ? getRequestErrorStatus(requestError)
            : undefined;
          const knownRejected =
            status !== undefined && status >= 400 && status < 500;
          const reason =
            requestError?.data?.reason ?? requestError?.response?.data?.reason;
          if (knownRejected && isCurrent()) pending.current = undefined;
          if (isCurrent())
            setState({ identity, status: knownRejected ? 'idle' : 'unknown' });
          throw new Error(
            status === 409 && concurrentChangeReasons.has(reason ?? '')
              ? '这个订单已经发生变动，请重新刷新。'
              : requestError
                ? getRequestErrorMessage(
                    requestError,
                    knownRejected
                      ? '保存失败，输入已保留'
                      : '未能确认保存结果，请重试同一次提交',
                  )
                : '保存失败，输入已保留',
          );
        }
      }
      if (!isCurrent()) return false;
      try {
        await reload(isCurrent);
      } catch (error) {
        if (isCurrent()) setState({ identity, status: 'refresh' });
        throw error;
      }
      if (!isCurrent()) return false;
      pending.current = undefined;
      setState({ identity, status: 'idle' });
      return true;
    } finally {
      if (isCurrent()) savingRef.current = false;
    }
  };
  return { status, submit, frozen: status !== 'idle' };
}
