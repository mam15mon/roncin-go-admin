import { App } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { orderFeeServiceListFeeOptions } from '@/services/roncin/orderFeeService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';

/** 加载订单档案与费用录入候选项、财务锁定状态。 */
export function useOrderFeeOptions(orderId?: string) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(Boolean(orderId));
  const [order, setOrder] = useState<API.Order>();
  const [loadedOrderId, setLoadedOrderId] = useState<string | undefined>();
  const [failedOrderId, setFailedOrderId] = useState<string | undefined>();
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
    if (!orderId) {
      setOrder(undefined);
      setLoadedOrderId(undefined);
      setFailedOrderId(undefined);
      setCurrencies([]);
      setSettlementParties([]);
      setFeeSettings([]);
      setBillingUnits([]);
      setFinanceLocked(false);
      setFinanceLockReason('');
      setFinanceLockCommissionNos([]);
      setCustomerName('');
      setLoading(false);
      return;
    }
    const currentRequestId = ++requestIdRef.current;
    const currentOrderId = orderId;
    setLoading(true);
    setFailedOrderId(undefined);
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
      setLoadedOrderId(currentOrderId);
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
        setOrder(undefined);
        setLoadedOrderId(undefined);
        setFailedOrderId(currentOrderId);
        setCurrencies([]);
        setSettlementParties([]);
        setFeeSettings([]);
        setBillingUnits([]);
        setFinanceLocked(false);
        setFinanceLockReason('');
        setFinanceLockCommissionNos([]);
        setCustomerName('');
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

  const isOrderMatched = Boolean(orderId && loadedOrderId === orderId);
  const effectiveOrder = isOrderMatched ? order : undefined;
  const effectiveCurrencies = isOrderMatched ? currencies : [];
  const effectiveSettlementParties = isOrderMatched ? settlementParties : [];
  const effectiveFeeSettings = isOrderMatched ? feeSettings : [];
  const effectiveBillingUnits = isOrderMatched ? billingUnits : [];
  const effectiveCustomerName = isOrderMatched ? customerName : '';
  const isPending =
    Boolean(orderId) && !isOrderMatched && failedOrderId !== orderId;
  const effectiveLoading = loading || isPending;

  return {
    loading: effectiveLoading,
    order: effectiveOrder,
    loadedOrderId,
    currencies: effectiveCurrencies,
    settlementParties: effectiveSettlementParties,
    setSettlementParties,
    feeSettings: effectiveFeeSettings,
    setFeeSettings,
    billingUnits: effectiveBillingUnits,
    financeLocked: isOrderMatched ? financeLocked : false,
    financeLockReason: isOrderMatched ? financeLockReason : '',
    financeLockCommissionNos: isOrderMatched ? financeLockCommissionNos : [],
    customerName: effectiveCustomerName,
    loadData,
  };
}
