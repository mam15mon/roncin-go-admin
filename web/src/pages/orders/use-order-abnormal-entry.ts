import { type RefObject, useEffect, useRef } from 'react';
import { useLocation } from 'react-router';
import { history } from '@/router/history';
import type { AbnormalCasePanelRef } from './abnormal-case-panel';

type EntryOptions = {
  order?: API.Order;
  orderId?: string;
  kind?: string;
  loading: boolean;
  error: boolean;
  canRead: boolean;
  panelRef: RefObject<AbnormalCasePanelRef | null>;
};

/** 直达参数只消费一次，普通订单重查不再次打开异常面板。 */
export function useOrderAbnormalEntry({
  order,
  orderId,
  kind,
  loading,
  error,
  canRead,
  panelRef,
}: EntryOptions) {
  const location = useLocation();
  const consumedEntryRef = useRef<string | undefined>(undefined);
  useEffect(() => {
    const query = new URLSearchParams(location.search);
    if (query.get('panel') !== 'abnormal' || !kind || !orderId) return;
    if (location.pathname !== `/orders/${kind}/${orderId}`) return;
    const entry = `${location.key}:${location.pathname}:${location.search}`;
    if (consumedEntryRef.current === entry || loading) return;
    if (canRead && !error && order?.id === orderId) {
      if (!panelRef.current) return;
      panelRef.current.open(order);
    } else if (canRead && !error) {
      return;
    }
    consumedEntryRef.current = entry;
    query.delete('panel');
    const search = query.toString();
    void history.replace(
      `${location.pathname}${search ? `?${search}` : ''}${location.hash}`,
    );
  }, [location, order, orderId, kind, loading, error, canRead, panelRef]);
}
