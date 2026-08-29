import { useRef, useState } from 'react';
import { orderFeeServiceResolveFeeExchangeRate } from '@/services/roncin/orderFeeService';
import { trimExactDecimal } from './order-fee-decimal';

type ExchangeRateStatus = 'idle' | 'loading' | 'resolved' | 'missing' | 'error';

/**
 * 订单费用抽屉面板的汇率解析与预览状态。
 * 注意：与费用工作台的 useFeeExchangePreview 语义不同——面板版不区分
 * FEE_EXCHANGE_RATE_MISSING，任何未命中或失败都转为手动录入。
 */
export function useOrderFeePanelExchangeRate(editingFee?: API.OrderFee) {
  const exchangeRateRequestRef = useRef(0);
  const [totalPreview, setTotalPreview] = useState<string>();
  const [exchangeRatePreview, setExchangeRatePreview] = useState<string>();
  const [exchangeRateStatus, setExchangeRateStatus] =
    useState<ExchangeRateStatus>('idle');
  const [manualExchangeRate, setManualExchangeRate] = useState(false);

  const resetPreview = () => {
    setTotalPreview(undefined);
    setExchangeRatePreview(undefined);
    setExchangeRateStatus('idle');
    setManualExchangeRate(false);
  };

  const seedFromFee = (fee: API.OrderFee) => {
    setTotalPreview(trimExactDecimal(fee.totalAmount));
    setExchangeRatePreview(
      fee.exchangeRate ? trimExactDecimal(fee.exchangeRate) : undefined,
    );
    setExchangeRateStatus(fee.exchangeRate ? 'resolved' : 'missing');
    setManualExchangeRate(fee.exchangeRateSource === 'MANUAL');
  };

  const resolveExchangeRate = (
    orderId: string,
    direction: number,
    currency: string,
    expenseDate: string,
  ) => {
    const currentRequestId = ++exchangeRateRequestRef.current;
    setExchangeRateStatus('loading');
    orderFeeServiceResolveFeeExchangeRate({
      orderId,
      direction,
      currency,
      expenseDate,
    })
      .then((response) => {
        if (currentRequestId !== exchangeRateRequestRef.current) return;
        if (response.success && response.exchangeRate) {
          setExchangeRateStatus('resolved');
          setExchangeRatePreview(trimExactDecimal(response.exchangeRate));
          if (!editingFee) {
            setManualExchangeRate(false);
          }
        } else {
          setExchangeRateStatus('missing');
          setExchangeRatePreview(undefined);
          setManualExchangeRate(true);
        }
      })
      .catch(() => {
        if (currentRequestId !== exchangeRateRequestRef.current) return;
        setExchangeRateStatus('error');
        setExchangeRatePreview(undefined);
        setManualExchangeRate(true);
      });
  };

  return {
    totalPreview,
    exchangeRatePreview,
    exchangeRateStatus,
    manualExchangeRate,
    setManualExchangeRate,
    resetPreview,
    seedFromFee,
    resolveExchangeRate,
  };
}
