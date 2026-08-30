import { describe, expect, it } from 'vitest';
import { RECEIVABLE } from './feeConstants';
import { feeBaseColumns } from './feeBaseColumns';

describe('feeBaseColumns', () => {
  it('保留费用工作台的列顺序与宽度', () => {
    const columns = feeBaseColumns({
      variant: 'workbench',
      direction: RECEIVABLE,
    });

    expect(columns.map((column) => column.dataIndex)).toEqual([
      'status',
      'feeCode',
      'feeName',
      'settlementPartyName',
      'currency',
      'unitPrice',
      'quantity',
      'billingUnit',
      'totalAmount',
      'exchangeRate',
      'expenseDate',
      'note',
    ]);
    expect(columns.map((column) => column.width)).toEqual([
      90,
      120,
      140,
      180,
      80,
      100,
      80,
      90,
      130,
      100,
      220,
      undefined,
    ]);
  });

  it('保留费用抽屉的方向列与展示密度', () => {
    const columns = feeBaseColumns({ variant: 'panel' });

    expect(columns.map((column) => column.dataIndex)).toEqual([
      'status',
      'direction',
      'feeCode',
      'feeName',
      'settlementPartyName',
      'billingUnit',
      'quantity',
      'unitPrice',
      'totalAmount',
      'exchangeRate',
      'expenseDate',
      'note',
    ]);
    expect(columns.map((column) => column.width)).toEqual([
      90,
      90,
      130,
      150,
      190,
      90,
      110,
      130,
      150,
      160,
      110,
      180,
    ]);
  });
});
