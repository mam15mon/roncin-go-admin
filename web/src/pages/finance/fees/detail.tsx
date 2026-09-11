import { PageContainer } from '@ant-design/pro-components';
import { history, useParams } from '@umijs/max';
import { App, Descriptions, Table, Tag } from 'antd';
import React, { useEffect, useState } from 'react';
import { PageHeaderShell, SectionCard } from '@/components/ui';
import { settlementServiceGetFeeLedgerOrderDetail } from '@/services/roncin/settlementService';

export default function FinanceFeeDetailPage() {
  const { orderId = '' } = useParams<{ orderId: string }>();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<API.FeeLedgerOrderDetail>();

  useEffect(() => {
    if (!orderId) return;
    let cancelled = false;
    setLoading(true);
    void settlementServiceGetFeeLedgerOrderDetail({ orderId })
      .then((response) => {
        if (!cancelled) setDetail(response.data);
      })
      .catch((error: Error) => {
        if (!cancelled) message.error(error.message || '加载费用台账详情失败');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [message, orderId]);

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
            {item.baseCurrency || '-'}：应收 {item.receivableBaseAmount || '0'}{' '}
            / 应付 {item.payableBaseAmount || '0'} / 毛利{' '}
            {item.profitBaseAmount || '0'}
          </Tag>
        ))}
      </SectionCard>
      <SectionCard title="费用明细" style={{ marginTop: 16 }}>
        <Table<API.FeeLedgerItem>
          rowKey="id"
          loading={loading}
          pagination={false}
          dataSource={detail?.fees ?? []}
          columns={[
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
                `${row.totalAmount || '0'} ${row.currency || ''}`,
            },
            {
              title: '折本币',
              key: 'base',
              render: (_, row) =>
                `${row.baseCurrencyAmount || '0'} ${row.baseCurrency || ''}`,
            },
            { title: '发生日期', dataIndex: 'expenseDate' },
          ]}
        />
      </SectionCard>
    </PageContainer>
  );
}
