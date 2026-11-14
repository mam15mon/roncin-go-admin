import { describe, expect, it } from 'vitest';
import {
  buildSaveRow,
  feeRowChanged,
  parseFeeSaveError,
  SaveRequestKeyHolder,
  taxRateChanged,
  validateFeeRow,
} from './feeSave';

const baseline: API.OrderFee = {
  id: 'f1',
  version: '4',
  feeSettingId: 's1',
  settlementPartyId: 'p1',
  billingUnitId: 'u1',
  currency: 'CNY',
  quantity: '1.0000',
  unitPrice: '100.0000',
  taxRate: '6.00',
};
const units: API.BillingUnit[] = [{ id: 'u1', name: '票' }];

describe('整表保存纯逻辑', () => {
  it('改动判定：尾零不同不算改动，真实改动才算', () => {
    expect(feeRowChanged({ ...baseline, quantity: '1' }, baseline)).toBe(false);
    expect(feeRowChanged({ ...baseline, note: '  ' }, baseline)).toBe(false);
    expect(feeRowChanged({ ...baseline, unitPrice: '100.5' }, baseline)).toBe(
      true,
    );
    expect(feeRowChanged({ ...baseline, currency: 'USD' }, baseline)).toBe(
      true,
    );
  });

  it('税率：空白视为未输入；与基线数值相同不算更改', () => {
    expect(taxRateChanged({ taxRate: '' }, '6.00')).toBe(false);
    expect(taxRateChanged({ taxRate: '6' }, '6.00')).toBe(false);
    expect(taxRateChanged({ taxRate: '9' }, '6.00')).toBe(true);
    expect(taxRateChanged({ taxRate: '6' }, undefined)).toBe(true);
  });

  it('新增行按目录税率判定，已保存行按自身基线判定', () => {
    const added = buildSaveRow({
      clientRowId: 'new_1',
      row: { ...baseline, id: undefined, taxRate: '9' },
      catalogTaxRate: '6.00',
    });
    expect(added).not.toHaveProperty('feeId');
    expect(added.taxRate).toBe('9');
    const same = buildSaveRow({
      clientRowId: 'new_2',
      row: { ...baseline, taxRate: '6' },
      catalogTaxRate: '6.00',
    });
    expect(same).not.toHaveProperty('taxRate');
    const edited = buildSaveRow({
      clientRowId: 'f1',
      row: { ...baseline, taxRate: '0' },
      baseline,
    });
    expect(edited).toMatchObject({
      feeId: 'f1',
      expectedVersion: '4',
      taxRate: '0',
    });
  });

  it('提交前校验：必填、数量规则与税率范围', () => {
    expect(validateFeeRow(baseline, baseline, units, true)).toBeUndefined();
    expect(
      validateFeeRow({ ...baseline, unitPrice: '0' }, baseline, units, true),
    ).toMatchObject({ field: 'unitPrice' });
    expect(
      validateFeeRow({ ...baseline, taxRate: '100.5' }, baseline, units, true),
    ).toMatchObject({ field: 'taxRate' });
    // 无税率权限时不校验税率输入（回显值不参与提交）。
    expect(
      validateFeeRow({ ...baseline, taxRate: '999' }, baseline, units, false),
    ).toBeUndefined();
  });

  it('请求键：同内容沿用，内容变化换新，成功清除后换新', () => {
    let n = 0;
    const holder = new SaveRequestKeyHolder(() => `key-${++n}`);
    const rowsA = [
      buildSaveRow({ clientRowId: 'f1', row: baseline, baseline }),
    ];
    const rowsB = [
      buildSaveRow({
        clientRowId: 'f1',
        row: { ...baseline, unitPrice: '1' },
        baseline,
      }),
    ];
    expect(holder.keyFor(rowsA)).toBe('key-1');
    expect(holder.keyFor(rowsA)).toBe('key-1');
    expect(holder.keyFor(rowsB)).toBe('key-2');
    holder.clear();
    expect(holder.keyFor(rowsB)).toBe('key-3');
  });

  it('错误解析：单行 metadata、row_errors 数组、无 metadata 与无响应', () => {
    const single = parseFeeSaveError({
      response: {
        status: 400,
        data: {
          reason: 'R',
          message: 'M',
          metadata: { client_row_id: 'f1', field: 'currency', reason: 'R' },
        },
      },
    });
    expect(single.rejected).toBe(true);
    expect(single.rowErrors).toEqual([
      { clientRowId: 'f1', field: 'currency', reason: 'R', message: 'M' },
    ]);
    const multi = parseFeeSaveError({
      response: {
        data: {
          reason: 'R',
          message: 'M',
          metadata: {
            client_row_id: 'f1',
            row_errors: JSON.stringify([
              { client_row_id: 'f1', field: 'a', reason: 'R', message: 'm1' },
              { client_row_id: 'f2', field: 'b', reason: 'R', message: 'm2' },
              { field: 'ignored' },
            ]),
          },
        },
      },
    });
    expect(multi.rowErrors.map((item) => item.clientRowId)).toEqual([
      'f1',
      'f2',
    ]);
    // 未带 metadata：仍是明确拒绝，只展示整体消息，不凭空造行。
    const bare = parseFeeSaveError({
      response: { data: { reason: 'R', message: 'M' } },
    });
    expect(bare).toMatchObject({ rejected: true, message: 'M', rowErrors: [] });
    expect(parseFeeSaveError(new Error('timeout')).rejected).toBe(false);
  });
});
