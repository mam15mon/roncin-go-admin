import type { Key } from 'react';

export type VerificationCandidateScope = {
  organizationId: string;
  direction: string;
  settlementPartyId: string;
  currency: string;
};

/** 正常 ProtoJSON 空列表可省略 data/total；请求失败必须显式失败，不能冒充空候选。 */
export function verificationCandidatePage<
  T extends { id?: string; organizationId?: string },
>(
  response:
    | { success?: boolean; data?: T[]; total?: number | string }
    | undefined,
  message: string,
) {
  if (
    !response ||
    response.success === false ||
    (response.data !== undefined && !Array.isArray(response.data)) ||
    (response.data === undefined && response.success !== true)
  )
    throw new Error(message);
  const data = response.data ?? [];
  if (data.some((row) => !row?.id || !row.organizationId))
    throw new Error(message);
  const total = Number(response.total ?? 0);
  if (!Number.isSafeInteger(total) || total < data.length)
    throw new Error(message);
  return { data, total, success: true };
}

/** 跨页选择保留显式已选来源快照，不把当前页之外的其他记录加入核销。 */
export function verificationSelectedSnapshots<T extends { id?: string }>(
  keys: Key[],
  rows: T[],
  previous: T[],
): T[] {
  const byId = new Map(previous.map((row) => [row.id, row]));
  for (const row of rows) {
    if (row?.id) byId.set(row.id, row);
  }
  return keys.map((key) => {
    const row = byId.get(String(key));
    if (!row) throw new Error('所选核销来源缺少数据，请重新选择');
    return row;
  });
}
