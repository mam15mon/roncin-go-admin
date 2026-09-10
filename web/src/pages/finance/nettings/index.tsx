import { CheckOutlined, RollbackOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Descriptions, Drawer, Select, Space, Table, Tag } from 'antd';
import { useEffect, useRef, useState } from 'react';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
} from '@/components/ui';
import {
  FinanceNettingStatus,
  FinanceOrganizationPurpose,
} from '@/enums.generated';
import {
  settlementServiceCancelNetting,
  settlementServiceConfirmNetting,
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceListNettings,
  settlementServiceReverseNetting,
} from '@/services/roncin/settlementService';
import { toTableRequest, unwrapPage } from '@/utils/api';
import { makeVersionActions } from '@/utils/versionActions';

function nettingStatusTag(status?: number) {
  switch (status) {
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT:
      return <Tag color="gold">草稿</Tag>;
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_CONFIRMED:
      return <Tag color="green">已确认</Tag>;
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_CANCELLED:
      return <Tag color="default">已取消</Tag>;
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_REVERSED:
      return <Tag color="default">已反转</Tag>;
    default:
      return <Tag color="default">未知</Tag>;
  }
}

function nettingStatusText(status?: number) {
  switch (status) {
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT:
      return '草稿';
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_CONFIRMED:
      return '已确认';
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_CANCELLED:
      return '已取消';
    case FinanceNettingStatus.FINANCE_NETTING_STATUS_REVERSED:
      return '已反转';
    default:
      return '未知';
  }
}

export default function FinanceNettingsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [detail, setDetail] = useState<API.FinanceNetting>();
  const [organizationId, setOrganizationId] = useState<string>();
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  const [metricStats, setMetricStats] = useState({
    confirmedCount: 0,
    amountsByBaseCurrency: [] as API.FinanceNettingBaseCurrencyAmount[],
  });

  useEffect(() => {
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_NETTING_READ,
    })
      .then((response) => setOrganizationOptions(response.data ?? []))
      .catch(() => setOrganizationOptions([]));
  }, []);

  const reload = () => actionRef.current?.reload();
  const nettingActions = makeVersionActions<API.FinanceNetting>({
    modal,
    message,
  });

  const confirmNetting = (record: API.FinanceNetting) => {
    nettingActions.run(record, async ({ id, expectedVersion }) => {
      await settlementServiceConfirmNetting({ id }, { id, expectedVersion });
      message.success('对冲单已确认；双方账单未结余额已按抵销额扣减');
      reload();
    });
  };

  const cancelNetting = (record: API.FinanceNetting) => {
    nettingActions.confirm(
      record,
      `取消 ${record.organizationName || '所属公司未标识'} 的草稿对冲单 ${record.nettingNo || ''}？`,
      async ({ id, expectedVersion }, reason) => {
        await settlementServiceCancelNetting(
          { id },
          { id, expectedVersion, reason },
        );
        message.success('对冲单已取消；草稿未占用账单余额');
        reload();
      },
      {
        placeholder: '请输入取消原因（必填）',
        requiredMessage: '请输入原因',
      },
    );
  };

  const reverseNetting = (record: API.FinanceNetting) => {
    nettingActions.confirm(
      record,
      `反转 ${record.organizationName || '所属公司未标识'} 的已确认对冲单 ${record.nettingNo || ''}？双方账单未结余额将恢复，已开发票与账单折算快照不受影响。`,
      async ({ id, expectedVersion }, reason) => {
        await settlementServiceReverseNetting(
          { id },
          { id, expectedVersion, reason },
        );
        message.success('对冲单已反转；双方账单未结余额已精确恢复');
        reload();
      },
      {
        placeholder: '请输入反转原因（必填）',
        requiredMessage: '请输入原因',
      },
    );
  };

  const formatNettingBaseAmounts = () =>
    metricStats.amountsByBaseCurrency
      .map(
        (item) =>
          `${item.nettingBaseAmount ?? '0'} ${item.baseCurrency ?? '-'}`,
      )
      .join(' / ') || '-';

  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'confirmed-count',
      title: '已确认对冲笔数',
      value: metricStats.confirmedCount,
      suffix: '笔',
    },
    {
      key: 'confirmed-base-amount',
      title: '已确认抵销本币金额',
      value: formatNettingBaseAmounts(),
      valueColor: '#52c41a',
    },
  ];

  const columns: ProColumns<API.FinanceNetting>[] = [
    {
      title: '所属公司',
      dataIndex: 'organizationName',
      width: 150,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: '输入对冲单号或结算单位',
      },
    },
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 55,
      fixed: 'left',
    },
    {
      title: '对冲单号',
      dataIndex: 'nettingNo',
      width: 180,
      copyable: true,
      search: false,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      valueEnum: {
        [FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT]: {
          text: '草稿',
        },
        [FinanceNettingStatus.FINANCE_NETTING_STATUS_CONFIRMED]: {
          text: '已确认',
        },
        [FinanceNettingStatus.FINANCE_NETTING_STATUS_CANCELLED]: {
          text: '已取消',
        },
        [FinanceNettingStatus.FINANCE_NETTING_STATUS_REVERSED]: {
          text: '已反转',
        },
      },
      render: (_, record) => nettingStatusTag(record.status),
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 220,
      ellipsis: true,
      search: false,
    },
    {
      title: '币种',
      dataIndex: 'currency',
      width: 70,
      search: false,
    },
    {
      title: '抵销金额',
      dataIndex: 'amount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, record) => (
        <strong style={{ color: '#262626' }}>
          {record.amount} {record.currency}
        </strong>
      ),
    },
    {
      title: '本币抵销额',
      dataIndex: 'baseCurrencyAmount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, record) => (
        <strong style={{ color: '#52c41a' }}>
          {record.baseCurrencyAmount} {record.baseCurrency}
        </strong>
      ),
    },
    {
      title: '分摊数',
      search: false,
      width: 75,
      align: 'center',
      render: (_, record) => record.allocations?.length || 0,
    },
    {
      title: '关联批次',
      dataIndex: 'batchNo',
      width: 160,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 170,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 170,
      render: (_, record) => [
        <a key="view" onClick={() => setDetail(record)}>
          详情
        </a>,
        access.canConfirmFinanceNettings &&
        record.status === FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT ? (
          <a key="confirm" onClick={() => confirmNetting(record)}>
            <CheckOutlined /> 确认
          </a>
        ) : null,
        access.canReverseFinanceNettings &&
        record.status === FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT ? (
          <a key="cancel" onClick={() => cancelNetting(record)}>
            取消
          </a>
        ) : null,
        access.canReverseFinanceNettings &&
        record.status ===
          FinanceNettingStatus.FINANCE_NETTING_STATUS_CONFIRMED ? (
          <a
            key="reverse"
            onClick={() => reverseNetting(record)}
            style={{ color: '#ff4d4f' }}
          >
            <RollbackOutlined /> 反转
          </a>
        ) : null,
      ],
    },
  ];

  return (
    <>
      <div style={{ marginBottom: 12 }}>
        <Select
          allowClear
          placeholder="所属公司"
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
      </div>
      <FinanceLedgerTemplate<API.FinanceNetting>
        headerTitle="对冲结算单管理"
        actionRef={actionRef}
        columns={columns}
        metricCards={metricCards}
        scrollX={1700}
        request={async (params) => {
          const response = await settlementServiceListNettings({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            currency: params.currency,
            status: params.status ? Number(params.status) : undefined,
            organizationId,
          });
          const page = unwrapPage(response);
          setMetricStats({
            confirmedCount: response.summary?.confirmedCount
              ? Number(response.summary.confirmedCount)
              : 0,
            amountsByBaseCurrency:
              response.summary?.amountsByBaseCurrency ?? [],
          });
          return { ...toTableRequest(response), total: page.total };
        }}
      />

      <Drawer
        title={`对冲结算单详情 ${detail?.nettingNo || ''}`}
        open={Boolean(detail)}
        size={920}
        onClose={() => setDetail(undefined)}
      >
        {detail && (
          <>
            <Descriptions bordered size="small" column={2}>
              <Descriptions.Item label="所属公司">
                {detail.organizationName || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                {nettingStatusTag(detail.status)}
              </Descriptions.Item>
              <Descriptions.Item label="结算单位" span={2}>
                {detail.settlementPartyName}
              </Descriptions.Item>
              <Descriptions.Item label="账单币种">
                {detail.currency}
              </Descriptions.Item>
              <Descriptions.Item label="抵销金额">
                <strong style={{ color: '#262626' }}>
                  {detail.amount} {detail.currency}
                </strong>
              </Descriptions.Item>
              <Descriptions.Item label="本位币">
                {detail.baseCurrency}
              </Descriptions.Item>
              <Descriptions.Item label="本币抵销额">
                <strong style={{ color: '#52c41a' }}>
                  {detail.baseCurrencyAmount} {detail.baseCurrency}
                </strong>
              </Descriptions.Item>
              <Descriptions.Item label="关联批次">
                {detail.batchNo || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {detail.createdAt || '-'}
              </Descriptions.Item>
              {detail.confirmedAt ? (
                <Descriptions.Item label="确认时间">
                  {detail.confirmedAt}
                </Descriptions.Item>
              ) : null}
              {detail.reversedAt ? (
                <Descriptions.Item label="反转时间">
                  {detail.reversedAt}
                </Descriptions.Item>
              ) : null}
              {detail.cancelledAt ? (
                <Descriptions.Item label="取消时间">
                  {detail.cancelledAt}
                </Descriptions.Item>
              ) : null}
              <Descriptions.Item label="备注" span={2}>
                {detail.note || '-'}
              </Descriptions.Item>
              {detail.cancellationReason ? (
                <Descriptions.Item label="取消原因" span={2}>
                  {detail.cancellationReason}
                </Descriptions.Item>
              ) : null}
              {detail.reversalReason ? (
                <Descriptions.Item label="反转原因" span={2}>
                  {detail.reversalReason}
                </Descriptions.Item>
              ) : null}
            </Descriptions>

            <div
              style={{
                fontWeight: 600,
                marginBottom: 8,
                marginTop: 16,
                fontSize: 13,
              }}
            >
              对冲分摊明细（{nettingStatusText(detail.status)}
              {detail.status ===
              FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT
                ? '，确认后生效'
                : ''}
              ）
            </div>
            <Table<API.FinanceNettingAllocation>
              rowKey="id"
              size="small"
              bordered
              pagination={false}
              dataSource={detail.allocations || []}
              columns={[
                { title: '账单编号', dataIndex: 'billNo', width: 180 },
                {
                  title: '方向',
                  dataIndex: 'direction',
                  width: 80,
                  render: (value) =>
                    value === 'RECEIVABLE' ? (
                      <Tag color="blue" style={{ margin: 0 }}>
                        应收
                      </Tag>
                    ) : (
                      <Tag color="orange" style={{ margin: 0 }}>
                        应付
                      </Tag>
                    ),
                },
                {
                  title: '抵销金额',
                  dataIndex: 'amount',
                  align: 'right',
                  render: (value) => (
                    <strong>
                      {value} {detail.currency}
                    </strong>
                  ),
                },
                {
                  title: '本币抵销额',
                  dataIndex: 'baseCurrencyAmount',
                  align: 'right',
                  render: (value) => `${value || '0'} ${detail.baseCurrency}`,
                },
                {
                  title: '有效',
                  dataIndex: 'active',
                  width: 80,
                  align: 'center',
                  render: (value) =>
                    value ? (
                      <Tag color="green" style={{ margin: 0 }}>
                        有效
                      </Tag>
                    ) : (
                      <Tag style={{ margin: 0 }}>失效</Tag>
                    ),
                },
              ]}
            />
            <Space style={{ marginTop: 8 }}>
              <span style={{ color: '#8c8c8c', fontSize: 12 }}>
                只有有效分摊参与账单可用余额；反转后分摊失效并保留审计。
              </span>
            </Space>
          </>
        )}
      </Drawer>
    </>
  );
}
