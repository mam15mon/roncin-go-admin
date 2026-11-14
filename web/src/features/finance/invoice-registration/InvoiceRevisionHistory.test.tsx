import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { describe, expect, it } from 'vitest';
import InvoiceRevisionHistory from './InvoiceRevisionHistory';

const snapshot = JSON.stringify({
  number: 'INV-1',
  date: '2026-11-14',
  invoiceType: 'NORMAL',
  status: 'ISSUED',
  invoiceTitle: '原票抬头',
  currency: 'CNY',
  totalAmount: '100.00000000',
  netAmount: '93.00000000',
  taxAmount: '7.00000000',
  bills: [
    {
      billNo: 'BI-1',
      amount: '100.00000000',
      taxAmount: '7.00000000',
      fees: [
        {
          code: 'FEE',
          name: '海运费',
          amount: '100.00000000',
          netAmount: '93.00000000',
          taxAmount: '7.00000000',
        },
      ],
    },
  ],
});
describe('发票修订历史', () => {
  it('用业务快照展示修改前后，使用本地可读时间，不渲染JSON、内部版本或UUID', async () => {
    const record: API.FinanceInvoiceRevision = {
      id: 'internal-history-id',
      revisionNumber: '99',
      actorName: '财务甲',
      reason: '核对原票后修订',
      createdAt: '2026-11-14T08:00:00Z',
      beforeSnapshot: snapshot,
      afterSnapshot: snapshot,
    };
    renderWithApp(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <InvoiceRevisionHistory revisions={[record]} />
      </ConfigProvider>,
    );
    fireEvent.click(screen.getByText(/财务甲 · 核对原票后修订/));
    expect(await screen.findByText('修改前')).toBeInTheDocument();
    expect(screen.getByText('修改后')).toBeInTheDocument();
    expect(screen.getAllByText('原票抬头')).toHaveLength(2);
    expect(screen.queryByText(snapshot)).not.toBeInTheDocument();
    expect(screen.queryByText('internal-history-id')).not.toBeInTheDocument();
    expect(screen.queryByText('99')).not.toBeInTheDocument();
    expect(screen.queryByText('2026-11-14T08:00:00Z')).not.toBeInTheDocument();
  });
  it('损坏业务快照明确失败，不伪装空历史', async () => {
    renderWithApp(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <InvoiceRevisionHistory
          revisions={[
            {
              id: 'bad',
              actorName: '财务乙',
              reason: '核对',
              beforeSnapshot: '{}',
              afterSnapshot: 'null',
            },
          ]}
        />
      </ConfigProvider>,
    );
    fireEvent.click(screen.getByText(/财务乙 · 核对/));
    expect(
      await screen.findByText('修订快照不完整，请联系管理员核对'),
    ).toBeInTheDocument();
    expect(screen.getByText('新增登记，此前无记录')).toBeInTheDocument();
  });
  it('显示修订前后的原始汇率快照与折本币金额，不以当前值覆盖历史', async () => {
    const before = {
      ...JSON.parse(snapshot),
      currency: 'USD',
      baseCurrency: 'CNY',
      exchangeRate: '7.10000000',
      exchangeRateSource: 'BOC_SYNC',
      exchangeRateDate: '2026-09-10',
      baseCurrencyAmount: '710.00000000',
    };
    renderWithApp(
      <InvoiceRevisionHistory
        revisions={[
          {
            id: 'economic-history',
            actorName: '财务丙',
            reason: '更正票据日期',
            beforeSnapshot: JSON.stringify(before),
            afterSnapshot: JSON.stringify({
              ...before,
              exchangeRate: '7.20000000',
              exchangeRateSource: 'WEEKLY',
              exchangeRateDate: '2026-11-14',
              baseCurrencyAmount: '720.00000000',
            }),
          },
        ]}
      />,
    );
    fireEvent.click(screen.getByText(/财务丙 · 更正票据日期/));
    expect(await screen.findByText('7.1')).toBeInTheDocument();
    expect(screen.getByText('7.2')).toBeInTheDocument();
    expect(screen.getByText('710.00 CNY')).toBeInTheDocument();
    expect(screen.getByText('720.00 CNY')).toBeInTheDocument();
    expect(screen.getByText('中行同步')).toBeInTheDocument();
    expect(screen.getByText('公司周汇率')).toBeInTheDocument();
    expect(screen.getByText('2026-09-10')).toBeInTheDocument();
    expect(screen.queryByText('7.10000000')).not.toBeInTheDocument();
  });
  it('移除后的空分配数组仍是可审阅历史，不能误报快照损坏', async () => {
    renderWithApp(
      <InvoiceRevisionHistory
        revisions={[
          {
            id: 'removed-history',
            actorName: '财务丁',
            reason: '移除错误票据',
            beforeSnapshot: snapshot,
            afterSnapshot: JSON.stringify({
              ...JSON.parse(snapshot),
              status: 'CANCELLED',
              bills: [],
            }),
          },
        ]}
      />,
    );
    fireEvent.click(screen.getByText(/财务丁 · 移除错误票据/));
    expect(await screen.findByText('已移除，原记录保留')).toBeInTheDocument();
    expect(
      screen.queryByText('修订快照不完整，请联系管理员核对'),
    ).not.toBeInTheDocument();
  });
});
