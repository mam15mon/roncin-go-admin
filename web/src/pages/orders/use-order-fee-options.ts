import { App } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { orderFeeServiceListFeeOptions } from '@/services/roncin/orderFeeService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';

/** 加载订单档案与费用录入候选项、财务锁定状态。 */
export function useOrderFeeOptions(orderId?: string) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [order, setOrder] = useState<API.Order>();
  const activeOrderIdRef = useRef(orderId);
  activeOrderIdRef.current = orderId;
  const requestIdRef = useRef(0);
  const [currencies, setCurrencies] = useState<API.OrderFeeCurrencyOption[]>(
    [],
  );
  const [settlementParties, setSettlementParties] = useState<
    API.OrderFeeSettlementPartyOption[]
  >([]);
  const [feeSettings, setFeeSettings] = useState<API.OrderFeeSettingOption[]>(
    [],
  );
  const [billingUnits, setBillingUnits] = useState<
    API.OrderFeeBillingUnitOption[]
  >([]);
  const [financeLocked, setFinanceLocked] = useState(false);
  const [financeLockReason, setFinanceLockReason] = useState('');
  const [financeLockCommissionNos, setFinanceLockCommissionNos] = useState<
    string[]
  >([]);
  const [customerName, setCustomerName] = useState('');

  const loadData = useCallback(async () => {
    if (!orderId) return;
    const currentRequestId = ++requestIdRef.current;
    const currentOrderId = orderId;
    setLoading(true);
    try {
      const [orderRes, optionsRes] = await Promise.all([
        orderServiceGetOrder({ id: orderId }),
        orderFeeServiceListFeeOptions({ orderId }),
      ]);
      if (
        currentRequestId !== requestIdRef.current ||
        currentOrderId !== activeOrderIdRef.current
      ) {
        return;
      }
      setOrder(orderRes.data);
      setCurrencies(optionsRes.currencies ?? []);
      setSettlementParties(optionsRes.settlementParties ?? []);
      setFeeSettings(optionsRes.feeSettings ?? []);
      setBillingUnits(optionsRes.billingUnits ?? []);
      setFinanceLocked(Boolean(optionsRes.financeLocked));
      setFinanceLockReason(optionsRes.financeLockReason || '');
      setFinanceLockCommissionNos(optionsRes.financeLockCommissionNos || []);
      setCustomerName(optionsRes.customerName || '');
    } catch (error: any) {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current
      ) {
        message.error(error.message || '加载费用信息失败');
      }
    } finally {
      if (
        currentRequestId === requestIdRef.current &&
        currentOrderId === activeOrderIdRef.current
      ) {
        setLoading(false);
      }
    }
  }, [orderId, message]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  return {
    loading,
    order,
    currencies,
    settlementParties,
    setSettlementParties,
    feeSettings,
    setFeeSettings,
    billingUnits,
    financeLocked,
    financeLockReason,
    financeLockCommissionNos,
    customerName,
    loadData,
  };
}
