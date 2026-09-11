import type { ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import {
  Alert,
  Card,
  Col,
  DatePicker,
  Descriptions,
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
import { getCurrencyOptions, type SelectOption } from '@/utils/options';

const { Text } = Typography;

type BillGroupCardProps = {
  group: API.BillBatchPreviewGroup;
  organizationId: string;
  sessionIdentity: string;
  feeColumns: ProColumns<API.FeeLedgerItem>[];
  directionText: (dir?: string) => string;
  onConfigurationChange: () => void;
};

type SettlementAccountSelectProps = Pick<
  BillGroupCardProps,
  'group' | 'organizationId' | 'sessionIdentity' | 'onConfigurationChange'
>;

function settlementAccountLabel(account: API.FinanceSettlementAccountOption) {
  const identity = account.name || account.accountHolder || '未命名账户';
  const bank = account.bankName || '-';
  const suffix = account.accountNo ? ` · ${account.accountNo}` : '';
  return `${identity}｜${bank}${suffix}｜${account.currency || '-'}`;
}

/** 每个叶子以完整身份和单调序号隔离候选，旧请求不能回填新叶子。 */
function SettlementAccountSelect({
  group,
  organizationId,
  sessionIdentity,
  onConfigurationChange,
}: SettlementAccountSelectProps) {
  const form = Form.useFormInstance();
  const [loaded, setLoaded] = useState<{
    identity: string;
    options: API.FinanceSettlementAccountOption[];
  }>();
  const [loading, setLoading] = useState(false);
  const requestSequenceRef = useRef(0);
  const groupKey = group.groupKey || '';
  const finalCurrency = group.currency || '';
  const groupField = (field: 'settlementAccountId'): NamePath => [
    'groups',
    groupKey,
    field,
  ];
  const identity = [
    sessionIdentity,
    organizationId,
    groupKey,
    group.settlementPartyId || '',
    group.direction || '',
    finalCurrency || '',
  ].join(':');
  const visibleOptions = loaded?.identity === identity ? loaded.options : [];

  useEffect(() => {
    const requestSequence = ++requestSequenceRef.current;
    const requestIdentity = identity;
    setLoaded(undefined);
    if (
      !organizationId ||
      !groupKey ||
      !group.settlementPartyId ||
      !group.direction ||
      !finalCurrency
    ) {
      setLoading(false);
      return undefined;
    }
    setLoading(true);
    void settlementServiceListBillSettlementAccountCandidates({
      organizationId,
      settlementPartyId: group.settlementPartyId,
      direction: group.direction,
      currency: finalCurrency,
    })
      .then((response) => {
        if (
          requestSequence !== requestSequenceRef.current ||
          requestIdentity !== identity
        ) {
          return;
        }
        const accounts = unwrapList(response);
        setLoaded({ identity: requestIdentity, options: accounts });
        const selected = form.getFieldValue(groupField('settlementAccountId'));
        if (selected && accounts.some((account) => account.id === selected)) {
          return;
        }
        const defaultAccount = accounts.find((account) => account.isDefault);
        if (defaultAccount?.id) {
          form.setFieldValue(
            groupField('settlementAccountId'),
            defaultAccount.id,
          );
          onConfigurationChange();
        }
      })
      .catch(() => {
        if (
          requestSequence === requestSequenceRef.current &&
          requestIdentity === identity
        ) {
          setLoaded({ identity: requestIdentity, options: [] });
        }
      })
      .finally(() => {
        if (
          requestSequence === requestSequenceRef.current &&
          requestIdentity === identity
        ) {
          setLoading(false);
        }
      });
    return () => {
      requestSequenceRef.current += 1;
    };
  }, [
    form,
    group.direction,
    group.settlementPartyId,
    groupKey,
    identity,
    onConfigurationChange,
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
        placeholder="请选择与账单方向、最终币种一致的启用账户"
        options={visibleOptions.map((account) => ({
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
  sessionIdentity,
  feeColumns,
  directionText,
  onConfigurationChange,
}: BillGroupCardProps) {
  const groupKey = group.groupKey || '';
  const [currencyOptions, setCurrencyOptions] = useState<SelectOption[]>([]);

  useEffect(() => {
    let cancelled = false;
    void getCurrencyOptions()
      .then((options) => {
        if (!cancelled) setCurrencyOptions(options);
      })
      .catch(() => {
        if (!cancelled) setCurrencyOptions([]);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Card
      size="small"
      style={{ marginBottom: 16, border: '1px solid #e8e8e8', borderRadius: 6 }}
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
          {group.totalAmount} {group.currency || '未配置币种'}
        </Text>
      }
    >
      {!group.configurationComplete && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          title="该叶子尚未形成可创建快照"
          description={
            group.isTemporaryBillDate
              ? '请确认账单日期和结算账户；服务端将按正式账单日期重新解析账单币种至组织本位币的汇率。'
              : '请补齐结算账户或服务端要求的汇率配置后重新预览。'
          }
        />
      )}
      <Row gutter={16} style={{ marginBottom: 8 }}>
        <Col xs={24} md={8}>
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
            name={['groups', groupKey, 'billDate'] as NamePath}
            label="账单日期"
            rules={[{ required: true, message: '请选择账单日期' }]}
          >
            <DatePicker allowClear={false} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={5}>
          <Form.Item label="账单币种">
            <Input disabled value={group.currency || '-'} />
          </Form.Item>
        </Col>
        <Col xs={24} md={3}>
          <Form.Item
            name={['groups', groupKey, 'paymentTermsDays'] as NamePath}
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
        <Col xs={24} md={3}>
          <Form.Item
            name={['groups', groupKey, 'note'] as NamePath}
            label="备注"
            rules={[{ max: 500, message: '备注不能超过 500 字' }]}
          >
            <Input maxLength={500} placeholder="选填" />
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
      </Row>
      <Descriptions
        bordered
        size="small"
        column={3}
        style={{ marginBottom: 16 }}
      >
        <Descriptions.Item label="账单金额">
          {group.totalAmount || '-'} {group.currency || '-'}
        </Descriptions.Item>
        <Descriptions.Item label="组织本位币金额">
          {group.isTemporaryBillDate
            ? '选择账单日期后计算'
            : `${group.baseCurrencyAmount || '服务端未提供'} ${group.baseCurrency || ''}`.trim()}
        </Descriptions.Item>
        <Descriptions.Item label="预计开票币种">
          {group.estimatedInvoiceCurrency || '服务端未提供'}
        </Descriptions.Item>
        <Descriptions.Item label="预计开票汇率">
          {group.estimatedInvoiceRate || '服务端未提供'}
        </Descriptions.Item>
        <Descriptions.Item label="预计开票金额">
          {group.estimatedInvoiceAmount || '服务端未提供'}
        </Descriptions.Item>
      </Descriptions>
      <Card
        size="small"
        title="预计开票配置（仅用于预计，不改变固定账单币种）"
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
              <Input placeholder="服务端按预计口径计算，可调整" />
            </Form.Item>
          </Col>
          <Col xs={24} md={8}>
            <Text type="secondary">
              预计金额由服务端预览返回；实际开票将以开票日期独立重新确定。
            </Text>
          </Col>
        </Row>
      </Card>
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
