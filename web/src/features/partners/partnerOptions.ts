import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import type { SelectOption } from '@/types/select-option';
import { unwrapList } from '@/utils/api';

/** 往来单位候选项：在通用字段上补充散客契约标记。 */
export type PartnerOption = SelectOption & { isCasual?: boolean };

type PartnerSearchOptions = {
  role?: number;
  enabled?: boolean;
};

export async function searchPartnerOptions(
  keyword?: string,
  options: PartnerSearchOptions = {},
): Promise<PartnerOption[]> {
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
