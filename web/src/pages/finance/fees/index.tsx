import {
  CloudUploadOutlined,
  DownOutlined,
  DownloadOutlined,
  EyeOutlined,
  FileDoneOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import {
  ProTable,
  type ActionType,
  type ProColumns,
} from '@ant-design/pro-components';
import { history } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Col,
  Dropdown,
  Row,
  Space,
  Statistic,
  Tag,
  Tooltip,
} from 'antd';
import React, { useRef, useState } from 'react';
import { settlementServiceListFeeLedger } from '@/services/roncin/settlementService';
import BillCreationWorkbench from '@/pages/finance/bills/components/BillCreationWorkbench';
import FeeSummaryBoard from './components/FeeSummaryBoard';

const businessLabels: Record<string, string> = {
  SE: '海运出口',
  SI: '海运进口',
  AE: '空运出口',
  AI: '空运进口',
  LAND: '陆运',
  RAIL: '铁路',
};
const businessRoutes: Record<string, string> = {
  SE: 'sea-export',
  SI: 'sea-import',
  AE: 'air-export',
  AI: 'air-import',
};
const statusLabels: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'gold' },
  CONFIRMED: { text: '已确认', color: 'green' },
  BILLED: { text: '已进账单', color: 'blue' },
  CANCELLED: { text: '已作废', color: 'default' },
};

function amount(value?: string) {
  return Number(value || 0);
}

export default function FinanceFeeLedgerPage() {
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [summary, setSummary] = useState<API.FeeLedgerSummary>();
  const [currentData, setCurrentData] = useState<API.FeeLedgerItem[]>([]);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [selectedRows, setSelectedRows] = useState<API.FeeLedgerItem[]>([]);
  const [billWorkbenchOpen, setBillWorkbenchOpen] = useState(false);

  // 导出 CSV 功能
  const handleExport = () => {
    const list = selectedRows.length > 0 ? selectedRows : currentData;
    if (list.length === 0) {
      message.warning('当前无数据可导出');
      return;
    }
    const headers = [
      '订单编号',
      '业务类型',
      '收付方向',
      '状态',
      '费用科目',
      '费用代码',
      '结算单位',
      '币种',
      '单价',
      '数量',
      '计费单位',
      '含税金额',
      '汇率',
      '折本币金额',
      '税率',
      '税金',
      '不含税净额',
      '费用日期',
      '备注',
    ];
    const rows = list.map((item) => [
      `"${item.orderNo || ''}"`,
      `"${businessLabels[item.businessType || ''] || item.businessType || ''}"`,
      `"${item.direction === 'RECEIVABLE' ? '应收' : '应付'}"`,
      `"${statusLabels[item.status || 'DRAFT']?.text || item.status || ''}"`,
      `"${item.feeName || ''}"`,
      `"${item.feeCode || ''}"`,
      `"${item.settlementPartyName || ''}"`,
      `"${item.currency || ''}"`,
      `"${item.unitPrice || '0'}"`,
      `"${item.quantity || '0'}"`,
      `"${item.billingUnit || ''}"`,
      `"${item.totalAmount || '0'}"`,
      `"${item.exchangeRate || '1'}"`,
      `"${item.baseCurrencyAmount || '0'}"`,
      `"${item.taxRate ? `${Number(item.taxRate) * 100}%` : '0%'}"`,
      `"${item.taxAmount || '0'}"`,
      `"${item.netAmount || '0'}"`,
      `"${item.expenseDate || ''}"`,
      `"${(item.note || '').replace(/"/g, '""')}"`,
    ]);
    const csvContent =
      '\uFEFF' +
      [headers.join(','), ...rows.map((e) => e.join(','))].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute(
      'download',
      `费用明细导出_${new Date().toISOString().slice(0, 10)}.csv`,
    );
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    message.success(`已导出 ${list.length} 条费用明细`);
  };

  const columns: ProColumns<API.FeeLedgerItem>[] = [
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 55,
      fixed: 'left',
    },
    {
      title: '属性',
      dataIndex: 'direction',
      width: 75,
      fixed: 'left',
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '应收' }, PAYABLE: { text: '应付' } },
      render: (_, row) => (
        <Tag
          color={row.direction === 'RECEIVABLE' ? 'green' : 'volcano'}
          style={{ margin: 0 }}
        >
          {row.direction === 'RECEIVABLE' ? '应收' : '应付'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 85,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(statusLabels).map(([key, value]) => [
          key,
          { text: value.text },
        ]),
      ),
      render: (_, row) => {
        const value = statusLabels[row.status || 'DRAFT'];
        return (
          <Tag color={value.color} style={{ margin: 0 }}>
            {value.text}
          </Tag>
        );
      },
    },
    {
      title: '订单编号',
      dataIndex: 'orderNo',
      width: 155,
      copyable: true,
      render: (_, row) => {
        const route = businessRoutes[row.businessType || ''];
        return route ? (
          <a
            style={{ fontWeight: 500 }}
            onClick={() => history.push(`/orders/${route}/${row.orderId}`)}
          >
            {row.orderNo}
          </a>
        ) : (
          row.orderNo
        );
      },
    },
    {
      title: '业务类型',
      dataIndex: 'businessType',
      width: 95,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(businessLabels).map(([key, text]) => [key, { text }]),
      ),
    },
    {
      title: '费用科目',
      dataIndex: 'feeName',
      width: 130,
      search: false,
      render: (_, row) => (
        <span>
          <span style={{ fontWeight: 500 }}>{row.feeName}</span>
          <span style={{ color: '#8c8c8c', fontSize: 11, marginLeft: 4 }}>
            ({row.feeCode})
          </span>
        </span>
      ),
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 180,
      ellipsis: true,
      search: false,
      render: (val) => (
        <Tooltip title={val}>
          <span>{val}</span>
        </Tooltip>
      ),
    },
    {
      title: '币种',
      dataIndex: 'currency',
      width: 70,
      align: 'center',
      search: false,
      render: (val) => <Tag style={{ margin: 0 }}>{val}</Tag>,
    },
    {
      title: '单价',
      dataIndex: 'unitPrice',
      width: 90,
      align: 'right',
      search: false,
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      width: 75,
      align: 'right',
      search: false,
    },
    {
      title: '单位',
      dataIndex: 'billingUnit',
      width: 70,
      align: 'center',
      search: false,
    },
    {
      title: '原币金额',
      dataIndex: 'totalAmount',
      width: 125,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong style={{ color: '#262626' }}>
          {row.totalAmount} {row.currency}
        </strong>
      ),
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      width: 85,
      align: 'right',
      search: false,
    },
    {
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 135,
      align: 'right',
      search: false,
      render: (_, row) => (
        <strong
          style={{
            color: row.direction === 'RECEIVABLE' ? '#1677ff' : '#fa8c16',
          }}
        >
          {row.baseCurrencyAmount} {row.baseCurrency}
        </strong>
      ),
    },
    {
      title: '税率',
      dataIndex: 'taxRate',
      width: 75,
      align: 'right',
      search: false,
      render: (val) => (val ? `${Number(val) * 100}%` : '-'),
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      width: 95,
      align: 'right',
      search: false,
    },
    {
      title: '不含税金额',
      dataIndex: 'netAmount',
      width: 110,
      align: 'right',
      search: false,
    },
    {
      title: '费用日期',
      dataIndex: 'expenseDate',
      width: 110,
      valueType: 'dateRange',
      search: {
        transform: (value) => ({
          expenseDateFrom: value[0],
          expenseDateTo: value[1],
        }),
      },
    },
    {
      title: '备注说明',
      dataIndex: 'note',
      width: 130,
      ellipsis: true,
      search: false,
      render: (val) =>
        val ? (
          <Tooltip title={val}>
            <span>{val}</span>
          </Tooltip>
        ) : (
          '-'
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 110,
      fixed: 'right',
      render: (_, row) => [
        <a
          key="view-order"
          onClick={() => {
            const route = businessRoutes[row.businessType || ''];
            if (route) history.push(`/orders/${route}/${row.orderId}`);
          }}
        >
          查看订单
        </a>,
        <a
          key="to-bill"
          onClick={() => {
            if (row.id) {
              setSelectedRowKeys([row.id]);
              setBillWorkbenchOpen(true);
            }
          }}
        >
          转账单
        </a>,
      ],
    },
  ];

  return (
    <div style={{ paddingBottom: 24 }}>
      {/* 顶部宏观统计指标卡 */}
      <Row gutter={12} style={{ marginBottom: 12 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="有效费用总笔数"
              value={Number(summary?.activeCount || 0)}
              suffix="笔"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="应收折本币总池"
              value={amount(summary?.receivableBaseAmount)}
              precision={2}
              suffix={summary?.baseCurrency || 'CNY'}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="应付折本币总池"
              value={amount(summary?.payableBaseAmount)}
              precision={2}
              suffix={summary?.baseCurrency || 'CNY'}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="确认综合毛利"
              value={amount(summary?.profitBaseAmount)}
              precision={2}
              suffix={summary?.baseCurrency || 'CNY'}
              valueStyle={{
                color:
                  amount(summary?.profitBaseAmount) >= 0
                    ? '#52c41a'
                    : '#ff4d4f',
              }}
            />
          </Card>
        </Col>
      </Row>

      <ProTable<API.FeeLedgerItem>
        headerTitle="集运费用明细"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 2100 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        toolBarRender={() => [
          <Button
            key="create-bill"
            type="primary"
            icon={<FileDoneOutlined />}
            disabled={selectedRowKeys.length === 0}
            onClick={() => setBillWorkbenchOpen(true)}
          >
            创建账单{' '}
            {selectedRowKeys.length > 0 ? `(${selectedRowKeys.length})` : ''}
          </Button>,
          <Dropdown
            key="batch-ops"
            menu={{
              items: [
                {
                  key: 'batch-confirm',
                  label: '批量确认勾选费用',
                  onClick: () => {
                    if (selectedRowKeys.length === 0) {
                      message.warning('请先勾选需要确认的费用');
                      return;
                    }
                    message.info(
                      `已选 ${selectedRowKeys.length} 笔费用，可直接点击【创建账单】自动原子确认并生成批次`,
                    );
                  },
                },
              ],
            }}
          >
            <Button>
              批量操作 <DownOutlined />
            </Button>
          </Dropdown>,
          <Button
            key="export"
            icon={<DownloadOutlined />}
            onClick={handleExport}
          >
            导出清单
          </Button>,
          <Tooltip key="import-tip" title="支持通过 Excel 标准模板批量导入费用">
            <Button
              key="import"
              icon={<CloudUploadOutlined />}
              style={{
                backgroundColor: '#faad14',
                borderColor: '#faad14',
                color: '#fff',
              }}
              onClick={() => message.info('可通过 Excel 模板批量导入费用明细')}
            >
              导入费用
            </Button>
          </Tooltip>,
        ]}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys, rows) => {
            setSelectedRowKeys(keys);
            setSelectedRows(rows);
          },
        }}
        request={async (params) => {
          const response = await settlementServiceListFeeLedger({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            businessType: params.businessType,
            direction: params.direction,
            status: params.status,
            expenseDateFrom: params.expenseDateFrom,
            expenseDateTo: params.expenseDateTo,
          });
          const list = response.data || [];
          setCurrentData(list);
          setTotalCount(Number(response.total || 0));
          setSummary(response.summary);
          return {
            data: list,
            total: Number(response.total || 0),
            success: response.success ?? true,
          };
        }}
      />

      {/* 底部双层动态汇总看板 */}
      <FeeSummaryBoard
        selectedRows={selectedRows}
        allRows={currentData}
        totalCount={totalCount}
        globalSummary={summary}
      />

      {/* 批量转账单工作台 */}
      <BillCreationWorkbench
        open={billWorkbenchOpen}
        initialFeeIds={selectedRowKeys.map(String)}
        sourceLabel={`从费用明细勾选的 ${selectedRowKeys.length} 笔费用`}
        onClose={() => setBillWorkbenchOpen(false)}
        onCreated={() => {
          setBillWorkbenchOpen(false);
          setSelectedRowKeys([]);
          setSelectedRows([]);
          actionRef.current?.reload();
        }}
      />
    </div>
  );
}
