import type { ProLayoutProps } from '@ant-design/pro-components';

/**
 * 全局后台布局默认配置
 * 统一对齐企业级后台设计：左侧全高深色侧栏（216px）、顶部56px紧凑浅色顶栏
 */
const Settings: ProLayoutProps & {
  logo?: string;
} = {
  navTheme: 'realDark',
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
      colorBgCollapsedButton: '#1e293b',
      colorTextCollapsedButton: '#94a3b8',
      colorTextCollapsedButtonHover: '#f8fafc',
      colorBgMenuItemCollapsedElevated: '#0f172a',
      colorBgMenuItemHover: 'rgba(255, 255, 255, 0.06)',
      colorBgMenuItemSelected: 'rgba(59, 130, 246, 0.16)',
      colorTextMenu: '#94a3b8',
      colorTextMenuSelected: '#60a5fa',
      colorTextMenuItemHover: '#f8fafc',
      colorTextMenuTitle: '#f8fafc',
      colorMenuBackground: '#0f172a',
    },
    header: {
      colorBgHeader: '#ffffff',
      colorHeaderTitle: '#0f172a',
      colorTextMenu: '#64748b',
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
