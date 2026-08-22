import {
  CompassOutlined,
  GlobalOutlined,
  NodeIndexOutlined,
  NumberOutlined,
  RocketOutlined,
  SendOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { Tabs } from 'antd';
import React, { useState } from 'react';
import AirlinesPanel from './components/AirlinesPanel';
import AirportsPanel from './components/AirportsPanel';
import CitiesPanel from './components/CitiesPanel';
import CountriesPanel from './components/CountriesPanel';
import NumberRulesPanel from './components/NumberRulesPanel';
import PortsPanel from './components/PortsPanel';
import ShippingLinesPanel from './components/ShippingLinesPanel';
import MilestoneTemplatesPanel from './milestone-templates-panel';

export default function MasterDataPage() {
  const [activeTab, setActiveTab] = useState('ports');

  const tabItems = [
    {
      key: 'ports',
      label: (
        <span>
          <CompassOutlined style={{ marginRight: 6 }} />
          海运港口 (UN/LOCODE)
        </span>
      ),
      children: <PortsPanel />,
    },
    {
      key: 'airports',
      label: (
        <span>
          <SendOutlined style={{ marginRight: 6 }} />
          空运机场 (IATA)
        </span>
      ),
      children: <AirportsPanel />,
    },
    {
      key: 'airlines',
      label: (
        <span>
          <RocketOutlined style={{ marginRight: 6 }} />
          航空公司 (Airlines)
        </span>
      ),
      children: <AirlinesPanel />,
    },
    {
      key: 'shipping-lines',
      label: (
        <span>
          <GlobalOutlined style={{ marginRight: 6 }} />
          船公司 (Shipping Lines)
        </span>
      ),
      children: <ShippingLinesPanel />,
    },
    {
      key: 'countries',
      label: (
        <span>
          <GlobalOutlined style={{ marginRight: 6 }} />
          国家与地区 (Countries)
        </span>
      ),
      children: <CountriesPanel />,
    },
    {
      key: 'cities',
      label: (
        <span>
          <CompassOutlined style={{ marginRight: 6 }} />
          城市与区划 (Cities)
        </span>
      ),
      children: <CitiesPanel />,
    },
    {
      key: 'number-rules',
      label: (
        <span>
          <NumberOutlined style={{ marginRight: 6 }} />
          单号规则 (Number Rules)
        </span>
      ),
      children: <NumberRulesPanel />,
    },
    {
      key: 'milestones',
      label: (
        <span>
          <NodeIndexOutlined style={{ marginRight: 6 }} />
          业务里程碑 (Milestones)
        </span>
      ),
      children: <MilestoneTemplatesPanel />,
    },
  ];

  return (
    <PageContainer
      header={{
        title: '货代主数据管理中心',
        subTitle:
          '统一维护全球港口五字码、机场三字码、航司二字码、船司 SCAC、国家城市及单号流水规则',
      }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      <div style={{ marginTop: 8 }}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          type="card"
          items={tabItems}
          tabBarStyle={{
            marginBottom: 16,
            backgroundColor: '#ffffff',
            padding: '8px 12px 0',
            borderRadius: '8px 8px 0 0',
          }}
        />
      </div>
    </PageContainer>
  );
}
