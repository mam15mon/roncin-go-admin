import type { PartnerOption } from '@/features/partners';
import { settlementServiceListFinanceSettlementPartyOptions } from '@/services/roncin/settlementService';
import { ensureListResponse, unwrapList } from '@/utils/api';

export type FinanceSettlementPartnerOption = PartnerOption & {
  creditExceeded: boolean;
};

/** 收付与核销共用原用途候选接口，快捷新增后也必须经过同一资格校验。 */
export async function searchFinanceSettlementPartnerOptions(
  params: Pick<
    API.SettlementServiceListFinanceSettlementPartyOptionsParams,
    'organizationId' | 'purpose' | 'keyword'
  >,
  signal: AbortSignal,
): Promise<FinanceSettlementPartnerOption[]> {
  return unwrapList(
    ensureListResponse(
      await settlementServiceListFinanceSettlementPartyOptions(
        {
          ...params,
          page: 1,
          pageSize: 50,
        },
        { signal, skipErrorHandler: true },
      ),
      '结算单位候选加载失败',
    ),
  ).flatMap((item) =>
    item.id
      ? [
          {
            value: item.id,
            label:
              item.name && item.code
                ? `${item.name} (${item.code})`
                : item.name || item.code || item.id,
            code: item.code,
            name: item.name,
            isCasual: item.isCasual ?? false,
            creditExceeded: item.creditExceeded ?? false,
          },
        ]
      : [],
  );
}
