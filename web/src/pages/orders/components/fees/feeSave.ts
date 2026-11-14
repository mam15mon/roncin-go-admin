import Decimal from 'decimal.js';
import { normalizeDecimalInput, quantityOrPricePattern } from '@/utils/decimal';
import { feeQuantityRuleError } from './feeQuantityRule';
import { quantitySelectionForSave } from './feeQuantitySource';

/** 整表保存的纯逻辑：改动判定、行校验、请求体构造、请求键管理与错误解析。 */

/** 税率输入口径与服务端一致：0-100，最多两位小数。 */
export const taxRateInputPattern = /^(0|[1-9][0-9]{0,2})(\.[0-9]{1,2})?$/;

/** 页面内新增行的临时主键前缀；服务端落库前它同时是请求的 client_row_id。 */
export const NEW_FEE_ROW_PREFIX = 'new_';

export function isNewFeeRowKey(key: unknown): boolean {
  return String(key ?? '').startsWith(NEW_FEE_ROW_PREFIX);
}

/** 表单中一行的可编辑字段（保存前以 baseline 兜底补齐未渲染字段）。 */
export type FeeDraftRow = Partial<API.OrderFee>;

export type FeeRowIssue = { field: string; message: string };

function isDecimalText(value: string): boolean {
  return /^-?\d+(\.\d+)?$/.test(value);
}

/** 十进制文本等值比较：尾零不同视为相同；任一方无法解析时回退到字符串比较。 */
export function sameDecimalText(a?: string | null, b?: string | null): boolean {
  const left = normalizeDecimalInput(String(a ?? ''));
  const right = normalizeDecimalInput(String(b ?? ''));
  if (left === right) return true;
  if (isDecimalText(left) && isDecimalText(right)) {
    return new Decimal(left).eq(new Decimal(right));
  }
  return false;
}

function sameText(a?: string | null, b?: string | null): boolean {
  return String(a ?? '').trim() === String(b ?? '').trim();
}

/** 税率是否相对基线发生更改（空白输入视为未输入，不构成更改）。 */
export function taxRateChanged(
  row: FeeDraftRow,
  baselineTaxRate?: string,
): boolean {
  const input = String(row.taxRate ?? '').trim();
  if (input === '') return false;
  return !sameDecimalText(input, baselineTaxRate);
}

/**
 * 已保存行相对基线是否有改动。未改动行不得提交（服务端按「提交即修改」处理，
 * 无谓提交会递增版本并重算提成）。
 */
export function feeRowChanged(
  row: FeeDraftRow,
  baseline: API.OrderFee,
): boolean {
  return (
    !sameText(row.feeSettingId, baseline.feeSettingId) ||
    !sameText(row.settlementPartyId, baseline.settlementPartyId) ||
    !sameText(row.currency, baseline.currency) ||
    !sameText(row.billingUnitId, baseline.billingUnitId) ||
    !sameDecimalText(row.quantity, baseline.quantity) ||
    !sameDecimalText(row.unitPrice, baseline.unitPrice) ||
    !sameText(row.note, baseline.note) ||
    taxRateChanged(row, baseline.taxRate) ||
    JSON.stringify(row.quantitySource ?? null) !==
      JSON.stringify(baseline.quantitySource ?? null)
  );
}

/** 单行提交前校验；返回第一个问题及其所在字段。 */
export function validateFeeRow(
  row: FeeDraftRow,
  baseline: API.OrderFee | undefined,
  billingUnits: API.BillingUnit[],
  taxRateEditable: boolean,
): FeeRowIssue | undefined {
  if (!row.feeSettingId)
    return { field: 'feeSettingId', message: '请选择费用项目' };
  if (!row.settlementPartyId)
    return { field: 'settlementPartyId', message: '请选择结算单位' };
  if (!row.currency) return { field: 'currency', message: '请选择币种' };
  if (
    !row.unitPrice ||
    !quantityOrPricePattern.test(
      normalizeDecimalInput(String(row.unitPrice)),
    ) ||
    Number(row.unitPrice) <= 0
  )
    return { field: 'unitPrice', message: '单价必须为大于 0 的有效数值' };
  if (
    !row.quantity ||
    !quantityOrPricePattern.test(normalizeDecimalInput(String(row.quantity))) ||
    Number(row.quantity) <= 0
  )
    return { field: 'quantity', message: '数量必须为大于 0 的有效数值' };
  if (!row.billingUnitId)
    return { field: 'billingUnitId', message: '请选择计费单位' };
  const quantityRuleError = feeQuantityRuleError(
    row.quantity,
    row.billingUnitId,
    billingUnits,
    baseline,
  );
  if (quantityRuleError)
    return { field: 'quantity', message: quantityRuleError };
  const taxInput = String(row.taxRate ?? '').trim();
  if (taxRateEditable && taxInput !== '') {
    const normalized = normalizeDecimalInput(taxInput);
    if (!taxRateInputPattern.test(normalized) || Number(normalized) > 100) {
      return { field: 'taxRate', message: '税率必须为 0-100 且最多两位小数' };
    }
  }
  return undefined;
}

/**
 * 构造一行提交体。税率仅在确有更改时携带：已保存行与基线比较，新增行与所选费用项目
 * 的目录税率比较（相同即无更改，避免无权限用户因回显值触发 403）。
 */
export function buildSaveRow(args: {
  clientRowId: string;
  row: FeeDraftRow;
  baseline?: API.OrderFee;
  catalogTaxRate?: string;
}): API.SaveOrderFeeRow {
  const { clientRowId, row, baseline, catalogTaxRate } = args;
  const saveRow: API.SaveOrderFeeRow = {
    clientRowId,
    feeSettingId: row.feeSettingId ?? '',
    settlementPartyId: row.settlementPartyId ?? '',
    billingUnitId: row.billingUnitId ?? '',
    quantity: normalizeDecimalInput(String(row.quantity ?? '')),
    quantitySource: quantitySelectionForSave(row.quantitySource, row.quantity),
    unitPrice: normalizeDecimalInput(String(row.unitPrice ?? '')),
    currency: row.currency ?? '',
    note: row.note?.trim() || undefined,
    taxInclusive: true,
  };
  if (baseline) {
    saveRow.feeId = baseline.id;
    saveRow.expectedVersion = baseline.version;
  }
  const taxBaseline = baseline ? baseline.taxRate : catalogTaxRate;
  if (taxRateChanged(row, taxBaseline)) {
    saveRow.taxRate = normalizeDecimalInput(String(row.taxRate));
  }
  return saveRow;
}

/**
 * 同一保存意图沿用同一 request_key：内容完全一致（响应丢失或失败后原样重试）沿用，
 * 内容变化则换新键，避免「同键不同内容」的幂等冲突；成功后清除，下一次保存必换键。
 */
export class SaveRequestKeyHolder {
  private current?: { fingerprint: string; key: string };

  constructor(private readonly generate: () => string) {}

  keyFor(rows: API.SaveOrderFeeRow[]): string {
    const fingerprint = JSON.stringify(rows);
    if (this.current?.fingerprint !== fingerprint) {
      this.current = { fingerprint, key: this.generate() };
    }
    return this.current.key;
  }

  clear() {
    this.current = undefined;
  }
}

export type FeeSaveRowError = {
  clientRowId: string;
  field: string;
  reason?: string;
  message: string;
};

export type ParsedFeeSaveError = {
  /** 服务端已明确拒绝（收到业务报文）；否则可能是超时或网络中断，结果未知。 */
  rejected: boolean;
  reason?: string;
  message: string;
  rowErrors: FeeSaveRowError[];
  metadata?: Record<string, string>;
};

type ErrorLike = {
  message?: string;
  data?: ErrorEnvelopeLike;
  response?: { status?: number; data?: ErrorEnvelopeLike };
};
type ErrorEnvelopeLike = {
  code?: number;
  message?: string;
  reason?: string;
  metadata?: Record<string, string>;
};

/** 解析保存失败：行级定位信息取自错误 metadata 的 client_row_id/field/reason/row_errors。 */
export function parseFeeSaveError(error: unknown): ParsedFeeSaveError {
  const source = (error ?? {}) as ErrorLike;
  const envelope = source.data ?? source.response?.data;
  if (!envelope || (!envelope.reason && !envelope.message)) {
    return {
      rejected: false,
      message: '保存结果未确认，请检查网络；原样重试不会重复新增',
      rowErrors: [],
    };
  }
  const metadata = envelope.metadata;
  const rowErrors: FeeSaveRowError[] = [];
  if (metadata?.row_errors) {
    try {
      const parsed: unknown = JSON.parse(metadata.row_errors);
      if (Array.isArray(parsed)) {
        for (const item of parsed as Array<Record<string, unknown>>) {
          if (typeof item?.client_row_id !== 'string') continue;
          rowErrors.push({
            clientRowId: item.client_row_id,
            field: typeof item.field === 'string' ? item.field : '',
            reason: typeof item.reason === 'string' ? item.reason : undefined,
            message:
              typeof item.message === 'string' ? item.message : '保存失败',
          });
        }
      }
    } catch {
      // row_errors 损坏时退回单行信息，不影响整体失败展示。
    }
  }
  if (rowErrors.length === 0 && metadata?.client_row_id) {
    rowErrors.push({
      clientRowId: metadata.client_row_id,
      field: metadata.field ?? '',
      reason: metadata.reason ?? envelope.reason,
      message: envelope.message ?? '保存失败',
    });
  }
  return {
    rejected: true,
    reason: envelope.reason,
    message: envelope.message ?? '保存失败，输入已保留',
    rowErrors,
    metadata,
  };
}
