import {
  CheckOutlined,
  CloseCircleOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import {
  App,
  Button,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import { useRef, useState } from 'react';
import {
  settlementServiceCancelBill,
  settlementServiceConfirmBill,
  settlementServiceGetBill,
  settlementServiceListBills,
  settlementServiceUpdateBill,
} from '@/services/roncin/settlementService';
import BillCreationWorkbench from './components/BillCreationWorkbench';

const statusOptions: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'gold' },
  CONFIRMED: { text: '已确认', color: 'green' },
  CANCELLED: { text: '已取消', color: 'default' },
};

type BillFormValues = {
  billDate: Dayjs;
  statementTitle: string;
  paymentTermsDays?: number;
  note?: string;
};

export default function FinanceBillsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [form] = Form.useForm<BillFormValues>();
  const [workbenchOpen, setWorkbenchOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<API.FinanceBill>();
  const [submitting, setSubmitting] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<API.FinanceBill>();

  const reload = () => actionRef.current?.reload();
  const openCreate = () => {
    setWorkbenchOpen(true);
  };
  const openEdit = (bill: API.FinanceBill) => {
    setEditing(bill);
    form.setFieldsValue({
      billDate: bill.billDate ? dayjs(bill.billDate) : dayjs(),
      statementTitle: bill.statementTitle || bill.settlementPartyName || '',
      paymentTermsDays: bill.paymentTermsDays,
      note: bill.note,
    });
    setEditOpen(true);
  };
  const openDetail = async (bill: API.FinanceBill) => {
    if (!bill.id) return;
    setDetailOpen(true);
    setDetailLoading(true);
    try {
      const response = await settlementServiceGetBill({ id: bill.id });
      setDetail(response.data);
    } catch (error: any) {
      message.error(error.message || '加载账单详情失败');
    } finally {
      setDetailLoading(false);
    }
  };
  const submitBill = async () => {
    const values = await form.validateFields();
    if (!editing?.id) return;
    setSubmitting(true);
    try {
      const billDate = values.billDate.format('YYYY-MM-DD');
      await settlementServiceUpdateBill(
        { id: editing.id },
        {
          id: editing.id,
          billDate,
          statementTitle: values.statementTitle.trim(),
          paymentTermsDays: values.paymentTermsDays,
          note: values.note,
          expectedVersion: editing.version || '0',
        },
      );
      message.success('账单更新成功');
      setEditOpen(false);
      reload();
    } catch (error: any) {
      message.error(error.message || '保存账单失败');
    } finally {
      setSubmitting(false);
    }
  };
  const confirmBill = async (bill: API.FinanceBill) => {
    if (!bill.id || !bill.version) return;
    try {
      await settlementServiceConfirmBill(
        { id: bill.id },
        { id: bill.id, expectedVersion: bill.version },
      );
      message.success('账单已确认');
      reload();
    } catch (error: any) {
      message.error(error.message || '确认账单失败');
    }
  };
  const cancelBill = (bill: API.FinanceBill) => {
    const billID = bill.id;
    const version = bill.version;
    if (!billID || !version) return;
    let reason = '';
    modal.confirm({
      title: '取消账单并释放费用？',
      content: (
        <Input.TextArea
          autoFocus
          maxLength={500}
          showCount
          placeholder="请输入取消原因（必填）"
          onChange={(event) => {
            reason = event.target.value.trim();
          }}
        />
      ),
      okText: '确认取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        if (!reason) {
          message.warning('请输入取消原因');
          throw new Error('取消原因不能为空');
        }
        await settlementServiceCancelBill(
          { id: billID },
          { id: billID, expectedVersion: version, reason },
        );
        message.success('账单已取消，费用已释放回已确认状态');
        reload();
      },
    });
  };

  const columns: ProColumns<API.FinanceBill>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '账单号、结算单位或订单号' },
    },
    {
      title: '账单编号',
      dataIndex: 'billNo',
      width: 170,
      copyable: true,
      search: false,
    },
    {
      title: '建单批次',
      dataIndex: 'batchNo',
      width: 175,
      copyable: true,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 80,
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
      render: (_, row) => (
        <Tag color={row.direction === 'RECEIVABLE' ? 'green' : 'volcano'}>
          {row.direction === 'RECEIVABLE' ? '应收' : '应付'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(statusOptions).map(([key, value]) => [
          key,
          { text: value.text },
        ]),
      ),
      render: (_, row) => {
        const value = statusOptions[row.status || 'DRAFT'];
        return <Tag color={value.color}>{value.text}</Tag>;
      },
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 220,
      ellipsis: true,
      search: false,
    },
    {
      title: '对账抬头',
      dataIndex: 'statementTitle',
      width: 200,
      ellipsis: true,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '账单金额',
      dataIndex: 'totalAmount',
      width: 150,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong>
          {row.totalAmount} {row.currency}
        </strong>
      ),
    },
    {
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 160,
      align: 'right',
      search: false,
      render: (_, row) => `${row.baseCurrencyAmount} ${row.baseCurrency}`,
    },
    {
      title: '已核销',
      dataIndex: 'verifiedAmount',
      width: 145,
      align: 'right',
      search: false,
      render: (_, row) =>
        `${row.verifiedAmount || '0.00000000'} ${row.currency}`,
    },
    {
      title: '未核销',
      dataIndex: 'unverifiedAmount',
      width: 145,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong
          style={{
            color:
              Number(row.unverifiedAmount || 0) > 0 ? '#cf1322' : '#389e0d',
          }}
        >
          {row.unverifiedAmount || '0.00000000'} {row.currency}
        </strong>
      ),
    },
    { title: '费用数', dataIndex: 'feeCount', width: 80, search: false },
    {
      title: '账单日期',
      dataIndex: 'billDate',
      width: 120,
      valueType: 'dateRange',
      search: {
        transform: (value) => ({
          billDateFrom: value[0],
          billDateTo: value[1],
        }),
      },
    },
    {
      title: '到期日',
      dataIndex: 'dueDate',
      width: 110,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 230,
      render: (_, row) => [
        <a key="detail" onClick={() => void openDetail(row)}>
          <EyeOutlined /> 详情
        </a>,
        access.canUpdateFinanceBills && row.status === 'DRAFT' ? (
          <a key="edit" onClick={() => openEdit(row)}>
            <EditOutlined /> 编辑
          </a>
        ) : null,
        access.canConfirmFinanceBills && row.status === 'DRAFT' ? (
          <Popconfirm
            key="confirm"
            title="确认后账单将进入正式结算流程，是否继续？"
            onConfirm={() => void confirmBill(row)}
          >
            <a>
              <CheckOutlined /> 确认
            </a>
          </Popconfirm>
        ) : null,
        access.canUpdateFinanceBills && row.status !== 'CANCELLED' ? (
          <a
            key="cancel"
            style={{ color: '#ff4d4f' }}
            onClick={() => cancelBill(row)}
          >
            <CloseCircleOutlined /> 取消
          </a>
        ) : null,
      ],
    },
  ];
  return (
    <>
      <ProTable<API.FinanceBill>
        headerTitle="账单管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 1800 }}
        toolBarRender={() =>
          access.canCreateFinanceBills
            ? [
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openCreate}
                >
                  生成账单
                </Button>,
              ]
            : []
        }
        request={async (params) => {
          const response = await settlementServiceListBills({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            direction: params.direction,
            status: params.status,
            billDateFrom: params.billDateFrom,
            billDateTo: params.billDateTo,
          });
          return {
            data: response.data || [],
            total: Number(response.total || 0),
            success: response.success ?? true,
          };
        }}
      />
      <Modal
        title={`编辑账单 ${editing?.billNo || ''}`}
        open={editOpen}
        width={680}
        destroyOnHidden
        confirmLoading={submitting}
        okText="保存"
        onCancel={() => setEditOpen(false)}
        onOk={() => void submitBill()}
      >
        <Form form={form} layout="vertical">
          <Space size={16} align="start" wrap style={{ width: '100%' }}>
            <Form.Item
              name="statementTitle"
              label="对账抬头"
              rules={[
                { required: true, whitespace: true, message: '请输入对账抬头' },
                { max: 200, message: '对账抬头不能超过 200 字' },
              ]}
              style={{ minWidth: 260 }}
            >
              <Input maxLength={200} />
            </Form.Item>
            <Form.Item
              name="billDate"
              label="账单日期"
              rules={[{ required: true, message: '请选择账单日期' }]}
            >
              <DatePicker allowClear={false} />
            </Form.Item>
            <Form.Item name="paymentTermsDays" label="账期（天，可选）">
              <InputNumber min={0} max={3650} precision={0} />
            </Form.Item>
            <Form.Item name="note" label="备注" style={{ minWidth: 620 }}>
              <Input maxLength={500} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
      <BillCreationWorkbench
        open={workbenchOpen}
        onClose={() => setWorkbenchOpen(false)}
        onCreated={() => reload()}
      />
      <Drawer
        title={`账单详情 ${detail?.billNo || ''}`}
        open={detailOpen}
        width={980}
        loading={detailLoading}
        onClose={() => setDetailOpen(false)}
      >
        {detail && (
          <>
            <Descriptions
              bordered
              size="small"
              column={3}
              style={{ marginBottom: 16 }}
            >
              <Descriptions.Item label="状态">
                <Tag color={statusOptions[detail.status || 'DRAFT'].color}>
                  {statusOptions[detail.status || 'DRAFT'].text}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="方向">
                {detail.direction === 'RECEIVABLE' ? '应收' : '应付'}
              </Descriptions.Item>
              <Descriptions.Item label="结算单位">
                {detail.settlementPartyName}
              </Descriptions.Item>
              <Descriptions.Item label="建单批次">
                {detail.batchNo || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="对账抬头">
                {detail.statementTitle || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="账单金额">
                {detail.totalAmount} {detail.currency}
              </Descriptions.Item>
              <Descriptions.Item label="折本币">
                {detail.baseCurrencyAmount} {detail.baseCurrency}
              </Descriptions.Item>
              <Descriptions.Item label="税额">
                {detail.taxAmount} {detail.currency}
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
              <Descriptions.Item label="费用数">
                {detail.feeCount}
              </Descriptions.Item>
              <Descriptions.Item label="备注" span={3}>
                {detail.note || '-'}
              </Descriptions.Item>
              {detail.cancellationReason && (
                <Descriptions.Item label="取消原因" span={3}>
                  {detail.cancellationReason}
                </Descriptions.Item>
              )}
            </Descriptions>
            <Table<API.FinanceBillLine>
              rowKey="id"
              size="small"
              bordered
              pagination={false}
              dataSource={detail.lines || []}
              columns={[
                { title: '订单编号', dataIndex: 'orderNo', width: 150 },
                { title: '费用代码', dataIndex: 'feeCode', width: 110 },
                { title: '费用名称', dataIndex: 'feeName', width: 140 },
                {
                  title: '税率',
                  dataIndex: 'taxRate',
                  align: 'right',
                  width: 90,
                  render: (value) =>
                    value == null ? '-' : `${Number(value)}%`,
                },
                {
                  title: '原币金额',
                  render: (_, row) => `${row.totalAmount} ${row.currency}`,
                  align: 'right',
                },
                { title: '税额', dataIndex: 'taxAmount', align: 'right' },
                {
                  title: '折本币',
                  render: (_, row) =>
                    `${row.baseCurrencyAmount} ${row.baseCurrency}`,
                  align: 'right',
                },
                {
                  title: '关联状态',
                  render: (_, row) =>
                    row.active ? (
                      <Tag color="blue">有效</Tag>
                    ) : (
                      <Tag>已释放</Tag>
                    ),
                  width: 90,
                },
              ]}
            />
          </>
        )}
      </Drawer>
    </>
  );
}
