import { masterDataServiceListCurrencies } from '@/services/roncin/masterDataService';
import type { SelectOption } from '@/types/select-option';
import { unwrapList } from '@/utils/api';

let currenciesRequest: Promise<API.Currency[]> | undefined;

export function getCurrencies(forceRefresh = false): Promise<API.Currency[]> {
  if (!currenciesRequest || forceRefresh) {
    let createdRequest: Promise<API.Currency[]>;
    createdRequest = masterDataServiceListCurrencies({ enabledOnly: true })
      .then(unwrapList)
      .catch((error) => {
        // 仅当失败请求仍是当前缓存时才清空；强制刷新后旧请求迟到失败不得清掉新缓存。
        if (currenciesRequest === createdRequest) {
          currenciesRequest = undefined;
        }
        throw error;
      });
    currenciesRequest = createdRequest;
  }
  return currenciesRequest;
}

export async function getCurrencyOptions(): Promise<SelectOption[]> {
  const currencies = await getCurrencies();
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
