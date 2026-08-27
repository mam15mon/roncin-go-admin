import {
  CompassOutlined,
  DollarOutlined,
  GlobalOutlined,
  RocketOutlined,
  SendOutlined,
} from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import React from 'react';
import { MultiTabCenterTemplate, type MultiTabCenterTabItem } from '@/components/ui';
import AirlinesPanel from './components/AirlinesPanel';
import AirportsPanel from './components/AirportsPanel';
import CitiesPanel from './components/CitiesPanel';
import CountriesPanel from './components/CountriesPanel';
import CurrenciesPanel from './components/CurrenciesPanel';
import PortsPanel from './components/PortsPanel';
import ShippingLinesPanel from './components/ShippingLinesPanel';

export default function MasterDataPage() {
  const access = useAccess();

  const tabItems: MultiTabCenterTabItem[] = [
    {
      key: 'ports',
      label: '海运港口 (UN/LOCODE)',
      icon: <CompassOutlined />,
      visible: access.canReadMasterDataPorts,
      tooltip: '维护全球港口五字码 (UN/LOCODE)、所属国家地区及海陆铁多式联运枢纽属性',
      children: <PortsPanel />,
    },
    {
      key: 'airports',
      label: '空运机场 (IATA)',
      icon: <SendOutlined />,
      visible: access.canReadMasterDataAirports,
      tooltip: '维护国际航空运输协会 (IATA) 机场三字码、ICAO 四字码及城市空港基础资料',
      children: <AirportsPanel />,
    },
    {
      key: 'airlines',
      label: '航空公司 (Airlines)',
      icon: <RocketOutlined />,
      visible: access.canReadMasterDataAirlines,
      tooltip: '维护航司 IATA 二字码、ICAO 三字码、运单三位前缀及主营航线基础资料',
      children: <AirlinesPanel />,
    },
    {
      key: 'shipping-lines',
      label: '船公司 (Shipping Lines)',
      icon: <GlobalOutlined />,
      visible: access.canReadMasterDataShippingLines,
      tooltip: '维护船司标准载体代码 (SCAC)、英文缩写及订舱跟踪信息',
      children: <ShippingLinesPanel />,
    },
    {
      key: 'countries',
      label: '国家与地区 (Countries)',
      icon: <GlobalOutlined />,
      visible: access.canReadMasterDataItems,
      tooltip: '维护 ISO 3166-1 国家与地区二字码/三字码、中英文标准全称及大洲归属',
      children: <CountriesPanel />,
    },
    {
      key: 'cities',
      label: '城市与区划 (Cities)',
      icon: <CompassOutlined />,
      visible: access.canReadMasterDataAdministrativeRegions,
      tooltip: '维护行政区划代码、城市中英文名称、所属省州与时区基础数据',
      children: <CitiesPanel />,
    },
    {
      key: 'currencies',
      label: '货币与币种 (Currencies)',
      icon: <DollarOutlined />,
      visible: access.canReadMasterDataCurrencies,
      tooltip: '维护 ISO 4217 货币三字代码、货币符号、中文名称及小数精度位',
      children: <CurrenciesPanel />,
    },
  ];

  return (
    <MultiTabCenterTemplate
      title="货代主数据管理中心"
      subTitle="统一维护全球港口五字码、机场三字码、航司二字码、船司 SCAC、国家城市及货币币种基础资料"
      items={tabItems}
      defaultActiveKey="ports"
      syncUrlQuery
    />
  );
}
