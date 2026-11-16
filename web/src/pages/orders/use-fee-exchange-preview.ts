import type { ProFormInstance } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import React, { useState } from 'react';
import {
  calculateExactFeeTotal,
  quantityOrPricePattern,
} from '@/utils/decimal';
import { formatAmount, trimDecimal } from '@/utils/format';
import type { FeeFormValues } from './components/fees/feeFormValues';
import {
  type FeeExchangeRateError,
  type FeeExchangeRateInput,
  fetchFeeExchangeRate,
} from './fee-exchange-rate';

type ExchangeRateStatus = 'idle' | 'loading' | 'resolved' | 'missing' | 'error';

/** 金额在本地计算；系统汇率按当前订单与表单币种查询，手工输入保持独立。 */
export function useFeeExchangePreview(
  orderId?: string,
  formRef?: React.RefObject<ProFormInstance<FeeFormValues> | undefined>,
) {
  const [request, setRequest] = useState<FeeExchangeRateInput>();
  const [seed, setSeed] = useState<{ orderId?: string; rate?: string }>();
  const [totalPreview, setTotalPreview] = useState<string>();
  const [manualExchangeRate, setManualExchangeRate] = useState(false);
  const activeRequest = request?.orderId === orderId ? request : undefined;
  const rateQuery = useQuery({
    queryKey: ['orders', 'fee-exchange-rate', activeRequest ?? {}],
    enabled: Boolean(activeRequest),
    queryFn: ({ signal }) => {
      if (!activeRequest) throw new Error('缺少汇率查询参数');
      return fetchFeeExchangeRate(activeRequest, signal);
    },
  });
  const error = rateQuery.error as FeeExchangeRateError | null;
  const missing =
    error?.reason === 'ORDER_ETD_MISSING' ||
    error?.reason === 'ORDER_ETA_MISSING' ||
    error?.reason === 'FEE_EXCHANGE_RATE_MISSING';
  const seeded = seed && seed.orderId === orderId;
  const exchangeRateStatus: ExchangeRateStatus = !activeRequest
    ? seeded
      ? 'resolved'
      : 'idle'
    : rateQuery.isPending
      ? 'loading'
      : rateQuery.isError
        ? missing
          ? 'missing'
          : 'error'
        : 'resolved';
  const exchangeRatePreview = activeRequest
    ? rateQuery.data
    : seeded
      ? seed.rate
      : undefined;

  const resetPreview = () => {
    setRequest(undefined);
    setSeed(undefined);
    setTotalPreview(undefined);
    setManualExchangeRate(false);
  };

  const seedFromFee = (fee: API.OrderFee) => {
    const total = calculateExactFeeTotal(fee.quantity, fee.unitPrice);
    setTotalPreview(total ? formatAmount(total) : undefined);
    setRequest(undefined);
    setSeed({
      orderId,
      rate: fee.exchangeRate ? trimDecimal(fee.exchangeRate) : undefined,
    });
  };

  const handleValuesChange = (options?: { amountOnly?: boolean }) => {
    const values = formRef?.current?.getFieldsValue();
    if (!values) return;
    const { quantity, unitPrice, currency, direction } = values;
    const total =
      quantity &&
      unitPrice &&
      quantityOrPricePattern.test(quantity) &&
      quantityOrPricePattern.test(unitPrice)
        ? calculateExactFeeTotal(quantity, unitPrice)
        : undefined;
    setTotalPreview(total ? formatAmount(total) : undefined);
    if (options?.amountOnly) return;
    setSeed(undefined);
    setManualExchangeRate(false);
    formRef?.current?.setFieldValue('exchangeRateOverride', undefined);
    if (!orderId || !direction || !currency) {
      setRequest(undefined);
      return;
    }
    const next = { orderId, direction: Number(direction), currency };
    if (
      activeRequest?.direction === next.direction &&
      activeRequest.currency === next.currency
    ) {
      void rateQuery.refetch();
    } else {
      setRequest(next);
    }
  };

  return {
    totalPreview,
    exchangeRatePreview,
    exchangeRateStatus,
    manualExchangeRate,
    setManualExchangeRate,
    resetPreview,
    seedFromFee,
    handleValuesChange,
  };
}
