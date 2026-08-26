import {
  ArrowLeftOutlined,
  CheckCircleOutlined,
  FileDoneOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { history, useAccess } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Result,
  Row,
  Space,
  Steps,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useEffect, useMemo, useState } from 'react';
import {
  settlementServiceCreateBillBatch,
  settlementServiceConfirmBillBatch,
  settlementServiceListFeeLedger,
  settlementServicePreviewBillBatch,
} from '@/services/roncin/settlementService';

const { Text, Title } = Typography;

type GroupFormValue = {
  statementTitle: string;
  billDate: Dayjs;
  paymentTermsDays?: number;
  note?: string;
};

type WorkbenchFormValue = {
  groups: GroupFormValue[];
};

type RequestError = Error & {
  data?: { reason?: string; message?: string };
  response?: { data?: { reason?: string; message?: string } };
};

export type BillCreationWorkbenchProps = {
  open: boolean;
  initialFeeIds?: string[];
  sourceLabel?: string;
  onClose: () => void;
  onCreated?: (batch: API.FinanceBillBatch) => void;
};

const feeColumns: ProColumns<API.FeeLedgerItem>[] = [
  {
    title: '订单编号',
    dataIndex: 'orderNo',
    width: 150,
    copyable: true,
    search: false,
  },
  {
    title: '方向',
    dataIndex: 'direction',
    width: 75,
    valueType: 'select',
    search: false,
    valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
    render: (_, row) => (
      <Tag color={row.direction === 'RECEIVABLE' ? 'green' : 'volcano'}>
        {row.direction === 'RECEIVABLE' ? '应收' : '应付'}
      </Tag>
    ),
  },
  {
    title: '结算单位',
    dataIndex: 'settlementPartyName',
    width: 210,
    ellipsis: true,
    search: false,
  },
  { title: '费用名称', dataIndex: 'feeName', width: 140, search: false },
  {
    title: '税率',
    dataIndex: 'taxRate',
    width: 85,
    align: 'right',
    search: false,
    renderText: (value) => (value == null ? '-' : `${Number(value)}%`),
  },
  {
    title: '金额',
    dataIndex: 'totalAmount',
    width: 150,
    align: 'right',
    search: false,
    render: (_, row) => (
      <Text strong>
        {row.totalAmount} {row.currency}
      </Text>
    ),
  },
  {
    title: '费用日期',
    dataIndex: 'expenseDate',
    width: 115,
    search: false,
  },
];

const selectionFeeColumns: ProColumns<API.FeeLedgerItem>[] = [
  {
    title: '关键词',
    dataIndex: 'keyword',
    hideInTable: true,
    fieldProps: { placeholder: '订单号、费用或结算单位' },
  },
  {
    title: '方向',
    dataIndex: 'direction',
    hideInTable: true,
    valueType: 'select',
    valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
  },
  ...feeColumns,
];

function directionText(value?: string) {
  return value === 'RECEIVABLE' ? '应收' : '应付';
}

function requestReason(error: RequestError) {
  return error.data?.reason || error.response?.data?.reason;
}

function requestMessage(error: RequestError, fallback: string) {
  return (
    error.data?.message ||
    error.response?.data?.message ||
    error.message ||
    fallback
  );
}

export default function BillCreationWorkbench({
  open,
  initialFeeIds = [],
  sourceLabel,
  onClose,
  onCreated,
}: BillCreationWorkbenchProps) {
  const { message } = App.useApp();
  const access = useAccess();
  const [form] = Form.useForm<WorkbenchFormValue>();
  const [current, setCurrent] = useState(0);
  const [selectedFeeIds, setSelectedFeeIds] = useState<React.Key[]>([]);
  const [splitByOrder, setSplitByOrder] = useState(false);
  const [splitByTaxRate, setSplitByTaxRate] = useState(false);
  const [preview, setPreview] = useState<API.PreviewBillBatchResponse>();
  const [result, setResult] = useState<API.FinanceBillBatch>();
  const [loading, setLoading] = useState(false);
  const [idempotencyKey, setIdempotencyKey] = useState('');
  const [confirming, setConfirming] = useState(false);
  const initialFeeKey = initialFeeIds.join('|');
  const fixedSelection = initialFeeIds.length > 0;

  useEffect(() => {
    if (!open) return;
    setCurrent(0);
    setSelectedFeeIds(initialFeeKey ? initialFeeKey.split('|') : []);
    setSplitByOrder(false);
    setSplitByTaxRate(false);
    setPreview(undefined);
    setResult(undefined);
    setLoading(false);
    setConfirming(false);
    setIdempotencyKey(globalThis.crypto.randomUUID());
    form.resetFields();
  }, [form, open, initialFeeKey]);

  const selectedIds = useMemo(
    () => selectedFeeIds.map(String),
    [selectedFeeIds],
  );

  const loadPreview = async () => {
    if (selectedIds.length === 0) {
      message.warning('请至少选择一笔已确认费用');
      return false;
    }
    setLoading(true);
    try {
      const response = await settlementServicePreviewBillBatch(
        {
          feeIds: selectedIds,
          groupingPolicy: { splitByOrder, splitByTaxRate },
        },
        { skipErrorHandler: true },
      );
      const groups = response.data || [];
      if (!response.previewToken || groups.length === 0) {
        throw new Error('服务端未返回有效的拆单预览');
      }
      setPreview(response);
      form.setFieldsValue({
        groups: groups.map((group) => ({
          statementTitle: group.settlementPartyName || '',
          billDate: dayjs(),
          paymentTermsDays: undefined,
          note: undefined,
        })),
      });
      return true;
    } catch (rawError: unknown) {
      const error = rawError as RequestError;
      message.error(requestMessage(error, '拆单预览失败'));
      return false;
    } finally {
      setLoading(false);
    }
  };

  const next = async () => {
    if (current === 0) {
      if (selectedIds.length === 0) {
        message.warning('请至少选择一笔已确认费用');
        return;
      }
      setCurrent(1);
      return;
    }
    if (current === 1) {
      if (await loadPreview()) setCurrent(2);
    }
  };

  const createBatch = async () => {
    if (!preview?.previewToken || !preview.data?.length) return;
    const values = await form.validateFields();
    setLoading(true);
    try {
      const response = await settlementServiceCreateBillBatch(
        {
          feeIds: selectedIds,
          groupingPolicy: { splitByOrder, splitByTaxRate },
          previewToken: preview.previewToken,
          idempotencyKey,
          groups: preview.data.map((group, index) => {
            const value = values.groups[index];
            return {
              groupKey: group.groupKey || '',
              statementTitle: value.statementTitle.trim(),
              billDate: value.billDate.format('YYYY-MM-DD'),
              paymentTermsDays: value.paymentTermsDays,
              note: value.note?.trim() || undefined,
            };
          }),
        },
        { skipErrorHandler: true },
      );
      if (!response.data) throw new Error('服务端未返回建单结果');
      setResult(response.data);
      setCurrent(3);
      message.success(`批次 ${response.data.batchNo || ''} 已原子生成`);
      onCreated?.(response.data);
    } catch (rawError: unknown) {
      const error = rawError as RequestError;
      if (requestReason(error) === 'FINANCE_BILL_PREVIEW_STALE') {
        message.warning('费用已发生变化，请重新预览后再生成账单');
        setPreview(undefined);
        setCurrent(1);
      } else {
        message.error(requestMessage(error, '批量生成账单失败'));
      }
    } finally {
      setLoading(false);
    }
  };

  const confirmBatch = async () => {
    if (!result?.id || !result.bills?.length) return;
    setConfirming(true);
    try {
      const response = await settlementServiceConfirmBillBatch(
        { id: result.id },
        {
          id: result.id,
          bills: result.bills.map((bill) => ({
            billId: bill.id || '',
            expectedVersion: bill.version || '0',
          })),
        },
      );
      if (response.data) setResult(response.data);
      message.success('本批账单已全部确认，可以进入开票、收付款和核销流程');
      onCreated?.(response.data || result);
    } catch (error: any) {
      message.error(error.message || '批量确认账单失败');
    } finally {
      setConfirming(false);
    }
  };

  const footer = (
    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
      <Button onClick={onClose}>{current === 3 ? '关闭' : '取消'}</Button>
      {current < 3 && (
        <Space>
          {current > 0 && (
            <Button
              icon={<ArrowLeftOutlined />}
              disabled={loading}
              onClick={() => setCurrent((value) => value - 1)}
            >
              上一步
            </Button>
          )}
          {current < 2 && (
            <Button
              type="primary"
              loading={loading}
              onClick={() => void next()}
            >
              下一步
            </Button>
          )}
          {current === 2 && (
            <Button
              type="primary"
              icon={<FileDoneOutlined />}
              loading={loading}
              onClick={() => void createBatch()}
            >
              原子生成 {preview?.data?.length || 0} 张账单
            </Button>
          )}
        </Space>
      )}
    </div>
  );

  return (
    <Drawer
      title="费用批量转账单"
      open={open}
      width="min(1280px, 96vw)"
      destroyOnHidden
      maskClosable={false}
      footer={footer}
      onClose={onClose}
    >
      <Steps
        current={current}
        size="small"
        style={{ marginBottom: 24 }}
        items={[
          { title: '选择费用' },
          { title: '拆单策略' },
          { title: '账单资料' },
          { title: '生成完成' },
        ]}
      />

      {current === 0 &&
        (fixedSelection ? (
          <Card>
            <Alert
              type="info"
              showIcon
              message={`已从${sourceLabel || '业务页面'}带入 ${selectedIds.length} 笔已确认费用`}
              description="费用状态、结算维度和金额快照将在预览及最终建单事务中由服务端再次校验。"
            />
          </Card>
        ) : (
          <ProTable<API.FeeLedgerItem>
            rowKey="id"
            headerTitle="选择待结算费用"
            columns={selectionFeeColumns}
            size="small"
            bordered
            options={false}
            pagination={{ defaultPageSize: 15, showSizeChanger: true }}
            rowSelection={{
              selectedRowKeys: selectedFeeIds,
              preserveSelectedRowKeys: true,
              onChange: setSelectedFeeIds,
            }}
            tableAlertRender={({ selectedRowKeys }) => (
              <Text>已选择 {selectedRowKeys.length} 笔已确认费用</Text>
            )}
            request={async (params) => {
              const response = await settlementServiceListFeeLedger({
                page: params.current,
                pageSize: params.pageSize,
                keyword: params.keyword,
                direction: params.direction,
                status: 'CONFIRMED',
              });
              return {
                data: response.data || [],
                total: Number(response.total || 0),
                success: response.success ?? true,
              };
            }}
          />
        ))}

      {current === 1 && (
        <Card>
          <Title level={5}>拆单维度</Title>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 20 }}
            message="收付方向、结算单位、原币和本币始终是强制拆单维度"
            description="不同强制维度的费用绝不会进入同一张账单。下面两个开关只控制额外拆分，不会放宽服务端的财务边界。"
          />
          <Row gutter={[24, 16]}>
            <Col xs={24} lg={12}>
              <Card size="small" title="按订单拆分">
                <Space direction="vertical">
                  <Switch
                    checked={splitByOrder}
                    checkedChildren="已启用"
                    unCheckedChildren="未启用"
                    onChange={setSplitByOrder}
                  />
                  <Text type="secondary">
                    启用后每个业务订单独立成账；关闭时允许同一结算单位的多票订单汇总对账。
                  </Text>
                </Space>
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card size="small" title="按税率拆分">
                <Space direction="vertical">
                  <Switch
                    checked={splitByTaxRate}
                    checkedChildren="已启用"
                    unCheckedChildren="未启用"
                    onChange={setSplitByTaxRate}
                  />
                  <Text type="secondary">
                    启用后不同税率独立成账；关闭时税率仍会逐费用行固化，后续开票可按行处理。
                  </Text>
                </Space>
              </Card>
            </Col>
          </Row>
          <Descriptions size="small" column={2} style={{ marginTop: 20 }}>
            <Descriptions.Item label="已选费用">
              {selectedIds.length} 笔
            </Descriptions.Item>
            <Descriptions.Item label="预览机制">
              服务端实时拆分并签发快照令牌
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {current === 2 && preview?.data && (
        <Form form={form} layout="vertical">
          <Alert
            type="success"
            showIcon
            style={{ marginBottom: 16 }}
            message={`服务端已拆分为 ${preview.data.length} 张账单`}
            description="提交时将再次锁定费用并校验快照；任何一笔费用变化，整个批次都不会部分落库。"
            action={
              <Button
                size="small"
                icon={<ReloadOutlined />}
                loading={loading}
                onClick={() => void loadPreview()}
              >
                重新预览
              </Button>
            }
          />
          {preview.data.map((group, index) => (
            <Card
              key={group.groupKey}
              size="small"
              style={{ marginBottom: 16 }}
              title={
                <Space wrap>
                  <Tag
                    color={
                      group.direction === 'RECEIVABLE' ? 'green' : 'volcano'
                    }
                  >
                    {directionText(group.direction)}
                  </Tag>
                  <span>{group.settlementPartyName}</span>
                  <Text type="secondary">
                    {group.orderNo ? `订单 ${group.orderNo}` : '多订单汇总'}
                  </Text>
                  {group.taxRate != null && (
                    <Tag>{Number(group.taxRate)}% 税率</Tag>
                  )}
                </Space>
              }
              extra={
                <Text strong>
                  {group.totalAmount} {group.currency}
                </Text>
              }
            >
              <Row gutter={16}>
                <Col xs={24} md={8}>
                  <Form.Item
                    name={['groups', index, 'statementTitle']}
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
                    <Input maxLength={200} />
                  </Form.Item>
                </Col>
                <Col xs={24} md={5}>
                  <Form.Item
                    name={['groups', index, 'billDate']}
                    label="账单日期"
                    rules={[{ required: true, message: '请选择账单日期' }]}
                  >
                    <DatePicker allowClear={false} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col xs={24} md={4}>
                  <Form.Item
                    name={['groups', index, 'paymentTermsDays']}
                    label="账期（天，可选）"
                  >
                    <InputNumber
                      min={0}
                      max={3650}
                      precision={0}
                      style={{ width: '100%' }}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} md={7}>
                  <Form.Item
                    name={['groups', index, 'note']}
                    label="备注"
                    rules={[{ max: 500, message: '备注不能超过 500 字' }]}
                  >
                    <Input maxLength={500} />
                  </Form.Item>
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
                columns={feeColumns.filter(
                  (column) => column.dataIndex !== 'direction',
                )}
                dataSource={group.fees || []}
                scroll={{ x: 850 }}
              />
            </Card>
          ))}
        </Form>
      )}

      {current === 3 &&
        (result ? (
          <>
            <Result
              status="success"
              icon={<CheckCircleOutlined />}
              title={`批次 ${result.batchNo || ''} 生成成功`}
              subTitle={`${result.feeCount || 0} 笔费用已原子生成 ${result.billCount || 0} 张账单，当前${result.bills?.every((bill) => bill.status === 'CONFIRMED') ? '已全部确认' : '为草稿状态'}，未发生部分成功。`}
              extra={
                <Space wrap>
                  {access.canConfirmFinanceBills &&
                    result.bills?.every((bill) => bill.status === 'DRAFT') && (
                      <Button
                        type="primary"
                        loading={confirming}
                        onClick={() => void confirmBatch()}
                      >
                        确认本批全部账单
                      </Button>
                    )}
                  <Button onClick={() => history.push('/finance/invoices')}>
                    前往开票 / 来票
                  </Button>
                  <Button
                    onClick={() => history.push('/finance/verifications')}
                  >
                    前往核销管理
                  </Button>
                </Space>
              }
            />
            <Descriptions
              bordered
              size="small"
              column={4}
              style={{ marginBottom: 16 }}
            >
              <Descriptions.Item label="批次号">
                {result.batchNo}
              </Descriptions.Item>
              <Descriptions.Item label="费用数">
                {result.feeCount}
              </Descriptions.Item>
              <Descriptions.Item label="账单数">
                {result.billCount}
              </Descriptions.Item>
              <Descriptions.Item label="本币合计">
                {result.totalBaseAmount} {result.baseCurrency}
              </Descriptions.Item>
            </Descriptions>
            <Table<API.FinanceBill>
              rowKey="id"
              size="small"
              bordered
              pagination={false}
              dataSource={result.bills || []}
              columns={[
                { title: '账单编号', dataIndex: 'billNo', width: 180 },
                {
                  title: '状态',
                  dataIndex: 'status',
                  width: 90,
                  render: (value) =>
                    value === 'CONFIRMED' ? (
                      <Tag color="green">已确认</Tag>
                    ) : (
                      <Tag color="gold">草稿</Tag>
                    ),
                },
                {
                  title: '方向',
                  dataIndex: 'direction',
                  width: 80,
                  render: (value) => directionText(String(value)),
                },
                { title: '结算单位', dataIndex: 'settlementPartyName' },
                { title: '对账抬头', dataIndex: 'statementTitle' },
                {
                  title: '金额',
                  align: 'right',
                  render: (_, row) => `${row.totalAmount} ${row.currency}`,
                },
                { title: '到期日', dataIndex: 'dueDate', width: 120 },
              ]}
            />
          </>
        ) : (
          <Empty />
        ))}
    </Drawer>
  );
}
