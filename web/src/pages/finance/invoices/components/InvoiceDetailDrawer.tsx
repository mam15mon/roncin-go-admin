import { Descriptions, Drawer, Space, Table, Tag } from 'antd';
import React from 'react';
import { invoiceStates } from './invoiceConstants';

interface InvoiceDetailDrawerProps {
  detail?: API.FinanceInvoice;
  onClose: () => void;
}

export default function InvoiceDetailDrawer({
  detail,
  onClose,
}: InvoiceDetailDrawerProps) {
  return (
    <Drawer
      title={`开票详情 ${detail?.recordNo || ''}`}
      open={Boolean(detail)}
      size={760}
      onClose={onClose}
    >
      {detail && (
        <>
          <Descriptions bordered size="small" column={2}>
            <Descriptions.Item label="状态">
              <Tag color={invoiceStates[detail.status || 'DRAFT']?.color}>
                {invoiceStates[detail.status || 'DRAFT']?.text}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="税务发票号">
              {detail.taxInvoiceNo || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="结算单位">
              {detail.settlementPartyName}
            </Descriptions.Item>
            <Descriptions.Item label="发票抬头">
              {detail.invoiceTitle || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="纳税人识别号" span={2}>
              {detail.taxpayerIdentificationNo || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="注册地址">
              {detail.registeredAddress || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="注册电话">
              {detail.registeredPhone || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="开户银行">
              {detail.bankName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="银行账号">
              {detail.bankAccount || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="开票日期">
              {detail.invoiceDate || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="未税金额">
              {detail.netAmount
                ? `${detail.netAmount} ${detail.currency}`
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="税额汇总">
              {detail.taxAmount
                ? `${detail.taxAmount} ${detail.currency}`
                : '0.00'}
            </Descriptions.Item>
            <Descriptions.Item label="含税总额">
              <strong style={{ color: '#262626' }}>
                {detail.totalAmount} {detail.currency}
              </strong>
            </Descriptions.Item>
            <Descriptions.Item label="开票汇率">
              {detail.exchangeRate ? (
                <Space size={4}>
                  <span>{detail.exchangeRate}</span>
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
                <span style={{ color: '#8c8c8c' }}>草稿（开票时固化）</span>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="发票折本币">
              {detail.baseCurrencyAmount ? (
                <strong style={{ color: '#1677ff' }}>
                  {detail.baseCurrencyAmount} {detail.baseCurrency}
                </strong>
              ) : (
                '-'
              )}
            </Descriptions.Item>
            <Descriptions.Item label="汇率生效日期">
              {detail.exchangeRateDate || detail.invoiceDate || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="备注" span={2}>
              {detail.note || '-'}
            </Descriptions.Item>
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
          </Descriptions>
          <Table<API.FinanceInvoiceLine>
            rowKey="id"
            size="small"
            bordered
            pagination={false}
            style={{ marginTop: 16 }}
            dataSource={detail.lines || []}
            columns={[
              { title: '行号', dataIndex: 'lineNo', width: 65 },
              { title: '费用代码', dataIndex: 'itemCode', width: 110 },
              { title: '开票项目', dataIndex: 'itemName' },
              {
                title: '税率',
                dataIndex: 'taxRate',
                align: 'right',
                render: (value) => `${Number(value)}%`,
              },
              { title: '未税金额', dataIndex: 'netAmount', align: 'right' },
              { title: '税额', dataIndex: 'taxAmount', align: 'right' },
              { title: '含税金额', dataIndex: 'totalAmount', align: 'right' },
              { title: '来源行数', dataIndex: 'sourceLineCount', width: 90 },
            ]}
          />
          <Table
            rowKey="id"
            size="small"
            pagination={false}
            style={{ marginTop: 16 }}
            dataSource={detail.billLinks || []}
            columns={[
              { title: '账单编号', dataIndex: 'billNo' },
              { title: '金额', dataIndex: 'amount', align: 'right' },
              { title: '税额', dataIndex: 'taxAmount', align: 'right' },
              {
                title: '关联',
                render: (_, r: API.FinanceInvoiceBill) =>
                  r.active ? (
                    <Tag color="blue">有效</Tag>
                  ) : (
                    <Tag>已释放</Tag>
                  ),
              },
            ]}
          />
        </>
      )}
    </Drawer>
  );
}
