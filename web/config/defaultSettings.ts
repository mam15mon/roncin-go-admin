import type { ProLayoutProps } from '@ant-design/pro-components';

/**
 * 全局后台布局默认配置
 * 统一对齐企业级后台设计：左侧全高深色侧栏（216px）、顶部56px紧凑浅色顶栏
 */
const Settings: ProLayoutProps & {
  logo?: string;
} = {
  navTheme: 'light',
  colorPrimary: '#1677ff',
  layout: 'mix',
  contentWidth: 'Fluid',
  fixedHeader: true,
  fixSiderbar: true,
  colorWeak: false,
  title: 'Roncin 货代后台',
  logo: '/logo.svg',
  iconfontUrl: '',
  siderWidth: 216,
  splitMenus: false,
  token: {
    sider: {
      colorBgCollapsedButton: '#ffffff',
      colorTextCollapsedButton: 'rgba(0, 0, 0, 0.45)',
      colorTextCollapsedButtonHover: '#1677ff',
      colorBgMenuItemCollapsedElevated: '#ffffff',
      colorBgMenuItemHover: 'rgba(0, 0, 0, 0.04)',
      colorBgMenuItemSelected: '#e6f4ff',
      colorTextMenu: 'rgba(0, 0, 0, 0.65)',
      colorTextMenuSelected: '#1677ff',
      colorTextMenuItemHover: 'rgba(0, 0, 0, 0.88)',
      colorTextMenuTitle: 'rgba(0, 0, 0, 0.88)',
      colorMenuBackground: '#ffffff',
    },
    header: {
      colorBgHeader: '#ffffff',
      colorHeaderTitle: 'rgba(0, 0, 0, 0.88)',
      colorTextMenu: 'rgba(0, 0, 0, 0.65)',
      colorBgMenuItemHover: 'rgba(0, 0, 0, 0.04)',
      colorTextMenuSelected: '#1677ff',
      heightLayoutHeader: 56,
    },
    pageContainer: {
      paddingBlockPageContainerContent: 8,
      paddingInlinePageContainerContent: 12,
    },
  },
};

export default Settings;
