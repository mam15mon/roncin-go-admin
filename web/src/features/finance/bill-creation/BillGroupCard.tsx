import type { ProColumns } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Tag,
  Typography,
} from 'antd';
import type { NamePath } from 'antd/es/form/interface';
import dayjs from 'dayjs';
import React from 'react';
import { useAccess } from '@/app/access';
import { ResponsiveProTable as ProTable } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { InvoiceRegistrationEditor } from '@/features/finance/invoice-registration';
import { currencyOptionsQueryOptions } from '@/features/master-data/currencies';
import { formatAmount } from '@/utils/format';
import BillTermsCreditWarnings from './BillTermsCreditWarnings';
import BillVerificationFields from './BillVerificationFields';
import SettlementAccountSelect from './SettlementAccountSelect';

const { Text } = Typography;

type BillGroupCardProps = {
  group: API.BillBatchPreviewGroup;
  organizationId: string;
  sessionIdentity: string;
  feeColumns: ProColumns<API.FeeLedgerItem>[];
  directionText: (dir?: string) => string;
  onConfigurationChange: () => void;
};

export default function BillGroupCard({
  group,
  organizationId,
  sessionIdentity,
  feeColumns,
  directionText,
  onConfigurationChange,
}: BillGroupCardProps) {
  const form = Form.useFormInstance();
  const access = useAccess();
  const groupKey = group.groupKey || '';
  const currenciesQuery = useQuery({
    ...currencyOptionsQueryOptions(),
  });
  const currencyOptions = currenciesQuery.data ?? [];

  const paymentTermsDays = Form.useWatch(
    ['groups', groupKey, 'paymentTermsDays'],
    form,
  );
  const derivedDueDate =
    group.billDate && typeof paymentTermsDays === 'number'
      ? dayjs(group.billDate).add(paymentTermsDays, 'day').format('YYYY-MM-DD')
      : undefined;

  const feeColumnSettings = useColumnSettings<ProColumns<API.FeeLedgerItem>>({
    tableKey: 'finance:bill-group-fees',
    columns: feeColumns,
  });

  return (
    <Card
      size="small"
      style={{ marginBottom: 16 }}
      title={
        <Space wrap>
          <Tag color={group.direction === 'RECEIVABLE' ? 'green' : 'volcano'}>
            {directionText(group.direction)}
          </Tag>
          <span style={{ fontWeight: 600 }}>{group.settlementPartyName}</span>
          {group.isCasual && <Tag color="orange">散客</Tag>}
          <Text type="secondary">
            {group.orderNo ? `订单 ${group.orderNo}` : '多订单汇总'}
          </Text>
          {group.taxRate != null && <Tag>{Number(group.taxRate)}% 税率</Tag>}
          <Tag color="geekblue">{group.fees?.length || 0} 笔费用</Tag>
        </Space>
      }
      extra={
        <Text strong style={{ color: '#1677ff', fontSize: 14 }}>
          {formatAmount(group.totalAmount)} {group.currency || '未配置币种'}
        </Text>
      }
    >
      {!group.configurationComplete && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          title="这张账单还有必填资料没补全"
          description="请补齐结算账户或汇率后，点击「刷新账单」。"
        />
      )}
      <BillTermsCreditWarnings
        direction={group.direction}
        isCasual={group.isCasual}
        paymentTermsDays={paymentTermsDays}
        creditLimitAmount={group.creditLimitAmount}
        creditCurrency={group.creditCurrency}
        currentUnsettledAmount={group.currentUnsettledAmount}
        isCreditExceeded={group.isCreditExceeded}
        billAmount={group.totalAmount}
        billCurrency={group.currency}
      />
      <Row gutter={16} style={{ marginBottom: 8 }}>
        <Col xs={24} md={10}>
          <Form.Item
            name={['groups', groupKey, 'statementTitle'] as NamePath}
            label="对账抬头"
            rules={[
              { required: true, whitespace: true, message: '请输入对账抬头' },
              { max: 200, message: '对账抬头不能超过 200 字' },
            ]}
          >
            <Input placeholder="默认使用结算单位名称，可编辑" />
          </Form.Item>
        </Col>
        <Col xs={24} md={5}>
          <Form.Item
            label="创建日期"
            tooltip="由系统维护。未开票时增减费用会更新创建日期，开票后固定。"
          >
            <Text>{group.billDate || '-'}</Text>
          </Form.Item>
        </Col>
        <Col xs={24} md={4}>
          <Form.Item label="账单币种">
            <Input disabled value={group.currency || '-'} />
          </Form.Item>
        </Col>
        <Col xs={24} md={5}>
          <Form.Item
            name={['groups', groupKey, 'paymentTermsDays'] as NamePath}
            label="账期（天）"
            extra={
              group.isCasual && group.direction === 'RECEIVABLE' ? (
                <Text type="warning" style={{ fontSize: 12 }}>
                  建议现结
                  {derivedDueDate ? ` (到期: ${derivedDueDate})` : ''}
                </Text>
              ) : derivedDueDate ? (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  到期: {derivedDueDate}
                </Text>
              ) : undefined
            }
          >
            <InputNumber
              min={0}
              max={3650}
              precision={0}
              placeholder="天数"
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <SettlementAccountSelect
            group={group}
            organizationId={organizationId}
            sessionIdentity={sessionIdentity}
            onConfigurationChange={onConfigurationChange}
          />
        </Col>
        <Col xs={24} md={12}>
          <Form.Item
            name={['groups', groupKey, 'note'] as NamePath}
            label="备注"
            rules={[{ max: 500, message: '备注不能超过 500 字' }]}
          >
            <Input maxLength={500} placeholder="选填" />
          </Form.Item>
        </Col>
      </Row>
      <Card
        size="small"
        title="预计开票（仅作预计，不改变账单币种）"
        style={{ marginBottom: 16 }}
      >
        <Row gutter={16}>
          <Col xs={24} md={8}>
            <Form.Item
              name={['groups', groupKey, 'estimatedInvoiceCurrency']}
              label="预计开票币种"
            >
              <Select
                allowClear
                placeholder="默认使用账单币种，可调整"
                options={currencyOptions}
              />
            </Form.Item>
          </Col>
          <Col xs={24} md={8}>
            <Form.Item
              name={['groups', groupKey, 'estimatedInvoiceRate']}
              label="预计开票汇率"
              rules={[
                ({ getFieldValue }) => ({
                  validator: async (_, rate) => {
                    const currency = getFieldValue([
                      'groups',
                      groupKey,
                      'estimatedInvoiceCurrency',
                    ]);
                    const normalizedRate = rate?.trim();
                    if (!currency) {
                      if (normalizedRate) {
                        throw new Error(
                          '填写预计开票汇率时必须选择预计开票币种',
                        );
                      }
                      return;
                    }
                    if (currency !== group.currency && !normalizedRate) {
                      throw new Error(
                        '预计开票币种与账单币种不同时必须填写预计开票汇率',
                      );
                    }
                    if (normalizedRate) {
                      const num = Number(normalizedRate);
                      if (Number.isNaN(num) || num <= 0) {
                        throw new Error('预计开票汇率必须为大于 0 的有效数字');
                      }
                    }
                    if (
                      currency === group.currency &&
                      normalizedRate &&
                      Number(normalizedRate) !== 1
                    ) {
                      throw new Error(
                        '预计开票币种与账单币种相同时，汇率必须为 1',
                      );
                    }
                  },
                }),
              ]}
            >
              <Input placeholder="可调整" />
            </Form.Item>
          </Col>
          <Col xs={24} md={8}>
            <Space orientation="vertical" size={2}>
              <Text>
                预计开票金额：
                {group.estimatedInvoiceAmount
                  ? `${formatAmount(group.estimatedInvoiceAmount)} ${group.estimatedInvoiceCurrency || ''}`.trim()
                  : '-'}
              </Text>
              <Text>
                折本位币金额：
                {group.baseCurrencyAmount
                  ? `${formatAmount(group.baseCurrencyAmount)} ${group.baseCurrency || ''}`.trim()
                  : '-'}
              </Text>
              <Text type="secondary" style={{ fontSize: 12 }}>
                实际开票时按开票日期重新确定。
              </Text>
            </Space>
          </Col>
        </Row>
      </Card>
      {access.canCreateFinanceInvoices &&
        access.canOperateOrganization(organizationId) && (
          <Card
            size="small"
            title="同次登记发票（可选）"
            style={{ marginBottom: 16 }}
          >
            <InvoiceRegistrationEditor
              name={['groups', groupKey, 'invoices']}
              organizationId={organizationId}
              settlementPartyId={group.settlementPartyId ?? ''}
              bills={[
                {
                  id: groupKey,
                  version: '1',
                  billNo: group.settlementPartyName,
                  currency: group.currency,
                  fees: (group.fees ?? []).map((fee) => ({
                    orderFeeId: fee.id ?? '',
                    feeCode: fee.feeCode,
                    feeName: fee.feeName,
                    totalAmount: fee.totalAmount,
                    invoiceableAmount: fee.totalAmount,
                    sourceTaxRate: fee.taxRate,
                  })),
                },
              ]}
            />
          </Card>
        )}
      <BillVerificationFields group={group} organizationId={organizationId} />
      <ProTable<API.FeeLedgerItem>
        rowKey="id"
        size="small"
        bordered
        search={false}
        options={{ reload: true, density: true, setting: false }}
        toolBarRender={() => [feeColumnSettings.entry]}
        pagination={false}
        columns={feeColumnSettings.columns}
        dataSource={group.fees || []}
      />
    </Card>
  );
}
