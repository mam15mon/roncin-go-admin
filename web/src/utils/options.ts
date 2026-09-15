import {
  masterDataServiceListCurrencies,
  masterDataServiceListShippingLines,
} from '@/services/roncin/masterDataService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import { unwrapList } from './api';

export type SelectOption = {
  label: string;
  value: string;
  code?: string;
  name?: string;
  isCasual?: boolean;
  /** 契约字段：该往来户已配置信用额度且折本币未核销总额超出额度（仅客户方向有意义）。 */
  creditExceeded?: boolean;
  disabled?: boolean;
};

/**
 * 直接干预模式下把超额客户候选项置灰禁用；
 * 仅提醒模式原样返回，标签与禁用状态都经契约字段渲染，不污染 label。
 */
export function disableCreditExceededOptions<T extends SelectOption>(
  options: T[],
  interventionActive: boolean,
): T[] {
  if (!interventionActive) {
    return options;
  }
  return options.map((option) =>
    option.creditExceeded ? { ...option, disabled: true } : option,
  );
}

type PartnerSearchOptions = {
  role?: number;
  enabled?: boolean;
};

let currenciesRequest: Promise<API.Currency[]> | undefined;

export async function searchPartnerOptions(
  keyword?: string,
  options: PartnerSearchOptions = {},
): Promise<SelectOption[]> {
  const response = await partnerServiceListPartners({
    page: 1,
    pageSize: 50,
    keyword,
    role: options.role,
    enabled: options.enabled,
  });
  return unwrapList(response)
    .map((partner) => ({
      label:
        partner.legalName && partner.code
          ? `${partner.legalName} (${partner.code})`
          : partner.legalName || partner.code || partner.id || '',
      value: partner.id || '',
      code: partner.code,
      name: partner.legalName,
      isCasual: partner.isCasual,
    }))
    .filter((option) => option.value !== '');
}

export async function searchShippingLineOptions(
  keyword?: string,
): Promise<SelectOption[]> {
  const response = await masterDataServiceListShippingLines({
    page: 1,
    pageSize: 50,
    keyword,
    enabled: true,
  });
  return unwrapList(response)
    .map((line) => ({
      label: `${line.nameZh || ''} / ${line.nameEn || ''} (${line.scacCode || ''})`,
      value: line.id || '',
      code: line.scacCode,
      name: line.nameZh || line.nameEn,
    }))
    .filter((option) => option.value !== '');
}

export function getCurrencies(forceRefresh = false): Promise<API.Currency[]> {
  if (!currenciesRequest || forceRefresh) {
    currenciesRequest = masterDataServiceListCurrencies({ enabledOnly: true })
      .then(unwrapList)
      .catch((error) => {
        currenciesRequest = undefined;
        throw error;
      });
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
