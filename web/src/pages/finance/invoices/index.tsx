import {
  CheckOutlined,
  CloseCircleOutlined,
  EyeOutlined,
  PlusOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import {
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import {
  FinanceLedgerTemplate,
  type FinanceLedgerMetricCard,
} from '@/components/ui';
import {
  App,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  Modal,
  Select,
  Table,
  Tag,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import { partnerServiceListPartnerInvoiceProfiles } from '@/services/roncin/partnerService';
import {
  settlementServiceCancelInvoice,
  settlementServiceCreateInvoice,
  settlementServiceGetInvoice,
  settlementServiceIssueInvoice,
  settlementServiceListBills,
  settlementServiceListInvoices,
  settlementServiceRedFlushInvoice,
} from '@/services/roncin/settlementService';

const { Text } = Typography;

const states: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '待开具', color: 'gold' },
  ISSUED: { text: '已开具', color: 'green' },
  CANCELLED: { text: '已取消/作废', color: 'default' },
  RED_FLUSHED: { text: '已红冲', color: 'red' },
};
type CreateValues = {
  invoiceProfileId: string;
  invoiceType: string;
  note?: string;
};
type IssueValues = { taxInvoiceNo: string; invoiceDate: Dayjs };
type RedFlushValues = {
  redInvoiceNo: string;
  redInvoiceDate: Dayjs;
  reason: string;
};

export default function FinanceInvoicesPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [createForm] = Form.useForm<CreateValues>();
  const [issueForm] = Form.useForm<IssueValues>();
  const [redFlushForm] = Form.useForm<RedFlushValues>();
  const [createOpen, setCreateOpen] = useState(false);
  const [issueTarget, setIssueTarget] = useState<API.FinanceInvoice>();
  const [redFlushTarget, setRedFlushTarget] = useState<API.FinanceInvoice>();
  const [selectedIDs, setSelectedIDs] = useState<React.Key[]>([]);
  const [selectedBills, setSelectedBills] = useState<API.FinanceBill[]>([]);
  const [availableProfiles, setAvailableProfiles] = useState<
    API.PartnerInvoiceProfile[]
  >([]);
  const [selectedProfile, setSelectedProfile] =
    useState<API.PartnerInvoiceProfile>();
  const [submitting, setSubmitting] = useState(false);
  const [detail, setDetail] = useState<API.FinanceInvoice>();
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    receivableTotal: 0,
    payableTotal: 0,
    issuedCount: 0,
  });
  const reload = () => actionRef.current?.reload();

  const loadSelectedProfiles = async (partnerId?: string) => {
    if (!partnerId) {
      setAvailableProfiles([]);
      setSelectedProfile(undefined);
      createForm.setFieldValue('invoiceProfileId', undefined);
      return;
    }
    try {
      const response = await partnerServiceListPartnerInvoiceProfiles(
        { partnerId },
        { skipErrorHandler: true },
      );
      const profiles = (response.data || []).filter((item) => item.enabled);
      setAvailableProfiles(profiles);
      const selected = profiles.find((item) => item.isDefault) || profiles[0];
      setSelectedProfile(selected);
      createForm.setFieldValue('invoiceProfileId', selected?.id);
      if (selected?.defaultInvoiceType) {
        createForm.setFieldValue('invoiceType', selected.defaultInvoiceType);
      }
      if (!selected) {
        message.warning(
          '该结算单位尚未配置可用开票抬头，请先到往来单位档案维护',
        );
      }
    } catch (rawError: any) {
      setAvailableProfiles([]);
      setSelectedProfile(undefined);
      createForm.setFieldValue('invoiceProfileId', undefined);
      message.error(rawError.message || '加载开票抬头失败');
    }
  };

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
        invoiceProfileId: values.invoiceProfileId,
        invoiceType: values.invoiceType,
        note: values.note,
        idempotencyKey: globalThis.crypto.randomUUID(),
      });
      message.success('开票记录已创建，账单已占用');
      setCreateOpen(false);
      reload();
    } catch (error: any) {
      const reason = error.data?.reason || error.response?.data?.reason;
      if (reason === 'FINANCE_INVOICE_PROFILE_REQUIRED') {
        message.error('请选择该结算单位下启用且完整的开票抬头');
      } else {
        message.error(error.message || '创建开票记录失败');
      }
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
  const redFlushInvoice = async () => {
    if (!redFlushTarget?.id || !redFlushTarget.version) return;
    const values = await redFlushForm.validateFields();
    setSubmitting(true);
    try {
      await settlementServiceRedFlushInvoice(
        { id: redFlushTarget.id },
        {
          id: redFlushTarget.id,
          expectedVersion: redFlushTarget.version,
          redInvoiceNo: values.redInvoiceNo,
          redInvoiceDate: values.redInvoiceDate.format('YYYY-MM-DD'),
          reason: values.reason,
        },
      );
      message.success('发票已红冲，原账单开票占用已释放');
      setRedFlushTarget(undefined);
      reload();
    } catch (error: any) {
      message.error(error.message || '发票红冲失败');
    } finally {
      setSubmitting(false);
    }
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
        access.canUpdateFinanceInvoices &&
        r.status !== 'CANCELLED' &&
        r.status !== 'RED_FLUSHED' ? (
          <a
            key="cancel"
            style={{ color: '#ff4d4f' }}
            onClick={() => cancelInvoice(r)}
          >
            <CloseCircleOutlined /> {r.status === 'ISSUED' ? '作废' : '取消'}
          </a>
        ) : null,
        access.canUpdateFinanceInvoices && r.status === 'ISSUED' ? (
          <a
            key="red-flush"
            style={{ color: '#cf1322' }}
            onClick={() => {
              setRedFlushTarget(r);
              redFlushForm.setFieldsValue({ redInvoiceDate: dayjs() });
            }}
          >
            <SwapOutlined /> 红冲
          </a>
        ) : null,
      ],
    },
  ];
  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'total-invoices',
      title: '发票总记录数',
      value: metricStats.totalCount,
      suffix: '笔',
    },
    {
      key: 'rec-invoices',
      title: '销项发票金额',
      value: metricStats.receivableTotal,
      precision: 2,
      suffix: 'CNY',
      valueColor: '#1677ff',
    },
    {
      key: 'pay-invoices',
      title: '进项发票金额',
      value: metricStats.payableTotal,
      precision: 2,
      suffix: 'CNY',
      valueColor: '#fa8c16',
    },
    {
      key: 'issued-count',
      title: '已正式开具',
      value: metricStats.issuedCount,
      suffix: '笔',
      valueColor: '#52c41a',
    },
  ];

  return (
    <>
      <FinanceLedgerTemplate<API.FinanceInvoice>
        headerTitle="发票明细管理"
        actionRef={actionRef}
        columns={columns}
        metricCards={metricCards}
        scrollX={1600}
        primaryActionText="新建开票记录"
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={
          access.canCreateFinanceInvoices
            ? () => {
                setSelectedIDs([]);
                setSelectedBills([]);
                setAvailableProfiles([]);
                setSelectedProfile(undefined);
                createForm.resetFields();
                createForm.setFieldsValue({ invoiceType: 'NORMAL' });
                setCreateOpen(true);
              }
            : undefined
        }
        request={async (p) => {
          const r = await settlementServiceListInvoices({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            direction: p.direction,
            status: p.status,
          });
          const list = r.data || [];
          let recTotal = 0;
          let payTotal = 0;
          let issued = 0;
          for (const item of list) {
            const amount = Number(item.totalAmount || 0);
            if (item.direction === 'RECEIVABLE') {
              recTotal += amount;
            } else if (item.direction === 'PAYABLE') {
              payTotal += amount;
            }
            if (item.status === 'ISSUED') {
              issued += 1;
            }
          }
          setMetricStats({
            totalCount: Number(r.total || 0),
            receivableTotal: recTotal,
            payableTotal: payTotal,
            issuedCount: issued,
          });
          return {
            data: list,
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
            name="invoiceProfileId"
            label="开票抬头"
            rules={[{ required: true, message: '请选择开票抬头' }]}
          >
            <Select
              style={{ width: 300 }}
              placeholder={
                selectedBills[0] ? '请选择该客户的开票抬头' : '请先选择账单'
              }
              disabled={!selectedBills[0]}
              options={availableProfiles.map((item) => ({
                value: item.id,
                label: `${item.invoiceTitle}${item.isDefault ? '（默认）' : ''}`,
              }))}
              onChange={(id) => {
                const profile = availableProfiles.find(
                  (item) => item.id === id,
                );
                setSelectedProfile(profile);
                if (profile?.defaultInvoiceType) {
                  createForm.setFieldValue(
                    'invoiceType',
                    profile.defaultInvoiceType,
                  );
                }
              }}
            />
          </Form.Item>
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
        {selectedBills[0] && (
          <Descriptions
            bordered
            size="small"
            column={3}
            style={{ marginBottom: 12 }}
          >
            <Descriptions.Item label="已选开票抬头" span={2}>
              {selectedProfile?.invoiceTitle || (
                <Text type="danger">未配置</Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="默认票种">
              {selectedProfile?.defaultInvoiceType === 'SPECIAL'
                ? '专用发票'
                : selectedProfile
                  ? '普通发票'
                  : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="纳税人识别号" span={3}>
              {selectedProfile?.taxpayerIdentificationNo || '-'}
            </Descriptions.Item>
          </Descriptions>
        )}
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
              const first = keys.map((key) => m.get(String(key))).find(Boolean);
              if (
                first?.settlementPartyId !== selectedBills[0]?.settlementPartyId
              ) {
                void loadSelectedProfiles(first?.settlementPartyId);
              } else if (!first) {
                void loadSelectedProfiles(undefined);
              }
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
      <Modal
        title={`红冲发票 ${redFlushTarget?.taxInvoiceNo || ''}`}
        open={Boolean(redFlushTarget)}
        confirmLoading={submitting}
        okButtonProps={{ danger: true }}
        okText="确认红冲"
        onCancel={() => setRedFlushTarget(undefined)}
        onOk={() => void redFlushInvoice()}
      >
        <Form form={redFlushForm} layout="vertical">
          <Form.Item
            name="redInvoiceNo"
            label="红字发票号码"
            rules={[{ required: true, message: '请输入红字发票号码' }]}
          >
            <Input maxLength={100} />
          </Form.Item>
          <Form.Item
            name="redInvoiceDate"
            label="红冲日期"
            rules={[{ required: true, message: '请选择红冲日期' }]}
          >
            <DatePicker />
          </Form.Item>
          <Form.Item
            name="reason"
            label="红冲原因"
            rules={[{ required: true, message: '请输入红冲原因' }]}
          >
            <Input.TextArea maxLength={500} rows={3} />
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
