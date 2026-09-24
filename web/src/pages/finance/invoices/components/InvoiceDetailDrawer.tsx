import { Descriptions, Space, Table, type TableColumnsType, Tag } from 'antd';
import React from 'react';
import { DescriptionsDetailDrawer, DItem } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { FinanceInvoiceStatus } from '@/enums.generated';
import { formatAmount, trimDecimal } from '@/utils/format';
import {
  invoiceIssueDateLabel,
  invoiceIssueVerb,
  invoiceRecordNoun,
  invoiceStates,
  invoiceStateText,
} from './invoiceConstants';

interface InvoiceDetailDrawerProps {
  detail?: API.FinanceInvoice;
  onClose: () => void;
}

export default function InvoiceDetailDrawer({
  detail,
  onClose,
}: InvoiceDetailDrawerProps) {
  const lineColumns: TableColumnsType<API.FinanceInvoiceLine> = [
    { title: '行号', dataIndex: 'lineNo', width: 65 },
    { title: '费用代码', dataIndex: 'itemCode', width: 110 },
    { title: '开票项目', dataIndex: 'itemName' },
    {
      title: '税率',
      dataIndex: 'taxRate',
      align: 'right',
      render: (value) => `${Number(value)}%`,
    },
    {
      title: '未税金额',
      dataIndex: 'netAmount',
      align: 'right',
      render: (val) => formatAmount(val),
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      align: 'right',
      render: (val) => formatAmount(val),
    },
    {
      title: '含税金额',
      dataIndex: 'totalAmount',
      align: 'right',
      render: (val) => formatAmount(val),
    },
    { title: '来源行数', dataIndex: 'sourceLineCount', width: 90 },
  ];

  const billLinkColumns: TableColumnsType<API.FinanceInvoiceBill> = [
    { title: '账单编号', dataIndex: 'billNo' },
    {
      title: '金额',
      dataIndex: 'amount',
      align: 'right',
      render: (val) => formatAmount(val),
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      align: 'right',
      render: (val) => formatAmount(val),
    },
    {
      title: '关联',
      render: (_, r) =>
        r.active ? <Tag color="blue">有效</Tag> : <Tag>已释放</Tag>,
    },
  ];

  const lineColumnSettings = useColumnSettings<
    TableColumnsType<API.FinanceInvoiceLine>[number]
  >({
    tableKey: 'finance:invoice-lines',
    columns: lineColumns,
  });

  const billLinkColumnSettings = useColumnSettings<
    TableColumnsType<API.FinanceInvoiceBill>[number]
  >({
    tableKey: 'finance:invoice-records',
    columns: billLinkColumns,
  });

  return (
    <DescriptionsDetailDrawer
      title={(current) =>
        `${invoiceRecordNoun(current?.direction)}详情 ${current?.recordNo || ''}`
      }
      open={Boolean(detail)}
      detail={detail}
      size={760}
      column={2}
      onClose={onClose}
      descriptions={(detail) => (
        <>
          <Descriptions.Item label="状态">
            <Tag
              color={
                invoiceStates[
                  detail.status ??
                    FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT
                ]?.color
              }
            >
              {invoiceStateText(detail.status, detail.direction)}
            </Tag>
          </Descriptions.Item>
          <DItem label="税务发票号">{detail.taxInvoiceNo}</DItem>
          <Descriptions.Item label="结算单位">
            {detail.settlementPartyName}
          </Descriptions.Item>
          <Descriptions.Item label="所属公司">
            {detail.organizationName || '-'}
          </Descriptions.Item>
          <DItem label="发票抬头">{detail.invoiceTitle}</DItem>
          <DItem label="纳税人识别号" span={2}>
            {detail.taxpayerIdentificationNo}
          </DItem>
          <DItem label="注册地址">{detail.registeredAddress}</DItem>
          <DItem label="注册电话">{detail.registeredPhone}</DItem>
          <DItem label="开户银行">{detail.bankName}</DItem>
          <DItem label="银行账号">{detail.bankAccount}</DItem>
          <Descriptions.Item label={invoiceIssueDateLabel(detail.direction)}>
            {detail.invoiceDate || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="未税金额">
            {detail.netAmount
              ? `${formatAmount(detail.netAmount)} ${detail.currency}`
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="税额汇总">
            {detail.taxAmount
              ? `${formatAmount(detail.taxAmount)} ${detail.currency}`
              : '0.00'}
          </Descriptions.Item>
          <Descriptions.Item label="含税总额">
            <strong style={{ color: '#262626' }}>
              {formatAmount(detail.totalAmount)} {detail.currency}
            </strong>
          </Descriptions.Item>
          <Descriptions.Item label="开票汇率">
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
              <span style={{ color: '#8c8c8c' }}>
                草稿（{invoiceIssueVerb(detail.direction)}时固化）
              </span>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="发票折本币">
            {detail.baseCurrencyAmount ? (
              <strong style={{ color: '#1677ff' }}>
                {formatAmount(detail.baseCurrencyAmount)} {detail.baseCurrency}
              </strong>
            ) : (
              '-'
            )}
          </Descriptions.Item>
          <Descriptions.Item label="汇率生效日期">
            {detail.exchangeRateDate || detail.invoiceDate || '-'}
          </Descriptions.Item>
          <DItem label="备注" span={2}>
            {detail.note}
          </DItem>
          {detail.cancellationReason && (
            <Descriptions.Item label="取消原因" span={2}>
              {detail.cancellationReason}
            </Descriptions.Item>
          )}
          {detail.redInvoiceNo && (
            <>
              <Descriptions.Item label="红字发票号">
                {detail.redInvoiceNo}
              </Descriptions.Item>
              <Descriptions.Item label="红冲日期">
                {detail.redInvoiceDate}
              </Descriptions.Item>
              <Descriptions.Item label="红冲原因" span={2}>
                {detail.redFlushReason}
              </Descriptions.Item>
            </>
          )}
        </>
      )}
    >
      {(detail) => (
        <>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            {lineColumnSettings.entry}
          </div>
          <Table<API.FinanceInvoiceLine>
            rowKey="id"
            size="small"
            bordered
            pagination={false}
            style={{ marginTop: 16 }}
            dataSource={detail.lines || []}
            columns={lineColumnSettings.columns}
          />
          {lineColumnSettings.modal}
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            {billLinkColumnSettings.entry}
          </div>
          <Table
            rowKey="id"
            size="small"
            pagination={false}
            style={{ marginTop: 16 }}
            dataSource={detail.billLinks || []}
            columns={billLinkColumnSettings.columns}
          />
          {billLinkColumnSettings.modal}
        </>
      )}
    </DescriptionsDetailDrawer>
  );
}
