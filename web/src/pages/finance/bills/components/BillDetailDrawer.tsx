import { Descriptions, Space, Table, type TableColumnsType, Tag } from 'antd';
import React from 'react';
import { DescriptionsDetailDrawer, DItem } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { statusTag } from '@/constants/statusMeta';
import { FinanceBillStatus } from '@/enums.generated';
import { billStatusMeta } from '@/features/finance/bill-status';
import { formatAmount, trimDecimal } from '@/utils/format';

interface BillDetailDrawerProps {
  open: boolean;
  loading: boolean;
  detail?: API.FinanceBill;
  onClose: () => void;
}

export default function BillDetailDrawer({
  open,
  loading,
  detail,
  onClose,
}: BillDetailDrawerProps) {
  const lineColumns: TableColumnsType<API.FinanceBillLine> = [
    { title: '订单编号', dataIndex: 'orderNo', width: 150 },
    { title: '费用代码', dataIndex: 'feeCode', width: 100 },
    { title: '费用名称', dataIndex: 'feeName', width: 130 },
    {
      title: '税率',
      dataIndex: 'taxRate',
      align: 'right',
      width: 80,
      render: (value) => (value == null ? '-' : `${Number(value)}%`),
    },
    {
      title: '不含税金额',
      dataIndex: 'netAmount',
      align: 'right',
      render: (val, row) =>
        val ? `${formatAmount(val)} ${row.currency}` : '-',
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      align: 'right',
      render: (val, row) =>
        val ? `${formatAmount(val)} ${row.currency}` : '-',
    },
    {
      title: '含税金额',
      dataIndex: 'totalAmount',
      render: (_, row) => (
        <strong>
          {formatAmount(row.totalAmount)} {row.currency}
        </strong>
      ),
      align: 'right',
    },
    {
      title: '费用折本币',
      render: (_, row) =>
        `${formatAmount(row.baseCurrencyAmount)} ${row.baseCurrency}`,
      align: 'right',
    },
    {
      title: '关联状态',
      render: (_, row) =>
        row.active ? <Tag color="blue">有效</Tag> : <Tag>已释放</Tag>,
      width: 85,
    },
  ];

  const lineColumnSettings = useColumnSettings<
    TableColumnsType<API.FinanceBillLine>[number]
  >({
    tableKey: 'finance:bill-lines',
    columns: lineColumns,
  });

  return (
    <DescriptionsDetailDrawer
      title={(current) => `账单详情 ${current?.billNo || ''}`}
      open={open}
      detail={detail}
      size={1020}
      loading={loading}
      column={3}
      onClose={onClose}
      descriptions={(detail) => (
        <>
          <Descriptions.Item label="状态">
            {statusTag(
              billStatusMeta,
              detail.status ?? FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT,
            )}
          </Descriptions.Item>
          <Descriptions.Item label="方向">
            {detail.direction === 'RECEIVABLE' ? '应收' : '应付'}
          </Descriptions.Item>
          <Descriptions.Item label="所属公司">
            {detail.organizationName || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="结算单位">
            {detail.settlementPartyName}
          </Descriptions.Item>
          <Descriptions.Item label="结算账户">
            {detail.settlementAccountName ||
              detail.settlementAccountHolder ||
              '-'}
          </Descriptions.Item>
          <Descriptions.Item label="账户户名">
            {detail.settlementAccountHolder || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="开户银行">
            {detail.settlementBankName || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="银行账号">
            {detail.settlementBankAccount || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="账户币种">
            {detail.settlementAccountCurrency || '-'}
          </Descriptions.Item>
          {detail.settlementSwiftCode && (
            <Descriptions.Item label="SWIFT Code">
              {detail.settlementSwiftCode}
            </Descriptions.Item>
          )}
          <DItem label="建单批次">{detail.batchNo}</DItem>
          <DItem label="对账抬头">{detail.statementTitle}</DItem>
          <Descriptions.Item label="含税总额">
            <strong style={{ color: '#262626' }}>
              {formatAmount(detail.totalAmount)} {detail.currency}
            </strong>
          </Descriptions.Item>
          <Descriptions.Item label="不含税金额">
            {detail.netAmount
              ? `${formatAmount(detail.netAmount)} ${detail.currency}`
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="税额汇总">
            {detail.taxAmount
              ? `${formatAmount(detail.taxAmount)} ${detail.currency}`
              : '0.00'}
          </Descriptions.Item>
          <Descriptions.Item label="账单汇率">
            {detail.exchangeRate ? (
              <Space size={4}>
                <span>{trimDecimal(detail.exchangeRate)}</span>
                <Tag
                  color={
                    detail.exchangeRateSource === 'MANUAL'
                      ? 'purple'
                      : detail.exchangeRateSource === 'BASE_CURRENCY'
                        ? 'default'
                        : 'blue'
                  }
                >
                  {detail.exchangeRateSource === 'MANUAL'
                    ? '手工'
                    : detail.exchangeRateSource === 'BASE_CURRENCY'
                      ? '本币'
                      : '系统'}
                </Tag>
              </Space>
            ) : (
              '-'
            )}
          </Descriptions.Item>
          <Descriptions.Item label="折本币总额">
            <strong style={{ color: '#1677ff' }}>
              {formatAmount(detail.baseCurrencyAmount)} {detail.baseCurrency}
            </strong>
          </Descriptions.Item>
          <Descriptions.Item label="汇率生效日期">
            {detail.exchangeRateDate || detail.billDate || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="账单日期">
            {detail.billDate}
          </Descriptions.Item>
          <Descriptions.Item label="到期日">
            {detail.dueDate || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="账期">
            {detail.paymentTermsDays == null
              ? '-'
              : `${detail.paymentTermsDays} 天`}
          </Descriptions.Item>
          <Descriptions.Item label="费用笔数">
            {detail.feeCount} 笔
          </Descriptions.Item>
          <Descriptions.Item label="预计开票币种">
            {detail.estimatedInvoiceCurrency || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="预计开票汇率">
            {detail.estimatedInvoiceRate
              ? trimDecimal(detail.estimatedInvoiceRate)
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="预计开票金额">
            {detail.estimatedInvoiceAmount
              ? `${formatAmount(detail.estimatedInvoiceAmount)} ${detail.estimatedInvoiceCurrency || ''}`.trim()
              : '-'}
          </Descriptions.Item>
          {/* column=3 下「预计开票金额」独占新行首列，备注补满剩余两列，避免行合计超出 column */}
          <DItem label="备注" span={2}>
            {detail.note}
          </DItem>
          {detail.cancellationReason && (
            <Descriptions.Item label="取消原因" span={3}>
              {detail.cancellationReason}
            </Descriptions.Item>
          )}
        </>
      )}
    >
      {(detail) => (
        <>
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              marginBottom: 8,
            }}
          >
            {lineColumnSettings.entry}
          </div>
          <Table<API.FinanceBillLine>
            rowKey="id"
            size="small"
            bordered
            pagination={false}
            dataSource={detail.lines || []}
            columns={lineColumnSettings.columns}
          />
          {lineColumnSettings.modal}
        </>
      )}
    </DescriptionsDetailDrawer>
  );
}
