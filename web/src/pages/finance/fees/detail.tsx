import { PageContainer } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Descriptions, type TableColumnsType, Tag } from 'antd';
import React from 'react';
import { useParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import {
  PageHeaderShell,
  SectionCard,
  ResponsiveTable as Table,
} from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { FeeCorrectionSection } from '@/features/orders/fee-correction';
import { history } from '@/router/history';
import { settlementServiceGetFeeLedgerOrderDetail } from '@/services/roncin/settlementService';
import { formatAmount, formatDate } from '@/utils/format';

export default function FinanceFeeDetailPage() {
  const { orderId = '' } = useParams<{ orderId: string }>();
  const { initialState } = useInitialState();
  const query = useQuery({
    queryKey: [
      'finance',
      'fee-detail',
      initialState?.currentUser?.currentOrganization?.id,
      orderId,
    ],
    enabled: Boolean(orderId),
    queryFn: async () => {
      const response = await settlementServiceGetFeeLedgerOrderDetail({
        orderId,
      });
      if (!response?.data) throw new Error('费用台账详情加载失败');
      return response.data;
    },
  });
  const detail = query.data;
  const loading = query.isFetching;

  const feeColumns: TableColumnsType<API.FeeLedgerItem> = [
    { title: '费用名称', dataIndex: 'feeName' },
    {
      title: '方向',
      dataIndex: 'direction',
      render: (value) => (value === 'RECEIVABLE' ? '应收' : '应付'),
    },
    { title: '结算单位', dataIndex: 'settlementPartyName' },
    {
      title: '原币金额',
      key: 'amount',
      render: (_, row) =>
        `${formatAmount(row.totalAmount)} ${row.currency || ''}`,
    },
    {
      title: '折本币',
      key: 'base',
      render: (_, row) =>
        `${formatAmount(row.baseCurrencyAmount)} ${row.baseCurrency || ''}`,
    },
    {
      title: '录入日期',
      dataIndex: 'createdAt',
      render: (value?: string) => formatDate(value, 'minute'),
    },
  ];

  const feeColumnSettings = useColumnSettings<
    TableColumnsType<API.FeeLedgerItem>[number]
  >({
    tableKey: 'finance:fee-detail-lines',
    columns: feeColumns,
  });

  return (
    <PageContainer
      title={false}
      breadcrumbRender={false}
      header={{ title: false, style: { padding: 0 } }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      <PageHeaderShell
        title="费用台账详情"
        subTitle="仅展示当前财务费用读取权限范围内的订单费用"
        onBack={() => history.push('/finance/fees')}
      />
      <SectionCard title="订单与所属公司">
        <Descriptions
          size="small"
          column={4}
          items={[
            {
              key: 'organization',
              label: '所属公司',
              children: detail?.organizationName || '-',
            },
            {
              key: 'orderNo',
              label: '订单编号',
              children: detail?.orderNo || '-',
            },
            {
              key: 'business',
              label: '业务类型',
              children: detail?.businessType || '-',
            },
            {
              key: 'customer',
              label: '委托单位',
              children: detail?.customerName || '-',
            },
          ]}
        />
      </SectionCard>
      <SectionCard title="按本位币汇总" style={{ marginTop: 16 }}>
        {(detail?.amountsByBaseCurrency ?? []).map((item) => (
          <Tag key={item.baseCurrency} color="blue">
            {item.baseCurrency || '-'}：应收{' '}
            {formatAmount(item.receivableBaseAmount ?? '0')} / 应付{' '}
            {formatAmount(item.payableBaseAmount ?? '0')} / 毛利{' '}
            {formatAmount(item.profitBaseAmount ?? '0')}
          </Tag>
        ))}
      </SectionCard>
      <SectionCard title="费用明细" style={{ marginTop: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            marginBottom: 8,
          }}
        >
          {feeColumnSettings.entry}
        </div>
        <Table<API.FeeLedgerItem>
          rowKey="id"
          loading={loading}
          pagination={false}
          dataSource={detail?.fees ?? []}
          columns={feeColumnSettings.columns}
        />
      </SectionCard>
      <SectionCard title="费用更正记录与审核">
        <FeeCorrectionSection key={orderId} orderId={orderId} />
      </SectionCard>
    </PageContainer>
  );
}
