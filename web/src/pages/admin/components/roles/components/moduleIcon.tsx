import {
  AppstoreOutlined,
  DatabaseOutlined,
  DollarOutlined,
  IdcardOutlined,
  SettingOutlined,
  ShoppingCartOutlined,
} from '@ant-design/icons';

/** 按服务模块名称返回对应的彩色图标。 */
export function getModuleIcon(name: string) {
  switch (name) {
    case '订单管理':
      return <ShoppingCartOutlined style={{ color: '#1677ff' }} />;
    case '费用管理':
      return <DollarOutlined style={{ color: '#52c41a' }} />;
    case '业务资料':
      return <IdcardOutlined style={{ color: '#722ed1' }} />;
    case '主数据':
      return <DatabaseOutlined style={{ color: '#fa8c16' }} />;
    case '系统管理':
      return <SettingOutlined style={{ color: '#13c2c2' }} />;
    default:
      return <AppstoreOutlined style={{ color: '#64748b' }} />;
  }
}
