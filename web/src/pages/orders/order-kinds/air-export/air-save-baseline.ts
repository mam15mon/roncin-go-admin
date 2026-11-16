/** 正文中的非 optional 标量由 ProtoJSON 省略零值，按实际 wire 语义比较。 */
export function sameAirWaybillContent(
  left?: API.AirWaybillContent,
  right?: API.AirWaybillContent,
): boolean {
  const normalize = (value: unknown): unknown => {
    if (value == null || value === '' || value === false) return undefined;
    if (Array.isArray(value))
      return value.length ? value.map(normalize) : undefined;
    if (typeof value !== 'object') return value;
    const entries = Object.entries(value)
      .sort(([a], [b]) => a.localeCompare(b))
      .flatMap(([key, entry]) => {
        const normalized = normalize(entry);
        return normalized === undefined ? [] : [[key, normalized]];
      });
    return entries.length ? Object.fromEntries(entries) : undefined;
  };
  // 运费、其他费归属合计及总计由服务端按明细派生；编辑基线比较只看真实可编辑事实。
  const editable = (content?: API.AirWaybillContent) =>
    content
      ? {
          ...content,
          prepaid: content.prepaid
            ? {
                valuationCharge: content.prepaid.valuationCharge,
                tax: content.prepaid.tax,
              }
            : undefined,
          collect: content.collect
            ? {
                valuationCharge: content.collect.valuationCharge,
                tax: content.collect.tax,
              }
            : undefined,
        }
      : undefined;
  return (
    JSON.stringify(normalize(editable(left))) ===
    JSON.stringify(normalize(editable(right)))
  );
}

function versionAdvanced(
  previous: string | undefined,
  saved: string | undefined,
  increment: bigint,
): boolean {
  return Boolean(
    previous &&
      saved &&
      /^\d+$/.test(previous) &&
      /^\d+$/.test(saved) &&
      BigInt(saved) === BigInt(previous) + increment,
  );
}

/** 只采纳本次整票写入的版本，不能把提交后重读到的其他写入借给旧输入。 */
export function acceptsAirSaveBaseline(
  orderId: string,
  organizationId: string | undefined,
  previous: API.AirOrderInfo | undefined,
  payload: API.UpdateOrderRequest,
  saved: API.Order,
): boolean {
  const air = saved.airExport;
  if (
    !organizationId ||
    saved.id !== orderId ||
    saved.organizationId !== organizationId ||
    !versionAdvanced(payload.expectedVersion, saved.version, 1n) ||
    !previous?.master?.id ||
    previous.master.id !== air?.master?.id ||
    !versionAdvanced(previous.master.version, air?.master?.version, 0n) ||
    previous.documentMode !== air?.documentMode ||
    (previous.active ?? false) !== (air?.active ?? false) ||
    !previous.linkId ||
    previous.linkId !== air?.linkId
  )
    return false;
  if (previous.documentMode === 'DIRECT')
    return (
      !previous.house &&
      !air?.house &&
      versionAdvanced(previous.linkVersion, air?.linkVersion, 0n)
    );
  const input = payload.airExport?.house;
  const house = previous.house;
  const savedHouse = air?.house;
  if (
    previous.documentMode !== 'HOUSE' ||
    !input ||
    !house?.id ||
    house.id !== savedHouse?.id
  )
    return false;
  const content = input.content ?? house.content;
  const status = input.status ?? house.status;
  const houseNo = input.houseNo ?? house.houseNo;
  const changed =
    houseNo !== house.houseNo ||
    status !== house.status ||
    !sameAirWaybillContent(content, house.content);
  return (
    savedHouse?.houseNo === houseNo &&
    savedHouse.status === status &&
    sameAirWaybillContent(savedHouse.content, content) &&
    versionAdvanced(house.version, savedHouse.version, changed ? 1n : 0n) &&
    versionAdvanced(previous.linkVersion, air?.linkVersion, changed ? 1n : 0n)
  );
}
