import { queryOptions } from '@tanstack/react-query';
import { masterDataServiceListCurrencies } from '@/services/roncin/masterDataService';
import type { SelectOption } from '@/types/select-option';
import { ensureListResponse, unwrapList } from '@/utils/api';
import type { RequestOptions } from '@/utils/requestClient';

export const CURRENCY_QUERY_PREFIX = ['master-data', 'currencies'] as const;

async function fetchCurrencies(
  enabledOnly: boolean,
  options?: RequestOptions,
): Promise<API.Currency[]> {
  return unwrapList(
    ensureListResponse(
      await masterDataServiceListCurrencies({ enabledOnly }, options),
      '币种数据加载失败，请稍后重试',
    ),
  );
}

/** 原生表单 request 等按需入口只负责请求，不持有模块级结果缓存。 */
export function getCurrencies(
  requestOptions?: RequestOptions,
): Promise<API.Currency[]> {
  return fetchCurrencies(true, requestOptions);
}

function toCurrencyOptions(currencies: API.Currency[]): SelectOption[] {
  return currencies
    .filter((currency) => Boolean(currency.enabled) && currency.code)
    .map((currency) => ({
      label: currency.name
        ? `${currency.code} - ${currency.name}`
        : currency.code || '',
      value: currency.code || '',
      code: currency.code,
      name: currency.name,
    }));
}

export async function getCurrencyOptions(): Promise<SelectOption[]> {
  return fetchCurrencyOptions();
}

/** Query 消费方显式传递取消与错误处理选项，原生字段 request 仍使用零参数入口。 */
export async function fetchCurrencyOptions(
  requestOptions?: RequestOptions,
): Promise<SelectOption[]> {
  return toCurrencyOptions(await getCurrencies(requestOptions));
}

/** 认证业务从当前 QueryClient 读取币种，启停后按统一前缀失效。 */
export function currencyQueryOptions(enabledOnly = true) {
  return queryOptions({
    queryKey: [...CURRENCY_QUERY_PREFIX, 'list', { enabledOnly }],
    queryFn: ({ signal }) =>
      fetchCurrencies(enabledOnly, { signal, skipErrorHandler: true }),
    meta: { errorMessage: '币种数据加载失败，请稍后重试' },
  });
}

/** 下拉与列表共用原始缓存，候选映射仅作为观察者投影。 */
export function currencyOptionsQueryOptions() {
  return { ...currencyQueryOptions(), select: toCurrencyOptions };
}
