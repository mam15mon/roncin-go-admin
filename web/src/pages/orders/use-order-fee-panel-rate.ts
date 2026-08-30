import { App } from 'antd';
import { useRef, useState } from 'react';
import { financeErrorReasons } from '@/errorReasons.generated';
import { orderFeeServiceResolveFeeExchangeRate } from '@/services/roncin/orderFeeService';
import { trimExactDecimal } from '@/utils/decimal';

type ExchangeRateStatus = 'idle' | 'loading' | 'resolved' | 'missing' | 'error';

type FeeRequestError = Error & {
  data?: { reason?: string };
  response?: { data?: { reason?: string } };
};

/** 订单费用抽屉面板的汇率解析与预览状态。 */
export function useOrderFeePanelExchangeRate(editingFee?: API.OrderFee) {
  const { message } = App.useApp();
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
    setExchangeRatePreview(undefined);
    setManualExchangeRate(false);
    orderFeeServiceResolveFeeExchangeRate(
      { orderId, direction, currency, expenseDate },
      { skipErrorHandler: true },
    )
      .then((response) => {
        if (currentRequestId !== exchangeRateRequestRef.current) return;
        if (response.success && response.exchangeRate) {
          setExchangeRateStatus('resolved');
          setExchangeRatePreview(trimExactDecimal(response.exchangeRate));
          if (!editingFee) {
            setManualExchangeRate(false);
          }
        } else {
          setExchangeRateStatus('error');
          message.error('汇率解析结果不完整');
        }
      })
      .catch((error: FeeRequestError) => {
        if (currentRequestId !== exchangeRateRequestRef.current) return;
        const reason = error.data?.reason ?? error.response?.data?.reason;
        if (reason === financeErrorReasons.FEE_EXCHANGE_RATE_MISSING) {
          setExchangeRateStatus('missing');
          setManualExchangeRate(true);
          return;
        }
        setExchangeRateStatus('error');
        message.error(error.message || '汇率解析失败');
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
