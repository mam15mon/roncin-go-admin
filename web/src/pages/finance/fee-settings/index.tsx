import {
  AccountBookOutlined,
  CalculatorOutlined,
  FileTextOutlined,
  SlidersOutlined,
} from '@ant-design/icons';
import React from 'react';
import { useAccess } from '@/app/access';
import {
  type MultiTabCenterTabItem,
  MultiTabCenterTemplate,
} from '@/components/ui';
import BillingUnitsPanel from './components/BillingUnitsPanel';
import CustomSettingsPanel from './components/CustomSettingsPanel';
import FeeItemsPanel from './components/FeeItemsPanel';
import FeeTemplatesPanel from './components/FeeTemplatesPanel';
import TaxableServicesPanel from './components/TaxableServicesPanel';

/**
 * /finance/fee-settings 费用设置中心
 * 聚合基础费用科目字典、计量计费单位基准、发票货物与应税劳务税率、自定义控制规则
 */
export default function FeeSettingsPage() {
  const access = useAccess();

  const tabItems: MultiTabCenterTabItem[] = [
    {
      key: 'fee-settings',
      label: access.isSystemWorkspace ? '初始费用目录' : '费用科目',
      icon: <AccountBookOutlined />,
      visible: access.canReadFeeSettings,
      tooltip:
        '维护基础费用科目字典（如海运费、港杂费、报关费、拖车费等）、默认收付币种与税率规则',
      children: access.isSystemWorkspace ? (
        <FeeTemplatesPanel />
      ) : (
        <FeeItemsPanel />
      ),
    },
    {
      key: 'billing-units',
      label: '计费单位',
      icon: <CalculatorOutlined />,
      visible: access.canReadFeeSettings,
      tooltip:
        '定义计费计量基准（如按票、CBM、车、箱量等），并区分常规计量单位与集装箱箱型单位',
      children: <BillingUnitsPanel />,
    },
    {
      key: 'taxable-services',
      label: '货物或应税劳务',
      icon: <FileTextOutlined />,
      visible: access.canOperateBusiness && access.canReadFeeSettings,
      tooltip: '维护商品编码、发票货物或应税劳务名称与默认开票税率',
      children: <TaxableServicesPanel />,
    },
    {
      key: 'custom-settings',
      label: '自定义规则',
      icon: <SlidersOutlined />,
      visible: access.canOperateBusiness && access.canReadFinanceBills,
      tooltip: '配置账单创建后允许修改的费用字段等组织级自定义财务规则',
      children: <CustomSettingsPanel />,
    },
  ];

  return (
    <MultiTabCenterTemplate
      title="费用设置"
      subTitle="集中维护基础费用科目字典、计量计费单位基准、发票劳务税率与账单控制规则"
      items={tabItems}
      defaultActiveKey="fee-settings"
      syncUrlQuery
    />
  );
}
