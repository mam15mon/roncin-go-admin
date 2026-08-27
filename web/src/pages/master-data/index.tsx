import {
  CompassOutlined,
  DollarOutlined,
  GlobalOutlined,
  RocketOutlined,
  SendOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Tabs } from 'antd';
import React, { useState } from 'react';
import AirlinesPanel from './components/AirlinesPanel';
import AirportsPanel from './components/AirportsPanel';
import CitiesPanel from './components/CitiesPanel';
import CountriesPanel from './components/CountriesPanel';
import CurrenciesPanel from './components/CurrenciesPanel';
import PortsPanel from './components/PortsPanel';
import ShippingLinesPanel from './components/ShippingLinesPanel';

export default function MasterDataPage() {
  const access = useAccess();
  const [activeTab, setActiveTab] = useState('ports');

  const tabItems = [
    {
      key: 'ports',
      visible: access.canReadMasterDataPorts,
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
      visible: access.canReadMasterDataAirports,
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
      visible: access.canReadMasterDataAirlines,
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
      visible: access.canReadMasterDataShippingLines,
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
      visible: access.canReadMasterDataItems,
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
      visible: access.canReadMasterDataAdministrativeRegions,
      label: (
        <span>
          <CompassOutlined style={{ marginRight: 6 }} />
          城市与区划 (Cities)
        </span>
      ),
      children: <CitiesPanel />,
    },
    {
      key: 'currencies',
      visible: access.canReadMasterDataCurrencies,
      label: (
        <span>
          <DollarOutlined style={{ marginRight: 6 }} />
          货币与币种 (Currencies)
        </span>
      ),
      children: <CurrenciesPanel />,
    },
  ]
    .filter((item) => item.visible)
    .map(({ visible: _visible, ...item }) => item);

  const visibleActiveTab = tabItems.some((item) => item.key === activeTab)
    ? activeTab
    : tabItems[0]?.key;

  return (
    <PageContainer
      header={{
        title: '货代主数据管理中心',
        subTitle:
          '统一维护全球港口五字码、机场三字码、航司二字码、船司 SCAC、国家城市及货币币种基础资料',
      }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      <div style={{ marginTop: 8 }}>
        <Tabs
          activeKey={visibleActiveTab}
          onChange={setActiveTab}
          type="card"
          items={tabItems}
          tabBarStyle={{
            position: 'sticky',
            top: 84,
            zIndex: 18,
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
