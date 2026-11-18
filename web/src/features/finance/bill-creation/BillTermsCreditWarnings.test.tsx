import { render, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it } from 'vitest';
import BillTermsCreditWarnings from './BillTermsCreditWarnings';
import { billDueResult, initialBillDueSettings } from './billDueSettings';

describe('散客按最终到期日提醒', () => {
  it.each([
    ['账期10天', { paymentTermsDays: 10 }, true, 'RECEIVABLE', true],
    ['指定晚于账单日', { dueDate: '2026-11-28' }, true, 'RECEIVABLE', true],
    ['指定当天', { dueDate: '2026-11-18' }, true, 'RECEIVABLE', false],
    ['正式客户', { dueDate: '2026-11-28' }, false, 'RECEIVABLE', false],
    ['散客应付', { dueDate: '2026-11-28' }, true, 'PAYABLE', false],
  ])('%s', (_, source, isCasual, direction, warning) => {
    const billDate = '2026-11-18';
    const { effectiveDueDate } = billDueResult(
      billDate,
      initialBillDueSettings(source),
    );
    render(
      <BillTermsCreditWarnings
        billDate={billDate}
        effectiveDueDate={effectiveDueDate}
        isCasual={isCasual}
        direction={direction}
      />,
    );
    const alert = screen.queryByText(/该客户为单次合作散客，建议现结/);
    if (warning)
      expect(alert).toHaveTextContent(
        '当前到期日为 2026-11-28（账单日后 10 天）',
      );
    else expect(alert).not.toBeInTheDocument();
  });
});
