import {
  MinusCircleOutlined,
  PlusOutlined,
  QuestionCircleOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Col,
  DatePicker,
  Form,
  Input,
  Row,
  Select,
  Tooltip,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import { useMemo, useState } from 'react';
import { useAccess } from '@/app/access';
import { paymentMethodOptions } from '@/features/finance/cashflow-entry';
import { settlementServiceListVerificationCashflowCandidates } from '@/services/roncin/settlementService';
import { formatAmount } from '@/utils/format';
import {
  type BillInvoiceDraft,
  hasBillGroupInvoiceDraft,
} from './billWorkbenchHelpers';

const positiveDecimal = {
  pattern: /^(?:0|[1-9]\d{0,19})(?:\.\d{1,8})?$/,
  message: '请输入有效金额，最多8位小数',
};

type Props = { group: API.BillBatchPreviewGroup; organizationId: string };

/** 建账附带核销仅填写草稿，最终提交才创建核销与差额流水。 */
export default function BillVerificationFields({
  group,
  organizationId,
}: Props) {
  const access = useAccess();
  const groupKey = group.groupKey ?? '';
  const form = Form.useFormInstance();
  const invoices = Form.useWatch(['groups', groupKey, 'invoices'], {
    form,
    preserve: true,
  }) as BillInvoiceDraft[] | undefined;
  const hasInvoice = Boolean(
    access.canCreateFinanceInvoices && hasBillGroupInvoiceDraft(invoices),
  );
  const canUse =
    access.canCreateFinanceVerifications &&
    access.canOperateOrganization(organizationId);
  const canShortfall =
    canUse &&
    access.canCreateFinanceCashflows &&
    access.canUpdateFinanceCashflows;
  const [keyword, setKeyword] = useState('');
  const [chosenCashflows, setChosenCashflows] = useState<API.FinanceCashflow[]>(
    [],
  );
  const query = useQuery({
    queryKey: [
      'finance',
      'bill-combination-cashflows',
      organizationId,
      group.direction,
      group.settlementPartyId,
      group.currency,
      keyword,
    ],
    enabled: Boolean(
      canUse && hasInvoice && group.settlementPartyId && group.currency,
    ),
    meta: { errorMessage: '加载本组核销资金候选失败' },
    queryFn: async () => {
      const response =
        await settlementServiceListVerificationCashflowCandidates({
          organizationId,
          direction: group.direction ?? '',
          settlementPartyId: group.settlementPartyId ?? '',
          currency: group.currency ?? '',
          keyword: keyword || undefined,
          page: 1,
          pageSize: 50,
        });
      if (
        !response ||
        response.success === false ||
        (response.data === undefined && response.success !== true)
      )
        throw new Error('加载本组核销资金候选失败');
      const rows = response.data ?? [];
      if (rows.some((row) => !row.id || row.organizationId !== organizationId))
        throw new Error('核销资金候选公司或标识不正确');
      return rows;
    },
  });
  const options = useMemo(() => {
    const rows = new Map(chosenCashflows.map((item) => [item.id, item]));
    for (const item of query.data ?? []) rows.set(item.id, item);
    return [...rows.values()].map((item) => ({
      value: item.id ?? '',
      label: `${item.flowNo ?? ''} · 余额 ${formatAmount(item.unverifiedAmount)} ${group.currency ?? ''}`,
    }));
  }, [chosenCashflows, query.data, group.currency]);
  if (!canUse) return null;
  return (
    <Card size="small" title="同次核销" style={{ marginBottom: 16 }}>
      <div style={{ textAlign: 'right', marginBottom: 8 }}>
        <Tooltip
          placement="topRight"
          title="核销与本组建账一起保存；任一记录失败，这张账单的核销全部撤销。"
        >
          <Button type="link" size="small" icon={<QuestionCircleOutlined />}>
            填写说明
          </Button>
        </Tooltip>
      </div>
      {!hasInvoice && (
        <Alert
          type="info"
          showIcon
          title="请先在本组登记发票，再添加同次核销。已有核销输入会保留，补齐发票后可继续提交。"
          style={{ marginBottom: 12 }}
        />
      )}
      <Form.List
        name={['groups', groupKey, 'verifications']}
        rules={[
          {
            validator: async (_, records) => {
              if (
                records?.length &&
                (!access.canCreateFinanceInvoices ||
                  !hasBillGroupInvoiceDraft(
                    form.getFieldValue(['groups', groupKey, 'invoices']),
                  ))
              )
                throw new Error('账单尚未开票，请先登记发票后核销');
            },
          },
        ]}
      >
        {(fields, { add, remove }, { errors }) => (
          <>
            {fields.map((field) => (
              <Card
                key={field.key}
                size="small"
                title={`第${field.name + 1}笔核销`}
                extra={
                  <Button
                    type="text"
                    danger
                    icon={<MinusCircleOutlined />}
                    onClick={() => remove(field.name)}
                  >
                    移除核销
                  </Button>
                }
                style={{ marginBottom: 12 }}
              >
                <Row gutter={12}>
                  <Col xs={24} md={8}>
                    <Form.Item
                      name={[field.name, 'verificationDate']}
                      label="核销日期"
                      rules={[{ required: true, message: '请选择核销日期' }]}
                    >
                      <DatePicker style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={16}>
                    <Form.Item
                      name={[field.name, 'note']}
                      label="核销备注"
                      rules={[{ max: 500 }]}
                    >
                      <Input />
                    </Form.Item>
                  </Col>
                </Row>
                <Form.List name={[field.name, 'allocations']}>
                  {(allocations, actions, errors) => (
                    <>
                      {allocations.map((allocation) => (
                        <Row key={allocation.key} gutter={12} align="middle">
                          <Col xs={24} md={14}>
                            <Form.Item
                              name={[allocation.name, 'cashflowId']}
                              label="既有资金流水"
                              rules={[
                                { required: true, message: '请选择资金流水' },
                              ]}
                            >
                              <Select
                                showSearch={{
                                  filterOption: false,
                                  onSearch: setKeyword,
                                }}
                                options={options}
                                loading={query.isFetching}
                                placeholder="输入流水号、银行参考号或单位搜索"
                                onChange={(id) => {
                                  const chosen = query.data?.find(
                                    (item) => item.id === id,
                                  );
                                  if (chosen)
                                    setChosenCashflows((rows) => [
                                      ...rows.filter((item) => item.id !== id),
                                      chosen,
                                    ]);
                                }}
                              />
                            </Form.Item>
                          </Col>
                          <Col xs={20} md={8}>
                            <Form.Item
                              name={[allocation.name, 'amount']}
                              label={`本次金额（${group.currency}）`}
                              rules={[
                                { required: true, message: '请输入金额' },
                                positiveDecimal,
                              ]}
                            >
                              <Input />
                            </Form.Item>
                          </Col>
                          <Col xs={4} md={2}>
                            <Button
                              type="text"
                              danger
                              icon={<MinusCircleOutlined />}
                              aria-label="移除资金分配"
                              onClick={() => actions.remove(allocation.name)}
                            />
                          </Col>
                        </Row>
                      ))}
                      <Form.ErrorList errors={errors.errors} />
                      <Button
                        size="small"
                        icon={<PlusOutlined />}
                        onClick={() => actions.add({ amount: '' })}
                      >
                        添加既有资金分配
                      </Button>
                    </>
                  )}
                </Form.List>
                {canShortfall && (
                  <>
                    <Form.Item
                      name={[field.name, 'shortfallEnabled']}
                      valuePropName="checked"
                      style={{ marginTop: 12 }}
                    >
                      <Checkbox>补录差额收付款</Checkbox>
                    </Form.Item>
                    <Form.Item
                      noStyle
                      shouldUpdate={(previous, next) =>
                        previous.groups?.[groupKey]?.verifications?.[field.name]
                          ?.shortfallEnabled !==
                        next.groups?.[groupKey]?.verifications?.[field.name]
                          ?.shortfallEnabled
                      }
                    >
                      {({ getFieldValue }) =>
                        getFieldValue([
                          'groups',
                          groupKey,
                          'verifications',
                          field.name,
                          'shortfallEnabled',
                        ]) && (
                          <Row gutter={12}>
                            <Col xs={24} md={8}>
                              <Form.Item
                                name={[field.name, 'shortfall', 'amount']}
                                label="差额金额（全额用于本次核销）"
                                rules={[{ required: true }, positiveDecimal]}
                              >
                                <Input />
                              </Form.Item>
                            </Col>
                            <Col xs={24} md={8}>
                              <Form.Item
                                name={[
                                  field.name,
                                  'shortfall',
                                  'transactionDate',
                                ]}
                                label="交易日期"
                                rules={[{ required: true }]}
                              >
                                <DatePicker style={{ width: '100%' }} />
                              </Form.Item>
                            </Col>
                            <Col xs={24} md={8}>
                              <Form.Item
                                name={[field.name, 'shortfall', 'ourAccount']}
                                label="我方账户"
                                rules={[{ required: true, whitespace: true }]}
                              >
                                <Input />
                              </Form.Item>
                            </Col>
                            <Col xs={24} md={8}>
                              <Form.Item
                                name={[
                                  field.name,
                                  'shortfall',
                                  'paymentMethod',
                                ]}
                                label="支付方式"
                                rules={[{ required: true }]}
                              >
                                <Select options={paymentMethodOptions} />
                              </Form.Item>
                            </Col>
                            <Col xs={24} md={8}>
                              <Form.Item
                                name={[
                                  field.name,
                                  'shortfall',
                                  'counterpartyAccount',
                                ]}
                                label="对方账户（散客付款必填）"
                                rules={[
                                  {
                                    required:
                                      group.isCasual &&
                                      group.direction === 'PAYABLE',
                                    whitespace: true,
                                  },
                                ]}
                              >
                                <Input />
                              </Form.Item>
                            </Col>
                            <Col xs={24} md={8}>
                              <Form.Item
                                name={[
                                  field.name,
                                  'shortfall',
                                  'bankReferenceNo',
                                ]}
                                label="银行参考号"
                              >
                                <Input />
                              </Form.Item>
                            </Col>
                            <Col span={24}>
                              <Typography.Text type="secondary">
                                使用本组公司、方向、结算单位和币种，金额由你明确填写，不会按并发余额变化自动补大。
                              </Typography.Text>
                            </Col>
                          </Row>
                        )
                      }
                    </Form.Item>
                  </>
                )}
              </Card>
            ))}
            <Form.ErrorList errors={errors} />
            <Button
              disabled={!hasInvoice}
              icon={<PlusOutlined />}
              onClick={() =>
                add({
                  verificationDate: dayjs(),
                  allocations: [],
                  shortfallEnabled: false,
                  shortfall: {
                    transactionDate: dayjs(),
                    paymentMethod: 'BANK_TRANSFER',
                  },
                })
              }
            >
              添加同次核销
            </Button>
          </>
        )}
      </Form.List>
    </Card>
  );
}
