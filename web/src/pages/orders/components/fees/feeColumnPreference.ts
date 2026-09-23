/**
 * 订单费用页列设置：列清单、默认显隐顺序与同浏览器偏好存取。
 *
 * 偏好按「用户 ID + 组织 ID」隔离在 localStorage，应收/应付两表共享一份，
 * 切换订单继续生效；只存列 key、显隐与顺序，不含任何费用数据或身份秘密。
 */

/** 订单费用表稳定列 key（与表格列 dataIndex 一一对应；操作列固定为 option）。 */
export const FEE_COLUMN_KEYS = [
  'status',
  'feeCode',
  'feeSettingId',
  'settlementPartyId',
  'currency',
  'unitPrice',
  'netUnitPrice',
  'quantity',
  'billingUnitId',
  'totalAmount',
  'exchangeRate',
  'expenseDate',
  'note',
  'taxRate',
  'taxAmount',
  'netAmount',
  'baseCurrencyAmount',
  'tags',
  'billNo',
  'financialProgress',
  'option',
] as const;

export type FeeColumnKey = (typeof FEE_COLUMN_KEYS)[number];

/** 单列设置元数据：标题、是否必显、默认是否展示、是否依赖财务读取权限。 */
export interface FeeColumnDef {
  key: FeeColumnKey;
  title: string;
  /** 录入必备列，用户不可隐藏，避免必填编辑项丢失。 */
  lockVisible: boolean;
  defaultVisible: boolean;
  /** 仅在具备财务费用读取权限时提供（账单号、关联账单财务进度）。 */
  financeOnly?: boolean;
}

/** 默认列保持当前表格相对顺序；费用标签与关联账单列默认隐藏，含税单价折算列与税额四列默认可见。 */
export const DEFAULT_FEE_COLUMN_DEFS: FeeColumnDef[] = [
  { key: 'status', title: '状态', lockVisible: false, defaultVisible: true },
  {
    key: 'feeCode',
    // 行内选科时以费用代码列作为代码预览载体，默认保持可见（仍可隐藏）。
    title: '费用代码',
    lockVisible: false,
    defaultVisible: true,
  },
  {
    key: 'feeSettingId',
    title: '费用名称',
    lockVisible: true,
    defaultVisible: true,
  },
  {
    key: 'settlementPartyId',
    title: '结算单位',
    lockVisible: true,
    defaultVisible: true,
  },
  { key: 'currency', title: '币种', lockVisible: true, defaultVisible: true },
  { key: 'unitPrice', title: '单价', lockVisible: true, defaultVisible: true },
  {
    key: 'netUnitPrice',
    title: '不含税单价',
    lockVisible: false,
    defaultVisible: true,
  },
  { key: 'quantity', title: '数量', lockVisible: true, defaultVisible: true },
  {
    key: 'billingUnitId',
    title: '计费单位',
    lockVisible: true,
    defaultVisible: true,
  },
  {
    key: 'totalAmount',
    title: '总金额',
    lockVisible: false,
    defaultVisible: true,
  },
  {
    key: 'exchangeRate',
    title: '汇率',
    lockVisible: false,
    defaultVisible: true,
  },
  {
    key: 'expenseDate',
    title: '发生日期',
    lockVisible: true,
    defaultVisible: true,
  },
  { key: 'note', title: '备注', lockVisible: false, defaultVisible: true },
  {
    key: 'taxRate',
    title: '税率(%)',
    lockVisible: false,
    defaultVisible: true,
  },
  {
    key: 'taxAmount',
    title: '税金',
    lockVisible: false,
    defaultVisible: true,
  },
  {
    key: 'netAmount',
    title: '不含税总额',
    lockVisible: false,
    defaultVisible: true,
  },
  {
    key: 'baseCurrencyAmount',
    title: '折本币金额',
    lockVisible: false,
    defaultVisible: true,
  },
  {
    key: 'tags',
    title: '费用标签',
    lockVisible: false,
    defaultVisible: false,
  },
  {
    key: 'billNo',
    title: '账单号',
    lockVisible: false,
    defaultVisible: false,
    financeOnly: true,
  },
  {
    key: 'financialProgress',
    title: '关联账单财务进度',
    lockVisible: false,
    defaultVisible: false,
    financeOnly: true,
  },
  { key: 'option', title: '操作', lockVisible: true, defaultVisible: true },
];

/** 用户提交的列偏好：完整顺序 + 被隐藏的列。 */
export interface FeeColumnPreference {
  order: FeeColumnKey[];
  hidden: FeeColumnKey[];
}

/** 列偏好在同一浏览器内的隔离范围：用户 + 当前组织。 */
export interface FeeColumnPreferenceScope {
  userId?: string;
  organizationId?: string;
}

/** 当前上下文实际可用的列（财务权限列按需裁剪），保持默认顺序。 */
export function effectiveFeeColumnDefs(financeAvailable: boolean) {
  return DEFAULT_FEE_COLUMN_DEFS.filter(
    (def) => !def.financeOnly || financeAvailable,
  );
}

/** 默认偏好：全部列按默认顺序排列，默认隐藏的列进入 hidden。 */
export function defaultFeeColumnPreference(
  financeAvailable: boolean,
): FeeColumnPreference {
  const defs = effectiveFeeColumnDefs(financeAvailable);
  return {
    order: defs.map((def) => def.key),
    hidden: defs.filter((def) => !def.defaultVisible).map((def) => def.key),
  };
}

function isSamePreference(a: FeeColumnPreference, b: FeeColumnPreference) {
  return (
    a.order.length === b.order.length &&
    a.order.every((key, index) => key === b.order[index]) &&
    a.hidden.length === b.hidden.length &&
    a.hidden.every((key) => b.hidden.includes(key))
  );
}

/** 判断偏好是否等价于默认列（用于“恢复默认”时显式删除本地偏好）。 */
export function isDefaultFeeColumnPreference(
  preference: FeeColumnPreference,
  financeAvailable: boolean,
): boolean {
  return isSamePreference(
    preference,
    defaultFeeColumnPreference(financeAvailable),
  );
}

/**
 * 合并存储偏好与当前可用列：
 * - 忽略未知 key 与损坏结构；
 * - 偏好未提及的新增列按其默认可见性兜底（默认隐藏的可选列不会因升级突然出现）；
 * - 必显列无论存储内容如何都不允许隐藏。
 */
export function resolveFeeColumnPreference(
  stored: FeeColumnPreference | null | undefined,
  financeAvailable: boolean,
): FeeColumnPreference {
  if (!stored) return defaultFeeColumnPreference(financeAvailable);
  const defs = effectiveFeeColumnDefs(financeAvailable);
  const knownKeys = new Set<FeeColumnKey>(defs.map((def) => def.key));
  const lockedVisible = new Set<FeeColumnKey>(
    defs.filter((def) => def.lockVisible).map((def) => def.key),
  );

  const storedOrder = (stored.order ?? []).filter((key) =>
    knownKeys.has(key as FeeColumnKey),
  ) as FeeColumnKey[];
  const orderedKeys = new Set(storedOrder);
  const order: FeeColumnKey[] = [
    ...storedOrder,
    ...defs.filter((def) => !orderedKeys.has(def.key)).map((def) => def.key),
  ];

  const mentioned = new Set<string>([
    ...(stored.order ?? []),
    ...(stored.hidden ?? []),
  ]);
  const hidden = new Set<FeeColumnKey>(
    ((stored.hidden ?? []) as string[]).filter(
      (key: string) =>
        knownKeys.has(key as FeeColumnKey) &&
        !lockedVisible.has(key as FeeColumnKey),
    ) as FeeColumnKey[],
  );
  for (const def of defs) {
    if (!mentioned.has(def.key) && !def.defaultVisible) {
      hidden.add(def.key);
    }
  }
  return { order, hidden: order.filter((key) => hidden.has(key)) };
}

function feeColumnStorageKey(scope: FeeColumnPreferenceScope): string {
  return `roncin:order-fee-columns:v1:${scope.userId || 'anonymous'}:${
    scope.organizationId || 'default'
  }`;
}

function isFeeColumnKeyArray(value: unknown): value is string[] {
  return (
    Array.isArray(value) &&
    value.every((key) => typeof key === 'string' && key.length > 0)
  );
}

/** 读取本浏览器偏好；结构损坏或存储不可用时回退 null（即默认列）。 */
export function loadFeeColumnPreference(
  scope: FeeColumnPreferenceScope,
): FeeColumnPreference | null {
  try {
    const raw = window.localStorage.getItem(feeColumnStorageKey(scope));
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== 'object' || parsed === null) return null;
    const { order, hidden } = parsed as Record<string, unknown>;
    if (!isFeeColumnKeyArray(order) || !isFeeColumnKeyArray(hidden)) {
      return null;
    }
    return {
      order: order as FeeColumnKey[],
      hidden: hidden as FeeColumnKey[],
    };
  } catch {
    return null;
  }
}

/** 保存偏好；返回是否持久化成功，失败时调用方需提示本次设置未持久保存。 */
export function saveFeeColumnPreference(
  scope: FeeColumnPreferenceScope,
  preference: FeeColumnPreference,
): boolean {
  try {
    window.localStorage.setItem(
      feeColumnStorageKey(scope),
      JSON.stringify({ order: preference.order, hidden: preference.hidden }),
    );
    return true;
  } catch {
    return false;
  }
}

/** 删除本场景偏好（恢复默认时调用）。 */
export function clearFeeColumnPreference(
  scope: FeeColumnPreferenceScope,
): boolean {
  try {
    window.localStorage.removeItem(feeColumnStorageKey(scope));
    return true;
  } catch {
    return false;
  }
}
