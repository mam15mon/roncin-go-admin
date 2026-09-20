import { masterDataServiceListShippingLines } from '@/services/roncin/masterDataService';
import type { SelectOption } from '@/types/select-option';
import { unwrapList } from '@/utils/api';

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
