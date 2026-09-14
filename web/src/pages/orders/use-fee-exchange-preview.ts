import type { ProFormInstance } from '@ant-design/pro-components';
import { App } from 'antd';
import dayjs from 'dayjs';
import React, { useRef, useState } from 'react';
import { orderFeeServiceResolveFeeExchangeRate } from '@/services/roncin/orderFeeService';
import {
  calculateExactFeeTotal,
  quantityOrPricePattern,
} from '@/utils/decimal';
import { trimDecimal } from '@/utils/format';
import type { FeeFormValues } from './components/fees/FeeFormModal';

type ExchangeRateStatus = 'idle' | 'loading' | 'resolved' | 'missing' | 'error';

type FeeRequestError = Error & {
  data?: { message?: string; reason?: string };
  response?: { data?: { message?: string; reason?: string } };
};

/** 费用表单的金额合计预览与汇率解析状态（含手动补录汇率开关）。 */
export function useFeeExchangePreview(
  orderId?: string,
  formRef?: React.RefObject<ProFormInstance<FeeFormValues> | undefined>,
) {
  const { message } = App.useApp();
  const exchangeRateRequestRef = useRef(0);
  const [totalPreview, setTotalPreview] = useState<string>();
  const [exchangeRatePreview, setExchangeRatePreview] = useState<string>();
  const [exchangeRateStatus, setExchangeRateStatus] =
    useState<ExchangeRateStatus>('idle');
  const [manualExchangeRate, setManualExchangeRate] = useState(false);
  // 漏配容灾：命中最近历史周行时置位，供费用表单展示「暂沿用上周汇率」黄色提示。
  const [inheritedLastWeek, setInheritedLastWeek] = useState(false);

  const resetPreview = () => {
    exchangeRateRequestRef.current += 1;
    setTotalPreview(undefined);
    setExchangeRatePreview(undefined);
    setExchangeRateStatus('idle');
    setManualExchangeRate(false);
    setInheritedLastWeek(false);
  };

  const seedFromFee = (fee: API.OrderFee) => {
    setTotalPreview(calculateExactFeeTotal(fee.quantity, fee.unitPrice));
    setExchangeRatePreview(
      fee.exchangeRate ? trimDecimal(fee.exchangeRate) : undefined,
    );
    setExchangeRateStatus('resolved');
  };

  const resolveExchangeRate = (
    currentOrderId: string,
    direction: number,
    currency: string,
    expenseDate: string,
  ) => {
    const requestSequence = ++exchangeRateRequestRef.current;
    setExchangeRateStatus('loading');
    setExchangeRatePreview(undefined);
    setManualExchangeRate(false);
    setInheritedLastWeek(false);
    formRef?.current?.setFieldValue('exchangeRateOverride', undefined);
    void orderFeeServiceResolveFeeExchangeRate(
      { orderId: currentOrderId, direction, currency, expenseDate },
      { skipErrorHandler: true },
    )
      .then((response) => {
        if (requestSequence !== exchangeRateRequestRef.current) return;
        if (!response.exchangeRate) {
          setExchangeRateStatus('error');
          message.error('汇率解析结果不完整');
          return;
        }
        setExchangeRatePreview(trimDecimal(response.exchangeRate));
        setExchangeRateStatus('resolved');
        setInheritedLastWeek(
          response.exchangeRateSource === 'INHERITED_LAST_WEEK',
        );
      })
      .catch((error: FeeRequestError) => {
        if (requestSequence !== exchangeRateRequestRef.current) return;
        const code = error.data?.reason || error.response?.data?.reason;
        if (code === 'FEE_EXCHANGE_RATE_MISSING') {
          setExchangeRateStatus('missing');
          setManualExchangeRate(true);
          return;
        }
        setExchangeRateStatus('error');
        message.error(error.message || '汇率解析失败');
      });
  };

  const handleValuesChange = () => {
    const values = formRef?.current?.getFieldsValue();
    if (!values) return;
    const { quantity, unitPrice, currency, expenseDate, direction } = values;
    if (
      quantity &&
      unitPrice &&
      quantityOrPricePattern.test(quantity) &&
      quantityOrPricePattern.test(unitPrice)
    ) {
      setTotalPreview(calculateExactFeeTotal(quantity, unitPrice));
    } else {
      setTotalPreview(undefined);
    }
    if (orderId && direction && currency && expenseDate) {
      resolveExchangeRate(
        orderId,
        Number(direction),
        currency,
        dayjs(expenseDate).format('YYYY-MM-DD'),
      );
    }
  };

  return {
    totalPreview,
    exchangeRatePreview,
    exchangeRateStatus,
    manualExchangeRate,
    setManualExchangeRate,
    inheritedLastWeek,
    resetPreview,
    seedFromFee,
    handleValuesChange,
  };
}
