import {
  CheckOutlined,
  CloseCircleOutlined,
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
  Modal,
  Select,
  Table,
  Tag,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  settlementServiceCancelInvoice,
  settlementServiceCreateInvoice,
  settlementServiceGetInvoice,
  settlementServiceIssueInvoice,
  settlementServiceListBills,
  settlementServiceListInvoices,
} from '@/services/roncin/settlementService';

const states: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '待开具', color: 'gold' },
  ISSUED: { text: '已开具', color: 'green' },
  CANCELLED: { text: '已取消', color: 'default' },
};
type CreateValues = { invoiceType: string; note?: string };
type IssueValues = { taxInvoiceNo: string; invoiceDate: Dayjs };

export default function FinanceInvoicesPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [createForm] = Form.useForm<CreateValues>();
  const [issueForm] = Form.useForm<IssueValues>();
  const [createOpen, setCreateOpen] = useState(false);
  const [issueTarget, setIssueTarget] = useState<API.FinanceInvoice>();
  const [selectedIDs, setSelectedIDs] = useState<React.Key[]>([]);
  const [selectedBills, setSelectedBills] = useState<API.FinanceBill[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [detail, setDetail] = useState<API.FinanceInvoice>();
  const reload = () => actionRef.current?.reload();

  const showDetail = async (row: API.FinanceInvoice) => {
    if (!row.id) return;
    try {
      setDetail((await settlementServiceGetInvoice({ id: row.id })).data);
    } catch (error: any) {
      message.error(error.message || '加载开票详情失败');
    }
  };
  const createInvoice = async () => {
    const values = await createForm.validateFields();
    if (!selectedIDs.length) {
      message.warning('请至少选择一张已确认账单');
      return;
    }
    setSubmitting(true);
    try {
      await settlementServiceCreateInvoice({
        billIds: selectedIDs.map(String),
        invoiceType: values.invoiceType,
        note: values.note,
        idempotencyKey: globalThis.crypto.randomUUID(),
      });
      message.success('开票记录已创建，账单已占用');
      setCreateOpen(false);
      reload();
    } catch (error: any) {
      message.error(error.message || '创建开票记录失败');
    } finally {
      setSubmitting(false);
    }
  };
  const issueInvoice = async () => {
    if (!issueTarget?.id || !issueTarget.version) return;
    const values = await issueForm.validateFields();
    setSubmitting(true);
    try {
      await settlementServiceIssueInvoice(
        { id: issueTarget.id },
        {
          id: issueTarget.id,
          expectedVersion: issueTarget.version,
          taxInvoiceNo: values.taxInvoiceNo,
          invoiceDate: values.invoiceDate.format('YYYY-MM-DD'),
        },
      );
      message.success('发票已确认开具');
      setIssueTarget(undefined);
      reload();
    } catch (error: any) {
      message.error(error.message || '确认开具失败');
    } finally {
      setSubmitting(false);
    }
  };
  const cancelInvoice = (row: API.FinanceInvoice) => {
    const id = row.id,
      version = row.version;
    if (!id || !version) return;
    let reason = '';
    modal.confirm({
      title: '取消开票记录并释放账单？',
      content: (
        <Input.TextArea
          placeholder="请输入取消原因（必填）"
          maxLength={500}
          onChange={(e) => {
            reason = e.target.value.trim();
          }}
        />
      ),
      okButtonProps: { danger: true },
      onOk: async () => {
        if (!reason) {
          message.warning('请输入取消原因');
          throw new Error('取消原因不能为空');
        }
        await settlementServiceCancelInvoice(
          { id },
          { id, expectedVersion: version, reason },
        );
        message.success('开票记录已取消，账单已释放');
        reload();
      },
    });
  };
  const columns: ProColumns<API.FinanceInvoice>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '记录编号、发票号或结算单位' },
    },
    {
      title: '记录编号',
      dataIndex: 'recordNo',
      width: 160,
      copyable: true,
      search: false,
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 80,
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '销项' }, PAYABLE: { text: '进项' } },
      render: (_, r) => (
        <Tag color={r.direction === 'RECEIVABLE' ? 'blue' : 'purple'}>
          {r.direction === 'RECEIVABLE' ? '销项' : '进项'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(states).map(([k, v]) => [k, { text: v.text }]),
      ),
      render: (_, r) => {
        const v = states[r.status || 'DRAFT'];
        return <Tag color={v.color}>{v.text}</Tag>;
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
      title: '发票类型',
      dataIndex: 'invoiceType',
      width: 100,
      search: false,
      renderText: (v) => (v === 'SPECIAL' ? '专用发票' : '普通发票'),
    },
    {
      title: '金额',
      dataIndex: 'totalAmount',
      width: 150,
      align: 'right',
      search: false,
      render: (_, r) => (
        <strong>
          {r.totalAmount} {r.currency}
        </strong>
      ),
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      width: 120,
      align: 'right',
      search: false,
    },
    {
      title: '税务发票号',
      dataIndex: 'taxInvoiceNo',
      width: 160,
      search: false,
      renderText: (v) => v || '-',
    },
    {
      title: '开票日期',
      dataIndex: 'invoiceDate',
      width: 110,
      search: false,
      renderText: (v) => v || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 210,
      render: (_, r) => [
        <a key="detail" onClick={() => void showDetail(r)}>
          <EyeOutlined /> 详情
        </a>,
        access.canUpdateFinanceInvoices && r.status === 'DRAFT' ? (
          <a
            key="issue"
            onClick={() => {
              setIssueTarget(r);
              issueForm.setFieldsValue({ invoiceDate: dayjs() });
            }}
          >
            <CheckOutlined /> 确认开具
          </a>
        ) : null,
        access.canUpdateFinanceInvoices && r.status !== 'CANCELLED' ? (
          <a
            key="cancel"
            style={{ color: '#ff4d4f' }}
            onClick={() => cancelInvoice(r)}
          >
            <CloseCircleOutlined /> 取消
          </a>
        ) : null,
      ],
    },
  ];
  return (
    <>
      <ProTable<API.FinanceInvoice>
        headerTitle="开票记录"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 1450 }}
        toolBarRender={() =>
          access.canCreateFinanceInvoices
            ? [
                <Button
                  key="new"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setSelectedIDs([]);
                    setSelectedBills([]);
                    createForm.setFieldsValue({ invoiceType: 'NORMAL' });
                    setCreateOpen(true);
                  }}
                >
                  新建开票记录
                </Button>,
              ]
            : []
        }
        request={async (p) => {
          const r = await settlementServiceListInvoices({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            direction: p.direction,
            status: p.status,
          });
          return {
            data: r.data || [],
            total: Number(r.total || 0),
            success: r.success ?? true,
          };
        }}
      />
      <Modal
        title="从已确认账单创建开票记录"
        open={createOpen}
        width={1050}
        confirmLoading={submitting}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void createInvoice()}
      >
        <Form form={createForm} layout="inline" style={{ marginBottom: 12 }}>
          <Form.Item
            name="invoiceType"
            label="发票类型"
            rules={[{ required: true }]}
          >
            <Select
              style={{ width: 150 }}
              options={[
                { value: 'NORMAL', label: '普通发票' },
                { value: 'SPECIAL', label: '专用发票' },
              ]}
            />
          </Form.Item>
          <Form.Item name="note" label="备注">
            <Input style={{ width: 360 }} maxLength={500} />
          </Form.Item>
        </Form>
        <ProTable<API.FinanceBill>
          rowKey="id"
          options={false}
          size="small"
          bordered
          columns={[
            { title: '账单编号', dataIndex: 'billNo' },
            {
              title: '方向',
              dataIndex: 'direction',
              renderText: (v) => (v === 'RECEIVABLE' ? '销项' : '进项'),
            },
            { title: '结算单位', dataIndex: 'settlementPartyName' },
            {
              title: '金额',
              render: (_, r) => `${r.totalAmount} ${r.currency}`,
            },
            { title: '税额', dataIndex: 'taxAmount' },
          ]}
          rowSelection={{
            selectedRowKeys: selectedIDs,
            preserveSelectedRowKeys: true,
            onChange: (keys, rows) => {
              const m = new Map(selectedBills.map((x) => [x.id, x]));
              rows.forEach((x) => {
                m.set(x.id, x);
              });
              setSelectedIDs(keys);
              setSelectedBills(
                keys
                  .map((k) => m.get(String(k)))
                  .filter(Boolean) as API.FinanceBill[],
              );
            },
            getCheckboxProps: (r) => {
              const f = selectedBills[0];
              return {
                disabled:
                  Boolean(f) &&
                  (r.direction !== f.direction ||
                    r.settlementPartyId !== f.settlementPartyId ||
                    r.currency !== f.currency),
              };
            },
          }}
          request={async (p) => {
            const r = await settlementServiceListBills({
              page: p.current,
              pageSize: p.pageSize,
              status: 'CONFIRMED',
            });
            return {
              data: r.data || [],
              total: Number(r.total || 0),
              success: r.success ?? true,
            };
          }}
        />
      </Modal>
      <Modal
        title="确认开具发票"
        open={Boolean(issueTarget)}
        confirmLoading={submitting}
        onCancel={() => setIssueTarget(undefined)}
        onOk={() => void issueInvoice()}
      >
        <Form form={issueForm} layout="vertical">
          <Form.Item
            name="taxInvoiceNo"
            label="税务发票号码"
            rules={[{ required: true, message: '请输入税务发票号码' }]}
          >
            <Input maxLength={100} />
          </Form.Item>
          <Form.Item
            name="invoiceDate"
            label="开票日期"
            rules={[{ required: true, message: '请选择开票日期' }]}
          >
            <DatePicker />
          </Form.Item>
        </Form>
      </Modal>
      <Drawer
        title={`开票详情 ${detail?.recordNo || ''}`}
        open={Boolean(detail)}
        width={760}
        onClose={() => setDetail(undefined)}
      >
        {detail && (
          <>
            <Descriptions bordered size="small" column={2}>
              <Descriptions.Item label="状态">
                <Tag color={states[detail.status || 'DRAFT'].color}>
                  {states[detail.status || 'DRAFT'].text}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="税务发票号">
                {detail.taxInvoiceNo || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="结算单位">
                {detail.settlementPartyName}
              </Descriptions.Item>
              <Descriptions.Item label="金额">
                {detail.totalAmount} {detail.currency}
              </Descriptions.Item>
              <Descriptions.Item label="开票日期">
                {detail.invoiceDate || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="税额">
                {detail.taxAmount}
              </Descriptions.Item>
              <Descriptions.Item label="备注" span={2}>
                {detail.note || '-'}
              </Descriptions.Item>
              {detail.cancellationReason && (
                <Descriptions.Item label="取消原因" span={2}>
                  {detail.cancellationReason}
                </Descriptions.Item>
              )}
            </Descriptions>
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
                    r.active ? <Tag color="blue">有效</Tag> : <Tag>已释放</Tag>,
                },
              ]}
            />
          </>
        )}
      </Drawer>
    </>
  );
}
