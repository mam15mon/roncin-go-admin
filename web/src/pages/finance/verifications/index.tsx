import { PlusOutlined, RollbackOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Descriptions, Drawer, Select, Space, Table, Tag } from 'antd';
import { useEffect, useRef, useState } from 'react';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
} from '@/components/ui';
import {
  FinanceOrganizationPurpose,
  FinanceVerificationStatus,
} from '@/enums.generated';
import {
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceListVerifications,
  settlementServiceReverseVerification,
} from '@/services/roncin/settlementService';
import { toTableRequest, unwrapPage } from '@/utils/api';
import { formatDate } from '@/utils/format';
import { makeVersionActions } from '@/utils/versionActions';
import VerificationWorkbench from './VerificationWorkbench';

/** 台账内置搜索表单提交的筛选字段（keyword 等分页字段由模板统一注入，不在此声明） */
type VerificationLedgerFilterParams = {
  status?: string;
};

export default function FinanceVerificationsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [open, setOpen] = useState(false);
  const [detail, setDetail] = useState<API.FinanceVerification>();
  const [organizationId, setOrganizationId] = useState<string>();
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  useEffect(() => {
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_READ,
    }).then((response) => setOrganizationOptions(response.data ?? []));
  }, []);
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    amountsByBaseCurrency: [] as API.FinanceBaseCurrencyAmount[],
  });
  const formatBaseCurrencyAmounts = (
    field: 'receivableBaseAmount' | 'payableBaseAmount',
  ) =>
    metricStats.amountsByBaseCurrency
      .map((item) => `${item[field] ?? '0'} ${item.baseCurrency ?? '-'}`)
      .join(' / ') || '-';

  const reload = () => actionRef.current?.reload();
  const verificationActions = makeVersionActions<API.FinanceVerification>({
    modal,
    message,
  });
  const reverse = (r: API.FinanceVerification) => {
    verificationActions.confirm(
      r,
      `反核销 ${r.organizationName || '所属公司未标识'} 的该批分配？`,
      async ({ id, expectedVersion }, reason) => {
        await settlementServiceReverseVerification(
          { id },
          { id, expectedVersion, reason },
        );
        message.success(
          '反核销成功；相关未支付提成已自动取消，资金与账单余额已释放',
        );
        reload();
      },
      {
        placeholder: '请输入反核销原因（必填）',
        requiredMessage: '请输入原因',
      },
    );
  };

  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'total-verifications',
      title: '有效核销总笔数',
      value: metricStats.totalCount,
      suffix: '笔',
    },
    {
      key: 'rec-verifications',
      title: '应收核销总金额',
      value: formatBaseCurrencyAmounts('receivableBaseAmount'),
      valueColor: '#1677ff',
    },
    {
      key: 'pay-verifications',
      title: '应付核销总金额',
      value: formatBaseCurrencyAmounts('payableBaseAmount'),
      valueColor: '#fa8c16',
    },
  ];

  const columns: ProColumns<API.FinanceVerification>[] = [
    {
      title: '所属公司',
      dataIndex: 'organizationName',
      width: 150,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 55,
      fixed: 'left',
    },
    {
      title: '核销编号',
      dataIndex: 'verificationNo',
      width: 170,
      copyable: true,
      search: false,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      valueEnum: {
        [FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_ACTIVE]: {
          text: '有效',
        },
        [FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_REVERSED]: {
          text: '已反核销',
        },
      },
      render: (_, r) => (
        <Tag
          color={
            r.status ===
            FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_ACTIVE
              ? 'green'
              : 'default'
          }
          style={{ margin: 0 }}
        >
          {r.status ===
          FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_ACTIVE
            ? '有效'
            : '已反核销'}
        </Tag>
      ),
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 90,
      search: false,
      renderText: (v) => (v === 'RECEIVABLE' ? '应收核销' : '应付核销'),
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 220,
      ellipsis: true,
      search: false,
    },
    {
      title: '核销金额',
      dataIndex: 'amount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, r) => (
        <strong style={{ color: '#262626' }}>
          {r.amount} {r.currency}
        </strong>
      ),
    },
    {
      title: '流水本位币合计',
      dataIndex: 'baseAmount',
      width: 150,
      align: 'right',
      search: false,
      render: (_, r) =>
        r.baseAmount ? (
          <strong
            style={{
              color: r.direction === 'RECEIVABLE' ? '#1677ff' : '#fa8c16',
            }}
          >
            {r.baseAmount} {r.baseCurrency}
          </strong>
        ) : (
          '-'
        ),
    },
    {
      title: '汇兑损益',
      dataIndex: 'exchangeGainLoss',
      width: 145,
      align: 'right',
      search: false,
      render: (_, r) => {
        const val = Number(r.exchangeGainLoss || 0);
        if (val > 0) {
          return (
            <Tag color="green">
              +{r.exchangeGainLoss} {r.baseCurrency || 'CNY'}
            </Tag>
          );
        }
        if (val < 0) {
          return (
            <Tag color="red">
              {r.exchangeGainLoss} {r.baseCurrency || 'CNY'}
            </Tag>
          );
        }
        return (
          <span style={{ color: '#8c8c8c' }}>
            0.00 {r.baseCurrency || 'CNY'}
          </span>
        );
      },
    },
    {
      title: '分配数',
      search: false,
      render: (_, r) => r.allocations?.length || 0,
      width: 75,
      align: 'center',
    },
    {
      title: '核销日期',
      dataIndex: 'verificationDate',
      width: 110,
      search: false,
    },
    {
      title: '关联明细',
      search: false,
      width: 260,
      ellipsis: true,
      render: (_, r) =>
        (r.allocations || [])
          .map((x) => `${x.cashflowNo} → ${x.billNo}: ${x.amount}`)
          .join('；'),
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 140,
      render: (_, r) => [
        <a key="view" onClick={() => setDetail(r)}>
          详情
        </a>,
        access.canReverseFinanceVerifications &&
        access.canOperateOrganization(r.organizationId) &&
        r.status ===
          FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_ACTIVE ? (
          <a
            key="reverse"
            onClick={() => reverse(r)}
            style={{ color: '#ff4d4f' }}
          >
            <RollbackOutlined /> 反核销
          </a>
        ) : null,
      ],
    },
  ];

  return (
    <>
      <FinanceLedgerTemplate<
        API.FinanceVerification,
        VerificationLedgerFilterParams
      >
        pageTitle="核销管理"
        pageSubTitle="应收/应付账单对账核销与多币种汇差结算"
        topBar={
          <Space size={8} align="center">
            <span style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.65)' }}>
              所属公司：
            </span>
            <Select
              allowClear
              placeholder="请选择所属公司"
              style={{ minWidth: 220 }}
              value={organizationId}
              options={organizationOptions.map((item) => ({
                value: item.id,
                label: item.name ?? item.code ?? item.id,
              }))}
              onChange={(value) => {
                setOrganizationId(value);
                actionRef.current?.reload();
              }}
            />
          </Space>
        }
        headerTitle="核销记录列表"
        actionRef={actionRef}
        columns={columns}
        metricCards={metricCards}
        scrollX={1770}
        primaryActionText="新建核销"
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={
          access.canCreateFinanceVerifications ? () => setOpen(true) : undefined
        }
        request={async (p) => {
          const r = await settlementServiceListVerifications({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            status: p.status ? Number(p.status) : undefined,
            organizationId,
          });
          const page = unwrapPage(r);
          setMetricStats({
            totalCount: page.total,
            amountsByBaseCurrency: r.summary?.amountsByBaseCurrency ?? [],
          });
          return { ...toTableRequest(r), total: page.total };
        }}
      />
      <VerificationWorkbench
        open={open}
        onClose={() => setOpen(false)}
        onCreated={reload}
      />

      <Drawer
        title={`核销记录详情 ${detail?.verificationNo || ''}`}
        open={Boolean(detail)}
        size={920}
        onClose={() => setDetail(undefined)}
      >
        {detail && (
          <>
            <Descriptions
              bordered
              size="small"
              column={2}
              style={{ marginBottom: 16 }}
            >
              <Descriptions.Item label="所属公司">
                {detail.organizationName || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="核销状态">
                <Tag
                  color={
                    detail.status ===
                    FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_ACTIVE
                      ? 'green'
                      : 'default'
                  }
                >
                  {detail.status ===
                  FinanceVerificationStatus.FINANCE_VERIFICATION_STATUS_ACTIVE
                    ? '有效'
                    : '已反核销'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="核销方向">
                {detail.direction === 'RECEIVABLE' ? '应收核销' : '应付核销'}
              </Descriptions.Item>
              <Descriptions.Item label="往来结算单位" span={2}>
                {detail.settlementPartyName}
              </Descriptions.Item>
              <Descriptions.Item label="核销原币总额">
                <strong style={{ color: '#262626' }}>
                  {detail.amount} {detail.currency}
                </strong>
              </Descriptions.Item>
              <Descriptions.Item label="流水本位币合计">
                <strong style={{ color: '#1677ff' }}>
                  {detail.baseAmount} {detail.baseCurrency}
                </strong>
              </Descriptions.Item>
              <Descriptions.Item label="账单账面本币">
                {detail.billBaseAmount
                  ? `${detail.billBaseAmount} ${detail.baseCurrency}`
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="资金折算本币">
                {detail.cashflowBaseAmount
                  ? `${detail.cashflowBaseAmount} ${detail.baseCurrency}`
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="已实现汇兑损益">
                {(() => {
                  const val = Number(detail.exchangeGainLoss || 0);
                  if (val > 0) {
                    return (
                      <Tag color="green">
                        +{detail.exchangeGainLoss} {detail.baseCurrency} (收益)
                      </Tag>
                    );
                  }
                  if (val < 0) {
                    return (
                      <Tag color="red">
                        {detail.exchangeGainLoss} {detail.baseCurrency} (损失)
                      </Tag>
                    );
                  }
                  return <span>0.00 {detail.baseCurrency}</span>;
                })()}
              </Descriptions.Item>
              <Descriptions.Item label="核销日期">
                {detail.verificationDate}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {formatDate(detail.createdAt)}
              </Descriptions.Item>
              <Descriptions.Item label="备注" span={2}>
                {detail.note || '-'}
              </Descriptions.Item>
              {detail.reversalReason && (
                <Descriptions.Item label="反核销原因" span={2}>
                  {detail.reversalReason}
                </Descriptions.Item>
              )}
            </Descriptions>

            <div style={{ fontWeight: 600, marginBottom: 8, fontSize: 13 }}>
              核销分摊明细
            </div>
            <Table<API.FinanceVerificationAllocation>
              rowKey="id"
              size="small"
              bordered
              pagination={false}
              dataSource={detail.allocations || []}
              columns={[
                { title: '资金流水号', dataIndex: 'cashflowNo', width: 160 },
                { title: '对账单号', dataIndex: 'billNo', width: 160 },
                {
                  title: '原币分摊金额',
                  dataIndex: 'amount',
                  align: 'right',
                  render: (val) => (
                    <strong>
                      {val} {detail.currency}
                    </strong>
                  ),
                },
                {
                  title: '账单账面本币',
                  dataIndex: 'billBaseAmount',
                  align: 'right',
                  render: (val) =>
                    val ? `${val} ${detail.baseCurrency}` : '-',
                },
                {
                  title: '资金实收付本币',
                  dataIndex: 'cashflowBaseAmount',
                  align: 'right',
                  render: (val) =>
                    val ? `${val} ${detail.baseCurrency}` : '-',
                },
                {
                  title: '分摊汇兑损益',
                  dataIndex: 'exchangeGainLoss',
                  align: 'right',
                  render: (val) => {
                    const num = Number(val || 0);
                    if (num > 0) return <Tag color="green">+{val} (收益)</Tag>;
                    if (num < 0) return <Tag color="red">{val} (损失)</Tag>;
                    return <span style={{ color: '#8c8c8c' }}>0.00</span>;
                  },
                },
                {
                  title: '状态',
                  render: (_, r) =>
                    r.active ? <Tag color="blue">有效</Tag> : <Tag>已冲销</Tag>,
                  width: 80,
                  align: 'center',
                },
              ]}
            />
          </>
        )}
      </Drawer>
    </>
  );
}
