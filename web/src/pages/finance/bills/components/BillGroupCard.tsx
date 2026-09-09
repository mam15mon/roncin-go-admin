import type { ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import {
  Card,
  Col,
  DatePicker,
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
import React, { useEffect, useRef, useState } from 'react';
import { settlementServiceListBillSettlementAccountCandidates } from '@/services/roncin/settlementService';
import { unwrapList } from '@/utils/api';

const { Text } = Typography;

type BillGroupCardProps = {
  group: API.BillBatchPreviewGroup;
  organizationId: string;
  feeColumns: ProColumns<API.FeeLedgerItem>[];
  directionText: (dir?: string) => string;
};

type SettlementAccountSelectProps = {
  group: API.BillBatchPreviewGroup;
  organizationId: string;
};

function settlementAccountLabel(account: API.FinanceSettlementAccountOption) {
  const identity = account.name || account.accountHolder || '未命名账户';
  const bank = account.bankName || '-';
  const suffix = account.accountNo ? ` · ${account.accountNo}` : '';
  return `${identity}｜${bank}${suffix}｜${account.currency || '-'}`;
}

/** 每个叶子的候选独立加载，避免并行叶子互相取消，也避免旧身份迟到回填。 */
function SettlementAccountSelect({
  group,
  organizationId,
}: SettlementAccountSelectProps) {
  const form = Form.useFormInstance();
  const [options, setOptions] = useState<API.FinanceSettlementAccountOption[]>(
    [],
  );
  const [loading, setLoading] = useState(false);
  const requestSequenceRef = useRef(0);
  const identityRef = useRef<string | undefined>(undefined);
  const groupKey = group.groupKey || '';
  const groupField = (field: 'settlementAccountId'): NamePath => [
    'groups',
    groupKey,
    field,
  ];
  const identity = [
    organizationId,
    groupKey,
    group.settlementPartyId || '',
    group.direction || '',
    group.currency || '',
  ].join(':');

  useEffect(() => {
    const requestSequence = ++requestSequenceRef.current;
    const identityChanged =
      identityRef.current !== undefined && identityRef.current !== identity;
    identityRef.current = identity;
    if (identityChanged) {
      form.setFieldValue(groupField('settlementAccountId'), undefined);
    }
    setOptions([]);
    if (
      !organizationId ||
      !groupKey ||
      !group.settlementPartyId ||
      !group.direction ||
      !group.currency
    ) {
      setLoading(false);
      return undefined;
    }
    setLoading(true);
    void settlementServiceListBillSettlementAccountCandidates({
      organizationId,
      settlementPartyId: group.settlementPartyId,
      direction: group.direction,
      currency: group.currency,
    })
      .then((response) => {
        if (requestSequence !== requestSequenceRef.current) return;
        const accounts = unwrapList(response);
        setOptions(accounts);
        const selected = form.getFieldValue(groupField('settlementAccountId'));
        const selectedStillAvailable = accounts.some(
          (account) => account.id === selected,
        );
        if (selected && selectedStillAvailable) return;

        const defaultAccount = accounts.find((account) => account.isDefault);
        form.setFieldValue(
          groupField('settlementAccountId'),
          defaultAccount?.id,
        );
      })
      .catch(() => {
        if (requestSequence === requestSequenceRef.current) setOptions([]);
      })
      .finally(() => {
        if (requestSequence === requestSequenceRef.current) setLoading(false);
      });
    return () => {
      requestSequenceRef.current += 1;
    };
  }, [
    form,
    group.currency,
    group.direction,
    group.settlementPartyId,
    groupKey,
    identity,
    organizationId,
  ]);

  return (
    <Form.Item
      name={groupField('settlementAccountId')}
      label="结算账户"
      rules={[{ required: true, message: '请选择结算账户' }]}
    >
      <Select
        allowClear
        loading={loading}
        placeholder="请选择与账单方向、币种一致的启用账户"
        options={options.map((account) => ({
          value: account.id,
          label: settlementAccountLabel(account),
        }))}
      />
    </Form.Item>
  );
}

export default function BillGroupCard({
  group,
  organizationId,
  feeColumns,
  directionText,
}: BillGroupCardProps) {
  return (
    <Card
      key={group.groupKey}
      size="small"
      style={{
        marginBottom: 16,
        border: '1px solid #e8e8e8',
        borderRadius: 6,
      }}
      title={
        <Space wrap>
          <Tag color={group.direction === 'RECEIVABLE' ? 'green' : 'volcano'}>
            {directionText(group.direction)}
          </Tag>
          <span style={{ fontWeight: 600 }}>{group.settlementPartyName}</span>
          <Text type="secondary">
            {group.orderNo ? `订单 ${group.orderNo}` : '多订单汇总'}
          </Text>
          {group.taxRate != null && <Tag>{Number(group.taxRate)}% 税率</Tag>}
          <Tag color="geekblue">{group.fees?.length || 0} 笔费用</Tag>
        </Space>
      }
      extra={
        <Text strong style={{ color: '#1677ff', fontSize: 14 }}>
          {group.totalAmount} {group.currency}
        </Text>
      }
    >
      <Row gutter={16} style={{ marginBottom: 8 }}>
        <Col xs={24} md={8}>
          <Form.Item
            name={
              ['groups', group.groupKey || '', 'statementTitle'] as NamePath
            }
            label="对账抬头"
            rules={[
              {
                required: true,
                whitespace: true,
                message: '请输入对账抬头',
              },
              { max: 200, message: '对账抬头不能超过 200 字' },
            ]}
          >
            <Input placeholder="默认使用结算单位名称，可编辑" />
          </Form.Item>
        </Col>
        <Col xs={24} md={5}>
          <Form.Item
            name={['groups', group.groupKey || '', 'billDate'] as NamePath}
            label="账单日期"
            rules={[{ required: true, message: '请选择账单日期' }]}
          >
            <DatePicker allowClear={false} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={4}>
          <Form.Item
            name={
              ['groups', group.groupKey || '', 'paymentTermsDays'] as NamePath
            }
            label="账期（天）"
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
        <Col xs={24} md={7}>
          <Form.Item
            name={['groups', group.groupKey || '', 'note'] as NamePath}
            label="备注"
            rules={[{ max: 500, message: '备注不能超过 500 字' }]}
          >
            <Input maxLength={500} placeholder="选填，账单备注" />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <SettlementAccountSelect
            group={group}
            organizationId={organizationId}
          />
        </Col>
      </Row>
      <ProTable<API.FeeLedgerItem>
        rowKey="id"
        size="small"
        bordered
        search={false}
        options={false}
        toolBarRender={false}
        pagination={false}
        columns={feeColumns}
        dataSource={group.fees || []}
        scroll={{ x: 880 }}
      />
    </Card>
  );
}
