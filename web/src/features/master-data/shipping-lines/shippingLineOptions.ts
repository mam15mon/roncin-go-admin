import { masterDataServiceListShippingLines } from '@/services/roncin/masterDataService';
import type { SelectOption } from '@/types/select-option';
import { ensureListResponse, unwrapList } from '@/utils/api';
import type { RequestOptions } from '@/utils/requestClient';

export async function searchShippingLineOptions(
  keyword?: string,
  requestOptions?: RequestOptions,
): Promise<SelectOption[]> {
  const response = await masterDataServiceListShippingLines(
    {
      page: 1,
      pageSize: 50,
      keyword,
      enabled: true,
    },
    requestOptions,
  );
  return unwrapList(ensureListResponse(response, '船公司候选加载失败'))
    .map((line) => ({
      label: `${line.nameZh || ''} / ${line.nameEn || ''} (${line.scacCode || ''})`,
      value: line.id || '',
      code: line.scacCode,
      name: line.nameZh || line.nameEn,
    }))
    .filter((option) => option.value !== '');
}
