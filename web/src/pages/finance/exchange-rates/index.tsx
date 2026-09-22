import { PageContainer } from '@ant-design/pro-components';
import React from 'react';

import { ExchangeRatesPanel } from './components/ExchangeRatesPanel';

/**
 * /finance/exchange-rates 财务 · 汇率页：各核算组织维护本组织周汇率行
 * （应收/应付双轨点差），系统管理额外维护 NULL 基线兜底行；一键同步入口对本币
 * CNY 组织显示「从中国银行同步周汇率」，非 CNY 本币组织显示「一键同步周汇率」。
 */
export default function ExchangeRatesPage() {
  return (
    <PageContainer
      header={{
        title: '汇率管理',
        subTitle: '周汇率 · 应收/应付双轨点差 · 中国银行一键同步',
      }}
    >
      <ExchangeRatesPanel />
    </PageContainer>
  );
}
