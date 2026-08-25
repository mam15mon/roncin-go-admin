import {
  AccountBookOutlined,
  AlertOutlined,
  CalculatorOutlined,
  DollarOutlined,
  FileTextOutlined,
  NodeIndexOutlined,
  NumberOutlined,
} from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import React from 'react';
import { ParameterSettingTemplate } from '@/components/ui';
import AbnormalCasesPanel from './components/AbnormalCasesPanel';
import BillingUnitsPanel from './components/BillingUnitsPanel';
import ExchangeRatesPanel from './components/ExchangeRatesPanel';
import FeeItemsPanel from './components/FeeItemsPanel';
import MilestoneTemplatesPanel from './components/MilestoneTemplatesPanel';
import NumberRulesPanel from './components/NumberRulesPanel';
import TaxableServicesPanel from './components/TaxableServicesPanel';

export default function SettingsPage() {
  const access = useAccess();

  const tabItems = [
    {
      key: 'fee-settings',
      label: '费用设置',
      icon: <AccountBookOutlined />,
      visible: access.canReadFeeSettings,
      tooltip: '维护基础费用科目字典（如海运费、港杂费、报关费、拖车费等）、默认收付币种与税率规则',
      children: <FeeItemsPanel />,
    },
    {
      key: 'exchange-rates',
      label: '汇率设置',
      icon: <DollarOutlined />,
      visible: access.canReadExchangeRates,
      tooltip: '配置多币种汇率管理规则，包括基准货币、折本币与结算汇率及时间标准取值优先级',
      children: <ExchangeRatesPanel />,
    },
    {
      key: 'billing-units',
      label: '计费单位设置',
      icon: <CalculatorOutlined />,
      visible: access.canReadFeeSettings,
      tooltip: '定义计费计量基准（如按票、CBM、车、箱量等），并区分常规计量单位与集装箱箱型单位',
      children: <BillingUnitsPanel />,
    },
    {
      key: 'abnormal-cases',
      label: '异常情况设置',
      icon: <AlertOutlined />,
      visible: access.canReadMasterDataItems,
      tooltip: '定义业务执行中的异常事件类型（如延航、查验、甩柜、货损、扣关等）及对应标识',
      children: <AbnormalCasesPanel />,
    },
    {
      key: 'number-rules',
      label: '编号规则设置',
      icon: <NumberOutlined />,
      visible: access.canReadMasterDataNumberRules,
      tooltip: '自定义各类业务单据的自动生成规则（如订单号、提单号、发票号、账单号等），支持前后缀、日期格式与流水号配置',
      children: <NumberRulesPanel />,
    },
    {
      key: 'taxable-services',
      label: '货物或应税劳务',
      icon: <FileTextOutlined />,
      visible: access.canReadFeeSettings,
      tooltip: '维护商品编码、发票货物或应税劳务名称与默认开票税率',
      children: <TaxableServicesPanel />,
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
      subTitle="统一维护费用科目字典、多币种汇率、计费单位基准、业务异常类型与单据自动编号规则"
      items={tabItems}
      defaultActiveKey="fee-settings"
      syncUrlQuery
    />
  );
}
