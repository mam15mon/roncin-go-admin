import { getRequestErrorMessage } from '@/requestErrorConfig';
import { orderFeeServiceResolveFeeExchangeRate } from '@/services/roncin/orderFeeService';
import { trimDecimal } from '@/utils/format';

export type FeeExchangeRateInput = {
  orderId: string;
  direction: number;
  currency: string;
};

export type FeeExchangeRateError = Error & { reason?: string };

/** 表单和行内查询共用汇率错误语义，取消仍由 Query 自身的生命周期处理。 */
export async function fetchFeeExchangeRate(
  input: FeeExchangeRateInput,
  signal: AbortSignal,
): Promise<string> {
  try {
    const response = await orderFeeServiceResolveFeeExchangeRate(input, {
      signal,
      skipErrorHandler: true,
    });
    if (!response?.exchangeRate || response.success === false)
      throw new Error('汇率解析结果不完整');
    return trimDecimal(response.exchangeRate);
  } catch (error) {
    const failure = error as {
      data?: { reason?: string };
      response?: { data?: { reason?: string } };
    };
    const reason = failure?.data?.reason ?? failure?.response?.data?.reason;
    const message =
      reason === 'ORDER_ETD_MISSING'
        ? '请先填写订单 ETD 后再解析汇率'
        : reason === 'ORDER_ETA_MISSING'
          ? '请先填写订单 ETA 后再解析汇率'
          : getRequestErrorMessage(
              error,
              reason === 'FEE_EXCHANGE_RATE_MISSING'
                ? '请先维护汇率'
                : error instanceof Error
                  ? error.message
                  : '汇率解析失败',
            );
    throw Object.assign(new Error(message), { reason });
  }
}
