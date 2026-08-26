import { PageContainer } from '@ant-design/pro-components';
import { useLocation } from '@umijs/max';
import { Result } from 'antd';
import React from 'react';

const descriptions: Record<string, [string, string]> = {
  '/finance/bills': ['账单管理', '按结算单位、收付方向和币种聚合已确认费用。'],
  '/finance/invoices': ['开票记录', '记录发票开具、作废、红冲及账单金额分配。'],
  '/finance/cashflows': ['收付管理', '登记银行流水、收付款单及资金认领。'],
  '/finance/verifications': [
    '核销管理',
    '将收付款金额多对多分配到应收应付账单。',
  ],
  '/finance/commissions': ['提成管理', '按单票毛利和人员规则计算业务提成。'],
};

export default function SettlementPlaceholderPage() {
  const location = useLocation();
  const [title, description] = descriptions[location.pathname] || [
    '费用管理',
    '财务结算模块',
  ];
  return (
    <PageContainer title={title}>
      <Result status="info" title={`${title}正在接入`} subTitle={description} />
    </PageContainer>
  );
}
