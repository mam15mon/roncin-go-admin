import { masterDataServiceListCurrencies } from '@/services/roncin/masterDataService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import { unwrapList } from './api';

export type SelectOption = {
  label: string;
  value: string;
  code?: string;
  name?: string;
};

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
    }))
    .filter((option) => option.value !== '');
}

export function getCurrencies(): Promise<API.Currency[]> {
  if (!currenciesRequest) {
    currenciesRequest = masterDataServiceListCurrencies()
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
    .filter((currency) => currency.enabled !== false && currency.code)
    .map((currency) => ({
      label: currency.name
        ? `${currency.code} - ${currency.name}`
        : currency.code || '',
      value: currency.code || '',
      code: currency.code,
      name: currency.name,
    }));
}
