import {
  AccountBookOutlined,
  DollarOutlined,
  NodeIndexOutlined,
  NumberOutlined,
} from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import React from 'react';
import { ParameterSettingTemplate } from '@/components/ui';
import ExchangeRatesPanel from './components/ExchangeRatesPanel';
import FeeSettingsPanel from './components/FeeSettingsPanel';
import MilestoneTemplatesPanel from './components/MilestoneTemplatesPanel';
import NumberRulesPanel from './components/NumberRulesPanel';

export default function SettingsPage() {
  const access = useAccess();

  const tabItems = [
    {
      key: 'number-rules',
      label: '单据编号规则',
      icon: <NumberOutlined />,
      visible: access.canReadMasterDataNumberRules,
      tooltip: '12 类业务单据独立编号规则与实时预览',
      children: <NumberRulesPanel />,
    },
    {
      key: 'fee-settings',
      label: '费用与科目设置',
      icon: <AccountBookOutlined />,
      visible: access.canReadFeeSettings,
      tooltip: '费用代码、计费单位、应税劳务及异常情况科目维护',
      children: <FeeSettingsPanel />,
    },
    {
      key: 'exchange-rates',
      label: '汇率与时间标准',
      icon: <DollarOutlined />,
      visible: access.canReadExchangeRates,
      tooltip: '折本币/开票/结算汇率及时间标准取值优先级',
      children: <ExchangeRatesPanel />,
    },
    {
      key: 'milestones',
      label: '业务履约里程碑',
      icon: <NodeIndexOutlined />,
      visible: access.canReadMasterDataMilestoneTemplates,
      tooltip: '各业务类型履约进度节点与时序流程模板',
      children: <MilestoneTemplatesPanel />,
    },
  ];

  return (
    <ParameterSettingTemplate
      title="业务参数与规则设置"
      subTitle="集中统一维护业务单据流水编号、费用科目字典、财务汇率基准与订单履约里程碑规则"
      items={tabItems}
      defaultActiveKey="number-rules"
      syncUrlQuery
    />
  );
}
